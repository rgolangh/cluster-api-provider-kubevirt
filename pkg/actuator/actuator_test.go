package actuator

import (
	"testing"

	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	// Add types to scheme
	machinev1.AddToScheme(scheme.Scheme)
}

func TestMachineEvents(t *testing.T) {
	// TODO
}

func TestHandleMachineErrors(t *testing.T) {
	// TODO implement

}
