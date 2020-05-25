module github.com/kubevirt/cluster-api-provider-kubevirt

go 1.13

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/golang/mock v1.2.0
	github.com/openshift/machine-api-operator v0.2.1-0.20200402110321-4f3602b96da3
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	// k8s.io/client-go v12.0.0+incompatible
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	kubevirt.io/client-go v0.29.0
	kubevirt.io/containerized-data-importer v1.10.6
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/yaml v1.2.0
)

replace k8s.io/client-go => k8s.io/client-go v0.18.3

replace sigs.k8s.io/controller-runtime => github.com/munnerz/controller-runtime v0.1.8-0.20200318092001-e22ac1073450
