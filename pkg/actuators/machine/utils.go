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
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	machinecontroller "github.com/openshift/machine-api-operator/pkg/controller/machine"
	"k8s.io/klog"
	kubevirtproviderv1 "kubevirt.io/client-go/api/v1"
	awsproviderv1 "sigs.k8s.io/cluster-api-provider-aws/pkg/apis/awsprovider/v1beta1"
	awsclient "sigs.k8s.io/cluster-api-provider-aws/pkg/client"
)

// upstreamMachineClusterIDLabel is the label that a machine must have to identify the cluster to which it belongs
const upstreamMachineClusterIDLabel = "sigs.k8s.io/cluster-api-cluster"

// existingInstanceStates returns the list of states an EC2 instance can be in
// while being considered "existing", i.e. mostly anything but "Terminated".
func existingInstanceStates() []*string {
	return []*string{
		kubevirtproviderv1.VirtualMachineInstancePhaseToString(kubevirtproviderv1.VmPhaseUnset),
		kubevirtproviderv1.VirtualMachineInstancePhaseToString(kubevirtproviderv1.Pending),
		kubevirtproviderv1.VirtualMachineInstancePhaseToString(kubevirtproviderv1.Scheduling),
		kubevirtproviderv1.VirtualMachineInstancePhaseToString(kubevirtproviderv1.Scheduled),
		kubevirtproviderv1.VirtualMachineInstancePhaseToString(kubevirtproviderv1.Running),
		kubevirtproviderv1.VirtualMachineInstancePhaseToString(kubevirtproviderv1.Succeeded),
		kubevirtproviderv1.VirtualMachineInstancePhaseToString(kubevirtproviderv1.Failed),
		kubevirtproviderv1.VirtualMachineInstancePhaseToString(kubevirtproviderv1.Unknown),
	}
}

// getRunningFromInstances returns all running instances from a list of instances.
func getRunningFromInstances(instances []*ec2.Instance) []*ec2.Instance {
	var runningInstances []*ec2.Instance
	for _, instance := range instances {
		if *instance.State.Name == ec2.InstanceStateNameRunning {
			runningInstances = append(runningInstances, instance)
		}
	}
	return runningInstances
}

// getStoppedInstances returns all stopped instances that have a tag matching our machine name,
// and cluster ID.
func getStoppedInstances(machine *machinev1.Machine, client awsclient.Client) ([]*ec2.Instance, error) {
	stoppedInstanceStateFilter := []*string{aws.String(ec2.InstanceStateNameStopped), aws.String(ec2.InstanceStateNameStopping)}
	return getInstances(machine, client, stoppedInstanceStateFilter)
}

// getExistingInstances returns all instances not terminated
func getExistingInstances(machine *machinev1.Machine, client awsclient.Client) ([]*ec2.Instance, error) {
	return getInstances(machine, client, existingInstanceStates())
}

func getExistingInstanceByID(id string, client awsclient.Client) (*ec2.Instance, error) {
	return getInstanceByID(id, client, existingInstanceStates())
}

func instanceHasAllowedState(instance *ec2.Instance, instanceStateFilter []*string) error {
	if instance.InstanceId == nil {
		return fmt.Errorf("instance has nil ID")
	}

	if instance.State == nil {
		return fmt.Errorf("instance %s has nil state", *instance.InstanceId)
	}

	if len(instanceStateFilter) == 0 {
		return nil
	}

	actualState := aws.StringValue(instance.State.Name)
	for _, allowedState := range instanceStateFilter {
		if aws.StringValue(allowedState) == actualState {
			return nil
		}
	}

	allowedStates := make([]string, 0, len(instanceStateFilter))
	for _, allowedState := range instanceStateFilter {
		allowedStates = append(allowedStates, aws.StringValue(allowedState))
	}
	return fmt.Errorf("instance %s state %q is not in %s", *instance.InstanceId, actualState, strings.Join(allowedStates, ", "))
}

// getInstanceByID returns the instance with the given ID if it exists.
func getInstanceByID(id string, client awsclient.Client, instanceStateFilter []*string) (*ec2.Instance, error) {
	if id == "" {
		return nil, fmt.Errorf("instance-id not specified")
	}

	request := &ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{id}),
	}

	result, err := client.DescribeInstances(request)
	if err != nil {
		return nil, err
	}

	if len(result.Reservations) != 1 {
		return nil, fmt.Errorf("found %d reservations for instance-id %s", len(result.Reservations), id)
	}

	reservation := result.Reservations[0]

	if len(reservation.Instances) != 1 {
		return nil, fmt.Errorf("found %d instances for instance-id %s", len(reservation.Instances), id)
	}

	instance := reservation.Instances[0]

	return instance, instanceHasAllowedState(instance, instanceStateFilter)
}

// getInstances returns all instances that have a tag matching our machine name,
// and cluster ID.
func getInstances(machine *machinev1.Machine, client awsclient.Client, instanceStateFilter []*string) ([]*ec2.Instance, error) {
	clusterID, ok := getClusterID(machine)
	if !ok {
		return []*ec2.Instance{}, fmt.Errorf("unable to get cluster ID for machine: %q", machine.Name)
	}

	requestFilters := []*ec2.Filter{
		{
			Name:   awsTagFilter("Name"),
			Values: aws.StringSlice([]string{machine.Name}),
		},
		clusterFilter(clusterID),
	}

	request := &ec2.DescribeInstancesInput{
		Filters: requestFilters,
	}

	result, err := client.DescribeInstances(request)
	if err != nil {
		return []*ec2.Instance{}, err
	}

	instances := make([]*ec2.Instance, 0, len(result.Reservations))

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			err := instanceHasAllowedState(instance, instanceStateFilter)
			if err != nil {
				klog.Errorf("Excluding instance matching %s: %v", machine.Name, err)
			} else {
				instances = append(instances, instance)
			}
		}
	}

	return instances, nil
}

// terminateInstances terminates all provided instances with a single EC2 request.
func terminateInstances(client awsclient.Client, instances []*ec2.Instance) ([]*ec2.InstanceStateChange, error) {
	instanceIDs := []*string{}
	// Cleanup all older instances:
	for _, instance := range instances {
		klog.Infof("Cleaning up extraneous instance for machine: %v, state: %v, launchTime: %v", *instance.InstanceId, *instance.State.Name, *instance.LaunchTime)
		instanceIDs = append(instanceIDs, instance.InstanceId)
	}
	for _, instanceID := range instanceIDs {
		klog.Infof("Terminating %v instance", *instanceID)
	}

	terminateInstancesRequest := &ec2.TerminateInstancesInput{
		InstanceIds: instanceIDs,
	}
	output, err := client.TerminateInstances(terminateInstancesRequest)
	if err != nil {
		klog.Errorf("Error terminating instances: %v", err)
		return nil, fmt.Errorf("error terminating instances: %v", err)
	}

	if output == nil {
		return nil, nil
	}

	return output.TerminatingInstances, nil
}

// setKubevirtMachineProviderCondition sets the condition for the machine and
// returns the new slice of conditions.
// If the machine does not already have a condition with the specified type,
// a condition will be added to the slice
// If the machine does already have a condition with the specified type,
// the condition will be updated if either of the following are true.
func setKubevirtMachineProviderCondition(condition kubevirtproviderv1.VirtualMachineCondition, conditions []kubevirtproviderv1.VirtualMachineCondition) []kubevirtproviderv1.VirtualMachineCondition {
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

func findProviderCondition(conditions []kubevirtproviderv1.VirtualMachineCondition, conditionType kubevirtproviderv1.VirtualMachineConditionType) *kubevirtproviderv1.VirtualMachineCondition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

func updateExistingCondition(newCondition, existingCondition *kubevirtproviderv1.VirtualMachineCondition) {
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

func shouldUpdateCondition(newCondition, existingCondition *kubevirtproviderv1.VirtualMachineCondition) bool {
	return newCondition.Reason != existingCondition.Reason || newCondition.Message != existingCondition.Message
}

// TODO Nir do we need to extract node adresses?
// The network info is saved in the vmi
// extractNodeAddresses maps the instance information from EC2 to an array of NodeAddresses
func extractNodeAddresses(vm *kubevirtproviderv1.VirtualMachine) ([]corev1.NodeAddress, error) {
	// Not clear if the order matters here, but we might as well indicate a sensible preference order

	if vm == nil {
		return nil, fmt.Errorf("nil vm passed to extractNodeAddresses")
	}

	addresses := []corev1.NodeAddress{}

	// handle internal network interfaces
	for _, networkInterface := range vm.Status. {
		// TODO Nir - In kubevirt there is not network state and IPV6
		// // skip network interfaces that are not currently in use
		// if aws.StringValue(networkInterface.Status) != ec2.NetworkInterfaceStatusInUse {
		// 	continue
		// }

		// TODO Does kubevirt use IPV6?
		// // Treating IPv6 addresses as type NodeInternalIP to match what the KNI
		// // patch to the AWS cloud-provider code is doing:
		// //
		// // https://github.com/openshift-kni/origin/commit/7db21c1e26a344e25ae1b825d4f21e7bef5c3650
		// for _, ipv6Address := range networkInterface.Ipv6Addresses {
		// 	if addr := aws.StringValue(ipv6Address.Ipv6Address); addr != "" {
		// 		ip := net.ParseIP(addr)
		// 		if ip == nil {
		// 			return nil, fmt.Errorf("EC2 instance had invalid IPv6 address: %s (%q)", aws.StringValue(instance.InstanceId), addr)
		// 		}
		// 		addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: ip.String()})
		// 	}
		// }

		for _, ipAddress := range networkInterface.IPs {
			if ipAddress != "" {
				ip := net.ParseIP(ipAddress)
				if ip == nil {
					return nil, fmt.Errorf("KubeVirt instance had invalid IP address: %s (%q)", string(instance.UID), ipAddress)
				}
				// TODO Nir - Modify NodeAddressType according to ip info (public, private, etc ...)
				addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: ip.String()})
			}
		}
	}

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

func conditionSuccess() kubevirtproviderv1.VirtualMachineInstanceCondition {
	return kubevirtproviderv1.VirtualMachineInstanceCondition{
		Type:    kubevirtproviderv1.VirtualMachineInstanceReady,
		Status:  corev1.ConditionTrue,
		Reason:  "MachineCreationSucceeded",
		Message: "Machine successfully created",
	}
}

func conditionFailed() kubevirtproviderv1.VirtualMachineInstanceCondition {
	return kubevirtproviderv1.VirtualMachineInstanceCondition{
		Type:   awsproviderv1.MachineCreation,
		Status: corev1.ConditionFalse,
		Reason: awsproviderv1.MachineCreationFailed,
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
