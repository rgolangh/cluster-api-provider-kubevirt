package machine

import (
	"fmt"
	"time"

	machinecontroller "github.com/openshift/machine-api-operator/pkg/controller/machine"
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

	//instance, err := launchInstance(r.machine, r.providerSpec, userData, r.awsClient)
	namespace := r.machine.GetNamespace()
	vm, createVMErr := createVM(r.virtualMachine, r.kubevirtClient, namespace)
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

	// TODO implement
	// // Get all instances not terminated.
	// existingInstances, err := r.getMachineInstances()
	// if err != nil {
	// 	klog.Errorf("%s: error getting existing instances: %v", r.machine.Name, err)
	// 	return err
	// }

	// existingLen := len(existingInstances)
	// klog.Infof("%s: found %d existing instances for machine", r.machine.Name, existingLen)
	// if existingLen == 0 {
	// 	klog.Warningf("%s: no instances found to delete for machine", r.machine.Name)
	// 	return nil
	// }

	// terminatingInstances, err := terminateInstances(r.kubevirtClient, existingInstances)
	// if err != nil {
	// 	return fmt.Errorf("failed to delete instaces: %w", err)
	// }

	// if len(terminatingInstances) == 1 {
	// 	if terminatingInstances[0] != nil && terminatingInstances[0].CurrentState != nil && terminatingInstances[0].CurrentState.Name != nil {
	// 		r.machine.Annotations[machinecontroller.MachineInstanceStateAnnotationName] = aws.StringValue(terminatingInstances[0].CurrentState.Name)
	// 	}
	// }

	// klog.Infof("Deleted machine %v", r.machine.Name)

	return nil
}

// update finds a vm and reconciles the machine resource status against it.
func (r *Reconciler) update() error {
	klog.Infof("%s: updating machine", r.machine.GetName())

	// TODO implement
	// if err := validateMachine(*r.machine); err != nil {
	// 	return fmt.Errorf("%v: failed validating machine provider spec: %v", r.machine.GetName(), err)
	// }

	// // Get all instances not terminated.
	// existingInstances, err := r.getMachineInstances()
	// if err != nil {
	// 	klog.Errorf("%s: error getting existing instances: %v", r.machine.Name, err)
	// 	return err
	// }

	// existingLen := len(existingInstances)
	// if existingLen == 0 {
	// 	if r.machine.Spec.ProviderID != nil && *r.machine.Spec.ProviderID != "" && (r.machine.Status.LastUpdated == nil || r.machine.Status.LastUpdated.Add(requeueAfterSeconds*time.Second).After(time.Now())) {
	// 		klog.Infof("%s: Possible eventual-consistency discrepancy; returning an error to requeue", r.machine.Name)
	// 		return &machinecontroller.RequeueAfterError{RequeueAfter: requeueAfterSeconds * time.Second}
	// 	}

	// 	klog.Warningf("%s: attempted to update machine but no instances found", r.machine.Name)

	// 	// Update status to clear out machine details.
	// 	r.machineScope.setProviderStatus(nil, conditionSuccess())
	// 	// This is an unrecoverable error condition.  We should delay to
	// 	// minimize unnecessary API calls.
	// 	return &machinecontroller.RequeueAfterError{RequeueAfter: requeueAfterFatalSeconds * time.Second}
	// }

	// sortInstances(existingInstances)
	// runningInstances := getRunningFromInstances(existingInstances)
	// runningLen := len(runningInstances)
	// var newestInstance *ec2.Instance

	// if runningLen > 0 {
	// 	// It would be very unusual to have more than one here, but it is
	// 	// possible if someone manually provisions a machine with same tag name.
	// 	klog.Infof("%s: found %d running instances for machine", r.machine.Name, runningLen)
	// 	newestInstance = runningInstances[0]

	// 	err = r.updateLoadBalancers(newestInstance)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to updated update load balancers: %w", err)
	// 	}
	// } else {
	// 	// Didn't find any running instances, just newest existing one.
	// 	// In most cases, there should only be one existing Instance.
	// 	newestInstance = existingInstances[0]
	// }

	// if err = r.setProviderID(newestInstance); err != nil {
	// 	return fmt.Errorf("failed to update machine object with providerID: %w", err)
	// }

	// if err = r.setMachineCloudProviderSpecifics(newestInstance); err != nil {
	// 	return fmt.Errorf("failed to set machine cloud provider specifics: %w", err)
	// }

	// klog.Infof("Updated machine %s", r.machine.Name)

	// r.machineScope.setProviderStatus(newestInstance, conditionSuccess())

	// return r.requeueIfInstancePending(newestInstance)
	return nil
}

// exists returns true if machine exists.
func (r *Reconciler) exists() (bool, error) {
	namespace := r.machine.GetNamespace()
	existingVM, err := vmExists(r.machine.GetName(), r.kubevirtClient, namespace)
	// OR
	//existingVm, err := vmExists(r.virtualMachine.Name, r.kubevirtClient, namespace)
	if err != nil || existingVM == nil {
		klog.Errorf("%s: error getting existing vms: %v", r.machine.GetName(), err)
	}
	return true, err
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
	providerID := fmt.Sprintf("kubevirt:///%s", string(vm.UID))

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
