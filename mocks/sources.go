// Code generated by MockGen. DO NOT EDIT.
// Source: sources.go

// Package mock_externalissuesource is a generated GoMock package.
package mock_externalissuesource

import (
	externalissuesource "github.com/aimeelaplant/externalissuesource"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockExternalSource is a mock of ExternalSource interface
type MockExternalSource struct {
	ctrl     *gomock.Controller
	recorder *MockExternalSourceMockRecorder
}

// MockExternalSourceMockRecorder is the mock recorder for MockExternalSource
type MockExternalSourceMockRecorder struct {
	mock *MockExternalSource
}

// NewMockExternalSource creates a new mock instance
func NewMockExternalSource(ctrl *gomock.Controller) *MockExternalSource {
	mock := &MockExternalSource{ctrl: ctrl}
	mock.recorder = &MockExternalSourceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockExternalSource) EXPECT() *MockExternalSourceMockRecorder {
	return m.recorder
}

// Issue mocks base method
func (m *MockExternalSource) Issue(url string) (*externalissuesource.Issue, error) {
	ret := m.ctrl.Call(m, "Issue", url)
	ret0, _ := ret[0].(*externalissuesource.Issue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Issue indicates an expected call of Issue
func (mr *MockExternalSourceMockRecorder) Issue(url interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Issue", reflect.TypeOf((*MockExternalSource)(nil).Issue), url)
}

// CharacterPage mocks base method
func (m *MockExternalSource) CharacterPage(url string) (*externalissuesource.CharacterPage, error) {
	ret := m.ctrl.Call(m, "CharacterPage", url)
	ret0, _ := ret[0].(*externalissuesource.CharacterPage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CharacterPage indicates an expected call of CharacterPage
func (mr *MockExternalSourceMockRecorder) CharacterPage(url interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CharacterPage", reflect.TypeOf((*MockExternalSource)(nil).CharacterPage), url)
}

// SearchCharacter mocks base method
func (m *MockExternalSource) SearchCharacter(query string) (externalissuesource.CharacterSearchResult, error) {
	ret := m.ctrl.Call(m, "SearchCharacter", query)
	ret0, _ := ret[0].(externalissuesource.CharacterSearchResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SearchCharacter indicates an expected call of SearchCharacter
func (mr *MockExternalSourceMockRecorder) SearchCharacter(query interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SearchCharacter", reflect.TypeOf((*MockExternalSource)(nil).SearchCharacter), query)
}
