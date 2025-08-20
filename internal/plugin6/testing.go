package plugin6

import (
	"context"
	"io"

	proto "github.com/hashicorp/terraform/internal/tfplugin6"

	"google.golang.org/grpc/metadata"
)

// newMockReadStateBytesClient returns a mock that will return data in
// defined chunks. Each element in the array will be returned in separate
// calls to the client's Recv method, and subsequent calls will return
// io.EOF errors.
func newMockReadStateBytesClient(chunks []string) mockReadStateBytesClient {
	// Calculate the total length of the chunks together when in byte form
	var totalLength int64
	chunkMap := map[int][]byte{}
	for i, chunk := range chunks {
		chunkBytes := []byte(chunk)

		chunkMap[i] = chunkBytes
		totalLength += int64(len(chunkBytes))
	}

	recvCount := 0
	return mockReadStateBytesClient{
		chunks:      chunkMap,
		totalLength: totalLength,
		recvCount:   &recvCount,
	}
}

var _ proto.Provider_ReadStateBytesClient = mockReadStateBytesClient{}

type mockReadStateBytesClient struct {
	chunks      map[int][]byte
	totalLength int64

	// Need a pointer because all methods have value receivers; need to track despite that
	recvCount *int
}

var _ proto.Provider_ReadStateBytesClient = mockReadStateBytesClient{}

func (m mockReadStateBytesClient) CloseSend() error {
	panic("not implemented") // Not invoked by code under test
}

func (m mockReadStateBytesClient) Context() context.Context {
	panic("not implemented") // Not invoked by code under test
}

func (m mockReadStateBytesClient) Header() (metadata.MD, error) {
	panic("not implemented") // Not invoked by code under test
}

// Recv returns the bytes in m.chunks (map[int][]byte) that correspond to how many times
// this method has been invoked. When no bytes are found for an invocation the method will
// act like it's reached the end of the available data and return an io.EOF error.
func (m mockReadStateBytesClient) Recv() (*proto.ReadStateBytes_ResponseChunk, error) {
	chunk := proto.ReadStateBytes_ResponseChunk{
		TotalLength: m.totalLength,
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

	delete(m.chunks, *m.recvCount)
	*m.recvCount++

	return &chunk, nil
}

func (m mockReadStateBytesClient) RecvMsg(a any) error {
	panic("not implemented") // Not invoked by code under test
}

func (m mockReadStateBytesClient) SendMsg(a any) error {
	panic("not implemented") // Not invoked by code under test
}

func (m mockReadStateBytesClient) Trailer() metadata.MD {
	panic("not implemented") // Not invoked by code under test
}
