// Code generated by MockGen. DO NOT EDIT.
// Source: ./client.go

// Package mock is a generated GoMock package.
package mock

import (
	gomock "github.com/golang/mock/gomock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	v10 "kubevirt.io/client-go/api/v1"
	reflect "reflect"
)

// MockClient is a mock of Client interface
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// CreateVirtualMachine mocks base method
func (m *MockClient) CreateVirtualMachine(namespace string, newVM *v10.VirtualMachine) (*v10.VirtualMachine, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVirtualMachine", namespace, newVM)
	ret0, _ := ret[0].(*v10.VirtualMachine)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVirtualMachine indicates an expected call of CreateVirtualMachine
func (mr *MockClientMockRecorder) CreateVirtualMachine(namespace, newVM interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVirtualMachine", reflect.TypeOf((*MockClient)(nil).CreateVirtualMachine), namespace, newVM)
}

// DeleteVirtualMachine mocks base method
func (m *MockClient) DeleteVirtualMachine(namespace, name string, options *v1.DeleteOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVirtualMachine", namespace, name, options)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVirtualMachine indicates an expected call of DeleteVirtualMachine
func (mr *MockClientMockRecorder) DeleteVirtualMachine(namespace, name, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVirtualMachine", reflect.TypeOf((*MockClient)(nil).DeleteVirtualMachine), namespace, name, options)
}

// GetVirtualMachine mocks base method
func (m *MockClient) GetVirtualMachine(namespace, name string, options *v1.GetOptions) (*v10.VirtualMachine, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVirtualMachine", namespace, name, options)
	ret0, _ := ret[0].(*v10.VirtualMachine)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVirtualMachine indicates an expected call of GetVirtualMachine
func (mr *MockClientMockRecorder) GetVirtualMachine(namespace, name, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVirtualMachine", reflect.TypeOf((*MockClient)(nil).GetVirtualMachine), namespace, name, options)
}

// GetVirtualMachineInstance mocks base method
func (m *MockClient) GetVirtualMachineInstance(namespace, name string, options *v1.GetOptions) (*v10.VirtualMachineInstance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVirtualMachineInstance", namespace, name, options)
	ret0, _ := ret[0].(*v10.VirtualMachineInstance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVirtualMachineInstance indicates an expected call of GetVirtualMachineInstance
func (mr *MockClientMockRecorder) GetVirtualMachineInstance(namespace, name, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVirtualMachineInstance", reflect.TypeOf((*MockClient)(nil).GetVirtualMachineInstance), namespace, name, options)
}

// ListVirtualMachine mocks base method
func (m *MockClient) ListVirtualMachine(namespace string, options *v1.ListOptions) (*v10.VirtualMachineList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListVirtualMachine", namespace, options)
	ret0, _ := ret[0].(*v10.VirtualMachineList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListVirtualMachine indicates an expected call of ListVirtualMachine
func (mr *MockClientMockRecorder) ListVirtualMachine(namespace, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListVirtualMachine", reflect.TypeOf((*MockClient)(nil).ListVirtualMachine), namespace, options)
}

// UpdateVirtualMachine mocks base method
func (m *MockClient) UpdateVirtualMachine(namespace string, vm *v10.VirtualMachine) (*v10.VirtualMachine, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateVirtualMachine", namespace, vm)
	ret0, _ := ret[0].(*v10.VirtualMachine)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateVirtualMachine indicates an expected call of UpdateVirtualMachine
func (mr *MockClientMockRecorder) UpdateVirtualMachine(namespace, vm interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateVirtualMachine", reflect.TypeOf((*MockClient)(nil).UpdateVirtualMachine), namespace, vm)
}

// PatchVirtualMachine mocks base method
func (m *MockClient) PatchVirtualMachine(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*v10.VirtualMachine, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{namespace, name, pt, data}
	for _, a := range subresources {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PatchVirtualMachine", varargs...)
	ret0, _ := ret[0].(*v10.VirtualMachine)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PatchVirtualMachine indicates an expected call of PatchVirtualMachine
func (mr *MockClientMockRecorder) PatchVirtualMachine(namespace, name, pt, data interface{}, subresources ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{namespace, name, pt, data}, subresources...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PatchVirtualMachine", reflect.TypeOf((*MockClient)(nil).PatchVirtualMachine), varargs...)
}

// RestartVirtualMachine mocks base method
func (m *MockClient) RestartVirtualMachine(namespace, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RestartVirtualMachine", namespace, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// RestartVirtualMachine indicates an expected call of RestartVirtualMachine
func (mr *MockClientMockRecorder) RestartVirtualMachine(namespace, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RestartVirtualMachine", reflect.TypeOf((*MockClient)(nil).RestartVirtualMachine), namespace, name)
}

// StartVirtualMachine mocks base method
func (m *MockClient) StartVirtualMachine(namespace, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StartVirtualMachine", namespace, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// StartVirtualMachine indicates an expected call of StartVirtualMachine
func (mr *MockClientMockRecorder) StartVirtualMachine(namespace, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartVirtualMachine", reflect.TypeOf((*MockClient)(nil).StartVirtualMachine), namespace, name)
}

// StopVirtualMachine mocks base method
func (m *MockClient) StopVirtualMachine(namespace, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StopVirtualMachine", namespace, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// StopVirtualMachine indicates an expected call of StopVirtualMachine
func (mr *MockClientMockRecorder) StopVirtualMachine(namespace, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StopVirtualMachine", reflect.TypeOf((*MockClient)(nil).StopVirtualMachine), namespace, name)
}
