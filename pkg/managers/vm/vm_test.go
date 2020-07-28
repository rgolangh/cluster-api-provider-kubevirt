package vm

import (
	"errors"
	"fmt"
	"testing"

	kubevirtapiv1 "kubevirt.io/client-go/api/v1"

	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"

	"github.com/golang/mock/gomock"
	"github.com/kubevirt/cluster-api-provider-kubevirt/pkg/clients/overkube"
	mockoverkube "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/clients/overkube/mock"
	"github.com/kubevirt/cluster-api-provider-kubevirt/pkg/clients/underkube"
	mockunderkube "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/clients/underkube/mock"
	"gotest.tools/assert"
)

func initializeMachine(t *testing.T, mockUnderkube *mockunderkube.MockClient, labels map[string]string, providerID string) *machinev1.Machine {
	machine, stubMachineErr := stubMachine(labels, providerID)

	if stubMachineErr != nil {
		t.Fatalf("Unable to build test machine manifest: %v", stubMachineErr)
		return nil
	}

	return machine
}

func TestCreate(t *testing.T) {
	// TODO add a case of setProviderID and setMachineAnnotationsAndLabels failure
	// TODO add excpect times per
	cases := []struct {
		name                     string
		wantValidateMachineErr   string
		wantCreateVMErr          string
		wantCreateServiceErr     string
		ClientCreateVMError      error
		ClientCreateServiceError error
		labels                   map[string]string
		providerID               string
		wantVMToBeReady          bool
	}{
		{
			name:                     "Create a VM",
			wantValidateMachineErr:   "",
			wantCreateVMErr:          "",
			ClientCreateVMError:      nil,
			labels:                   nil,
			providerID:               "",
			wantVMToBeReady:          true,
			wantCreateServiceErr:     "",
			ClientCreateServiceError: nil,
		},
		{
			name:                     "Create a VM but fail on create a sevice",
			wantValidateMachineErr:   "",
			wantCreateVMErr:          "",
			ClientCreateVMError:      nil,
			labels:                   nil,
			providerID:               "",
			wantVMToBeReady:          true,
			wantCreateServiceErr:     "failed to create service: client error",
			ClientCreateServiceError: errors.New("client error"),
		},
		{
			name:                   "Create a VM from unlabeled machine and fail",
			wantValidateMachineErr: fmt.Sprintf("%s: failed validating machine provider spec: %v: missing %q label", mahcineName, mahcineName, machinev1.MachineClusterIDLabel),
			wantCreateVMErr:        "",
			ClientCreateVMError:    nil,
			labels:                 map[string]string{machinev1.MachineClusterIDLabel: ""},
			providerID:             "",
			wantVMToBeReady:        true,
		},
		{
			name:                   "Create a VM with an error in the client-go and fail",
			wantValidateMachineErr: "",
			wantCreateVMErr:        "failed to create virtual machine: client error",
			ClientCreateVMError:    errors.New("client error"),
			labels:                 nil,
			providerID:             "",
			wantVMToBeReady:        true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockUnderkube := mockunderkube.NewMockClient(mockCtrl)
			mockOvernderkube := mockoverkube.NewMockClient(mockCtrl)
			machine := initializeMachine(t, mockUnderkube, tc.labels, tc.providerID)
			if machine == nil {
				t.Fatalf("Unable to create the stub machine object")
			}

			kubevirtClientMockBuilder := func(overkubeClient overkube.Client, secretName, namespace string) (underkube.Client, error) {
				return mockUnderkube, nil
			}

			machineScope, err := stubMachineScope(machine, mockOvernderkube, kubevirtClientMockBuilder)
			if err != nil {
				t.Fatalf("Unable to build virtual machine with error: %v", err)
			}

			virtualMachine := stubVirtualMachine(machineScope)
			vmi, _ := stubVmi(virtualMachine)

			returnVM := stubVirtualMachine(machineScope)
			returnVM.Status.Ready = tc.wantVMToBeReady

			// TODO: test negative flow, return err != nil
			mockUnderkube.EXPECT().CreateVirtualMachine(clusterID, virtualMachine).Return(returnVM, tc.ClientCreateVMError).AnyTimes()
			mockUnderkube.EXPECT().GetVirtualMachineInstance(clusterID, virtualMachine.Name, gomock.Any()).Return(vmi, nil).AnyTimes()

			if tc.wantCreateServiceErr == "" {
				mockUnderkube.EXPECT().CreateService(gomock.Any(), virtualMachine.Namespace).Return(stubService(virtualMachine.Name), nil).AnyTimes()
			} else {
				mockUnderkube.EXPECT().CreateService(gomock.Any(), virtualMachine.Namespace).Return(nil, tc.ClientCreateServiceError).AnyTimes()
			}

			mockOvernderkube.EXPECT().PatchMachine(machine, machine.DeepCopy()).Return(nil).AnyTimes()
			mockOvernderkube.EXPECT().StatusPatchMachine(machine, machine.DeepCopy()).Return(nil).AnyTimes()
			mockOvernderkube.EXPECT().GetSecret(workerUserDataSecretName, machine.Namespace).Return(stubSecret(), nil).AnyTimes()

			providerVMInstance := New(kubevirtClientMockBuilder, mockOvernderkube)
			err = providerVMInstance.Create(machine)
			if tc.wantValidateMachineErr != "" {
				assert.Equal(t, tc.wantValidateMachineErr, err.Error())
			} else if tc.wantCreateVMErr != "" {
				assert.Equal(t, tc.wantCreateVMErr, err.Error())
			} else if tc.wantCreateServiceErr != "" {
				assert.Equal(t, tc.wantCreateServiceErr, err.Error())
			} else {
				assert.Equal(t, err, nil)
			}
		})
	}

}

func TestDelete(t *testing.T) {
	// TODO add a case of setProviderID and setMachineAnnotationsAndLabels failure
	cases := []struct {
		name                     string
		wantValidateMachineErr   string
		wantGetVMErr             string
		wantDeleteVMErr          string
		wantGetServiceErr        string
		wantDeleteServiceErr     string
		clientGetVMError         error
		clientDeleteVMError      error
		ClientDeleteServiceError error
		ClientGetServiceError    error
		emptyGetVM               bool
		labels                   map[string]string
		providerID               string
	}{
		{
			name:                     "Delete a VM successfully",
			wantValidateMachineErr:   "",
			wantGetVMErr:             "",
			clientGetVMError:         nil,
			wantDeleteVMErr:          "",
			clientDeleteVMError:      nil,
			emptyGetVM:               false,
			labels:                   nil,
			providerID:               "",
			wantDeleteServiceErr:     "",
			wantGetServiceErr:        "",
			ClientDeleteServiceError: nil,
			ClientGetServiceError:    nil,
		},
		{
			name:                     "Delete a VM successfully but fail on get service",
			wantValidateMachineErr:   "",
			wantGetVMErr:             "",
			clientGetVMError:         nil,
			wantDeleteVMErr:          "",
			clientDeleteVMError:      nil,
			emptyGetVM:               false,
			labels:                   nil,
			providerID:               "",
			wantDeleteServiceErr:     "",
			wantGetServiceErr:        "failed to delete the service of VM: client error",
			ClientDeleteServiceError: nil,
			ClientGetServiceError:    errors.New("client error"),
		},
		{
			name:                     "Delete a VM successfully but fail on delete service",
			wantValidateMachineErr:   "",
			wantGetVMErr:             "",
			clientGetVMError:         nil,
			wantDeleteVMErr:          "",
			clientDeleteVMError:      nil,
			emptyGetVM:               false,
			labels:                   nil,
			providerID:               "",
			wantDeleteServiceErr:     "failed to delete the service of VM: client error",
			wantGetServiceErr:        "",
			ClientDeleteServiceError: errors.New("client error"),
			ClientGetServiceError:    nil,
		},
		{
			name:                   "Delete a VM from unlabeled machine and fail",
			wantValidateMachineErr: fmt.Sprintf("%s: failed validating machine provider spec: %v: missing %q label", mahcineName, mahcineName, machinev1.MachineClusterIDLabel),
			wantGetVMErr:           "",
			clientGetVMError:       nil,
			wantDeleteVMErr:        "",
			clientDeleteVMError:    nil,
			emptyGetVM:             false,
			labels:                 map[string]string{machinev1.MachineClusterIDLabel: ""},
			providerID:             "",
		},
		{
			name:                   "Delete deleting VM with getting error and fail",
			wantValidateMachineErr: "",
			wantGetVMErr:           "client error",
			clientGetVMError:       errors.New("client error"),
			wantDeleteVMErr:        "",
			clientDeleteVMError:    nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             "",
		},
		{
			name:                   "Delete a nonexistent VM and fail",
			wantValidateMachineErr: "",
			wantGetVMErr:           "",
			clientGetVMError:       nil,
			wantDeleteVMErr:        "",
			clientDeleteVMError:    nil,
			emptyGetVM:             true,
			labels:                 nil,
			providerID:             "",
		},
		{
			name:                   "Delete a VM with an error in the client-go and fail",
			wantValidateMachineErr: "",
			wantGetVMErr:           "",
			clientGetVMError:       nil,
			wantDeleteVMErr:        "failed to delete VM: client error",
			clientDeleteVMError:    errors.New("client error"),
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockUnderkube := mockunderkube.NewMockClient(mockCtrl)
			mockOvernderkube := mockoverkube.NewMockClient(mockCtrl)

			machine := initializeMachine(t, mockUnderkube, tc.labels, tc.providerID)
			if machine == nil {
				t.Fatalf("Unable to create the stub machine object")
			}

			kubevirtClientMockBuilder := func(kubernetesClient overkube.Client, secretName, namespace string) (underkube.Client, error) {
				return mockUnderkube, nil
			}

			machineScope, err := stubMachineScope(machine, mockOvernderkube, kubevirtClientMockBuilder)
			if err != nil {
				t.Fatalf("Unable to build virtual machine with error: %v", err)
			}

			virtualMachine := stubVirtualMachine(machineScope)
			vmi, _ := stubVmi(virtualMachine)

			var returnVM *kubevirtapiv1.VirtualMachine
			if !tc.emptyGetVM {
				returnVM = virtualMachine
			}

			//underkube mocks
			mockUnderkube.EXPECT().GetVirtualMachine(clusterID, virtualMachine.Name, gomock.Any()).Return(returnVM, tc.clientGetVMError).AnyTimes()
			mockUnderkube.EXPECT().DeleteVirtualMachine(clusterID, virtualMachine.Name, gomock.Any()).Return(tc.clientDeleteVMError).AnyTimes()
			mockUnderkube.EXPECT().GetVirtualMachineInstance(clusterID, virtualMachine.Name, gomock.Any()).Return(vmi, nil).AnyTimes()

			if tc.wantGetServiceErr == "" {
				mockUnderkube.EXPECT().GetService(virtualMachine.Name, virtualMachine.Namespace, gomock.Any()).Return(stubService(virtualMachine.Name), nil).AnyTimes()
			} else {
				mockUnderkube.EXPECT().GetService(virtualMachine.Name, virtualMachine.Namespace, gomock.Any()).Return(nil, tc.ClientGetServiceError).AnyTimes()
			}
			if tc.wantDeleteServiceErr == "" {
				mockUnderkube.EXPECT().DeleteService(virtualMachine.Name, virtualMachine.Namespace, gomock.Any()).Return(nil).AnyTimes()
			} else {
				mockUnderkube.EXPECT().DeleteService(virtualMachine.Name, virtualMachine.Namespace, gomock.Any()).Return(tc.ClientDeleteServiceError).AnyTimes()
			}

			//overkube mocks
			// TODO: test negative flow, return err != nil
			mockOvernderkube.EXPECT().PatchMachine(machine, machine.DeepCopy()).Return(nil).AnyTimes()
			mockOvernderkube.EXPECT().StatusPatchMachine(machine, machine.DeepCopy()).Return(nil).AnyTimes()
			mockOvernderkube.EXPECT().GetSecret(workerUserDataSecretName, machine.Namespace).Return(stubSecret(), nil).AnyTimes()

			providerVMInstance := New(kubevirtClientMockBuilder, mockOvernderkube)
			err = providerVMInstance.Delete(machine)

			// getServicErr
			// deleteServiceErr
			if tc.wantValidateMachineErr != "" {
				assert.Equal(t, tc.wantValidateMachineErr, err.Error())
			} else if tc.wantGetVMErr != "" {
				assert.Equal(t, tc.wantGetVMErr, err.Error())
			} else if tc.wantDeleteVMErr != "" {
				assert.Equal(t, tc.wantDeleteVMErr, err.Error())
			} else if tc.wantGetServiceErr != "" {
				assert.Equal(t, tc.wantGetServiceErr, err.Error())
			} else if tc.wantDeleteServiceErr != "" {
				assert.Equal(t, tc.wantDeleteServiceErr, err.Error())
			} else {
				assert.Equal(t, err, nil)
			}
		})
	}

}

func TestExists(t *testing.T) {
	// TODO add a case of setProviderID and setMachineAnnotationsAndLabels failure
	cases := []struct {
		name           string
		clientGetError error
		emptyGetVM     bool
		isExist        bool
		labels         map[string]string
		providerID     string
	}{
		{
			name:           "Validate existence VM",
			clientGetError: nil,
			emptyGetVM:     false,
			isExist:        true,
			labels:         nil,
			providerID:     "",
		},
		{
			name:           "Validate non existence VM",
			clientGetError: nil,
			emptyGetVM:     true,
			isExist:        false,
			labels:         nil,
			providerID:     "",
		},
		{
			name:           "Validate an error in get VM",
			clientGetError: errors.New("client error"),
			emptyGetVM:     true,
			isExist:        false,
			labels:         nil,
			providerID:     "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockUnderkube := mockunderkube.NewMockClient(mockCtrl)
			mockOvernderkube := mockoverkube.NewMockClient(mockCtrl)

			machine := initializeMachine(t, mockUnderkube, tc.labels, tc.providerID)
			if machine == nil {
				t.Fatalf("Unable to create the stub machine object")
			}

			kubevirtClientMockBuilder := func(kubernetesClient overkube.Client, secretName, namespace string) (underkube.Client, error) {
				return mockUnderkube, nil
			}

			machineScope, err := stubMachineScope(machine, mockOvernderkube, kubevirtClientMockBuilder)
			if err != nil {
				t.Fatalf("Unable to build virtual machine with error: %v", err)
			}

			virtualMachine := stubVirtualMachine(machineScope)
			vmi, _ := stubVmi(virtualMachine)

			var returnVM *kubevirtapiv1.VirtualMachine
			if !tc.emptyGetVM {
				returnVM = virtualMachine
			}

			//underkube mocks
			mockUnderkube.EXPECT().GetVirtualMachine(clusterID, virtualMachine.Name, gomock.Any()).Return(returnVM, tc.clientGetError).AnyTimes()
			mockUnderkube.EXPECT().GetVirtualMachineInstance(clusterID, virtualMachine.Name, gomock.Any()).Return(vmi, nil).AnyTimes()
			mockOvernderkube.EXPECT().GetSecret(workerUserDataSecretName, machine.Namespace).Return(stubSecret(), nil).AnyTimes()

			providerVMInstance := New(kubevirtClientMockBuilder, mockOvernderkube)
			existsVM, err := providerVMInstance.Exists(machine)

			if tc.clientGetError != nil {
				assert.Equal(t, tc.clientGetError.Error(), err.Error())
			} else if tc.emptyGetVM {
				assert.Equal(t, err, nil)
				assert.Equal(t, existsVM, false)
			} else {
				assert.Equal(t, err, nil)
				assert.Equal(t, existsVM, tc.isExist)
			}
		})
	}

}

func TestUpdate(t *testing.T) {
	// TODO add a case of setProviderID and setMachineAnnotationsAndLabels failure
	cases := []struct {
		name                     string
		wantValidateMachineErr   string
		wantUpdateVMErr          string
		wantCreateServiceErr     string
		wantGetServiceErr        string
		clientGetVMError         error
		clientUpdateVMError      error
		clientCreateServiceError error
		clientGetServiceError    error
		emptyGetVM               bool
		labels                   map[string]string
		providerID               string
		wantVMToBeReady          bool
	}{
		{
			name:                   "Update a VM and create a service",
			wantValidateMachineErr: "",
			wantUpdateVMErr:        "",
			clientGetVMError:       nil,
			clientUpdateVMError:    nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             fmt.Sprintf("kubevirt:///%s/%s", defaultNamespace, mahcineName),
			wantVMToBeReady:        true,
			wantGetServiceErr:      "service not found",
			clientGetServiceError:  errors.New("service not found"),
		},
		{
			name:                   "Update a VM but fail on get service",
			wantValidateMachineErr: "",
			wantUpdateVMErr:        "",
			clientGetVMError:       nil,
			clientUpdateVMError:    nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             fmt.Sprintf("kubevirt:///%s/%s", defaultNamespace, mahcineName),
			wantVMToBeReady:        true,
			wantGetServiceErr:      "machine-test: error getting service of VM: client error",
			clientGetServiceError:  errors.New("client error"),
		},
		{
			name:                     "Update a VM but fail on create service",
			wantValidateMachineErr:   "",
			wantUpdateVMErr:          "",
			clientGetVMError:         nil,
			clientUpdateVMError:      nil,
			emptyGetVM:               false,
			labels:                   nil,
			providerID:               fmt.Sprintf("kubevirt:///%s/%s", defaultNamespace, mahcineName),
			wantVMToBeReady:          true,
			wantCreateServiceErr:     "failed to create service: client error",
			clientCreateServiceError: errors.New("client error"),
			wantGetServiceErr:        "service not found",
			clientGetServiceError:    errors.New("service not found"),
		},
		// TODO: enable that test after pushing the PR: https://github.com/kubevirt/kubevirt/pull/3889 so update wouldn't override the vm Status
		//{
		//	name:                   "Update a VM that never be ready",
		//	wantValidateMachineErr: "",
		//	wantUpdateVMErr:        "",
		//	clientGetVMError:       nil,
		//	clientUpdateVMError:    nil,
		//	emptyGetVM:             false,
		//	labels:                 nil,
		//	providerID:             "",
		//	wantVMToBeReady:        false,
		//},
		{
			name:                   "Update a VM from unlabeled machine and fail",
			wantValidateMachineErr: fmt.Sprintf("%s: failed validating machine provider spec: %v: missing %q label", mahcineName, mahcineName, machinev1.MachineClusterIDLabel),
			wantUpdateVMErr:        "",
			clientGetVMError:       nil,
			clientUpdateVMError:    nil,
			emptyGetVM:             false,
			labels:                 map[string]string{machinev1.MachineClusterIDLabel: ""},
			providerID:             "",
			wantVMToBeReady:        false,
		},
		{
			name:                   "Update a VM with getting error and fail",
			wantValidateMachineErr: "",
			wantUpdateVMErr:        "",
			clientGetVMError:       errors.New("client error"),
			clientUpdateVMError:    nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             "",
			wantVMToBeReady:        false,
		},
		{
			name:                   "Update a nonexistent VM and fail",
			wantValidateMachineErr: "",
			wantUpdateVMErr:        "",
			clientGetVMError:       nil,
			clientUpdateVMError:    nil,
			emptyGetVM:             true,
			labels:                 nil,
			providerID:             "",
			wantVMToBeReady:        false,
		},
		{
			name:                   "Delete a VM with an error in the client-go and fail",
			wantValidateMachineErr: "",
			clientGetVMError:       nil,
			clientUpdateVMError:    errors.New("client error"),
			wantUpdateVMErr:        "failed to update VM: client error",
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             "",
			wantVMToBeReady:        false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockUnderkube := mockunderkube.NewMockClient(mockCtrl)
			mockOvernderkube := mockoverkube.NewMockClient(mockCtrl)

			machine := initializeMachine(t, mockUnderkube, tc.labels, tc.providerID)
			if machine == nil {
				t.Fatalf("Unable to create the stub machine object")
			}

			kubevirtClientMockBuilder := func(kubernetesClient overkube.Client, secretName, namespace string) (underkube.Client, error) {
				return mockUnderkube, nil
			}

			machineScope, err := stubMachineScope(machine, mockOvernderkube, kubevirtClientMockBuilder)
			if err != nil {
				t.Fatalf("Unable to build virtual machine with error: %v", err)
			}

			virtualMachine := stubVirtualMachine(machineScope)
			vmi, _ := stubVmi(virtualMachine)
			var getReturnVM *kubevirtapiv1.VirtualMachine
			if !tc.emptyGetVM {
				returnVMResult := stubVirtualMachine(machineScope)
				getReturnVM = returnVMResult
				getReturnVM.Status = kubevirtapiv1.VirtualMachineStatus{
					Created: true,
					Ready:   tc.wantVMToBeReady,
				}
				getReturnVM.Status.Created = true
				getReturnVM.Status.Ready = tc.wantVMToBeReady

			}

			updateReturnVM := stubVirtualMachine(machineScope)
			updateReturnVM.Status = kubevirtapiv1.VirtualMachineStatus{
				Created: true,
				Ready:   tc.wantVMToBeReady,
			}

			mockUnderkube.EXPECT().GetVirtualMachine(clusterID, virtualMachine.Name, gomock.Any()).Return(getReturnVM, tc.clientGetVMError).AnyTimes()
			mockUnderkube.EXPECT().UpdateVirtualMachine(clusterID, getReturnVM).Return(updateReturnVM, tc.clientUpdateVMError).AnyTimes()
			mockUnderkube.EXPECT().GetVirtualMachineInstance(clusterID, virtualMachine.Name, gomock.Any()).Return(vmi, nil).AnyTimes()

			if tc.wantGetServiceErr == "" {
				mockUnderkube.EXPECT().GetService(virtualMachine.Name, virtualMachine.Namespace, gomock.Any()).Return(stubService(virtualMachine.Name), nil).AnyTimes()
			} else {
				mockUnderkube.EXPECT().GetService(virtualMachine.Name, virtualMachine.Namespace, gomock.Any()).Return(nil, tc.clientGetServiceError).AnyTimes()
			}
			if tc.wantCreateServiceErr == "" {
				mockUnderkube.EXPECT().CreateService(gomock.Any(), virtualMachine.Namespace).Return(stubService(virtualMachine.Name), nil).AnyTimes()

			} else {
				mockUnderkube.EXPECT().CreateService(gomock.Any(), virtualMachine.Namespace).Return(nil, tc.clientCreateServiceError).AnyTimes()
			}

			// TODO: test negative flow, return err != nil
			mockOvernderkube.EXPECT().PatchMachine(machine, machine.DeepCopy()).Return(nil).AnyTimes()
			mockOvernderkube.EXPECT().StatusPatchMachine(machine, machine.DeepCopy()).Return(nil).AnyTimes()
			mockOvernderkube.EXPECT().GetSecret(workerUserDataSecretName, machine.Namespace).Return(stubSecret(), nil).AnyTimes()

			providerVMInstance := New(kubevirtClientMockBuilder, mockOvernderkube)
			// TODO: test the bool wasUpdated
			_, err = providerVMInstance.Update(machine)

			if tc.wantValidateMachineErr != "" {
				assert.Equal(t, tc.wantValidateMachineErr, err.Error())
			} else if tc.clientGetVMError != nil {
				assert.Equal(t, tc.clientGetVMError.Error(), err.Error())
			} else if tc.wantUpdateVMErr != "" {
				assert.Equal(t, tc.wantUpdateVMErr, err.Error())
			} else if tc.emptyGetVM {
				assert.Equal(t, err.Error(), "requeue in: 3m0s")
			} else if !tc.wantVMToBeReady {
				assert.Equal(t, err.Error(), "requeue in: 20s")
			} else if tc.wantCreateServiceErr != "" {
				assert.Equal(t, tc.wantCreateServiceErr, err.Error())
			} else if tc.wantGetServiceErr != "" && tc.wantGetServiceErr != "service not found" {
				assert.Equal(t, tc.wantGetServiceErr, err.Error())
			} else {
				assert.Equal(t, err, nil)
				//providerID := fmt.Sprintf("kubevirt:///%s/%s", machineScope.machine.GetNamespace(), machineScope.virtualMachine.GetName())
				assert.Equal(t, *machine.Spec.ProviderID, tc.providerID)
			}
		})
	}

}

// func DefaultVirtualMachine(started bool) (*kubevirtapiv1.VirtualMachine, *kubevirtapiv1.VirtualMachineInstance) {
// 	return DefaultVirtualMachineWithNames(started, "testvmi", "testvmi")
// }

// func DefaultVirtualMachineWithNames(started bool, vmName string, vmiName string) (*kubevirtapiv1.VirtualMachine, *kubevirtapiv1.VirtualMachineInstance) {
// 	vmi := kubevirtapiv1.NewMinimalVMI(vmiName)
// 	vmi.Status.Phase = kubevirtapiv1.Running
// 	vm := VirtualMachineFromVMI(vmName, vmi, started)
// 	t := true
// 	vmi.OwnerReferences = []metav1.OwnerReference{{
// 		APIVersion:         kubevirtapiv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
// 		Kind:               kubevirtapiv1.VirtualMachineGroupVersionKind.Kind,
// 		Name:               vm.ObjectMeta.Name,
// 		UID:                vm.ObjectMeta.UID,
// 		Controller:         &t,
// 		BlockOwnerDeletion: &t,
// 	}}
// 	return vm, vmi
// }

// func VirtualMachineFromVMI(name string, vmi *kubevirtapiv1.VirtualMachineInstance, started bool) *kubevirtapiv1.VirtualMachine {
// 	vm := &kubevirtapiv1.VirtualMachine{
// 		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vmi.ObjectMeta.Namespace, ResourceVersion: "1"},
// 		Spec: kubevirtapiv1.VirtualMachineSpec{
// 			Running: &started,
// 			Template: &kubevirtapiv1.VirtualMachineInstanceTemplateSpec{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:   vmi.ObjectMeta.Name,
// 					Labels: vmi.ObjectMeta.Labels,
// 				},
// 				Spec: vmi.Spec,
// 			},
// 		},
// 	}
// 	return vm
// }
