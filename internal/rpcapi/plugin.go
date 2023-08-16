package rpcapi

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/rpcapi/dynrpcserver"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"google.golang.org/grpc"
)

type corePlugin struct {
	plugin.Plugin

	experimentsAllowed bool
}

func (p *corePlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	// This codebase only provides a server implementation of this plugin.
	// Clients must live elsewhere.
	return nil, fmt.Errorf("there is no client implementation in this codebase")
}

func (p *corePlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// We initially only register the setup server, because the registration
	// of other services can vary depending on the capabilities negotiated
	// during handshake.
	setup := newSetupServer(p.handshakeFunc(s))
	terraform1.RegisterSetupServer(s, setup)
	return nil
}

func (p *corePlugin) handshakeFunc(s *grpc.Server) func(context.Context, *terraform1.ClientCapabilities) (*terraform1.ServerCapabilities, error) {
	dependencies := dynrpcserver.NewDependenciesStub()
	terraform1.RegisterDependenciesServer(s, dependencies)
	stacks := dynrpcserver.NewStacksStub()
	terraform1.RegisterStacksServer(s, stacks)

	return func(ctx context.Context, clientCaps *terraform1.ClientCapabilities) (*terraform1.ServerCapabilities, error) {
		// All of our servers will share a common handles table so that objects
		// can be passed from one service to another.
		handles := newHandleTable()

		// If handshaking is successful (which it currently always is, because
		// we don't have any special capabilities to negotiate yet) then we
		// will initialize all of the other services so the client can begin
		// doing real work. In future the details of what we register here
		// might vary based on the negotiated capabilities.
		dependencies.ActivateRPCServer(newDependenciesServer(handles))
		stacks.ActivateRPCServer(newStacksServer(handles))

		// If the client requested any extra capabililties that we're going
		// to honor then we should announce them in this result.
		return &terraform1.ServerCapabilities{}, nil
	}
}
