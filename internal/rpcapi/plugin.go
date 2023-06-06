package rpcapi

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-plugin"
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
	return func(ctx context.Context, clientCaps *terraform1.ClientCapabilities) (*terraform1.ServerCapabilities, error) {

		// If handshaking is successful (which it currently always is, because
		// we don't have any special capabilities to negotiate yet) then we
		// will register all of the other services so the client can being
		// doing real work. In future the details of what we register here
		// might vary based on the negotiated capabilities.
		terraform1.RegisterDependenciesServer(s, &dependenciesServer{})
		return &terraform1.ServerCapabilities{}, nil
	}
}
