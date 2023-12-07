// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloudplugin1

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform/internal/cloudplugin/cloudproto1"
	"github.com/hashicorp/terraform/internal/cloudplugin/mock_cloudproto1"
	"github.com/hashicorp/terraform/internal/terminal"
)

var mockError = "this is a mock error"

func testGRPCloudClient(t *testing.T, ctrl *gomock.Controller, client *mock_cloudproto1.MockCommandService_ExecuteClient, streams *terminal.Streams, executeError error) *GRPCCloudClient {
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
		streams: streams,
	}
}

func Test_GRPCCloudClient_ExecuteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	testStreams, done := terminal.StreamsForTesting(t)
	gRPCClient := testGRPCloudClient(t, ctrl, nil, testStreams, errors.New(mockError))

	exitCode := gRPCClient.Execute([]string{"example"})

	if exitCode != 1 {
		t.Fatalf("expected exit %d, got %d", 1, exitCode)
	}

	errOutput := done(t).Stderr()
	if errOutput != mockError {
		t.Errorf("expected error %q, got %q", mockError, errOutput)
	}
}

func Test_GRPCCloudClient_Execute_RecvError(t *testing.T) {
	ctrl := gomock.NewController(t)
	testStreams, done := terminal.StreamsForTesting(t)
	executeClient := mock_cloudproto1.NewMockCommandService_ExecuteClient(ctrl)
	executeClient.EXPECT().Recv().Return(nil, errors.New(mockError))

	gRPCClient := testGRPCloudClient(t, ctrl, executeClient, testStreams, nil)

	exitCode := gRPCClient.Execute([]string{"example"})

	if exitCode != 1 {
		t.Fatalf("expected exit %d, got %d", 1, exitCode)
	}

	mockRecvError := fmt.Sprintf("Failed to receive command response from cloudplugin: %s", mockError)

	errOutput := done(t).Stderr()
	if errOutput != mockRecvError {
		t.Errorf("expected error %q, got %q", mockRecvError, errOutput)
	}
}

func Test_GRPCCloudClient_Execute_Invalid_Exit(t *testing.T) {
	ctrl := gomock.NewController(t)
	executeClient := mock_cloudproto1.NewMockCommandService_ExecuteClient(ctrl)
	testStreams, _ := terminal.StreamsForTesting(t)

	executeClient.EXPECT().Recv().Return(
		&cloudproto1.CommandResponse{
			Data: &cloudproto1.CommandResponse_ExitCode{
				ExitCode: 3_000,
			},
		}, nil,
	)

	gRPCClient := testGRPCloudClient(t, ctrl, executeClient, testStreams, nil)

	exitCode := gRPCClient.Execute([]string{"example"})

	if exitCode != 255 {
		t.Fatalf("expected exit %q, got %q", 255, exitCode)
	}
}

func Test_GRPCCloudClient_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	testStreams, done := terminal.StreamsForTesting(t)
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

	gRPCClient := testGRPCloudClient(t, ctrl, executeClient, testStreams, nil)

	exitCode := gRPCClient.Execute([]string{"example"})

	if exitCode != 99 {
		t.Fatalf("expected exit %q, got %q", 99, exitCode)
	}

	output := done(t).Stdout()
	if output != "firstcall\nsecondcall\n" {
		t.Errorf("expected output %q, got %q", "firstcall\nsecondcall\n", output)
	}
}
