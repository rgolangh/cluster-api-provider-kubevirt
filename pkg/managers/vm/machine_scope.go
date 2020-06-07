package vm

import (
	"fmt"
	"time"

	kubevirtproviderv1 "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/apis/kubevirtprovider/v1"
	kubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	machineapierros "github.com/openshift/machine-api-operator/pkg/controller/machine"
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesclient "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	kubevirtapiv1 "kubevirt.io/client-go/api/v1"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

const (
	userDataSecretKey                 = "userData"
	pvcRequestsStorage                = "35Gi"
	defaultRequestedMemory            = "2048M"
	defaultPersistentVolumeAccessMode = corev1.ReadWriteOnce
	defaultDataVolumeDiskName         = "datavolumedisk1"
)

type machineScope struct {
	// api server controller runtime client
	kubevirtClient kubevirtclient.Client
	// machine resource
	machine        *machinev1.Machine
	virtualMachine *kubevirtapiv1.VirtualMachine
}

func newMachineScope(machine *machinev1.Machine, kubernetesClient *kubernetesclient.Clientset, kubevirtClientBuilder kubevirtclient.KubevirtClientBuilderFuncType) (*machineScope, error) {
	if validateMachineErr := validateMachine(*machine); validateMachineErr != nil {
		return nil, fmt.Errorf("%v: failed validating machine provider spec: %w", machine.GetName(), validateMachineErr)
	}

	providerSpec, err := kubevirtproviderv1.ProviderSpecFromRawExtension(machine.Spec.ProviderSpec.Value)
	if err != nil {
		return nil, machineapierros.InvalidMachineConfiguration("failed to get machine config: %v", err)
	}

	kubevirtClient, err := kubevirtClientBuilder(kubernetesClient, providerSpec.SecretName, machine.GetNamespace())

	if err != nil {
		return nil, machineapierros.InvalidMachineConfiguration("failed to create aKubeVirt client: %v", err.Error())
	}

	virtualMachine, virtualMachineErr := machineToVirtualMachine(machine, providerSpec.SourcePvcName)
	if virtualMachineErr != nil {
		return nil, virtualMachineErr
	}

	// TODO Nir - Add other virtualMachine params

	return &machineScope{
		kubevirtClient: kubevirtClient,
		machine:        machine,
		virtualMachine: virtualMachine,
	}, nil
}

func machineToVirtualMachine(machine *machinev1.Machine, sourcePvcName string) (*kubevirtapiv1.VirtualMachine, error) {
	//runningState := true
	runAlways := kubevirtapiv1.RunStrategyAlways

	providerStatus, err := kubevirtproviderv1.ProviderStatusFromRawExtension(machine.Status.ProviderStatus)
	if err != nil {
		return nil, machineapierros.InvalidMachineConfiguration("failed to get machine provider status: %v", err.Error())
	}

	virtualMachine := kubevirtapiv1.VirtualMachine{
		Spec: kubevirtapiv1.VirtualMachineSpec{
			//Running: &runningState,
			RunStrategy: &runAlways,
			DataVolumeTemplates: []cdiv1.DataVolume{
				*buildBootVolumeDataVolumeTemplate(machine.GetName(), sourcePvcName, machine.GetNamespace()),
			},
			Template: buildVMITemplate(machine.GetName()),
		},
		Status: providerStatus.VirtualMachineStatus,
	}

	virtualMachine.TypeMeta = machine.TypeMeta
	virtualMachine.ObjectMeta = metav1.ObjectMeta{
		Name: machine.Name,
		//GenerateName:               "",
		Namespace:       machine.Namespace,
		Labels:          machine.Labels,
		Annotations:     machine.Annotations,
		OwnerReferences: machine.OwnerReferences,
		ClusterName:     machine.ClusterName,
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

func (s *machineScope) updateAllowed() bool {
	// validate that updates come in the right order
	// if there is an update that was supposes to be done after that update - return an error
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

	// TODO which labels/annotations need to assign here?
	// Reaching to machine provider config since the region is not directly
	// providing by *kubevirtapiv1.VirtualMachine object
	//memory
	//storage
	//cpu
	////labels
	//machineProviderConfig, err := kubevirtproviderv1.ProviderSpecFromRawExtension(p.machine.Spec.ProviderSpec.Value)
	//
	//if err != nil {
	//	return fmt.Errorf("error decoding MachineProviderConfig: %w", err)
	//}
	//
	//p.machine.Labels[machinecontroller.MachineRegionLabelName] = machineProviderConfig.Placement.Region

	// if instance.Placement != nil {
	// 	p.machine.Labels[machinecontroller.MachineAZLabelName] = aws.StringValue(instance.Placement.AvailabilityZone)
	// }

	// if instance.InstanceType != nil {
	// 	p.machine.Labels[machinecontroller.MachineInstanceTypeLabelName] = aws.StringValue(instance.InstanceType)
	// }

	// if instance.State != nil && instance.State.Name != nil {
	// 	p.machine.Annotations[machinecontroller.MachineInstanceStateAnnotationName] = aws.StringValue(instance.State.Name)
	// }

	// if instance.InstanceLifecycle != nil && *instance.InstanceLifecycle == ec2.InstanceLifecycleTypeSpot {
	// 	// Label on the Spec so that it is propogated to the Node
	// 	p.machine.Spec.Labels[machinecontroller.MachineInterruptibleInstanceLabelName] = ""
	// }

	return nil
}

func buildVMITemplate(virtualMachineName string) *kubevirtapiv1.VirtualMachineInstanceTemplateSpec {
	template := &kubevirtapiv1.VirtualMachineInstanceTemplateSpec{}

	template.ObjectMeta = metav1.ObjectMeta{
		Labels: map[string]string{"kubevirt.io/vm": virtualMachineName},
	}

	template.Spec = kubevirtapiv1.VirtualMachineInstanceSpec{}
	template.Spec.Volumes = []kubevirtapiv1.Volume{
		{
			// TODO : use the machine-name in order to determine the volume
			Name: defaultDataVolumeDiskName,
			VolumeSource: kubevirtapiv1.VolumeSource{
				DataVolume: &kubevirtapiv1.DataVolumeSource{
					Name: buildBootVolumeName(virtualMachineName),
				},
			},
		},
	}

	template.Spec.Domain = kubevirtapiv1.DomainSpec{}

	requests := corev1.ResourceList{}
	//Memory:
	requests[corev1.ResourceMemory] = apiresource.MustParse(defaultRequestedMemory)

	//CPU:
	//requests[corev1.ResourceCPU] = apiresource.MustParse("2")

	template.Spec.Domain.Resources = kubevirtapiv1.ResourceRequirements{
		Requests: requests,
	}
	// TODO: get the machine type from machine.yaml
	template.Spec.Domain.Machine = kubevirtapiv1.Machine{Type: ""}
	template.Spec.Domain.Devices = kubevirtapiv1.Devices{
		Disks: []kubevirtapiv1.Disk{
			{
				Name: defaultDataVolumeDiskName,
			},
		},
	}

	return template
}

func buildBootVolumeName(virtualMachineName string) string {
	return virtualMachineName + "-bootvolume"
}
func buildBootVolumeDataVolumeTemplate(virtualMachineName string, pvcName string, namespace string) *cdiv1.DataVolume {
	// TODO Nir - add spec to data volume
	return &cdiv1.DataVolume{
		TypeMeta: metav1.TypeMeta{APIVersion: cdiv1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildBootVolumeName(virtualMachineName),
			Namespace: namespace,
		},
		// TODO: ERROR: [spec.pvc.resources.requests: Required value, spec.pvc.accessModes: Required value
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				PVC: &cdiv1.DataVolumeSourcePVC{
					Name:      pvcName,
					Namespace: namespace,
				},
			},
			PVC: &corev1.PersistentVolumeClaimSpec{
				// TODO: Need to determin it by the type of storage class
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

	//TODO: update the status -> from vm to the machine
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

// GetMachineName return the name of the provided Machine
func GetMachineName(machine *machinev1.Machine) string {
	return machine.GetName()
}

// GetMachineResourceVersion return the ResourceVersion of the provided Machine
func GetMachineResourceVersion(machine *machinev1.Machine) string {
	return machine.ResourceVersion
}
