// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	plugin "github.com/hashicorp/go-plugin"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/auth"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/internal/stacksplugin/mock_stacksproto1"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1"
	"github.com/hashicorp/terraform/version"
	"go.uber.org/mock/gomock"
)

var mockError = "this is a mock error"
var mockBrokerIDs = brokerIDs{
	packagesBrokerID:     1,
	dependenciesBrokerID: 2,
	stacksBrokerID:       3,
}

func mockDisco() *disco.Disco {
	mux := http.NewServeMux()
	s := httptest.NewServer(mux)

	host, _ := url.Parse(s.URL)
	defaultHostname := "app.terraform.io"
	tfeHost := svchost.Hostname(defaultHostname)
	services := map[string]interface{}{
		"stacksplugin.v1": fmt.Sprintf("%s/api/stacksplugin/v1/", s.URL),
		"tfe.v2":          fmt.Sprintf("%s/api/v2/", s.URL),
	}

	credsSrc := auth.StaticCredentialsSource(map[svchost.Hostname]map[string]interface{}{
		tfeHost: {"token": "test-auth-token"},
	})

	d := disco.NewWithCredentialsSource(credsSrc)
	d.SetUserAgent(httpclient.TerraformUserAgent(version.String()))
	d.ForceHostServices(tfeHost, services)
	d.ForceHostServices(svchost.Hostname(host.Host), services)

	return d
}

func mockGRPStacksClient(t *testing.T, ctrl *gomock.Controller, client *mock_stacksproto1.MockCommandService_ExecuteClient, executeError error) *GRPCStacksClient {
	t.Helper()

	if client != nil && executeError != nil {
		t.Fatal("one of client or executeError must be nil")
	}

	result := mock_stacksproto1.NewMockCommandServiceClient(ctrl)

	result.EXPECT().Execute(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(client, executeError)

	return &GRPCStacksClient{
		Client:   result,
		Context:  context.Background(),
		Services: mockDisco(),
		Broker:   &plugin.GRPCBroker{},
	}
}

func Test_GRPCStacksClient_ExecuteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	gRPCClient := mockGRPStacksClient(t, ctrl, nil, errors.New(mockError))

	buffer := bytes.Buffer{}
	// use executeWithBrokerIDs instead of Execute to allow mocking of the broker IDs
	// This is necessary because the plugin.GRPCBroker cannot be mocked except the actual plugin process is started.
	exitCode := gRPCClient.executeWithBrokers(mockBrokerIDs, []string{"init"}, io.Discard, &buffer)

	if exitCode != 1 {
		t.Fatalf("expected exit %d, got %d", 1, exitCode)
	}

	if buffer.String() != mockError {
		t.Errorf("expected error %q, got %q", mockError, buffer.String())
	}
}

func Test_GRPCStacksClient_Execute_RecvError(t *testing.T) {
	ctrl := gomock.NewController(t)
	executeClient := mock_stacksproto1.NewMockCommandService_ExecuteClient(ctrl)
	executeClient.EXPECT().Recv().Return(nil, errors.New(mockError))

	gRPCClient := mockGRPStacksClient(t, ctrl, executeClient, nil)

	buffer := bytes.Buffer{}
	// use executeWithBrokerIDs instead of Execute to allow mocking of the broker IDs
	// This is necessary because the plugin.GRPCBroker cannot be mocked except the actual plugin process is started.
	exitCode := gRPCClient.executeWithBrokers(mockBrokerIDs, []string{"init"}, io.Discard, &buffer)

	if exitCode != 1 {
		t.Fatalf("expected exit %d, got %d", 1, exitCode)
	}

	mockRecvError := fmt.Sprintf("Failed to receive command response from stacksplugin: %s", mockError)

	if buffer.String() != mockRecvError {
		t.Errorf("expected error %q, got %q", mockRecvError, buffer.String())
	}
}

func Test_GRPCStacksClient_Execute_HandleEOFError(t *testing.T) {
	ctrl := gomock.NewController(t)
	executeClient := mock_stacksproto1.NewMockCommandService_ExecuteClient(ctrl)
	executeClient.EXPECT().Recv().Return(&stacksproto1.CommandResponse{
		Data: &stacksproto1.CommandResponse_ExitCode{
			ExitCode: 0,
		},
	}, io.EOF)

	gRPCClient := mockGRPStacksClient(t, ctrl, executeClient, nil)

	var logBuffer bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuffer)
	defer log.SetOutput(originalOutput)

	// use executeWithBrokerIDs instead of Execute to allow mocking of the broker IDs
	// This is necessary because the plugin.GRPCBroker cannot be mocked except the actual plugin process is started.
	exitCode := gRPCClient.executeWithBrokers(mockBrokerIDs, []string{"init"}, io.Discard, io.Discard)
	if exitCode != 1 {
		t.Fatalf("expected exit %d, got %d", 1, exitCode)
	}

	recvLog := "[DEBUG] received EOF from stacksplugin\n"
	if logBuffer.String() != recvLog {
		t.Errorf("expected EOF message %q, got %q", recvLog, logBuffer.String())
	}
}

func Test_GRPCStacksClient_Execute_Invalid_Exit(t *testing.T) {
	ctrl := gomock.NewController(t)
	executeClient := mock_stacksproto1.NewMockCommandService_ExecuteClient(ctrl)

	executeClient.EXPECT().Recv().Return(
		&stacksproto1.CommandResponse{
			Data: &stacksproto1.CommandResponse_ExitCode{
				ExitCode: 3_000,
			},
		}, nil,
	)
	gRPCClient := mockGRPStacksClient(t, ctrl, executeClient, nil)

	var logBuffer bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuffer)
	defer log.SetOutput(originalOutput)

	// use executeWithBrokerIDs instead of Execute to allow mocking of the broker IDs
	// This is necessary because the plugin.GRPCBroker cannot be mocked except the actual plugin process is started.
	exitCode := gRPCClient.executeWithBrokers(mockBrokerIDs, []string{"init"}, io.Discard, io.Discard)
	if exitCode != 255 {
		t.Fatalf("expected exit %q, got %q", 255, exitCode)
	}

	recvLog := "[TRACE] received exit code: 3000\n[ERROR] stacksplugin returned an invalid error code 3000\n"
	if logBuffer.String() != recvLog {
		t.Errorf("expected error message %q, got %q", recvLog, logBuffer.String())
	}
}

func Test_GRPCStacksClient_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	executeClient := mock_stacksproto1.NewMockCommandService_ExecuteClient(ctrl)

	gomock.InOrder(
		executeClient.EXPECT().Recv().Return(
			&stacksproto1.CommandResponse{
				Data: &stacksproto1.CommandResponse_Stdout{
					Stdout: []byte("firstresponse\n"),
				},
			}, nil,
		),
		executeClient.EXPECT().Recv().Return(
			&stacksproto1.CommandResponse{
				Data: &stacksproto1.CommandResponse_Stdout{
					Stdout: []byte("secondresponse\n"),
				},
			}, nil,
		),
		executeClient.EXPECT().Recv().Return(
			&stacksproto1.CommandResponse{
				Data: &stacksproto1.CommandResponse_ExitCode{
					ExitCode: 99,
				},
			}, nil,
		),
	)

	gRPCClient := mockGRPStacksClient(t, ctrl, executeClient, nil)

	stdoutBuffer := bytes.Buffer{}
	// use executeWithBrokerIDs instead of Execute to allow mocking of the broker IDs
	// This is necessary because the plugin.GRPCBroker cannot be mocked except the actual plugin process is started.
	exitCode := gRPCClient.executeWithBrokers(mockBrokerIDs, []string{"example"}, &stdoutBuffer, io.Discard)
	if exitCode != 99 {
		t.Fatalf("expected exit %q, got %q", 99, exitCode)
	}

	recvResponse := "firstresponse\nsecondresponse\n"
	if stdoutBuffer.String() != recvResponse {
		t.Errorf("expected output %q, got %q", recvResponse, stdoutBuffer.String())
	}
}
