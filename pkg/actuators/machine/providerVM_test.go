package machine

import (
	"errors"
	"testing"

	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesclient "k8s.io/client-go/kubernetes"

	"github.com/golang/mock/gomock"
	kubevirtClient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client"
	mockkubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client/mock"
	"gotest.tools/assert"
	v1 "kubevirt.io/client-go/api/v1"
)

func initializeTest(t *testing.T, mockKubevirtClient *mockkubevirtclient.MockClient, labels map[string]string, providerID string) *machineScope {
	machine, stubMachineErr := stubMachine(labels, providerID)

	if stubMachineErr != nil {
		t.Fatalf("Unable to build test machine manifest: %v", stubMachineErr)
		return nil
	}

	machineScope, newMachineScopeErr := newMachineScope(machineScopeParams{
		kubevirtClientBuilder: func(kubernetesClient *kubernetesclient.Clientset, secretName, namespace string) (kubevirtClient.Client, error) {
			return mockKubevirtClient, nil
		},
		machine:          machine,
		kubernetesClient: nil,
		Context:          nil,
	})

	if newMachineScopeErr != nil {
		t.Fatal(newMachineScopeErr)
		return nil
	}

	return machineScope

}
func TestCreate(t *testing.T) {
	// TODO add a case of setProviderID and setMachineCloudProviderSpecifics failure
	cases := []struct {
		name                   string
		wantValidateMachineErr error
		wantCreateErr          error
		labels                 map[string]string
		providerID             string
	}{
		{
			name:                   "Create a VM",
			wantValidateMachineErr: nil,
			wantCreateErr:          nil,
			labels:                 nil,
			providerID:             nil,
		},
		{
			name:                   "Create a VM from unlabeled machine and fail",
			wantValidateMachineErr: errors.New("failed validating machine provider spec"),
			wantCreateErr:          nil,
			labels:                 map[string]string{machinev1.MachineClusterIDLabel: ""},
			providerID:             nil,
		},
		{
			name:                   "Create a VM with an error in the client-go and fail",
			wantValidateMachineErr: nil,
			wantCreateErr:          errors.New("failed to create virtual machine"),
			labels:                 nil,
			providerID:             nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockKubevirtClient := mockkubevirtclient.NewMockClient(mockCtrl)
			machineScope := initializeTest(t, mockKubevirtClient, tc.labels, tc.providerID)
			if machineScope == nil {
				return
			}

			mockKubevirtClient.EXPECT().CreateVirtualMachine(defaultNamespace, machineScope.virtualMachine).Return(machineScope.virtualMachine, tc.wantCreateErr).Times(1)

			createVMErr := providerVM{machineScope}.create()
			if tc.wantValidateMachineErr != nil {
				assert.Equal(t, tc.wantValidateMachineErr, createVMErr)
			} else if tc.wantCreateErr != nil {
				assert.Equal(t, tc.wantCreateErr, createVMErr)
			} else {
				assert.Equal(t, createVMErr, nil)
			}
		})
	}

}

func TestDelete(t *testing.T) {
	// TODO add a case of setProviderID and setMachineCloudProviderSpecifics failure
	cases := []struct {
		name                   string
		wantValidateMachineErr error
		wantGetErr             error
		wantDeleteErr          error
		emptyGetVM             bool
		labels                 map[string]string
		providerID             string
	}{
		{
			name:                   "Delete a VM successfully",
			wantValidateMachineErr: nil,
			wantGetErr:             nil,
			wantDeleteErr:          nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             nil,
		},
		{
			name:                   "Delete a VM from unlabeled machine and fail",
			wantValidateMachineErr: errors.New("failed validating machine provider spec"),
			wantGetErr:             nil,
			wantDeleteErr:          nil,
			emptyGetVM:             false,
			labels:                 map[string]string{machinev1.MachineClusterIDLabel: ""},
			providerID:             nil,
		},
		{
			name:                   "Delete deleting VM with getting error and fail",
			wantValidateMachineErr: nil,
			wantGetErr:             errors.New("ferror getting existing VM"),
			wantDeleteErr:          nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             nil,
		},
		{
			name:                   "Delete a nonexistent VM and fail",
			wantValidateMachineErr: nil,
			wantGetErr:             nil,
			wantDeleteErr:          nil,
			emptyGetVM:             true,
			labels:                 nil,
			providerID:             nil,
		},
		{
			name:                   "Delete a VM with an error in the client-go and fail",
			wantValidateMachineErr: nil,
			wantGetErr:             nil,
			wantDeleteErr:          errors.New("failed to delete virtual machine"),
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockKubevirtClient := mockkubevirtclient.NewMockClient(mockCtrl)

			machineScope := initializeTest(t, mockKubevirtClient, tc.labels, tc.providerID)
			if machineScope == nil {
				return
			}

			returnVm := machineScope.virtualMachine
			if tc.emptyGetVM {
				returnVm = nil
			}
			mockKubevirtClient.EXPECT().GetVirtualMachine(defaultNamespace, machineScope.virtualMachine.Name).Return(returnVm, tc.wantGetErr).Times(1)

			//TODO : understand if is it good like that
			//gracePeriod := int64(10)
			//mockKubevirtClient.EXPECT().DeleteVirtualMachine(defaultNamespace, machineScope.virtualMachine, &k8smetav1.DeleteOptions{GracePeriodSeconds: &gracePeriod}).Return(machineScope.virtualMachine, nil).AnyTimes()
			mockKubevirtClient.EXPECT().DeleteVirtualMachine(defaultNamespace, machineScope.virtualMachine, gomock.Any()).Return(tc.wantDeleteErr).Times(1)

			deleteVMErr := providerVM{machineScope}.delete()

			if tc.wantValidateMachineErr != nil {
				assert.Equal(t, tc.wantValidateMachineErr, deleteVMErr)
			} else if tc.wantGetErr != nil {
				assert.Equal(t, tc.wantGetErr, deleteVMErr)
			} else if tc.wantDeleteErr != nil {
				assert.Equal(t, tc.wantDeleteErr, deleteVMErr)
			} else {
				assert.Equal(t, deleteVMErr, nil)
			}
		})
	}

}

func TestExists(t *testing.T) {
	// TODO add a case of setProviderID and setMachineCloudProviderSpecifics failure
	cases := []struct {
		name                   string
		wantValidateMachineErr error
		wantGetErr             error
		emptyGetVM             bool
		isExist                bool
		labels                 map[string]string
		providerID             string
	}{
		{
			name:                   "Validate existence VM",
			wantValidateMachineErr: nil,
			wantGetErr:             nil,
			emptyGetVM:             false,
			isExist:                true,
			labels:                 nil,
			providerID:             nil,
		},
		{
			name:                   "Validate non existence VM",
			wantValidateMachineErr: nil,
			wantGetErr:             nil,
			emptyGetVM:             true,
			isExist:                false,
			labels:                 nil,
			providerID:             nil,
		},
		{
			name:                   "Validate existence VM with unlabeled machine and fail",
			wantValidateMachineErr: errors.New("failed validating machine provider spec"),
			wantGetErr:             nil,
			emptyGetVM:             false,
			isExist:                true,
			labels:                 map[string]string{machinev1.MachineClusterIDLabel: ""},
			providerID:             nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockKubevirtClient := mockkubevirtclient.NewMockClient(mockCtrl)
			machineScope := initializeTest(t, mockKubevirtClient, tc.labels, tc.providerID)
			if machineScope == nil {
				return
			}

			returnVm := machineScope.virtualMachine
			if tc.emptyGetVM {
				returnVm = nil
			}
			mockKubevirtClient.EXPECT().GetVirtualMachine(defaultNamespace, machineScope.virtualMachine.Name).Return(returnVm, tc.wantGetErr).Times(1)

			existsVM, existsVMErr := providerVM{machineScope}.exists()

			if tc.wantValidateMachineErr != nil {
				assert.Equal(t, tc.wantValidateMachineErr, existsVMErr)
			} else if tc.wantGetErr != nil {
				assert.Equal(t, tc.wantGetErr, existsVMErr)
			} else if tc.emptyGetVM {
				assert.Equal(t, existsVMErr, nil)
				assert.Equal(t, existsVM, false)
			} else {
				assert.Equal(t, existsVMErr, nil)
				assert.Equal(t, existsVM, tc.isExist)
			}
		})
	}

}

func TestUpdate(t *testing.T) {
	// TODO add a case of setProviderID and setMachineCloudProviderSpecifics failure
	cases := []struct {
		name                   string
		wantValidateMachineErr error
		wantGetErr             error
		wantUpdateeErr         error
		emptyGetVM             bool
		labels                 map[string]string
		providerID             string
	}{
		{
			name:                   "Update a VM",
			wantValidateMachineErr: nil,
			wantGetErr:             nil,
			wantUpdateeErr:         nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             nil,
		},
		{
			name:                   "Update a VM from unlabeled machine and fail",
			wantValidateMachineErr: errors.New("failed validating machine provider spec"),
			wantGetErr:             nil,
			wantUpdateeErr:         nil,
			emptyGetVM:             false,
			labels:                 map[string]string{machinev1.MachineClusterIDLabel: ""},
			providerID:             nil,
		},
		{
			name:                   "Update a VM with getting error and fail",
			wantValidateMachineErr: nil,
			wantGetErr:             errors.New("error getting existing VM"),
			wantUpdateeErr:         nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             nil,
		},
		{
			name:                   "Update a nonexistent VM and fail",
			wantValidateMachineErr: nil,
			wantGetErr:             nil,
			wantUpdateeErr:         nil,
			emptyGetVM:             true,
			labels:                 nil,
			providerID:             nil,
		},

		{
			name:                   "Delete a VM with an error in the client-go and fail",
			wantValidateMachineErr: nil,
			wantGetErr:             nil,
			wantUpdateeErr:         errors.New("failed to update virtual machine"),
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockKubevirtClient := mockkubevirtclient.NewMockClient(mockCtrl)
			machineScope := initializeTest(t, mockKubevirtClient, tc.labels, tc.providerID)
			if machineScope == nil {
				return
			}

			returnVm := machineScope.virtualMachine
			if tc.emptyGetVM {
				returnVm = nil
			}
			mockKubevirtClient.EXPECT().GetVirtualMachine(defaultNamespace, machineScope.virtualMachine.Name).Return(returnVm, tc.wantGetErr).Times(1)
			mockKubevirtClient.EXPECT().UpdateVirtualMachine(defaultNamespace, machineScope.virtualMachine).Return(machineScope.virtualMachine, tc.wantUpdateeErr).Times(1)
			// TODO: added cases when existingVM == nil :
			// p.machine.Spec.ProviderID != nil && *p.machine.Spec.ProviderID != "" && (p.machine.Status.LastUpdated == nil || p.machine.Status.LastUpdated.Add(requeueAfterSeconds*time.Second).After(time.Now())) - return error
			// else - another error
			updateVMErr := providerVM{machineScope}.update()

			if tc.wantValidateMachineErr != nil {
				assert.Equal(t, tc.wantValidateMachineErr, updateVMErr)
			} else if tc.wantGetErr != nil {
				assert.Equal(t, tc.wantGetErr, updateVMErr)
			} else if tc.wantUpdateeErr != nil {
				assert.Equal(t, tc.wantUpdateeErr, updateVMErr)
			} else {
				assert.Equal(t, updateVMErr, nil)
				//providerID := fmt.Sprintf("kubevirt:///%s/%s", machineScope.machine.GetNamespace(), machineScope.virtualMachine.GetName())
				assert.Equal(t, machineScope.machine.Spec.ProviderID, tc.providerID)
			}
		})
	}

}

func DefaultVirtualMachine(started bool) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	return DefaultVirtualMachineWithNames(started, "testvmi", "testvmi")
}

func DefaultVirtualMachineWithNames(started bool, vmName string, vmiName string) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	vmi := v1.NewMinimalVMI(vmiName)
	vmi.Status.Phase = v1.Running
	vm := VirtualMachineFromVMI(vmName, vmi, started)
	t := true
	vmi.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               v1.VirtualMachineGroupVersionKind.Kind,
		Name:               vm.ObjectMeta.Name,
		UID:                vm.ObjectMeta.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}}
	return vm, vmi
}

func VirtualMachineFromVMI(name string, vmi *v1.VirtualMachineInstance, started bool) *v1.VirtualMachine {
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vmi.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: v1.VirtualMachineSpec{
			Running: &started,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vmi.ObjectMeta.Name,
					Labels: vmi.ObjectMeta.Labels,
				},
				Spec: vmi.Spec,
			},
		},
	}
	return vm
}
