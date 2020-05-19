package machine

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	kubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client"

	kubevirtapiv1 "kubevirt.io/client-go/api/v1"
)

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

func createVm(virtualMachine *kubevirtapiv1.VirtualMachine, underkubeclient kubevirtclient.Client, namespace string) (*kubevirtapiv1.VirtualMachine, error) {
	return underkubeclient.CreateVirtualMachine(namespace, virtualMachine)
}

type vmList []*ec2.Instance
