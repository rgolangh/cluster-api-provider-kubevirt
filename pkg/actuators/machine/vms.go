package machine

import (
	kubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client"

	kubevirtapiv1 "kubevirt.io/client-go/api/v1"
)

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

func createVM(virtualMachine *kubevirtapiv1.VirtualMachine, underkubeclient kubevirtclient.Client, namespace string) (*kubevirtapiv1.VirtualMachine, error) {
	return underkubeclient.CreateVirtualMachine(namespace, virtualMachine)
}

type vmList []*kubevirtapiv1.VirtualMachine
