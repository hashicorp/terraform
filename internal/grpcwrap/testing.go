// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package grpcwrap

import (
	"context"
	"log"
	"net"
	"testing"

	"github.com/hashicorp/terraform/internal/plugin6"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfplugin6"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewGRPCProvider wraps an internal providers.Interface
// implementation in a local gRPC server and returns a new providers.Interface
// that can be used to interact with the inner provider over gRPC and
// through the plugin6.GRPCProviderPlugin interface.
//
// This mimics the behavior of the Terraform CLI when it starts a provider plugin
// server and connects to it, allowing tests to use the same gRPC interface
// that the CLI uses to interact with providers.
func NewGRPCProvider(t *testing.T, inner providers.Interface) (providers.Interface, func()) {
	t.Helper()

	// Start a gRPC server on a random local port
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer()
	tfplugin6.RegisterProviderServer(server, Provider6(inner))

	go func() {
		if err := server.Serve(lis); err != nil {
			log.Printf("gRPC server exited: %v", err)
		}
	}()

	// Connect a client
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	// Construct the client-side object via the plugin6.GRPCProviderPlugin interface
	plugin := &plugin6.GRPCProviderPlugin{}
	clientIface, err := plugin.GRPCClient(context.Background(), nil, conn)
	if err != nil {
		t.Fatalf("failed to create GRPCProvider client: %v", err)
	}
	p, ok := clientIface.(*plugin6.GRPCProvider)
	if !ok {
		t.Fatalf("unexpected type from GRPCClient: %T", clientIface)
	}
	return p, func() {
		conn.Close()
		server.Stop()
		lis.Close()
	}
}
