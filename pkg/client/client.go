/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	networkclient "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	overKubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	metav1 "github.com/kubevirt/cluster-api-provider-kubevirt/vendor/k8s.io/apimachinery/pkg/apis/meta/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

//go:generate go run ../../vendor/github.com/golang/mock/mockgen -source=./client.go -destination=./mock/client_generated.go -package=mock

const (
	kubeconfig = "kubeconfig"
	secretName = "underKubeSecretClient"
)

// GetKubevirtClient is function type for building kubevirt client
func GetKubevirtClient(overKubeClient overKubeClient.Client, namespace string) (Client, error){
	secret, err := overKubeClient.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	//TODO: check that
	kubeconfig := secret.value['kubeconfig']
	optionalArgument := ""
	kubecliclient, err := kubecli.GetKubevirtClientFromFlags(optionalArgument, kubeconfig)
	resultClient := kubevirtclient{
		kubecliclient: kubecliclient,
		//TODO: fill it
		virtctlclient: "",
	}
	return resultClient, nil
}

// Client is a wrapper object for actual kubevirt clients: virtctl and the kubecli
type Client interface {
	CreateVirtualMachine(namespace string, newVM *v1.VirtualMachine) (*v1.VirtualMachine, error)
	DeleteVirtualMachine(namespace string, name string, options *k8smetav1.DeleteOptions) error
	//NetworkClient() networkclient.Interface
}

type kubevirtclient struct {
	kubecliclient kubecli.KubevirtClient
	//TODO: create a virtctl client also
	virtctlclient string
}

func (c *kubevirtclient) NetworkClient() networkclient.Interface {
	return c.NetworkClient()
}
func (c *kubevirtclient) CreateVirtualMachine(namespace string, newVM *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	return c.kubecliclient.VirtualMachine(namespace).Create(newVM)
}
func (c *kubevirtclient) DeleteVirtualMachine(namespace string, name string, options *k8smetav1.DeleteOptions) error {
	return c.kubecliclient.VirtualMachine(namespace).Delete(name, options)
}
func (c *kubevirtclient) GetVirtualMachine(namespace string, name string, options *k8smetav1.GetOptions) (*v1.VirtualMachine, error) {
	return c.kubecliclient.VirtualMachine(namespace).Get(name, options)
}
func (c *kubevirtclient) ListVirtualMachine(namespace string, options *k8smetav1.ListOptions) (*v1.VirtualMachineList, error) {
	return c.kubecliclient.VirtualMachine(namespace).List(options)
}
func (c *kubevirtclient) UpdateVirtualMachine(namespace string, vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	return c.kubecliclient.VirtualMachine(namespace).Update(vm)
}
func (c *kubevirtclient) PatchVirtualMachine(namespace string, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachine, err error){
	return c.kubecliclient.VirtualMachine(namespace).Patch(name, pt, data, subresources...)
}
func (c *kubevirtclient) RestartVirtualMachine(namespace string,name string) error  {
	return c.kubecliclient.VirtualMachine(namespace).Restart(name)
}
func (c *kubevirtclient) StartVirtualMachine(namespace string,  name string) error {
	return c.kubecliclient.VirtualMachine(namespace).Start(name)
}
func (c *kubevirtclient) StopVirtualMachine(namespace string, name string) error{
	return c.kubecliclient.VirtualMachine(namespace).Stop(name)
}
