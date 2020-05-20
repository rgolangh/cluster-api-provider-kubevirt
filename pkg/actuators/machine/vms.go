package machine

import (
	kubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtapiv1 "kubevirt.io/client-go/api/v1"
)

func createVM(virtualMachine *kubevirtapiv1.VirtualMachine, underkubeclient kubevirtclient.Client, namespace string) (*kubevirtapiv1.VirtualMachine, error) {
	return underkubeclient.CreateVirtualMachine(namespace, virtualMachine)
}

func getVM(vmName string, underkubeclient kubevirtclient.Client, namespace string) (*kubevirtapiv1.VirtualMachine, error) {
	return underkubeclient.GetVirtualMachine(namespace, vmName)
}

func deleteVM(vmName string, underkubeclient kubevirtclient.Client, namespace string) error {
	gracePeriod := int64(10)
	return underkubeclient.DeleteVirtualMachine(namespace, vmName, &k8smetav1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
}

func updateVM(updatedVm *kubevirtapiv1.VirtualMachine, underkubeclient kubevirtclient.Client, namespace string) (*kubevirtapiv1.VirtualMachine, error) {
	return underkubeclient.UpdateVirtualMachine(namespace, updatedVm)
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
func getRunningFromVms(vms []*kubevirtapiv1.VirtualMachine) []*kubevirtapiv1.VirtualMachine {
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
func getStoppedVms(machine *machinev1.Machine, client kubevirtclient.Client) ([]*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// stoppedInstanceStateFilter := []*string{aws.String(ec2.InstanceStateNameStopped), aws.String(ec2.InstanceStateNameStopping)}
	// return getInstances(machine, client, stoppedInstanceStateFilter)
	return nil, nil
}

// getExistingVms returns all vms not terminated
func getExistingVms(machine *machinev1.Machine, client kubevirtclient.Client) ([]*kubevirtapiv1.VirtualMachine, error) {
	// TODO implement
	// return getInstances(machine, client, existingInstanceStates())
	return nil, nil
}

func getExistingVMByID(id string, client kubevirtclient.Client) (*kubevirtapiv1.VirtualMachine, error) {
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
func getVMByID(id string, client kubevirtclient.Client, instanceStateFilter []*string) (*kubevirtapiv1.VirtualMachine, error) {
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
func getVms(machine *machinev1.Machine, client kubevirtclient.Client, vmStateFilter []*string) ([]*kubevirtapiv1.VirtualMachine, error) {
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
func terminateVms(client kubevirtclient.Client, vms []*kubevirtapiv1.VirtualMachine) error {
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
