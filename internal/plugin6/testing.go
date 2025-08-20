// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin6

import (
	"errors"
	"io"

	proto "github.com/hashicorp/terraform/internal/tfplugin6"

	"google.golang.org/grpc"
)

// newMockReadStateBytesClient returns a mock that will return data in
// defined chunks. Each element in the array will be returned in separate
// calls to the client's Recv method, and subsequent calls will return
// io.EOF errors.
func newMockReadStateBytesClient(chunks []string, opts mockOpts) mockReadStateBytesClient {
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

type mockOpts struct {
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
