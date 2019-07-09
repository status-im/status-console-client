// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/status-im/status-console-client/protocol/client (interfaces: Database)

// Package client is a generated GoMock package.
package client

import (
	ecdsa "crypto/ecdsa"
	gomock "github.com/golang/mock/gomock"
	client "github.com/status-im/status-console-client/protocol/client"
	v1 "github.com/status-im/status-console-client/protocol/v1"
	reflect "reflect"
	time "time"
)

// MockDatabase is a mock of Database interface
type MockDatabase struct {
	ctrl     *gomock.Controller
	recorder *MockDatabaseMockRecorder
}

// MockDatabaseMockRecorder is the mock recorder for MockDatabase
type MockDatabaseMockRecorder struct {
	mock *MockDatabase
}

// NewMockDatabase creates a new mock instance
func NewMockDatabase(ctrl *gomock.Controller) *MockDatabase {
	mock := &MockDatabase{ctrl: ctrl}
	mock.recorder = &MockDatabaseMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDatabase) EXPECT() *MockDatabaseMockRecorder {
	return m.recorder
}

// Close mocks base method
func (m *MockDatabase) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockDatabaseMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockDatabase)(nil).Close))
}

// ContactExist mocks base method
func (m *MockDatabase) ContactExist(arg0 client.Contact) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ContactExist", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ContactExist indicates an expected call of ContactExist
func (mr *MockDatabaseMockRecorder) ContactExist(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ContactExist", reflect.TypeOf((*MockDatabase)(nil).ContactExist), arg0)
}

// Contacts mocks base method
func (m *MockDatabase) Contacts() ([]client.Contact, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Contacts")
	ret0, _ := ret[0].([]client.Contact)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Contacts indicates an expected call of Contacts
func (mr *MockDatabaseMockRecorder) Contacts() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Contacts", reflect.TypeOf((*MockDatabase)(nil).Contacts))
}

// DeleteContact mocks base method
func (m *MockDatabase) DeleteContact(arg0 client.Contact) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteContact", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteContact indicates an expected call of DeleteContact
func (mr *MockDatabaseMockRecorder) DeleteContact(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteContact", reflect.TypeOf((*MockDatabase)(nil).DeleteContact), arg0)
}

// GetOneToOneChat mocks base method
func (m *MockDatabase) GetOneToOneChat(arg0 *ecdsa.PublicKey) (*client.Contact, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOneToOneChat", arg0)
	ret0, _ := ret[0].(*client.Contact)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOneToOneChat indicates an expected call of GetOneToOneChat
func (mr *MockDatabaseMockRecorder) GetOneToOneChat(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOneToOneChat", reflect.TypeOf((*MockDatabase)(nil).GetOneToOneChat), arg0)
}

// GetPublicChat mocks base method
func (m *MockDatabase) GetPublicChat(arg0 string) (*client.Contact, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPublicChat", arg0)
	ret0, _ := ret[0].(*client.Contact)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPublicChat indicates an expected call of GetPublicChat
func (mr *MockDatabaseMockRecorder) GetPublicChat(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPublicChat", reflect.TypeOf((*MockDatabase)(nil).GetPublicChat), arg0)
}

// Histories mocks base method
func (m *MockDatabase) Histories() ([]client.History, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Histories")
	ret0, _ := ret[0].([]client.History)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Histories indicates an expected call of Histories
func (mr *MockDatabaseMockRecorder) Histories() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Histories", reflect.TypeOf((*MockDatabase)(nil).Histories))
}

// LastMessageClock mocks base method
func (m *MockDatabase) LastMessageClock(arg0 client.Contact) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LastMessageClock", arg0)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LastMessageClock indicates an expected call of LastMessageClock
func (mr *MockDatabaseMockRecorder) LastMessageClock(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LastMessageClock", reflect.TypeOf((*MockDatabase)(nil).LastMessageClock), arg0)
}

// Messages mocks base method
func (m *MockDatabase) Messages(arg0 client.Contact, arg1, arg2 time.Time) ([]*v1.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Messages", arg0, arg1, arg2)
	ret0, _ := ret[0].([]*v1.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Messages indicates an expected call of Messages
func (mr *MockDatabaseMockRecorder) Messages(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Messages", reflect.TypeOf((*MockDatabase)(nil).Messages), arg0, arg1, arg2)
}

// NewMessages mocks base method
func (m *MockDatabase) NewMessages(arg0 client.Contact, arg1 int64) ([]*v1.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewMessages", arg0, arg1)
	ret0, _ := ret[0].([]*v1.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewMessages indicates an expected call of NewMessages
func (mr *MockDatabaseMockRecorder) NewMessages(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewMessages", reflect.TypeOf((*MockDatabase)(nil).NewMessages), arg0, arg1)
}

// SaveContacts mocks base method
func (m *MockDatabase) SaveContacts(arg0 []client.Contact) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveContacts", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveContacts indicates an expected call of SaveContacts
func (mr *MockDatabaseMockRecorder) SaveContacts(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveContacts", reflect.TypeOf((*MockDatabase)(nil).SaveContacts), arg0)
}

// SaveMessages mocks base method
func (m *MockDatabase) SaveMessages(arg0 client.Contact, arg1 []*v1.Message) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveMessages", arg0, arg1)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SaveMessages indicates an expected call of SaveMessages
func (mr *MockDatabaseMockRecorder) SaveMessages(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveMessages", reflect.TypeOf((*MockDatabase)(nil).SaveMessages), arg0, arg1)
}

// UnreadMessages mocks base method
func (m *MockDatabase) UnreadMessages(arg0 client.Contact) ([]*v1.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnreadMessages", arg0)
	ret0, _ := ret[0].([]*v1.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UnreadMessages indicates an expected call of UnreadMessages
func (mr *MockDatabaseMockRecorder) UnreadMessages(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnreadMessages", reflect.TypeOf((*MockDatabase)(nil).UnreadMessages), arg0)
}

// UpdateHistories mocks base method
func (m *MockDatabase) UpdateHistories(arg0 []client.History) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateHistories", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateHistories indicates an expected call of UpdateHistories
func (mr *MockDatabaseMockRecorder) UpdateHistories(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateHistories", reflect.TypeOf((*MockDatabase)(nil).UpdateHistories), arg0)
}
