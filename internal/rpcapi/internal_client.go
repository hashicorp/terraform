// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/dependencies"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/packages"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/setup"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
)

// Client is a client for the RPC API.
//
// This just wraps a raw gRPC client connection and provides a more convenient
// API to access its services.
type Client struct {
	// conn should be a connection to a server that has already completed
	// the Setup.Handshake call.
	conn *grpc.ClientConn
	// serverCaps should be from the result of the Setup.Handshake call
	// previously made to the server that conn is connected to.
	serverCaps *setup.ServerCapabilities

	close func(context.Context) error
}

// NewInternalClient returns a client for the RPC API that uses in-memory
// buffers to allow callers within the same Terraform CLI process to access
// the RPC API without any sockets or child processes.
//
// This is intended for exposing Terraform Core functionality through Terraform
// CLI, to establish an explicit interface between those two sides without
// the overhead of forking a child process containing exactly the same code.
//
// Callers should call the Close method of the returned client once they are
// done using it, or else they will leak goroutines.
func NewInternalClient(ctx context.Context, clientCaps *setup.ClientCapabilities) (*Client, error) {
	fakeListener := bufconn.Listen(4 * 1024 * 1024 /* buffer size */)
	srv := grpc.NewServer()
	registerGRPCServices(srv, &serviceOpts{})

	go func() {
		if err := srv.Serve(fakeListener); err != nil {
			// We can't actually return an error here, but this should
			// not arise with our fake listener anyway so we'll just panic.
			panic(err)
		}
	}()

	fakeDialer := func(ctx context.Context, fakeAddr string) (net.Conn, error) {
		return fakeListener.DialContext(ctx)
	}
	clientConn, err := grpc.DialContext(
		ctx, "testfake",
		grpc.WithContextDialer(fakeDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC API: %w", err)
	}

	// We perform the setup step on the caller's behalf, so that they can
	// immediately use the main services. (The caller would otherwise need
	// to do this immediately on return anyway, or the result would be
	// useless.)
	setupClient := setup.NewSetupClient(clientConn)
	setupResp, err := setupClient.Handshake(ctx, &setup.Handshake_Request{
		Capabilities: clientCaps,
	})
	if err != nil {
		return nil, fmt.Errorf("setup failed: %w", err)
	}

	var client *Client
	client = &Client{
		conn:       clientConn,
		serverCaps: setupResp.Capabilities,
		close: func(ctx context.Context) error {
			clientConn.Close()
			srv.Stop()
			fakeListener.Close()
			client.conn = nil
			client.serverCaps = nil
			client.close = func(context.Context) error {
				return nil
			}
			return nil
		},
	}

	return client, nil
}

// Close frees the internal buffers and terminates the goroutines that handle
// the internal RPC API connection.
//
// Any service clients previously returned by other methods become invalid
// as soon as this method is called, and must not be used any further.
func (c *Client) Close(ctx context.Context) error {
	return c.close(ctx)
}

// ServerCapabilities returns the server's response to capability negotiation.
//
// Callers must not modify anything reachable through the returned pointer.
func (c *Client) ServerCapabilities() *setup.ServerCapabilities {
	return c.serverCaps
}

// Dependencies returns a client for the Dependencies service of the RPC API.
func (c *Client) Dependencies() dependencies.DependenciesClient {
	return dependencies.NewDependenciesClient(c.conn)
}

// Packages returns a client for the Packages service of the RPC API.
func (c *Client) Packages() packages.PackagesClient {
	return packages.NewPackagesClient(c.conn)
}

// Stacks returns a client for the Stacks service of the RPC API.
func (c *Client) Stacks() stacks.StacksClient {
	return stacks.NewStacksClient(c.conn)
}
