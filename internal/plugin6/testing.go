// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin6

import (
	"errors"
	"io"
	"reflect"
	"testing"

	proto "github.com/hashicorp/terraform/internal/tfplugin6"
	"go.uber.org/mock/gomock"

	"google.golang.org/grpc"
)

// newMockReadStateBytesClient returns a mock that will return data in
// defined chunks. Each element in the array will be returned in separate
// calls to the client's Recv method, and subsequent calls will return
// io.EOF errors.
func newMockReadStateBytesClient(chunks []string, opts mockReadStateBytesOpts) mockReadStateBytesClient {
	// Calculate the total length of the chunks together when in byte form
	var totalLength int64
	chunkMap := map[int][]byte{}
	for i, chunk := range chunks {
		chunkBytes := []byte(chunk)

		chunkMap[i] = chunkBytes
		totalLength += int64(len(chunkBytes))
	}

	if opts.overrideTotalLength {
		// We're forcing this to be a given value
		totalLength = opts.newTotalLength
	}

	recvCount := 0
	return mockReadStateBytesClient{
		chunks:         chunkMap,
		totalLength:    totalLength,
		recvCount:      &recvCount,
		recvDiagnostic: opts.recvDiagnostic,
	}
}

var _ proto.Provider_ReadStateBytesClient = mockReadStateBytesClient{}

type mockReadStateBytesOpts struct {
	overrideTotalLength bool
	newTotalLength      int64
	recvDiagnostic      *proto.Diagnostic
}

type mockReadStateBytesClient struct {
	chunks      map[int][]byte
	totalLength int64

	// If recvDiagnostic is set, the Recv method will return this diagnostic
	// on the first invocation.
	recvDiagnostic *proto.Diagnostic

	// We need a pointer for tracking how many times the Recv method has been called
	// because all proto.Provider_ReadStateBytesClient methods have value receivers.
	recvCount *int

	// Embedding this interface helps minimize what's implemented in this mock
	// Any calls to the unimplemented methods will fail due to a nil pointer error,
	// as we aren't supplying an instance of the embedded type when making the
	// `mockReadStateBytesClient` used in tests
	grpc.ClientStream
}

var _ proto.Provider_ReadStateBytesClient = mockReadStateBytesClient{}

// Recv returns the bytes in m.chunks (map[int][]byte) that correspond to how many times
// this method has been invoked. When no bytes are found for an invocation the method will
// act like it's reached the end of the available data and return an io.EOF error.
func (m mockReadStateBytesClient) Recv() (*proto.ReadStateBytes_ResponseChunk, error) {
	var chunk proto.ReadStateBytes_ResponseChunk

	if m.recvDiagnostic != nil {
		chunk.Diagnostics = append(chunk.Diagnostics, m.recvDiagnostic)
		return &chunk, errors.New("returning error diagnostic supplied to mock client")
	}

	chunkBytes, exists := m.chunks[*m.recvCount]
	if !exists {
		if len(m.chunks) == 0 {
			// All data has been sent
			return nil, io.EOF
		}

		// There's still data, which suggests bad test setup or a bug in the mock
		return nil, io.ErrUnexpectedEOF
	}
	chunk.Bytes = chunkBytes
	chunk.TotalLength = m.totalLength

	delete(m.chunks, *m.recvCount)
	*m.recvCount++

	return &chunk, nil
}

// newMockReadStateBytesClient returns a mock that will write the data in the
// passed string.
func newMockWriteStateBytesClient(t *testing.T, opts mockWriteStateBytesOpts) mockWriteStateBytesClient {
	ctrl := gomock.NewController(t)
	client := mockWriteStateBytesClient{
		ctrl:       ctrl,
		receiverId: "mock-write-state-bytes-client", // See comments for mockWriteStateBytesClient.
	}
	client.recorder = &mockWriteStateBytesClientRecorder{
		mock: &client,
	}
	return client
}

var _ proto.Provider_WriteStateBytesClient = mockWriteStateBytesClient{}

type mockWriteStateBytesOpts struct{}

type mockWriteStateBytesClient struct {
	// Embedding this interface helps minimize what's implemented in this mock
	// Any calls to the unimplemented methods will fail due to a nil pointer error,
	// as we aren't supplying an instance of the embedded type when making the
	// `mockReadStateBytesClient` used in tests
	grpc.ClientStream

	// We use the field below to allow spying on each method call in this mock
	ctrl     *gomock.Controller
	recorder *mockWriteStateBytesClientRecorder

	// This is a hack to help use "go.uber.org/mock/gomock"
	// When a method on the mock is called there is logic matching that call to any
	// expected calls. That matching logic relies on data about a "receiver" being supplied.
	// The Send method that we're spying on through this mock has a value receiver, so the
	// receiver of the method changes each time it is called and breaks the matching logic.
	// If the method had a pointer receiver (like other RPC methods we mock ) then we'd pass the
	// method receiver as that value. However for Send we pass through this string that's set on
	// the receiver, as the string will be the same each time.
	receiverId string
}

type mockWriteStateBytesClientRecorder struct {
	mock *mockWriteStateBytesClient
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *mockWriteStateBytesClient) EXPECT() *mockWriteStateBytesClientRecorder {
	return m.recorder
}

// Send on the mockWriteStateBytesClient is the method invoked by the calling code that we're testing.
// The logic inside checks that this invocation of the method has been defined as expected in the test setup,
// and checks all the previously defined assertions.
// Fulfils the proto.Provider_WriteStateBytesClient interface.
func (m mockWriteStateBytesClient) Send(arg0 *proto.WriteStateBytes_RequestChunk) error {
	m.ctrl.T.Helper()
	varargs := []any{arg0}
	// We use m.receiverId as the recorded receiver of the method call, as the Send method uses
	// a value receiver. This means that there are different receivers on each call.
	ret := m.ctrl.Call(m.receiverId, "Send", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send on the mockWriteStateBytesClientRecorder is the method used during test setup to define assertions about
// if and how the Send method is invoked on the mockWriteStateBytesClient.
func (mr *mockWriteStateBytesClientRecorder) Send(arg0 any, arg1 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0}, arg1...)
	t := reflect.TypeOf(mockWriteStateBytesClient{}.Send) // Differs from other mocks as Send has value receiver
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock.receiverId, "Send", t, varargs...)
}

// Fulfils the proto.Provider_WriteStateBytesClient interface.
func (m mockWriteStateBytesClient) CloseAndRecv() (*proto.WriteStateBytes_Response, error) {
	resp := &proto.WriteStateBytes_Response{}
	return resp, nil
}
