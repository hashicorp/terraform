// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackscliplugin1

import (
	"context"
	"errors"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/stackscliplugin"
	"github.com/hashicorp/terraform/internal/stackscliplugin/stackscliproto1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// GRPCStacksCLIPlugin is the go-plugin implementation, but only the client
// implementation exists in this package.
type GRPCStacksCLIPlugin struct {
	plugin.GRPCPlugin
	Impl stackscliplugin.StacksCLI1
	// Any configuration metadata that the plugin executable needs in order to
	// do something useful, which will be passed along via gRPC metadata headers.
	Metadata metadata.MD
}

// Server always returns an error; we're only implementing the GRPCPlugin
// interface, not the Plugin interface.
func (p *GRPCStacksCLIPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return nil, errors.New("stackscliplugin only implements gRPC clients")
}

// Client always returns an error; we're only implementing the GRPCPlugin
// interface, not the Plugin interface.
func (p *GRPCStacksCLIPlugin) Client(*plugin.MuxBroker, *rpc.Client) (interface{}, error) {
	return nil, errors.New("stackscliplugin only implements gRPC clients")
}

// GRPCServer always returns an error; we're only implementing the client
// interface, not the server.
func (p *GRPCStacksCLIPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return errors.New("stackscliplugin only implements gRPC clients")
}

// GRPCClient returns a new GRPC client for interacting with the cloud plugin server.
func (p *GRPCStacksCLIPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	ctx = metadata.NewOutgoingContext(ctx, p.Metadata)
	return &GRPCStacksCLIClient{
		client:  stackscliproto1.NewCommandServiceClient(c),
		context: ctx,
	}, nil
}
