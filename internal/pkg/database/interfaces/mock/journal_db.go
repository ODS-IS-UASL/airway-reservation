package mock

import (
	"uasl-reservation/internal/pkg/database/interfaces"

	"github.com/golang/mock/gomock"
)

// MockJournalDBIF is a mock of JournalDBIF interface
type MockJournalDBIF struct {
	ctrl     *gomock.Controller
	recorder *MockJournalDBIFMockRecorder
}

// MockJournalDBIFMockRecorder is the mock recorder for MockJournalDBIF
type MockJournalDBIFMockRecorder struct {
	mock *MockJournalDBIF
}

// NewMockJournalDBIF creates a new mock instance
func NewMockJournalDBIF(ctrl *gomock.Controller) *MockJournalDBIF {
	mock := &MockJournalDBIF{ctrl: ctrl}
	mock.recorder = &MockJournalDBIFMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockJournalDBIF) EXPECT() *MockJournalDBIFMockRecorder {
	return m.recorder
}

// AddEvent mocks base method
func (m *MockJournalDBIF) AddEvent(event *interfaces.JournalEvent, tableName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddEvent", event, tableName)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddEvent indicates an expected call of AddEvent
func (mr *MockJournalDBIFMockRecorder) AddEvent(event, tableName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddEvent", nil, event, tableName)
}

// AddEventsTransact mocks base method
func (m *MockJournalDBIF) AddEventsTransact(events []*interfaces.JournalEvent, tableName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddEventsTransact", events, tableName)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddEventsTransact indicates an expected call of AddEventsTransact
func (mr *MockJournalDBIFMockRecorder) AddEventsTransact(events, tableName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddEventsTransact", nil, events, tableName)
}
