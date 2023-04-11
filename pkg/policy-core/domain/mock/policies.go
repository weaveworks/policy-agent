// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/weaveworks/policy-agent/pkg/policy-core/domain (interfaces: PolicyValidationSink)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	domain "github.com/weaveworks/policy-agent/pkg/policy-core/domain"
	gomock "github.com/golang/mock/gomock"
)

// MockPolicyValidationSink is a mock of PolicyValidationSink interface.
type MockPolicyValidationSink struct {
	ctrl     *gomock.Controller
	recorder *MockPolicyValidationSinkMockRecorder
}

// MockPolicyValidationSinkMockRecorder is the mock recorder for MockPolicyValidationSink.
type MockPolicyValidationSinkMockRecorder struct {
	mock *MockPolicyValidationSink
}

// NewMockPolicyValidationSink creates a new mock instance.
func NewMockPolicyValidationSink(ctrl *gomock.Controller) *MockPolicyValidationSink {
	mock := &MockPolicyValidationSink{ctrl: ctrl}
	mock.recorder = &MockPolicyValidationSinkMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPolicyValidationSink) EXPECT() *MockPolicyValidationSinkMockRecorder {
	return m.recorder
}

// Write mocks base method.
func (m *MockPolicyValidationSink) Write(arg0 context.Context, arg1 []domain.PolicyValidation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Write", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Write indicates an expected call of Write.
func (mr *MockPolicyValidationSinkMockRecorder) Write(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockPolicyValidationSink)(nil).Write), arg0, arg1)
}
