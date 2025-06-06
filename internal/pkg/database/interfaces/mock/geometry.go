// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/pkg/database/interfaces/geometry.go

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockGeometryIF is a mock of GeometryIF interface.
type MockGeometryIF struct {
	ctrl     *gomock.Controller
	recorder *MockGeometryIFMockRecorder
}

// MockGeometryIFMockRecorder is the mock recorder for MockGeometryIF.
type MockGeometryIFMockRecorder struct {
	mock *MockGeometryIF
}

// NewMockGeometryIF creates a new mock instance.
func NewMockGeometryIF(ctrl *gomock.Controller) *MockGeometryIF {
	mock := &MockGeometryIF{ctrl: ctrl}
	mock.recorder = &MockGeometryIFMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGeometryIF) EXPECT() *MockGeometryIFMockRecorder {
	return m.recorder
}

// GeometryEncode mocks base method.
func (m *MockGeometryIF) GeometryEncode(geometry string) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GeometryEncode", geometry)
	ret0, _ := ret[0].(string)
	return ret0
}

// GeometryEncode indicates an expected call of GeometryEncode.
func (mr *MockGeometryIFMockRecorder) GeometryEncode(geometry interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GeometryEncode", reflect.TypeOf((*MockGeometryIF)(nil).GeometryEncode), geometry)
}

// GetBufferFromGeoHex mocks base method.
func (m *MockGeometryIF) GetBufferFromGeoHex(geo string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBufferFromGeoHex", geo)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBufferFromGeoHex indicates an expected call of GetBufferFromGeoHex.
func (mr *MockGeometryIFMockRecorder) GetBufferFromGeoHex(geo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBufferFromGeoHex", reflect.TypeOf((*MockGeometryIF)(nil).GetBufferFromGeoHex), geo)
}

// GetGeoFromText mocks base method.
func (m *MockGeometryIF) GetGeoFromText(wkt string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGeoFromText", wkt)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGeoFromText indicates an expected call of GetGeoFromText.
func (mr *MockGeometryIFMockRecorder) GetGeoFromText(wkt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGeoFromText", reflect.TypeOf((*MockGeometryIF)(nil).GetGeoFromText), wkt)
}

// GetGeoJson mocks base method.
func (m *MockGeometryIF) GetGeoJson(geo string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGeoJson", geo)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGeoJson indicates an expected call of GetGeoJson.
func (mr *MockGeometryIFMockRecorder) GetGeoJson(geo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGeoJson", reflect.TypeOf((*MockGeometryIF)(nil).GetGeoJson), geo)
}

// GetWktFromGeo mocks base method.
func (m *MockGeometryIF) GetWktFromGeo(geo string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWktFromGeo", geo)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWktFromGeo indicates an expected call of GetWktFromGeo.
func (mr *MockGeometryIFMockRecorder) GetWktFromGeo(geo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWktFromGeo", reflect.TypeOf((*MockGeometryIF)(nil).GetWktFromGeo), geo)
}
