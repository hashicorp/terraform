// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloudplugin1

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform/internal/cloudplugin/cloudproto1"
	"github.com/hashicorp/terraform/internal/cloudplugin/mock_cloudproto1"
)

var mockError = "this is a mock error"

func testGRPCloudClient(t *testing.T, ctrl *gomock.Controller, client *mock_cloudproto1.MockCommandService_ExecuteClient, executeError error) *GRPCCloudClient {
	t.Helper()

	if client != nil && executeError != nil {
		t.Fatal("one of client or executeError must be nil")
	}

	result := mock_cloudproto1.NewMockCommandServiceClient(ctrl)

	result.EXPECT().Execute(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(client, executeError)

	return &GRPCCloudClient{
		client:  result,
		context: context.Background(),
	}
}

func Test_GRPCCloudClient_ExecuteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	gRPCClient := testGRPCloudClient(t, ctrl, nil, errors.New(mockError))

	buffer := bytes.Buffer{}
	exitCode := gRPCClient.Execute([]string{"example"}, io.Discard, &buffer)

	if exitCode != 1 {
		t.Fatalf("expected exit %d, got %d", 1, exitCode)
	}

	if buffer.String() != mockError {
		t.Errorf("expected error %q, got %q", mockError, buffer.String())
	}
}

func Test_GRPCCloudClient_Execute_RecvError(t *testing.T) {
	ctrl := gomock.NewController(t)
	executeClient := mock_cloudproto1.NewMockCommandService_ExecuteClient(ctrl)
	executeClient.EXPECT().Recv().Return(nil, errors.New(mockError))

	gRPCClient := testGRPCloudClient(t, ctrl, executeClient, nil)

	buffer := bytes.Buffer{}
	exitCode := gRPCClient.Execute([]string{"example"}, io.Discard, &buffer)

	if exitCode != 1 {
		t.Fatalf("expected exit %d, got %d", 1, exitCode)
	}

	mockRecvError := fmt.Sprintf("Failed to receive command response from cloudplugin: %s", mockError)

	if buffer.String() != mockRecvError {
		t.Errorf("expected error %q, got %q", mockRecvError, buffer.String())
	}
}

func Test_GRPCCloudClient_Execute_Invalid_Exit(t *testing.T) {
	ctrl := gomock.NewController(t)
	executeClient := mock_cloudproto1.NewMockCommandService_ExecuteClient(ctrl)

	executeClient.EXPECT().Recv().Return(
		&cloudproto1.CommandResponse{
			Data: &cloudproto1.CommandResponse_ExitCode{
				ExitCode: 3_000,
			},
		}, nil,
	)

	gRPCClient := testGRPCloudClient(t, ctrl, executeClient, nil)

	exitCode := gRPCClient.Execute([]string{"example"}, io.Discard, io.Discard)

	if exitCode != 255 {
		t.Fatalf("expected exit %q, got %q", 255, exitCode)
	}
}

func Test_GRPCCloudClient_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	executeClient := mock_cloudproto1.NewMockCommandService_ExecuteClient(ctrl)

	gomock.InOrder(
		executeClient.EXPECT().Recv().Return(
			&cloudproto1.CommandResponse{
				Data: &cloudproto1.CommandResponse_Stdout{
					Stdout: []byte("firstcall\n"),
				},
			}, nil,
		),
		executeClient.EXPECT().Recv().Return(
			&cloudproto1.CommandResponse{
				Data: &cloudproto1.CommandResponse_Stdout{
					Stdout: []byte("secondcall\n"),
				},
			}, nil,
		),
		executeClient.EXPECT().Recv().Return(
			&cloudproto1.CommandResponse{
				Data: &cloudproto1.CommandResponse_ExitCode{
					ExitCode: 99,
				},
			}, nil,
		),
	)

	gRPCClient := testGRPCloudClient(t, ctrl, executeClient, nil)

	stdoutBuffer := bytes.Buffer{}
	exitCode := gRPCClient.Execute([]string{"example"}, &stdoutBuffer, io.Discard)

	if exitCode != 99 {
		t.Fatalf("expected exit %q, got %q", 99, exitCode)
	}

	if stdoutBuffer.String() != "firstcall\nsecondcall\n" {
		t.Errorf("expected output %q, got %q", "firstcall\nsecondcall\n", stdoutBuffer.String())
	}
}
