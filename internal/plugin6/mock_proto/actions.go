// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mock_tfplugin6

import (
	context "context"
	"io"

	proto "github.com/hashicorp/terraform/internal/tfplugin6"
	metadata "google.golang.org/grpc/metadata"
)

var _ proto.Provider_InvokeActionClient = (*MockInvokeProtoClient)(nil)

type MockInvokeProtoClient struct {
	Events []*proto.InvokeAction_Event
}

func (m *MockInvokeProtoClient) Recv() (*proto.InvokeAction_Event, error) {
	if len(m.Events) == 0 {
		return nil, io.EOF
	}
	event := m.Events[0]
	m.Events = m.Events[1:]
	return event, nil
}

func (m *MockInvokeProtoClient) CloseSend() error {
	return nil
}

func (m *MockInvokeProtoClient) Context() context.Context {
	return context.TODO()
}

func (m *MockInvokeProtoClient) Header() (metadata.MD, error) {
	return nil, nil
}

func (m *MockInvokeProtoClient) RecvMsg(k any) error {
	return nil
}

func (m *MockInvokeProtoClient) SendMsg(k any) error {
	return nil
}
func (m *MockInvokeProtoClient) Trailer() metadata.MD {
	return nil
}
