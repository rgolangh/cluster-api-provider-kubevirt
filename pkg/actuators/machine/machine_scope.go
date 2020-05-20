package machine

import (
	"context"

	kubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client"

	kubevirtproviderv1 "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/apis/kubevirtprovider/v1"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	machineapierros "github.com/openshift/machine-api-operator/pkg/controller/machine"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesclient "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	kubevirtapiv1 "kubevirt.io/client-go/api/v1"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

const (
	userDataSecretKey = "userData"
)

// machineScopeParams defines the input parameters used to create a new MachineScope.
type machineScopeParams struct {
	context.Context

	kubevirtClientBuilder kubevirtclient.KubevirtClientBuilderFuncType
	// api server controller runtime client
	kubernetesClient *kubernetesclient.Clientset
	// machine resource
	machine *machinev1.Machine
}

type machineScope struct {
	context.Context

	// client for interacting with KubeVirt
	kubevirtClient kubevirtclient.Client
	// api server controller runtime client
	kubernetesClient *kubernetesclient.Clientset
	// machine resource
	machine        *machinev1.Machine
	virtualMachine *kubevirtapiv1.VirtualMachine
}

func newMachineScope(params machineScopeParams) (*machineScope, error) {
	providerSpec, err := kubevirtproviderv1.ProviderSpecFromRawExtension(params.machine.Spec.ProviderSpec.Value)
	if err != nil {
		return nil, machineapierros.InvalidMachineConfiguration("failed to get machine config: %v", err)
	}

	providerStatus, err := kubevirtproviderv1.ProviderStatusFromRawExtension(params.machine.Status.ProviderStatus)
	if err != nil {
		return nil, machineapierros.InvalidMachineConfiguration("failed to get machine provider status: %v", err.Error())
	}

	// TODO Nir - add secretName
	kubevirtClient, err := params.kubevirtClientBuilder(params.kubernetesClient, "", params.machine.GetNamespace())

	if err != nil {
		return nil, machineapierros.InvalidMachineConfiguration("failed to create aKubeVirt client: %v", err.Error())
	}

	virtualMachine := kubevirtapiv1.VirtualMachine{
		Spec:   kubevirtapiv1.VirtualMachineSpec{},
		Status: providerStatus.VirtualMachineStatus,
	}
	virtualMachine.TypeMeta = providerSpec.TypeMeta
	virtualMachine.ObjectMeta = providerSpec.ObjectMeta
	// TODO Nir - find pvc name
	virtualMachine.Spec.DataVolumeTemplates = []cdiv1.DataVolume{*buildBootVolumeDataVolumeTemplate(virtualMachine.Name, "pvc", params.machine.Namespace)}

	// TODO Nir - Add other virtualMachine params

	return &machineScope{
		Context:          params.Context,
		kubevirtClient:   kubevirtClient,
		kubernetesClient: params.kubernetesClient,
		machine:          params.machine,
		virtualMachine:   &virtualMachine,
	}, nil
}

func buildBootVolumeDataVolumeTemplate(virtualMachineName string, pvcName string, namespace string) *cdiv1.DataVolume {
	// TODO Nir - add spec to data volume
	return &cdiv1.DataVolume{
		TypeMeta: metav1.TypeMeta{APIVersion: cdiv1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      virtualMachineName + "BootVolume",
			Namespace: namespace,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				PVC: &cdiv1.DataVolumeSourcePVC{
					Name:      pvcName,
					Namespace: namespace,
				},
			},
			PVC: &corev1.PersistentVolumeClaimSpec{},
		},
	}
}

// Patch patches the machine spec and machine status after reconciling.
func (s *machineScope) patchMachine() error {
	// TODO implement if needed otherwise remove
	// klog.V(3).Infof("%v: patching machine", s.machine.GetName())

	// providerStatus, err := kubevirtproviderv1.RawExtensionFromProviderStatus(providerStatusFromVirtualMachineStatus(&s.virtualMachine.Status))
	// if err != nil {
	// 	return machineapierros.InvalidMachineConfiguration("failed to get machine provider status: %v", err.Error())
	// }
	// s.machine.Status.ProviderStatus = providerStatus

	// statusCopy := *s.machine.Status.DeepCopy()

	// // patch machine
	// if err := s.client.Patch(context.Background(), s.machine, s.machineToBePatched); err != nil {
	// 	klog.Errorf("Failed to patch machine %q: %v", s.machine.GetName(), err)
	// 	return err
	// }

	// s.machine.Status = statusCopy

	// // patch status
	// if err := s.client.Status().Patch(context.Background(), s.machine, s.machineToBePatched); err != nil {
	// 	klog.Errorf("Failed to patch machine status %q: %v", s.machine.GetName(), err)
	// 	return err
	// }

	return nil
}

func providerStatusFromVirtualMachineStatus(virtualMachineStatus *kubevirtapiv1.VirtualMachineStatus) *kubevirtproviderv1.KubevirtMachineProviderStatus {
	result := kubevirtproviderv1.KubevirtMachineProviderStatus{}
	result.VirtualMachineStatus = *virtualMachineStatus
	return &result
}

// TODO Nir - In Kubevirt just need to update local state with given state because its the same object
// Is other field need to be updated?
// Why in aws also update s.machine.Status.Addresses?
func (s *machineScope) setProviderStatus(vm *kubevirtapiv1.VirtualMachine, condition kubevirtapiv1.VirtualMachineCondition) error {
	klog.Infof("%s: Updating status", s.machine.GetName())

	networkAddresses := []corev1.NodeAddress{}

	if vm != nil {
		s.virtualMachine.Status = vm.Status

		// Copy specific adresses - only node adresses
		addresses, err := extractNodeAddresses(vm)
		if err != nil {
			klog.Errorf("%s: Error extracting vm IP addresses: %v", s.machine.GetName(), err)
			return err
		}

		networkAddresses = append(networkAddresses, addresses...)
		klog.Infof("%s: finished calculating KubeVirt status", s.machine.GetName())
	} else {
		klog.Infof("%s: couldn't calculate KubeVirt status - the provided vm is empty", s.machine.GetName())
	}

	s.machine.Status.Addresses = networkAddresses
	s.virtualMachine.Status.Conditions = setKubevirtMachineProviderCondition(condition, s.virtualMachine.Status.Conditions)

	return nil
}
