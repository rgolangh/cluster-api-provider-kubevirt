package machine

import (
	"fmt"
	"time"

	kubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"

	machinecontroller "github.com/openshift/machine-api-operator/pkg/controller/machine"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	kubevirtapiv1 "kubevirt.io/client-go/api/v1"
)

const (
	requeueAfterSeconds      = 20
	requeueAfterFatalSeconds = 180
	masterLabel              = "node-role.kubernetes.io/master"
)

// Reconciler runs the logic to reconciles a machine resource towards its desired state
type Reconciler struct {
	*machineScope
}

func newReconciler(scope *machineScope) *Reconciler {
	return &Reconciler{
		machineScope: scope,
	}
}

// create creates machine if it does not exists.
func (r *Reconciler) create() error {
	klog.Infof("%s: creating machine", r.machine.GetName())

	if validateMachineErr := validateMachine(*r.machine); validateMachineErr != nil {
		return fmt.Errorf("%v: failed validating machine provider spec: %w", r.machine.GetName(), validateMachineErr)
	}

	vm, createVMErr := r.createVM(r.virtualMachine)
	if createVMErr != nil {
		klog.Errorf("%s: error creating machine: %v", r.machine.GetName(), createVMErr)
		conditionFailed := conditionFailed()
		conditionFailed.Message = createVMErr.Error()
		r.machineScope.setProviderStatus(nil, conditionFailed)
		return fmt.Errorf("failed to create virtual machine: %w", createVMErr)
	}

	klog.Infof("Created Machine %v", r.machine.GetName())

	if setIDErr := r.setProviderID(vm); setIDErr != nil {
		return fmt.Errorf("failed to update machine object with providerID: %w", setIDErr)
	}

	if err := r.setMachineCloudProviderSpecifics(vm); err != nil {
		return fmt.Errorf("failed to set machine cloud provider specifics: %w", err)
	}

	r.machineScope.setProviderStatus(vm, conditionSuccess())

	return r.requeueIfInstancePending(vm)
}

// delete deletes machine
func (r *Reconciler) delete() error {
	klog.Infof("%s: deleting machine", r.machine.GetName())

	if validateMachineErr := validateMachine(*r.machine); validateMachineErr != nil {
		return fmt.Errorf("%v: failed validating machine provider spec: %w", r.machine.GetName(), validateMachineErr)
	}

	existingVM, err := r.getVM(r.virtualMachine.GetName())
	if err != nil {
		klog.Errorf("%s: error getting existing VM: %v", r.machine.GetName(), err)
		return err
	}

	if existingVM == nil {
		klog.Warningf("%s: VM not found to delete for machine", r.machine.Name)
		return nil
	}

	if err := r.deleteVM(r.virtualMachine.GetName()); err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	klog.Infof("Deleted machine %v", r.machine.GetName())

	return nil
}

// update finds a vm and reconciles the machine resource status against it.
func (r *Reconciler) update() error {
	klog.Infof("%s: updating machine", r.machine.GetName())

	if validateMachineErr := validateMachine(*r.machine); validateMachineErr != nil {
		return fmt.Errorf("%v: failed validating machine provider spec: %w", r.machine.GetName(), validateMachineErr)
	}

	existingVM, err := r.getVM(r.virtualMachine.GetName())
	if err != nil {
		klog.Errorf("%s: error getting existing VM: %v", r.machine.GetName(), err)
		return err
	}

	//TODO Danielle - update ProviderID to lowercase
	if existingVM == nil {
		// validate that updates come in the right order
		// if there is an update that was supposes to be done after that update - return an error
		if r.machine.Spec.ProviderID != nil && *r.machine.Spec.ProviderID != "" && (r.machine.Status.LastUpdated == nil || r.machine.Status.LastUpdated.Add(requeueAfterSeconds*time.Second).After(time.Now())) {
			klog.Infof("%s: Possible eventual-consistency discrepancy; returning an error to requeue", r.machine.Name)
			return &machinecontroller.RequeueAfterError{RequeueAfter: requeueAfterSeconds * time.Second}
		}
		klog.Warningf("%s: attempted to update machine but the VM found", r.machine.Name)
		//TODO Danielle - understand that case
		// Update status to clear out machine details.
		r.machineScope.setProviderStatus(nil, conditionSuccess())
		// This is an unrecoverable error condition.  We should delay to
		// minimize unnecessary API calls.
		return &machinecontroller.RequeueAfterError{RequeueAfter: requeueAfterFatalSeconds * time.Second}
	}

	updatedVm, updateVMErr := r.updateVM(r.virtualMachine)

	if updateVMErr != nil {
		return fmt.Errorf("failed to update VM : %w", err)
	}
	if setIDErr := r.setProviderID(updatedVm); setIDErr != nil {
		return fmt.Errorf("failed to update machine object with providerID: %w", setIDErr)
	}

	if err := r.setMachineCloudProviderSpecifics(updatedVm); err != nil {
		return fmt.Errorf("failed to set machine cloud provider specifics: %w", err)
	}

	klog.Infof("Updated machine %s", r.machine.Name)
	r.machineScope.setProviderStatus(updatedVm, conditionSuccess())

	return r.requeueIfInstancePending(updatedVm)
}

// exists returns true if machine exists.
func (r *Reconciler) exists() (bool, error) {
	existingVM, err := r.getVM(r.virtualMachine.GetName())
	if err != nil || existingVM == nil {
		klog.Errorf("%s: error getting existing VM: %v", r.machine.GetName(), err)
	}
	return true, err
}

func (r *Reconciler) createVM(virtualMachine *kubevirtapiv1.VirtualMachine) (*kubevirtapiv1.VirtualMachine, error) {
	return r.kubevirtClient.CreateVirtualMachine(r.machine.GetNamespace(), virtualMachine)
}

func (r *Reconciler) getVM(vmName string) (*kubevirtapiv1.VirtualMachine, error) {
	return r.kubevirtClient.GetVirtualMachine(r.machine.GetNamespace(), vmName)
}

func (r *Reconciler) deleteVM(vmName string) error {
	gracePeriod := int64(10)
	return r.kubevirtClient.DeleteVirtualMachine(r.machine.GetNamespace(), vmName, &k8smetav1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
}

func (r *Reconciler) updateVM(updatedVm *kubevirtapiv1.VirtualMachine) (*kubevirtapiv1.VirtualMachine, error) {
	return r.kubevirtClient.UpdateVirtualMachine(r.machine.GetNamespace(), updatedVm)
}

// isMaster returns true if the machine is part of a cluster's control plane
func (r *Reconciler) isMaster() (bool, error) {
	// TODO implement
	// if r.machine.Status.NodeRef == nil {
	// 	klog.Errorf("NodeRef not found in machine %s", r.machine.Name)
	// 	return false, nil
	// }
	// node := &corev1.Node{}
	// nodeKey := types.NamespacedName{
	// 	Namespace: r.machine.Status.NodeRef.Namespace,
	// 	Name:      r.machine.Status.NodeRef.Name,
	// }

	// err := r.client.Get(r.Context, nodeKey, node)
	// if err != nil {
	// 	return false, fmt.Errorf("failed to get node from machine %s", r.machine.Name)
	// }

	// if _, exists := node.Labels[masterLabel]; exists {
	// 	return true, nil
	// }
	return false, nil
}

// // updateLoadBalancers adds a given machine instance to the load balancers specified in its provider config
// func (r *Reconciler) updateLoadBalancers(instance *ec2.Instance) error {
// 	if len(r.providerSpec.LoadBalancers) == 0 {
// 		klog.V(4).Infof("%s: Instance %q has no load balancers configured. Skipping", r.machine.Name, *instance.InstanceId)
// 		return nil
// 	}
// 	errs := []error{}
// 	classicLoadBalancerNames := []string{}
// 	networkLoadBalancerNames := []string{}
// 	for _, loadBalancerRef := range r.providerSpec.LoadBalancers {
// 		switch loadBalancerRef.Type {
// 		case kubevirtproviderv1.NetworkLoadBalancerType:
// 			networkLoadBalancerNames = append(networkLoadBalancerNames, loadBalancerRef.Name)
// 		case kubevirtproviderv1.ClassicLoadBalancerType:
// 			classicLoadBalancerNames = append(classicLoadBalancerNames, loadBalancerRef.Name)
// 		}
// 	}

// 	var err error
// 	if len(classicLoadBalancerNames) > 0 {
// 		err := registerWithClassicLoadBalancers(r.awsClient, classicLoadBalancerNames, instance)
// 		if err != nil {
// 			klog.Errorf("%s: Failed to register classic load balancers: %v", r.machine.Name, err)
// 			errs = append(errs, err)
// 		}
// 	}
// 	if len(networkLoadBalancerNames) > 0 {
// 		err = registerWithNetworkLoadBalancers(r.awsClient, networkLoadBalancerNames, instance)
// 		if err != nil {
// 			klog.Errorf("%s: Failed to register network load balancers: %v", r.machine.Name, err)
// 			errs = append(errs, err)
// 		}
// 	}
// 	if len(errs) > 0 {
// 		return errorutil.NewAggregate(errs)
// 	}
// 	return nil
// }

// setProviderID adds providerID in the machine spec
func (r *Reconciler) setProviderID(vm *kubevirtapiv1.VirtualMachine) error {
	existingProviderID := r.machine.Spec.ProviderID
	if vm == nil {
		return nil
	}
	// TODO what is the right providerID structure?
	providerID := fmt.Sprintf("kubevirt:///%s/%s", r.machine.GetNamespace(), vm.GetName())

	if existingProviderID != nil && *existingProviderID == providerID {
		klog.Infof("%s: ProviderID already set in the machine Spec with value:%s", r.machine.GetName(), *existingProviderID)
		return nil
	}
	r.machine.Spec.ProviderID = &providerID
	klog.Infof("%s: ProviderID set at machine spec: %s", r.machine.GetName(), providerID)
	return nil
}

func (r *Reconciler) setMachineCloudProviderSpecifics(vm *kubevirtapiv1.VirtualMachine) error {
	if vm == nil {
		return nil
	}

	if r.machine.Labels == nil {
		r.machine.Labels = make(map[string]string)
	}

	if r.machine.Spec.Labels == nil {
		r.machine.Spec.Labels = make(map[string]string)
	}

	if r.machine.Annotations == nil {
		r.machine.Annotations = make(map[string]string)
	}

	// TODO which labels/annotations need to assign here?
	// Reaching to machine provider config since the region is not directly
	// providing by *kubevirtapiv1.VirtualMachine object
	// machineProviderConfig, err := kubevirtproviderv1.ProviderSpecFromRawExtension(r.machine.Spec.ProviderSpec.Value)
	// if err != nil {
	// 	return fmt.Errorf("error decoding MachineProviderConfig: %w", err)
	// }

	// r.machine.Labels[machinecontroller.MachineRegionLabelName] = machineProviderConfig.Placement.Region

	// if instance.Placement != nil {
	// 	r.machine.Labels[machinecontroller.MachineAZLabelName] = aws.StringValue(instance.Placement.AvailabilityZone)
	// }

	// if instance.InstanceType != nil {
	// 	r.machine.Labels[machinecontroller.MachineInstanceTypeLabelName] = aws.StringValue(instance.InstanceType)
	// }

	// if instance.State != nil && instance.State.Name != nil {
	// 	r.machine.Annotations[machinecontroller.MachineInstanceStateAnnotationName] = aws.StringValue(instance.State.Name)
	// }

	// if instance.InstanceLifecycle != nil && *instance.InstanceLifecycle == ec2.InstanceLifecycleTypeSpot {
	// 	// Label on the Spec so that it is propogated to the Node
	// 	r.machine.Spec.Labels[machinecontroller.MachineInterruptibleInstanceLabelName] = ""
	// }

	return nil
}

func (r *Reconciler) requeueIfInstancePending(vm *kubevirtapiv1.VirtualMachine) error {
	// If machine state is still pending, we will return an error to keep the controllers
	// attempting to update status until it hits a more permanent state. This will ensure
	// we get a public IP populated more quickly.
	if !vm.Status.Ready {
		klog.Infof("%s: VM status is not ready, returning an error to requeue", r.machine.GetName())
		return &machinecontroller.RequeueAfterError{RequeueAfter: requeueAfterSeconds * time.Second}
	}

	return nil
}

//
//handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=" + nodeName)
//labelSelector, err := labels.Parse(virtv1.AppLabel + " in (virt-handler)")
//k8smetav1.ListOptions{
//	FieldSelector: handlerNodeSelector.String(),
//	LabelSelector: labelSelector.String()}
//func listVMs(underkubeclient kubevirtclient.Client, namespace string) (*kubevirtapiv1.VirtualMachineList, error) {
//	//filter the vms with a specific tag '{}'
//	//machine.Labels map[string]string
//	labelSelector, _ := labels.Parse(kubevirtapiv1.AppLabel + " in (virt-handler)")
//
//	aa := k8smetav1.ListOptions{
//		LabelSelector: labelSelector.String(),
//	}
//
//	return underkubeclient.ListVirtualMachine(namespace, &aa)
//}

// getRunningFromVms returns all running vms from a list of vms.
func (r *Reconciler) getRunningFromVms(vms []*kubevirtapiv1.VirtualMachine) []*kubevirtapiv1.VirtualMachine {
	var runningVms []*kubevirtapiv1.VirtualMachine
	for _, vm := range vms {
		if vm.Status.Ready {
			runningVms = append(runningVms, vm)
		}
	}
	return runningVms
}

// getStoppedVms returns all stopped vms that have a tag matching our machine name,
// and cluster ID.
func (r *Reconciler) getStoppedVms(machine *machinev1.Machine, client kubevirtclient.Client) ([]*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// stoppedInstanceStateFilter := []*string{aws.String(ec2.InstanceStateNameStopped), aws.String(ec2.InstanceStateNameStopping)}
	// return getInstances(machine, client, stoppedInstanceStateFilter)
	return nil, nil
}

// getExistingVms returns all vms not terminated
func (r *Reconciler) getExistingVms(machine *machinev1.Machine, client kubevirtclient.Client) ([]*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// return getInstances(machine, client, existingInstanceStates())
	return nil, nil
}

func (r *Reconciler) getExistingVMByID(id string, client kubevirtclient.Client) (*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// return getInstanceByID(id, client, existingInstanceStates())
	return nil, nil
}

// func instanceHasAllowedState(instance *ec2.Instance, instanceStateFilter []*string) error {
// 	if instance.InstanceId == nil {
// 		return fmt.Errorf("instance has nil ID")
// 	}

// 	if instance.State == nil {
// 		return fmt.Errorf("instance %s has nil state", *instance.InstanceId)
// 	}

// 	if len(instanceStateFilter) == 0 {
// 		return nil
// 	}

// 	actualState := aws.StringValue(instance.State.Name)
// 	for _, allowedState := range instanceStateFilter {
// 		if aws.StringValue(allowedState) == actualState {
// 			return nil
// 		}
// 	}

// 	allowedStates := make([]string, 0, len(instanceStateFilter))
// 	for _, allowedState := range instanceStateFilter {
// 		allowedStates = append(allowedStates, aws.StringValue(allowedState))
// 	}
// 	return fmt.Errorf("instance %s state %q is not in %s", *instance.InstanceId, actualState, strings.Join(allowedStates, ", "))
// }

// getVmByID returns the vm with the given ID if it exists.
func (r *Reconciler) getVMByID(id string, client kubevirtclient.Client, instanceStateFilter []*string) (*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// if id == "" {
	// 	return nil, fmt.Errorf("instance-id not specified")
	// }

	// request := &ec2.DescribeInstancesInput{
	// 	InstanceIds: aws.StringSlice([]string{id}),
	// }

	// result, err := client.DescribeInstances(request)
	// if err != nil {
	// 	return nil, err
	// }

	// if len(result.Reservations) != 1 {
	// 	return nil, fmt.Errorf("found %d reservations for instance-id %s", len(result.Reservations), id)
	// }

	// reservation := result.Reservations[0]

	// if len(reservation.Instances) != 1 {
	// 	return nil, fmt.Errorf("found %d instances for instance-id %s", len(reservation.Instances), id)
	// }

	// instance := reservation.Instances[0]

	// return instance, instanceHasAllowedState(instance, instanceStateFilter)
	return nil, nil
}

// getVms returns all vms that have a tag matching our machine name,
// and cluster ID.
func (r *Reconciler) getVms(machine *machinev1.Machine, client kubevirtclient.Client, vmStateFilter []*string) ([]*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// clusterID, ok := getClusterID(machine)
	// if !ok {
	// 	return []*ec2.Instance{}, fmt.Errorf("unable to get cluster ID for machine: %q", machine.Name)
	// }

	// requestFilters := []*ec2.Filter{
	// 	{
	// 		Name:   awsTagFilter("Name"),
	// 		Values: aws.StringSlice([]string{machine.Name}),
	// 	},
	// 	clusterFilter(clusterID),
	// }

	// request := &ec2.DescribeInstancesInput{
	// 	Filters: requestFilters,
	// }

	// result, err := client.DescribeInstances(request)
	// if err != nil {
	// 	return []*ec2.Instance{}, err
	// }

	// instances := make([]*ec2.Instance, 0, len(result.Reservations))

	// for _, reservation := range result.Reservations {
	// 	for _, instance := range reservation.Instances {
	// 		err := instanceHasAllowedState(instance, instanceStateFilter)
	// 		if err != nil {
	// 			klog.Errorf("Excluding instance matching %s: %v", machine.Name, err)
	// 		} else {
	// 			instances = append(instances, instance)
	// 		}
	// 	}
	// }

	// return instances, nil
	return nil, nil
}

// terminateVms terminates all provided vms with a single virtctl request.
func (r *Reconciler) terminateVms(client kubevirtclient.Client, vms []*kubevirtapiv1.VirtualMachine) error {
	// TODO implement: should return similar to aws? []*kubevirtapiv1.VirtualMachineState
	// instanceIDs := []*string{}
	// // Cleanup all older instances:
	// for _, instance := range instances {
	// 	klog.Infof("Cleaning up extraneous instance for machine: %v, state: %v, launchTime: %v", *instance.InstanceId, *instance.State.Name, *instance.LaunchTime)
	// 	instanceIDs = append(instanceIDs, instance.InstanceId)
	// }
	// for _, instanceID := range instanceIDs {
	// 	klog.Infof("Terminating %v instance", *instanceID)
	// }

	// terminateInstancesRequest := &ec2.TerminateInstancesInput{
	// 	InstanceIds: instanceIDs,
	// }
	// output, err := client.TerminateInstances(terminateInstancesRequest)
	// if err != nil {
	// 	klog.Errorf("Error terminating instances: %v", err)
	// 	return nil, fmt.Errorf("error terminating instances: %v", err)
	// }

	// if output == nil {
	// 	return nil, nil
	// }

	// return output.TerminatingInstances, nil
	return nil
}
