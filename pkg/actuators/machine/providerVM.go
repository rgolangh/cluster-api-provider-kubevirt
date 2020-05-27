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

// providerVM runs the logic to reconciles a machine resource towards its desired state
type providerVM struct {
	*machineScope
}

func newProviderVM(scope *machineScope) *providerVM {
	return &providerVM{
		machineScope: scope,
	}
}

// create creates machine if it does not exists.
func (p *providerVM) create() error {
	klog.Infof("%s: creating machine", p.machine.GetName())

	if validateMachineErr := validateMachine(*p.machine); validateMachineErr != nil {
		return fmt.Errorf("%v: failed validating machine provider spec: %w", p.machine.GetName(), validateMachineErr)
	}

	vm, createVMErr := p.createVM(p.virtualMachine)
	if createVMErr != nil {
		klog.Errorf("%s: error creating machine: %v", p.machine.GetName(), createVMErr)
		conditionFailed := conditionFailed()
		conditionFailed.Message = createVMErr.Error()
		p.machineScope.setProviderStatus(nil, conditionFailed)
		return fmt.Errorf("failed to create virtual machine: %w", createVMErr)
	}

	klog.Infof("Created Machine %v", p.machine.GetName())

	if setIDErr := p.setProviderID(vm); setIDErr != nil {
		return fmt.Errorf("failed to update machine object with providerID: %w", setIDErr)
	}

	if err := p.setMachineCloudProviderSpecifics(vm); err != nil {
		return fmt.Errorf("failed to set machine cloud provider specifics: %w", err)
	}

	p.machineScope.setProviderStatus(vm, conditionSuccess())

	return p.requeueIfInstancePending(vm)
}

// delete deletes machine
func (p *providerVM) delete() error {
	klog.Infof("%s: deleting machine", p.machine.GetName())

	if validateMachineErr := validateMachine(*p.machine); validateMachineErr != nil {
		return fmt.Errorf("%v: failed validating machine provider spec: %w", p.machine.GetName(), validateMachineErr)
	}

	existingVM, existingVMErr := p.getVM(p.virtualMachine.GetName())
	if existingVMErr != nil {
		klog.Errorf("%s: error getting existing VM: %v", p.machine.GetName(), existingVMErr)
		return existingVMErr
	}

	if existingVM == nil {
		klog.Warningf("%s: VM not found to delete for machine", p.machine.Name)
		return nil
	}

	if deleteVMErr := p.deleteVM(p.virtualMachine.GetName()); deleteVMErr != nil {
		return fmt.Errorf("failed to delete VM: %w", deleteVMErr)
	}

	klog.Infof("Deleted machine %v", p.machine.GetName())

	return nil
}

// update finds a vm and reconciles the machine resource status against it.
func (p *providerVM) update() error {
	klog.Infof("%s: updating machine", p.machine.GetName())

	if validateMachineErr := validateMachine(*p.machine); validateMachineErr != nil {
		return fmt.Errorf("%v: failed validating machine provider spec: %w", p.machine.GetName(), validateMachineErr)
	}

	existingVM, existingVMErr := p.getVM(p.virtualMachine.GetName())
	if existingVMErr != nil {
		klog.Errorf("%s: error getting existing VM: %v", p.machine.GetName(), existingVMErr)
		return existingVMErr
	}

	//TODO Danielle - update ProviderID to lowercase
	if existingVM == nil {
		// validate that updates come in the right order
		// if there is an update that was supposes to be done after that update - return an error
		if p.machine.Spec.ProviderID != nil && *p.machine.Spec.ProviderID != "" && (p.machine.Status.LastUpdated == nil || p.machine.Status.LastUpdated.Add(requeueAfterSeconds*time.Second).After(time.Now())) {
			klog.Infof("%s: Possible eventual-consistency discrepancy; returning an error to requeue", p.machine.Name)
			return &machinecontroller.RequeueAfterError{RequeueAfter: requeueAfterSeconds * time.Second}
		}
		klog.Warningf("%s: attempted to update machine but the VM found", p.machine.Name)
		//TODO Danielle - understand that case
		// Update status to clear out machine details.
		p.machineScope.setProviderStatus(nil, conditionSuccess())
		// This is an unrecoverable error condition.  We should delay to
		// minimize unnecessary API calls.
		return &machinecontroller.RequeueAfterError{RequeueAfter: requeueAfterFatalSeconds * time.Second}
	}

	updatedVm, updateVMErr := p.updateVM(p.virtualMachine)

	if updateVMErr != nil {
		return fmt.Errorf("failed to update VM: %w", updateVMErr)
	}
	if setIDErr := p.setProviderID(updatedVm); setIDErr != nil {
		return fmt.Errorf("failed to update machine object with providerID: %w", setIDErr)
	}

	if err := p.setMachineCloudProviderSpecifics(updatedVm); err != nil {
		return fmt.Errorf("failed to set machine cloud provider specifics: %w", err)
	}

	klog.Infof("Updated machine %s", p.machine.Name)
	p.machineScope.setProviderStatus(updatedVm, conditionSuccess())

	return p.requeueIfInstancePending(updatedVm)
}

// exists returns true if machine exists.
func (p *providerVM) exists() (bool, error) {
	existingVM, existingVMErr := p.getVM(p.virtualMachine.GetName())
	if existingVMErr != nil {
		klog.Errorf("%s: error getting existing VM: %v", p.machine.GetName(), existingVMErr)
		return false, existingVMErr
	}
	if existingVM == nil {
		klog.Infof("%s: VM does not exist", p.machine.GetName())
		return false, nil
	}
	return true, existingVMErr
}

func (p *providerVM) createVM(virtualMachine *kubevirtapiv1.VirtualMachine) (*kubevirtapiv1.VirtualMachine, error) {
	return p.kubevirtClient.CreateVirtualMachine(p.machine.GetNamespace(), virtualMachine)
}

func (p *providerVM) getVM(vmName string) (*kubevirtapiv1.VirtualMachine, error) {
	return p.kubevirtClient.GetVirtualMachine(p.machine.GetNamespace(), vmName)
}

func (p *providerVM) deleteVM(vmName string) error {
	gracePeriod := int64(10)
	return p.kubevirtClient.DeleteVirtualMachine(p.machine.GetNamespace(), vmName, &k8smetav1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
}

func (p *providerVM) updateVM(updatedVm *kubevirtapiv1.VirtualMachine) (*kubevirtapiv1.VirtualMachine, error) {
	return p.kubevirtClient.UpdateVirtualMachine(p.machine.GetNamespace(), updatedVm)
}

// isMaster returns true if the machine is part of a cluster's control plane
func (p *providerVM) isMaster() (bool, error) {
	// TODO implement
	// if p.machine.Status.NodeRef == nil {
	// 	klog.Errorf("NodeRef not found in machine %s", p.machine.Name)
	// 	return false, nil
	// }
	// node := &corev1.Node{}
	// nodeKey := types.NamespacedName{
	// 	Namespace: p.machine.Status.NodeRef.Namespace,
	// 	Name:      p.machine.Status.NodeRef.Name,
	// }

	// err := p.client.Get(p.Context, nodeKey, node)
	// if err != nil {
	// 	return false, fmt.Errorf("failed to get node from machine %s", p.machine.Name)
	// }

	// if _, exists := node.Labels[masterLabel]; exists {
	// 	return true, nil
	// }
	return false, nil
}

// // updateLoadBalancers adds a given machine instance to the load balancers specified in its provider config
// func (r *providerVM) updateLoadBalancers(instance *ec2.Instance) error {
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
func (p *providerVM) setProviderID(vm *kubevirtapiv1.VirtualMachine) error {
	// TODO: return an error when the setting is failed
	existingProviderID := p.machine.Spec.ProviderID
	if vm == nil {
		return nil
	}
	providerID := fmt.Sprintf("kubevirt:///%s/%s", p.machine.GetNamespace(), vm.GetName())

	if existingProviderID != nil && *existingProviderID == providerID {
		klog.Infof("%s: ProviderID already set in the machine Spec with value:%s", p.machine.GetName(), *existingProviderID)
		return nil
	}
	p.machine.Spec.ProviderID = &providerID
	klog.Infof("%s: ProviderID set at machine spec: %s", p.machine.GetName(), providerID)
	return nil
}

func (p *providerVM) setMachineCloudProviderSpecifics(vm *kubevirtapiv1.VirtualMachine) error {
	if vm == nil {
		return nil
	}

	if p.machine.Labels == nil {
		p.machine.Labels = make(map[string]string)
	}

	if p.machine.Spec.Labels == nil {
		p.machine.Spec.Labels = make(map[string]string)
	}

	if p.machine.Annotations == nil {
		p.machine.Annotations = make(map[string]string)
	}

	// TODO which labels/annotations need to assign here?
	// Reaching to machine provider config since the region is not directly
	// providing by *kubevirtapiv1.VirtualMachine object
	//memory
	//storage
	//cpu
	////labels
	//machineProviderConfig, err := kubevirtproviderv1.ProviderSpecFromRawExtension(p.machine.Spec.ProviderSpec.Value)
	//
	//if err != nil {
	//	return fmt.Errorf("error decoding MachineProviderConfig: %w", err)
	//}
	//
	//p.machine.Labels[machinecontroller.MachineRegionLabelName] = machineProviderConfig.Placement.Region

	// if instance.Placement != nil {
	// 	p.machine.Labels[machinecontroller.MachineAZLabelName] = aws.StringValue(instance.Placement.AvailabilityZone)
	// }

	// if instance.InstanceType != nil {
	// 	p.machine.Labels[machinecontroller.MachineInstanceTypeLabelName] = aws.StringValue(instance.InstanceType)
	// }

	// if instance.State != nil && instance.State.Name != nil {
	// 	p.machine.Annotations[machinecontroller.MachineInstanceStateAnnotationName] = aws.StringValue(instance.State.Name)
	// }

	// if instance.InstanceLifecycle != nil && *instance.InstanceLifecycle == ec2.InstanceLifecycleTypeSpot {
	// 	// Label on the Spec so that it is propogated to the Node
	// 	p.machine.Spec.Labels[machinecontroller.MachineInterruptibleInstanceLabelName] = ""
	// }

	return nil
}

func (p *providerVM) requeueIfInstancePending(vm *kubevirtapiv1.VirtualMachine) error {
	// If machine state is still pending, we will return an error to keep the controllers
	// attempting to update status until it hits a more permanent state. This will ensure
	// we get a public IP populated more quickly.
	if !vm.Status.Ready {
		klog.Infof("%s: VM status is not ready, returning an error to requeue", p.machine.GetName())
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
func (p *providerVM) getRunningFromVms(vms []*kubevirtapiv1.VirtualMachine) []*kubevirtapiv1.VirtualMachine {
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
func (p *providerVM) getStoppedVms(machine *machinev1.Machine, client kubevirtclient.Client) ([]*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// stoppedInstanceStateFilter := []*string{aws.String(ec2.InstanceStateNameStopped), aws.String(ec2.InstanceStateNameStopping)}
	// return getInstances(machine, client, stoppedInstanceStateFilter)
	return nil, nil
}

// getExistingVms returns all vms not terminated
func (p *providerVM) getExistingVms(machine *machinev1.Machine, client kubevirtclient.Client) ([]*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// return getInstances(machine, client, existingInstanceStates())
	return nil, nil
}

func (p *providerVM) getExistingVMByID(id string, client kubevirtclient.Client) (*kubevirtapiv1.VirtualMachine, error) {
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
func (p *providerVM) getVMByID(id string, client kubevirtclient.Client, instanceStateFilter []*string) (*kubevirtapiv1.VirtualMachine, error) {
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
func (p *providerVM) getVms(machine *machinev1.Machine, client kubevirtclient.Client, vmStateFilter []*string) ([]*kubevirtapiv1.VirtualMachine, error) {
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
func (p *providerVM) terminateVms(client kubevirtclient.Client, vms []*kubevirtapiv1.VirtualMachine) error {
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
