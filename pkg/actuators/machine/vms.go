package machine

import (
	kubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client"
	k8smetav1 "github.com/kubevirt/cluster-api-provider-kubevirt/vendor/k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	kubevirtapiv1 "kubevirt.io/client-go/api/v1"
)

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

func createVM(virtualMachine *kubevirtapiv1.VirtualMachine, underkubeclient kubevirtclient.Client, namespace string) (*kubevirtapiv1.VirtualMachine, error) {
	return underkubeclient.CreateVirtualMachine(namespace, virtualMachine)
}

func vmExists(vmName string, underkubeclient kubevirtclient.Client, namespace string) (*kubevirtapiv1.VirtualMachine, error) {
	return underkubeclient.GetVirtualMachine(namespace, vmName)
}

func listVMs(underkubeclient kubevirtclient.Client, namespace string) (*kubevirtapiv1.VirtualMachineList, error) {
	//filter the vms with a specific tag '{}'
	handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=" + "123")
	labelSelector, _ := labels.Parse(kubevirtapiv1.AppLabel + " in (virt-handler)")
	aa := k8smetav1.ListOptions{
		FieldSelector: handlerNodeSelector.String(),
		LabelSelector: labelSelector.String(),
	}

	return underkubeclient.ListVirtualMachine(namespace, &aa)
}
