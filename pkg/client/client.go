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
)

//go:generate go run ../../vendor/github.com/golang/mock/mockgen -source=./client.go -destination=./mock/client_generated.go -package=mock

const (
	kubeconfig = "kubeconfig"
	secretName = ""
)

// KubevirtClientBuilderFuncType is function type for building kubevirt client
type KubevirtClientBuilderFuncType func(client kubecli.KubevirtClient, namespace string) (Client, error){
	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	//TODO: check that
	kubeconfig = secret.value['kubeconfig']
	optionalArgument = ""
	virtclient, err = kubecli.GetKubevirtClientFromFlags(optionalArgument, kubeconfig)
	return virtclient
}

// Client is a wrapper object for actual kubevirt clients: virtctl and the kubecli
type Client interface {
	CreateVirtualMachineInstance(namespace string, vmi kubecli.VirtualMachineInstance) (v1.VirtualMachineInstance, error)
	CreateVirtualMachine(namespace string, vmi kubecli.VirtualMachine) (v1.VirtualMachine, error)
	VirtualMachine(namespace string) kubecli.kubeVirtualMachineInterface
	NetworkClient() networkclient.Interface
}

type kubevirtclient struct {
	kubeclient Client
	virtctlclient string
}

func NewClientBuilder(kubeconfig string) (*ClientBuilder, error) {
	config, err := getRestConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return &ClientBuilder{
		config: config,
	}, nil
}

func (c *kubevirtclient) VirtualMachine(namespace string) kubecli.VirtualMachineInterface {
	return c.VirtualMachine(namespace)
}

func (c *kubevirtclient) NetworkClient() networkclient.Interface {
	return c.NetworkClient()
}

func (c *kubevirtclient) CreateVirtualMachineInstance(namespace string, vmi kubecli.VirtualMachineInstance) (v1.VirtualMachineInstance, error) {

	return c.VirtualMachineInstance(namespace).Create(vmi)
}

func (c *kubevirtclient) CreateVirtualMachine(namespace string, newVM *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	return c.kubeclient.VirtualMachine(namespace).Create(newVM)
}

func NewClient(ctrlRuntimeClient client.Client, namespace string) Client {
	return &kubevirtclient{namespace}
}
