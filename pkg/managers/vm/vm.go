package vm

import (
	"fmt"
	"strings"
	"time"

	kubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/clients/kubevirt"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"

	kubernetesclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/clients/kubernetes"
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

// ProviderVM runs the logic to reconciles a machine resource towards its desired state
type ProviderVM interface {
	Create(machine *machinev1.Machine) error
	Delete(machine *machinev1.Machine) error
	Update(machine *machinev1.Machine) (bool, error)
	Exists(machine *machinev1.Machine) (bool, error)
}

// manager is the struct which implement ProviderVM interface
// Use kubernetesClient to access secret params assigned by user
// Use kubevirtClientBuilder to create the UnderKube kubevirtClient used
type manager struct {
	kubevirtClientBuilder kubevirtclient.ClientBuilderFuncType
	kubernetesClient      kubernetesclient.Client
}

// New creates provider vm instance
func New(kubevirtClientBuilder kubevirtclient.ClientBuilderFuncType, kubernetesClient kubernetesclient.Client) ProviderVM {
	return &manager{
		kubernetesClient:      kubernetesClient,
		kubevirtClientBuilder: kubevirtClientBuilder,
	}
}

// Create creates machine if it does not exists.
func (m *manager) Create(machine *machinev1.Machine) (resultErr error) {
	machineScope, err := newMachineScope(machine, m.kubernetesClient, m.kubevirtClientBuilder)
	if err != nil {
		return err
	}

	virtualMachineFromMachine, err := machineScope.machineToVirtualMachine()
	if err != nil {
		return err
	}

	klog.Infof("%s: create machine", machineScope.getMachineName())

	defer func() {
		// After the operation is done (success or failure)
		// Update the machine object with the relevant changes
		if err := machineScope.patchMachine(); err != nil {
			resultErr = err
		}
	}()

	createdVM, err := m.createVM(virtualMachineFromMachine, machineScope)
	if err != nil {
		klog.Errorf("%s: error creating machine: %v", machineScope.getMachineName(), err)
		conditionFailed := conditionFailed()
		conditionFailed.Message = err.Error()
		return fmt.Errorf("failed to create virtual machine: %w", err)
	}

	klog.Infof("Created Machine %v", machineScope.getMachineName())

	if err := m.syncMachine(createdVM, machineScope); err != nil {
		klog.Errorf("%s: fail syncing machine from vm: %v", machineScope.getMachineName(), err)
		return err
	}

	return m.requeueIfInstancePending(createdVM, machineScope.getMachineName())
}

// delete deletes machine
func (m *manager) Delete(machine *machinev1.Machine) error {
	machineScope, err := newMachineScope(machine, m.kubernetesClient, m.kubevirtClientBuilder)
	if err != nil {
		return err
	}

	virtualMachineFromMachine, err := machineScope.machineToVirtualMachine()
	if err != nil {
		return err
	}

	klog.Infof("%s: delete machine", machineScope.getMachineName())

	existingVM, err := m.getVM(virtualMachineFromMachine.GetName(), virtualMachineFromMachine.GetNamespace(), machineScope)
	if err != nil {
		// TODO ask Nir how to check it
		if strings.Contains(err.Error(), "not found") {
			klog.Infof("%s: VM does not exist", machineScope.getMachineName())
			return nil
		}

		klog.Errorf("%s: error getting existing VM: %v", machineScope.getMachineName(), err)
		return err
	}

	if existingVM == nil {
		klog.Warningf("%s: VM not found to delete for machine", machineScope.getMachineName())
		return nil
	}

	if err := m.deleteVM(existingVM.GetName(), existingVM.GetNamespace(), machineScope); err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	klog.Infof("Deleted machine %v", machineScope.getMachineName())

	return nil
}

// update finds a vm and reconciles the machine resource status against it.
func (m *manager) Update(machine *machinev1.Machine) (wasUpdated bool, resultErr error) {
	machineScope, err := newMachineScope(machine, m.kubernetesClient, m.kubevirtClientBuilder)
	if err != nil {
		return false, err
	}

	virtualMachineFromMachine, err := machineScope.machineToVirtualMachine()
	if err != nil {
		return false, err
	}

	klog.Infof("%s: update machine", machineScope.getMachineName())

	defer func() {
		// After the operation is done (success or failure)
		// Update the machine object with the relevant changes
		if err := machineScope.patchMachine(); err != nil {
			resultErr = err
		}
	}()

	existingVM, err := m.getVM(virtualMachineFromMachine.GetName(), virtualMachineFromMachine.GetNamespace(), machineScope)
	if err != nil {
		klog.Errorf("%s: error getting existing VM: %v", machineScope.getMachineName(), err)
		return false, err
	}
	//TODO Danielle - update ProviderID to lowercase
	if existingVM == nil {
		if machineScope.updateAllowed() {
			klog.Infof("%s: Possible eventual-consistency discrepancy; returning an error to requeue", machineScope.getMachineName())
			return false, &machinecontroller.RequeueAfterError{RequeueAfter: requeueAfterSeconds * time.Second}
		}
		klog.Warningf("%s: attempted to update machine but the VM found", machineScope.getMachineName())

		// This is an unrecoverable error condition.  We should delay to
		// minimize unnecessary API calls.
		return false, &machinecontroller.RequeueAfterError{RequeueAfter: requeueAfterFatalSeconds * time.Second}
	}

	previousResourceVersion := existingVM.ResourceVersion
	virtualMachineFromMachine.ObjectMeta.ResourceVersion = previousResourceVersion

	updatedVM, err := m.updateVM(virtualMachineFromMachine, machineScope)
	if err != nil {
		return false, fmt.Errorf("failed to update VM: %w", err)
	}
	currentResourceVersion := updatedVM.ResourceVersion

	klog.Infof("Updated machine %s", machineScope.getMachineName())

	//Get an updatedVM with more details
	getUpdatedVM, err := m.getVM(updatedVM.GetName(), updatedVM.GetNamespace(), machineScope)
	if err != nil {
		klog.Errorf("%s: error getting updated VM: %v", machineScope.getMachineName(), err)
		getUpdatedVM = updatedVM
	}

	if err := m.syncMachine(getUpdatedVM, machineScope); err != nil {
		klog.Errorf("%s: fail syncing machine from vm: %v", machineScope.getMachineName(), err)
		return false, err
	}
	wasUpdated = previousResourceVersion != currentResourceVersion
	return wasUpdated, m.requeueIfInstancePending(getUpdatedVM, machineScope.getMachineName())
}

func (m *manager) syncMachine(vm *kubevirtapiv1.VirtualMachine, machineScope *machineScope) error {
	vmi, err := m.getVMI(vm.Name, vm.Namespace, machineScope)
	if err != nil {
		klog.Errorf("%s: error getting vmi for machine: %v", machineScope.getMachineName(), err)
	}
	if err := machineScope.SyncMachineFromVm(vm, vmi); err != nil {
		klog.Errorf("%s: fail syncing machine from vm: %v", machineScope.getMachineName(), err)
		return err
	}
	return nil
}

// exists returns true if machine exists.
func (m *manager) Exists(machine *machinev1.Machine) (bool, error) {
	machineScope, err := newMachineScope(machine, m.kubernetesClient, m.kubevirtClientBuilder)
	if err != nil {
		return false, err
	}

	klog.Infof("%s: check if machine exists", machineScope.getMachineName())
	existingVM, err := m.getVM(machine.GetName(), getVMNamespace(machine), machineScope)
	if err != nil {
		// TODO ask Nir how to check it
		if strings.Contains(err.Error(), "not found") {
			klog.Infof("%s: VM does not exist", machineScope.getMachineName())
			return false, nil
		}
		klog.Errorf("%s: error getting existing VM: %v", machineScope.getMachineName(), err)
		return false, err
	}

	if existingVM == nil {
		klog.Infof("%s: VM does not exist", machineScope.getMachineName())
		return false, nil
	}

	return true, nil
}

func (m *manager) createVM(virtualMachine *kubevirtapiv1.VirtualMachine, machineScope *machineScope) (*kubevirtapiv1.VirtualMachine, error) {
	return machineScope.kubevirtClient.CreateVirtualMachine(virtualMachine.Namespace, virtualMachine)
}

func (m *manager) getVM(vmName, vmNamespace string, machineScope *machineScope) (*kubevirtapiv1.VirtualMachine, error) {
	return machineScope.kubevirtClient.GetVirtualMachine(vmNamespace, vmName, &k8smetav1.GetOptions{})
}
func (m *manager) getVMI(vmName, vmNamespace string, machineScope *machineScope) (*kubevirtapiv1.VirtualMachineInstance, error) {
	return machineScope.kubevirtClient.GetVirtualMachineInstance(vmNamespace, vmName, &k8smetav1.GetOptions{})
}

func (m *manager) deleteVM(vmName, vmNamespace string, machineScope *machineScope) error {
	gracePeriod := int64(10)
	return machineScope.kubevirtClient.DeleteVirtualMachine(vmNamespace, vmName, &k8smetav1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
}

func (m *manager) updateVM(updatedVM *kubevirtapiv1.VirtualMachine, machineScope *machineScope) (*kubevirtapiv1.VirtualMachine, error) {
	return machineScope.kubevirtClient.UpdateVirtualMachine(updatedVM.Namespace, updatedVM)
}

// isMaster returns true if the machine is part of a cluster's control plane
func (m *manager) isMaster() (bool, error) {
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

func (m *manager) requeueIfInstancePending(vm *kubevirtapiv1.VirtualMachine, machineName string) error {
	// If machine state is still pending, we will return an error to keep the controllers
	// attempting to update status until it hits a more permanent state. This will ensure
	// we get a public IP populated more quickly.
	if !vm.Status.Ready {
		klog.Infof("%s: VM status is not ready, returning an error to requeue", machineName)
		return &machinecontroller.RequeueAfterError{RequeueAfter: requeueAfterSeconds * time.Second}
	}

	return nil
}

// getRunningFromVms returns all running vms from a list of vms.
func (m *manager) getRunningFromVms(vms []*kubevirtapiv1.VirtualMachine) []*kubevirtapiv1.VirtualMachine {
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
func (m *manager) getStoppedVms(machine *machinev1.Machine, client kubevirtclient.Client) ([]*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// stoppedInstanceStateFilter := []*string{aws.String(ec2.InstanceStateNameStopped), aws.String(ec2.InstanceStateNameStopping)}
	// return getInstances(machine, client, stoppedInstanceStateFilter)
	return nil, nil
}

// getExistingVms returns all vms not terminated
func (m *manager) getExistingVms(machine *machinev1.Machine, client kubevirtclient.Client) ([]*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// return getInstances(machine, client, existingInstanceStates())
	return nil, nil
}

func (m *manager) getExistingVMByID(id string, client kubevirtclient.Client) (*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// return getInstanceByID(id, client, existingInstanceStates())
	return nil, nil
}

// getVmByID returns the vm with the given ID if it exists.
func (m *manager) getVMByID(id string, client kubevirtclient.Client, instanceStateFilter []*string) (*kubevirtapiv1.VirtualMachine, error) {
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
func (m *manager) getVms(machine *machinev1.Machine, client kubevirtclient.Client, vmStateFilter []*string) ([]*kubevirtapiv1.VirtualMachine, error) {
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
func (m *manager) terminateVms(client kubevirtclient.Client, vms []*kubevirtapiv1.VirtualMachine) error {
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
