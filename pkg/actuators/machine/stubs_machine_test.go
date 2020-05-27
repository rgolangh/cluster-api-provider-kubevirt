package machine

import (
	"fmt"

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
