// Code generated by MockGen. DO NOT EDIT.
// Source: worker.go
//
// Generated by this command:
//
//	mockgen -source=worker.go -destination=../../mocks/stream/worker.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	downloader "github.com/amirdaaee/TGMon/internal/stream/downloader"
	tg "github.com/gotd/td/tg"
	gomock "go.uber.org/mock/gomock"
)

// MockIWorker is a mock of IWorker interface.
type MockIWorker struct {
	ctrl     *gomock.Controller
	recorder *MockIWorkerMockRecorder
	isgomock struct{}
}

// MockIWorkerMockRecorder is the mock recorder for MockIWorker.
type MockIWorkerMockRecorder struct {
	mock *MockIWorker
}

// NewMockIWorker creates a new mock instance.
func NewMockIWorker(ctrl *gomock.Controller) *MockIWorker {
	mock := &MockIWorker{ctrl: ctrl}
	mock.recorder = &MockIWorkerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIWorker) EXPECT() *MockIWorkerMockRecorder {
	return m.recorder
}

// GetDoc mocks base method.
func (m *MockIWorker) GetDoc(ctx context.Context, messageID int) (*tg.Document, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDoc", ctx, messageID)
	ret0, _ := ret[0].(*tg.Document)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDoc indicates an expected call of GetDoc.
func (mr *MockIWorkerMockRecorder) GetDoc(ctx, messageID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDoc", reflect.TypeOf((*MockIWorker)(nil).GetDoc), ctx, messageID)
}

// GetThumbnail mocks base method.
func (m *MockIWorker) GetThumbnail(ctx context.Context, messageID int) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetThumbnail", ctx, messageID)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetThumbnail indicates an expected call of GetThumbnail.
func (mr *MockIWorkerMockRecorder) GetThumbnail(ctx, messageID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetThumbnail", reflect.TypeOf((*MockIWorker)(nil).GetThumbnail), ctx, messageID)
}

// Stream mocks base method.
func (m *MockIWorker) Stream(ctx context.Context, reader *downloader.Reader) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stream", ctx, reader)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Stream indicates an expected call of Stream.
func (mr *MockIWorkerMockRecorder) Stream(ctx, reader any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stream", reflect.TypeOf((*MockIWorker)(nil).Stream), ctx, reader)
}
