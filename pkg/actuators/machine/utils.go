/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package machine

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	machinecontroller "github.com/openshift/machine-api-operator/pkg/controller/machine"
	kubevirtapiv1 "kubevirt.io/client-go/api/v1"
)

// upstreamMachineClusterIDLabel is the label that a machine must have to identify the cluster to which it belongs
const upstreamMachineClusterIDLabel = "sigs.k8s.io/cluster-api-cluster"

// existingInstanceStates returns the list of states an EC2 instance can be in
// while being considered "existing", i.e. mostly anything but "Terminated".
// func existingInstanceStates() []*string {
// 	return []*string{
// 		kubevirtapiv1.VirtualMachineInstancePhaseToString(kubevirtapiv1.VmPhaseUnset),
// 		kubevirtapiv1.VirtualMachineInstancePhaseToString(kubevirtapiv1.Pending),
// 		kubevirtapiv1.VirtualMachineInstancePhaseToString(kubevirtapiv1.Scheduling),
// 		kubevirtapiv1.VirtualMachineInstancePhaseToString(kubevirtapiv1.Scheduled),
// 		kubevirtapiv1.VirtualMachineInstancePhaseToString(kubevirtapiv1.Running),
// 		kubevirtapiv1.VirtualMachineInstancePhaseToString(kubevirtapiv1.Succeeded),
// 		kubevirtapiv1.VirtualMachineInstancePhaseToString(kubevirtapiv1.Failed),
// 		kubevirtapiv1.VirtualMachineInstancePhaseToString(kubevirtapiv1.Unknown),
// 	}
// }

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

// setKubevirtMachineProviderCondition sets the condition for the machine and
// returns the new slice of conditions.
// If the machine does not already have a condition with the specified type,
// a condition will be added to the slice
// If the machine does already have a condition with the specified type,
// the condition will be updated if either of the following are true.
func setKubevirtMachineProviderCondition(condition kubevirtapiv1.VirtualMachineCondition, conditions []kubevirtapiv1.VirtualMachineCondition) []kubevirtapiv1.VirtualMachineCondition {
	now := metav1.Now()

	if existingCondition := findProviderCondition(conditions, condition.Type); existingCondition == nil {
		condition.LastProbeTime = now
		condition.LastTransitionTime = now
		conditions = append(conditions, condition)
	} else {
		updateExistingCondition(&condition, existingCondition)
	}

	return conditions
}

func findProviderCondition(conditions []kubevirtapiv1.VirtualMachineCondition, conditionType kubevirtapiv1.VirtualMachineConditionType) *kubevirtapiv1.VirtualMachineCondition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

func updateExistingCondition(newCondition, existingCondition *kubevirtapiv1.VirtualMachineCondition) {
	if !shouldUpdateCondition(newCondition, existingCondition) {
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.LastTransitionTime = metav1.Now()
	}
	existingCondition.Status = newCondition.Status
	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
	existingCondition.LastProbeTime = newCondition.LastProbeTime
}

func shouldUpdateCondition(newCondition, existingCondition *kubevirtapiv1.VirtualMachineCondition) bool {
	return newCondition.Reason != existingCondition.Reason || newCondition.Message != existingCondition.Message
}

// TODO Nir do we need to extract node adresses?
// TODO Networks are located in vmi, need to add
// The network info is saved in the vmi
// extractNodeAddresses maps the instance information from Vmi to an array of NodeAddresses
func extractNodeAddresses(vm *kubevirtapiv1.VirtualMachine) ([]corev1.NodeAddress, error) {
	// Not clear if the order matters here, but we might as well indicate a sensible preference order

	if vm == nil {
		return nil, fmt.Errorf("nil vm passed to extractNodeAddresses")
	}

	addresses := []corev1.NodeAddress{}

	// TODO implement
	// // handle internal network interfaces
	// for _, networkInterface := range vm.Status. {
	// 	// TODO Nir - In kubevirt there is not network state and IPV6
	// 	// // skip network interfaces that are not currently in use
	// 	// if aws.StringValue(networkInterface.Status) != ec2.NetworkInterfaceStatusInUse {
	// 	// 	continue
	// 	// }

	// 	// TODO Does kubevirt use IPV6?
	// 	// // Treating IPv6 addresses as type NodeInternalIP to match what the KNI
	// 	// // patch to the AWS cloud-provider code is doing:
	// 	// //
	// 	// // https://github.com/openshift-kni/origin/commit/7db21c1e26a344e25ae1b825d4f21e7bef5c3650
	// 	// for _, ipv6Address := range networkInterface.Ipv6Addresses {
	// 	// 	if addr := aws.StringValue(ipv6Address.Ipv6Address); addr != "" {
	// 	// 		ip := net.ParseIP(addr)
	// 	// 		if ip == nil {
	// 	// 			return nil, fmt.Errorf("EC2 instance had invalid IPv6 address: %s (%q)", aws.StringValue(instance.InstanceId), addr)
	// 	// 		}
	// 	// 		addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: ip.String()})
	// 	// 	}
	// 	// }

	// 	for _, ipAddress := range networkInterface.IPs {
	// 		if ipAddress != "" {
	// 			ip := net.ParseIP(ipAddress)
	// 			if ip == nil {
	// 				return nil, fmt.Errorf("KubeVirt instance had invalid IP address: %s (%q)", string(instance.UID), ipAddress)
	// 			}
	// 			// TODO Nir - Modify NodeAddressType according to ip info (public, private, etc ...)
	// 			addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: ip.String()})
	// 		}
	// 	}
	// }

	// TODO Nir - Get DNS name, public and private
	// privateDNSName := aws.StringValue(instance.PrivateDnsName)
	// if privateDNSName != "" {
	// 	addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeInternalDNS, Address: privateDNSName})
	// 	addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeHostName, Address: privateDNSName})
	// }

	// publicDNSName := aws.StringValue(instance.PublicDnsName)
	// if publicDNSName != "" {
	// 	addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeExternalDNS, Address: publicDNSName})
	// }

	return addresses, nil
}

// TODO There is only one kind of VirtualMachineConditionType: VirtualMachineFailure
//      How should report on success?
//      Is Failure/false is good enough or need to add type to client-go?
func conditionSuccess() kubevirtapiv1.VirtualMachineCondition {
	return kubevirtapiv1.VirtualMachineCondition{
		Type:    kubevirtapiv1.VirtualMachineFailure,
		Status:  corev1.ConditionFalse,
		Reason:  "MachineCreationSucceeded",
		Message: "Machine successfully created",
	}
}

func conditionFailed() kubevirtapiv1.VirtualMachineCondition {
	return kubevirtapiv1.VirtualMachineCondition{
		Type:   kubevirtapiv1.VirtualMachineFailure,
		Status: corev1.ConditionTrue,
		Reason: "MachineCreationFailed",
	}
}

// validateMachine check the label that a machine must have to identify the cluster to which it belongs is present.
func validateMachine(machine machinev1.Machine) error {
	if machine.Labels[machinev1.MachineClusterIDLabel] == "" {
		return machinecontroller.InvalidMachineConfiguration("%v: missing %q label", machine.GetName(), machinev1.MachineClusterIDLabel)
	}

	return nil
}

// getClusterID get cluster ID by machine.openshift.io/cluster-api-cluster label
func getClusterID(machine *machinev1.Machine) (string, bool) {
	clusterID, ok := machine.Labels[machinev1.MachineClusterIDLabel]
	// TODO: remove 347-350
	// NOTE: This block can be removed after the label renaming transition to machine.openshift.io
	if !ok {
		clusterID, ok = machine.Labels[upstreamMachineClusterIDLabel]
	}
	return clusterID, ok
}
