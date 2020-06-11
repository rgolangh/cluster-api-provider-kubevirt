package vm

import (
	"fmt"

	kubevirtapiv1 "kubevirt.io/client-go/api/v1"

	machineapierros "github.com/openshift/machine-api-operator/pkg/controller/machine"

	kubernetesclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/clients/kubernetes"
	kubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/clients/kubevirt"

	kubevirtproviderv1 "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/apis/kubevirtprovider/v1"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultNamespace   = "default"
	mahcineName        = "machine-test"
	userDataSecretName = "kubevirt-actuator-user-data-secret"
	keyName            = "kubevirt-actuator-key-name"
	clusterID          = "kubevirt-actuator-cluster"
	clusterName        = "kubevirt-actuator-cluster"
)

func stubVmi(vm *kubevirtapiv1.VirtualMachine) (*kubevirtapiv1.VirtualMachineInstance, error) {
	vmi := kubevirtapiv1.VirtualMachineInstance{
		//TypeMeta:   v12.TypeMeta{},
		//ObjectMeta: v12.ObjectMeta{},
		Spec:   kubevirtapiv1.VirtualMachineInstanceSpec{},
		Status: kubevirtapiv1.VirtualMachineInstanceStatus{},
	}
	vmi.Name = vm.Name
	vmi.Namespace = vm.Namespace
	vmi.Spec = vm.Spec.Template.Spec
	return &vmi, nil
}

func stubMachineScope(machine *machinev1.Machine, kubernetesClient kubernetesclient.Client, kubevirtClientBuilder kubevirtclient.ClientBuilderFuncType) (*machineScope, error) {
	providerSpec, err := kubevirtproviderv1.ProviderSpecFromRawExtension(machine.Spec.ProviderSpec.Value)
	if err != nil {
		return nil, machineapierros.InvalidMachineConfiguration("failed to get machine config: %v", err)
	}

	providerStatus, err := kubevirtproviderv1.ProviderStatusFromRawExtension(machine.Status.ProviderStatus)
	if err != nil {
		return nil, machineapierros.InvalidMachineConfiguration("failed to get machine provider status: %v", err.Error())
	}

	kubevirtClient, err := kubevirtClientBuilder(kubernetesClient, providerSpec.SecretName, machine.GetNamespace())
	if err != nil {
		return nil, machineapierros.InvalidMachineConfiguration("failed to create aKubeVirt client: %v", err.Error())
	}

	return &machineScope{
		kubevirtClient:        kubevirtClient,
		kubernetesClient:      kubernetesClient,
		machine:               machine,
		originMachineCopy:     machine.DeepCopy(),
		machineProviderSpec:   providerSpec,
		machineProviderStatus: providerStatus,
	}, nil
}

func stubMachine(labels map[string]string, providerID string) (*machinev1.Machine, error) {
	providerSpecValue, providerSpecValueErr := kubevirtproviderv1.RawExtensionFromProviderSpec(&kubevirtproviderv1.KubevirtMachineProviderSpec{
		SourcePvcName: "SourceTestPvcName",
	})
	if labels == nil {
		labels = map[string]string{
			machinev1.MachineClusterIDLabel: clusterID,
		}
	}
	if providerSpecValueErr != nil {
		return nil, fmt.Errorf("codec.EncodeProviderSpec failed: %v", providerSpecValueErr)
	}
	machine := &machinev1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       mahcineName,
			Namespace:                  defaultNamespace,
			Generation:                 0,
			CreationTimestamp:          metav1.Time{},
			DeletionTimestamp:          nil,
			DeletionGracePeriodSeconds: nil,
			Labels:                     labels,
			//Annotations:                nil,
			ClusterName: clusterName,
		},
		Spec: machinev1.MachineSpec{
			ObjectMeta:   metav1.ObjectMeta{},
			ProviderSpec: machinev1.ProviderSpec{Value: providerSpecValue},
			ProviderID:   &providerID,
		},
		Status: machinev1.MachineStatus{},
	}

	return machine, nil
}
