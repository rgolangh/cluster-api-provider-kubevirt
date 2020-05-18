package machine

import (
	"encoding/base64"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	mapierrors "github.com/openshift/machine-api-operator/pkg/controller/machine"
	"k8s.io/klog"
	awsproviderv1 "sigs.k8s.io/cluster-api-provider-aws/pkg/apis/awsprovider/v1beta1"
	kubevirtclient "sigs.k8s.io/cluster-api-provider-kubevirt/pkg/client"

	v1 "kubevirt.io/client-go/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

// Scan machine tags, and return a deduped tags list
func removeDuplicatedTags(tags []*ec2.Tag) []*ec2.Tag {
	m := make(map[string]bool)
	result := []*ec2.Tag{}

	// look for duplicates
	for _, entry := range tags {
		if _, value := m[*entry.Key]; !value {
			m[*entry.Key] = true
			result = append(result, entry)
		}
	}
	return result
}

// removeStoppedMachine removes all instances of a specific machine that are in a stopped state.
func removeStoppedMachine(machine *machinev1.Machine, client kubevirtclient.Client) error {
	instances, err := getStoppedInstances(machine, client)
	if err != nil {
		klog.Errorf("Error getting stopped instances: %v", err)
		return fmt.Errorf("error getting stopped instances: %v", err)
	}

	if len(instances) == 0 {
		klog.Infof("No stopped instances found for machine %v", machine.Name)
		return nil
	}

	_, err = terminateInstances(client, instances)
	return err
}

func buildEC2Filters(inputFilters []awsproviderv1.Filter) []*ec2.Filter {
	filters := make([]*ec2.Filter, len(inputFilters))
	for i, f := range inputFilters {
		values := make([]*string, len(f.Values))
		for j, v := range f.Values {
			values[j] = aws.String(v)
		}
		filters[i] = &ec2.Filter{
			Name:   aws.String(f.Name),
			Values: values,
		}
	}
	return filters
}

func getSecurityGroupsIDs(securityGroups []awsproviderv1.AWSResourceReference, client kubevirtclient.Client) ([]*string, error) {
	var securityGroupIDs []*string
	for _, g := range securityGroups {
		// ID has priority
		if g.ID != nil {
			securityGroupIDs = append(securityGroupIDs, g.ID)
		} else if g.Filters != nil {
			klog.Info("Describing security groups based on filters")
			// Get groups based on filters
			describeSecurityGroupsRequest := ec2.DescribeSecurityGroupsInput{
				Filters: buildEC2Filters(g.Filters),
			}
			describeSecurityGroupsResult, err := client.DescribeSecurityGroups(&describeSecurityGroupsRequest)
			if err != nil {
				klog.Errorf("error describing security groups: %v", err)
				return nil, fmt.Errorf("error describing security groups: %v", err)
			}
			for _, g := range describeSecurityGroupsResult.SecurityGroups {
				groupID := *g.GroupId
				securityGroupIDs = append(securityGroupIDs, &groupID)
			}
		}
	}

	if len(securityGroups) == 0 {
		klog.Info("No security group found")
	}

	return securityGroupIDs, nil
}

func getSubnetIDs(subnet awsproviderv1.AWSResourceReference, availabilityZone string, client kubevirtclient.Client) ([]*string, error) {
	var subnetIDs []*string
	// ID has priority
	if subnet.ID != nil {
		subnetIDs = append(subnetIDs, subnet.ID)
	} else {
		var filters []awsproviderv1.Filter
		if availabilityZone != "" {
			// Improve error logging for better user experience.
			// Otherwise, during the process of minimizing API calls, this is a good
			// candidate for removal.
			_, err := client.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{
				ZoneNames: []*string{aws.String(availabilityZone)},
			})
			if err != nil {
				klog.Errorf("error describing availability zones: %v", err)
				return nil, fmt.Errorf("error describing availability zones: %v", err)
			}
			filters = append(filters, awsproviderv1.Filter{Name: "availabilityZone", Values: []string{availabilityZone}})
		}
		filters = append(filters, subnet.Filters...)
		klog.Info("Describing subnets based on filters")
		describeSubnetRequest := ec2.DescribeSubnetsInput{
			Filters: buildEC2Filters(filters),
		}
		describeSubnetResult, err := client.DescribeSubnets(&describeSubnetRequest)
		if err != nil {
			klog.Errorf("error describing subnetes: %v", err)
			return nil, fmt.Errorf("error describing subnets: %v", err)
		}
		for _, n := range describeSubnetResult.Subnets {
			subnetID := *n.SubnetId
			subnetIDs = append(subnetIDs, &subnetID)
		}
	}
	if len(subnetIDs) == 0 {
		return nil, fmt.Errorf("no subnet IDs were found")
	}
	return subnetIDs, nil
}

func getPvcName(image awsproviderv1.AWSResourceReference, client kubevirtclient.Client) (*string, error) {
	if image.ID != nil {
		imageID := image.ID
		klog.Infof("Using image %s", *imageID)
		return imageID, nil
	}
	if len(image.Filters) > 0 {
		klog.Info("Describing AMI based on filters")
		describeImagesRequest := ec2.DescribeImagesInput{
			Filters: buildEC2Filters(image.Filters),
		}
		describeAMIResult, err := client.DescribeImages(&describeImagesRequest)
		if err != nil {
			klog.Errorf("error describing AMI: %v", err)
			return nil, fmt.Errorf("error describing AMI: %v", err)
		}
		if len(describeAMIResult.Images) < 1 {
			klog.Errorf("no image for given filters not found")
			return nil, fmt.Errorf("no image for given filters not found")
		}
		latestImage := describeAMIResult.Images[0]
		latestTime, err := time.Parse(time.RFC3339, *latestImage.CreationDate)
		if err != nil {
			klog.Errorf("unable to parse time for %q AMI: %v", *latestImage.ImageId, err)
			return nil, fmt.Errorf("unable to parse time for %q AMI: %v", *latestImage.ImageId, err)
		}
		for _, image := range describeAMIResult.Images[1:] {
			imageTime, err := time.Parse(time.RFC3339, *image.CreationDate)
			if err != nil {
				klog.Errorf("unable to parse time for %q AMI: %v", *image.ImageId, err)
				return nil, fmt.Errorf("unable to parse time for %q AMI: %v", *image.ImageId, err)
			}
			if latestTime.Before(imageTime) {
				latestImage = image
				latestTime = imageTime
			}
		}
		return latestImage.ImageId, nil
	}
	return nil, fmt.Errorf("AMI ID or AMI filters need to be specified")
}
func buildSpecVolume(pvcName string, userData []byte) []v1.Volume{
	//TODO: move to machine_scope
	userDataEnc := base64.StdEncoding.EncodeToString(userData)
	return []v1.Volume{
		//{
		//	Name: "??",
		//	VolumeSource: v1.VolumeSource{
		//		ContainerDisk: &v1.ContainerDiskSource{
		//			Image: "my-image",
		//			Path:  "????",
		//		},
		//	},
		//},
		//
		{
			Name: "bootVolume0",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		}
	}
}


func createVm(machine *machinev1.Machine, machineProviderConfig *kubevirtproviderv1.KubeVirtMachineProviderConfig, userData []byte, client kubevirtclient.Client) (*v1.VirtualMachine, error) {
	//pvcName, err := getPvcName(machineProviderConfig.image, client)
	//if err != nil {
	//	return nil, mapierrors.InvalidMachineConfiguration("error getting AMI: %v", err)
	//}

	pvcName = machineProviderConfig.pvcName
	//TODO: for POC the network will be empty
	//TODO: for the poc we will create only the boot volume + userdata??
	//TODO: add labels
	//TODO: ignition of userData


	newVm := &v1.VirtualMachine{}
	newVm.Spec = machineProviderConfig.Spec
	//newVm.Status = machineProviderConfig.Status
	namespace := machine.Namespace
	newVm.Spec.Volumes = buildSpecVolume(pvcName, userData)
	return client.CreateVirtualMachine(namespace, newVM)
}

type instanceList []*ec2.Instance

func (il instanceList) Len() int {
	return len(il)
}

func (il instanceList) Swap(i, j int) {
	il[i], il[j] = il[j], il[i]
}

func (il instanceList) Less(i, j int) bool {
	if il[i].LaunchTime == nil && il[j].LaunchTime == nil {
		return false
	}
	if il[i].LaunchTime != nil && il[j].LaunchTime == nil {
		return false
	}
	if il[i].LaunchTime == nil && il[j].LaunchTime != nil {
		return true
	}
	return (*il[i].LaunchTime).After(*il[j].LaunchTime)
}

// sortInstances will sort a list of instance based on an instace launch time
// from the newest to the oldest.
// This function should only be called with running instances, not those which are stopped or
// terminated.
func sortInstances(instances []*ec2.Instance) {
	sort.Sort(instanceList(instances))
}

func getInstanceMarketOptionsRequest(providerConfig *awsproviderv1.AWSMachineProviderConfig) *ec2.InstanceMarketOptionsRequest {
	if providerConfig.SpotMarketOptions == nil {
		// Instance is not a Spot instance
		return nil
	}

	// Set required values for Spot instances
	spotOptions := &ec2.SpotMarketOptions{}
	// The following two options ensure that:
	// - If an instance is interrupted, it is terminated rather than hibernating or stopping
	// - No replacement instance will be created if the instance is interrupted
	// - If the spot request cannot immediately be fulfilled, it will not be created
	// This behaviour should satisfy the 1:1 mapping of Machines to Instances as
	// assumed by the machine API.
	spotOptions.SetInstanceInterruptionBehavior(ec2.InstanceInterruptionBehaviorTerminate)
	spotOptions.SetSpotInstanceType(ec2.SpotInstanceTypeOneTime)

	// Set the MaxPrice if specified by the providerConfig
	maxPrice := providerConfig.SpotMarketOptions.MaxPrice
	if maxPrice != nil && *maxPrice != "" {
		spotOptions.SetMaxPrice(*maxPrice)
	}

	instanceMarketOptionsRequest := &ec2.InstanceMarketOptionsRequest{}
	instanceMarketOptionsRequest.SetMarketType(ec2.MarketTypeSpot)
	instanceMarketOptionsRequest.SetSpotOptions(spotOptions)

	return instanceMarketOptionsRequest
}
