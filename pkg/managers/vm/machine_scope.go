package vm

import (
	"fmt"
	"time"

	machinecontroller "github.com/openshift/machine-api-operator/pkg/controller/machine"

	kubevirtproviderv1 "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/apis/kubevirtprovider/v1"
	kubernetesclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/clients/kubernetes"
	kubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/clients/kubevirt"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	machineapierros "github.com/openshift/machine-api-operator/pkg/controller/machine"
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/klog"
	kubevirtapiv1 "kubevirt.io/client-go/api/v1"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

type machineState string

const (
	userDataSecretKey                              = "userData"
	pvcRequestsStorage                             = "35Gi"
	defaultRequestedMemory                         = "2048M"
	defaultPersistentVolumeAccessMode              = corev1.ReadWriteOnce
	defaultDataVolumeDiskName                      = "datavolumedisk1"
	kubevirtIdAnnotationKey                        = "VmId"
	vmNotCreated                      machineState = "vmNotCreated"
	vmCreatedNotReady                 machineState = "vmWasCreatedButNotReady"
	vmCreatedAndReady                 machineState = "vmWasCreatedButAndReady"
)

type machineScope struct {
	kubevirtClient        kubevirtclient.Client
	kubernetesClient      kubernetesclient.Client
	machine               *machinev1.Machine
	originMachineCopy     *machinev1.Machine
	machineProviderSpec   *kubevirtproviderv1.KubevirtMachineProviderSpec
	machineProviderStatus *kubevirtproviderv1.KubevirtMachineProviderStatus
}

func newMachineScope(machine *machinev1.Machine, kubernetesClient kubernetesclient.Client, kubevirtClientBuilder kubevirtclient.ClientBuilderFuncType) (*machineScope, error) {
	if err := validateMachine(*machine); err != nil {
		return nil, fmt.Errorf("%v: failed validating machine provider spec: %w", machine.GetName(), err)
	}

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

func machineToVirtualMachine(machineScope *machineScope) (*kubevirtapiv1.VirtualMachine, error) {
	runAlways := kubevirtapiv1.RunStrategyAlways

	// use getClusterID as a namespace
	// TODO: if there isnt a cluster id - need to return an error
	namespace, ok := getClusterID(machineScope.machine)
	if !ok {
		namespace = machineScope.machine.Namespace
	}

	virtualMachine := kubevirtapiv1.VirtualMachine{
		Spec: kubevirtapiv1.VirtualMachineSpec{
			RunStrategy: &runAlways,
			DataVolumeTemplates: []cdiv1.DataVolume{
				*buildBootVolumeDataVolumeTemplate(machineScope.machine.GetName(), machineScope.machineProviderSpec.SourcePvcName, namespace, machineScope.machineProviderSpec.SourcePvcNamespace),
			},
			Template: buildVMITemplate(machineScope.machine.GetName(), machineScope.machineProviderSpec),
		},
		Status: machineScope.machineProviderStatus.VirtualMachineStatus,
	}

	virtualMachine.TypeMeta = machineScope.machine.TypeMeta
	virtualMachine.ObjectMeta = metav1.ObjectMeta{
		Name:            machineScope.machine.Name,
		Namespace:       namespace,
		Labels:          machineScope.machine.Labels,
		Annotations:     machineScope.machine.Annotations,
		OwnerReferences: machineScope.machine.OwnerReferences,
		ClusterName:     machineScope.machine.ClusterName,
		ResourceVersion: machineScope.machineProviderStatus.ResourceVersion,
	}

	return &virtualMachine, nil
}

func (s *machineScope) getMachineName() string {
	return s.machine.GetName()
}

func (s *machineScope) getMachineNamespace() string {
	return s.machine.GetNamespace()
}

// setProviderID adds providerID in the machine spec
func (s *machineScope) setProviderID(vm *kubevirtapiv1.VirtualMachine) {
	// TODO: return an error when the setting is failed
	existingProviderID := s.machine.Spec.ProviderID
	if vm == nil {
		return
	}

	providerID := fmt.Sprintf("kubevirt:///%s/%s", s.getMachineNamespace(), vm.GetName())

	if existingProviderID != nil && *existingProviderID == providerID {
		klog.Infof("%s: ProviderID already set in the machine Spec with value:%s", s.getMachineName(), *existingProviderID)
		return
	}

	s.machine.Spec.ProviderID = &providerID
	klog.Infof("%s: ProviderID set at machine spec: %s", s.getMachineName(), providerID)
}

// updateAllowed validates that updates come in the right order
// if there is an update that was supposes to be done after that update - return an error
func (s *machineScope) updateAllowed() bool {
	return s.machine.Spec.ProviderID != nil && *s.machine.Spec.ProviderID != "" && (s.machine.Status.LastUpdated == nil || s.machine.Status.LastUpdated.Add(requeueAfterSeconds*time.Second).After(time.Now()))
}

func (s *machineScope) setMachineCloudProviderSpecifics(vm *kubevirtapiv1.VirtualMachine) error {
	if vm == nil {
		return nil
	}

	if s.machine.Labels == nil {
		s.machine.Labels = make(map[string]string)
	}

	if s.machine.Spec.Labels == nil {
		s.machine.Spec.Labels = make(map[string]string)
	}

	if s.machine.Annotations == nil {
		s.machine.Annotations = make(map[string]string)
	}
	vmId := vm.UID
	vmType := vm.Spec.Template.Spec.Domain.Machine.Type
	vmCreated := vm.Status.Created
	vmReady := vm.Status.Ready

	vmState := vmNotCreated
	if vmCreated {
		vmState = vmCreatedNotReady
		if vmReady {
			vmState = vmCreatedAndReady
		}
	}

	s.machine.ObjectMeta.Annotations[kubevirtIdAnnotationKey] = string(vmId)
	s.machine.Labels[machinecontroller.MachineInstanceTypeLabelName] = vmType
	s.machine.Annotations[machinecontroller.MachineInstanceStateAnnotationName] = string(vmState)

	return nil
}

func buildDataVolumeDiskName(virtualMachineName string) string {
	return virtualMachineName + defaultDataVolumeDiskName
}

func buildVMITemplate(virtualMachineName string, providerSpec *kubevirtproviderv1.KubevirtMachineProviderSpec) *kubevirtapiv1.VirtualMachineInstanceTemplateSpec {
	template := &kubevirtapiv1.VirtualMachineInstanceTemplateSpec{}

	template.ObjectMeta = metav1.ObjectMeta{
		Labels: map[string]string{"kubevirt.io/vm": virtualMachineName},
	}

	template.Spec = kubevirtapiv1.VirtualMachineInstanceSpec{}
	template.Spec.Volumes = []kubevirtapiv1.Volume{
		{
			// TODO : use the machine-name in order to determine the volume
			Name: buildDataVolumeDiskName(virtualMachineName),
			VolumeSource: kubevirtapiv1.VolumeSource{
				DataVolume: &kubevirtapiv1.DataVolumeSource{
					Name: buildBootVolumeName(virtualMachineName),
				},
			},
		},
	}

	template.Spec.Domain = kubevirtapiv1.DomainSpec{}

	requests := corev1.ResourceList{}

	requestedMemory := providerSpec.RequestedMemory
	if requestedMemory == "" {
		requestedMemory = defaultRequestedMemory
	}
	requests[corev1.ResourceMemory] = apiresource.MustParse(requestedMemory)

	if providerSpec.RequestedCPU != "" {
		requests[corev1.ResourceCPU] = apiresource.MustParse(providerSpec.RequestedCPU)
	}

	template.Spec.Domain.Resources = kubevirtapiv1.ResourceRequirements{
		Requests: requests,
	}
	// TODO: get the machine type from machine.yaml
	template.Spec.Domain.Machine = kubevirtapiv1.Machine{Type: providerSpec.MachineType}
	template.Spec.Domain.Devices = kubevirtapiv1.Devices{
		Disks: []kubevirtapiv1.Disk{
			{
				Name: buildDataVolumeDiskName(virtualMachineName),
			},
		},
	}

	return template
}

func buildBootVolumeName(virtualMachineName string) string {
	return virtualMachineName + "-bootvolume"
}

func buildBootVolumeDataVolumeTemplate(virtualMachineName, pvcName, dvNamespace, pvcNamespace string) *cdiv1.DataVolume {
	return &cdiv1.DataVolume{
		TypeMeta: metav1.TypeMeta{APIVersion: cdiv1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildBootVolumeName(virtualMachineName),
			Namespace: dvNamespace,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				PVC: &cdiv1.DataVolumeSourcePVC{
					Name:      pvcName,
					Namespace: dvNamespace,
					//Namespace: pvcNamespace,
				},
			},
			PVC: &corev1.PersistentVolumeClaimSpec{
				// TODO: Need to determine it by the type of storage class: pvc.Spec.StorageClassName
				AccessModes: []corev1.PersistentVolumeAccessMode{
					defaultPersistentVolumeAccessMode,
				},
				// TODO: Where to get it?? - add as a list
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: apiresource.MustParse(pvcRequestsStorage),
					},
				},
			},
		},
	}
}

func (s *machineScope) SyncMachineFromVm(vm *kubevirtapiv1.VirtualMachine, vmi *kubevirtapiv1.VirtualMachineInstance) error {
	// TODO this function need to be removed
	s.setProviderID(vm)

	// TODO  this function need to be removed
	if err := s.setMachineCloudProviderSpecifics(vm); err != nil {
		return fmt.Errorf("failed to set machine cloud provider specifics: %w", err)
	}

	// TODO this function need to be removed
	klog.Infof("Updated machine %s", s.getMachineName())
	if err := s.setProviderStatus(vm, vmi, conditionSuccess()); err != nil {
		return machineapierros.InvalidMachineConfiguration("failed to set machine provider status: %v", err.Error())
	}

	return nil
}

// Patch patches the machine spec and machine status after reconciling.
func (s *machineScope) patchMachine() error {

	klog.V(3).Infof("%v: patching machine", s.machine.GetName())

	providerStatus, err := kubevirtproviderv1.RawExtensionFromProviderStatus(s.machineProviderStatus)
	if err != nil {
		return machineapierros.InvalidMachineConfiguration("failed to get machine provider status: %v", err.Error())
	}
	s.machine.Status.ProviderStatus = providerStatus

	// patch machine
	statusCopy := *s.machine.Status.DeepCopy()
	if err := s.kubernetesClient.PatchMachine(s.machine, s.originMachineCopy); err != nil {
		klog.Errorf("Failed to patch machine %q: %v", s.machine.GetName(), err)
		return err
	}

	s.machine.Status = statusCopy

	// patch status
	if err := s.kubernetesClient.StatusPatchMachine(s.machine, s.originMachineCopy); err != nil {
		klog.Errorf("Failed to patch machine status %q: %v", s.machine.GetName(), err)
		return err
	}

	return nil
}

func machineProviderStatusFromVirtualMachine(virtualMachine *kubevirtapiv1.VirtualMachine) *kubevirtproviderv1.KubevirtMachineProviderStatus {
	result := kubevirtproviderv1.KubevirtMachineProviderStatus{
		VirtualMachineStatus: virtualMachine.Status,
		ResourceVersion:      virtualMachine.ResourceVersion,
	}
	return &result
}

// Is other field need to be updated?
// Why in aws also update s.machine.Status.Addresses?
func (s *machineScope) setProviderStatus(vm *kubevirtapiv1.VirtualMachine, vmi *kubevirtapiv1.VirtualMachineInstance, condition kubevirtapiv1.VirtualMachineCondition) error {
	klog.Infof("%s: Updating status", s.machine.GetName())

	var networkAddresses []corev1.NodeAddress

	if vm != nil {
		s.machineProviderStatus = machineProviderStatusFromVirtualMachine(vm)
		//s.virtualMachine.Status = vm.Status
		// update nodeAddresses
		networkAddresses = append(networkAddresses, corev1.NodeAddress{Address: vm.Name, Type: corev1.NodeInternalDNS})
		// Copy specific addresses - only node addresses
		addresses, err := extractNodeAddresses(vmi)
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
	// TODO: update it
	//s.virtualMachine.Status.Conditions = setKubevirtMachineProviderCondition(condition, s.virtualMachine.Status.Conditions)

	return nil
}

// GetMachineName return the name of the provided Machine
func GetMachineName(machine *machinev1.Machine) string {
	return machine.GetName()
}
