// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	interfaces "uasl-reservation/internal/pkg/database/interfaces"
)

// MockTransactionDBIF is a mock of TransactionDBIF interface.
type MockTransactionDBIF struct {
	ctrl     *gomock.Controller
	recorder *MockTransactionDBIFMockRecorder
}

// MockTransactionDBIFMockRecorder is the mock recorder for MockTransactionDBIF.
type MockTransactionDBIFMockRecorder struct {
	mock *MockTransactionDBIF
}

// NewMockTransactionDBIF creates a new mock instance.
func NewMockTransactionDBIF(ctrl *gomock.Controller) *MockTransactionDBIF {
	mock := &MockTransactionDBIF{ctrl: ctrl}
	mock.recorder = &MockTransactionDBIFMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTransactionDBIF) EXPECT() *MockTransactionDBIFMockRecorder {
	return m.recorder
}

// GetDB mocks base method.
func (m *MockTransactionDBIF) GetDB() *interfaces.DB {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDB")
	ret0, _ := ret[0].(*interfaces.DB)
	return ret0
}

// GetDB indicates an expected call of GetDB.
func (mr *MockTransactionDBIFMockRecorder) GetDB() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDB", reflect.TypeOf((*MockTransactionDBIF)(nil).GetDB))
}

// TransactionScope mocks base method.
func (m *MockTransactionDBIF) TransactionScope(arg0 context.Context, arg1 func(context.Context, *interfaces.DB) error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TransactionScope", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// TransactionScope indicates an expected call of TransactionScope.
func (mr *MockTransactionDBIFMockRecorder) TransactionScope(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TransactionScope", reflect.TypeOf((*MockTransactionDBIF)(nil).TransactionScope), arg0, arg1)
}
