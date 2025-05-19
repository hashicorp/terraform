// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stacksplugin1

import (
	"context"
	"errors"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/pluginshared"
	"github.com/hashicorp/terraform/internal/rpcapi"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// GRPCCloudPlugin is the go-plugin implementation, but only the client
// implementation exists in this package.
type GRPCStacksPlugin struct {
	plugin.GRPCPlugin
	Metadata   metadata.MD
	Impl       pluginshared.CustomPluginClient
	Services   *disco.Disco
	ShutdownCh <-chan struct{}
}

// Server always returns an error; we're only implementing the GRPCPlugin
// interface, not the Plugin interface.
func (p *GRPCStacksPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return nil, errors.New("stacksplugin only implements gRPC clients")
}

// Client always returns an error; we're only implementing the GRPCPlugin
// interface, not the Plugin interface.
func (p *GRPCStacksPlugin) Client(*plugin.MuxBroker, *rpc.Client) (interface{}, error) {
	return nil, errors.New("stacksplugin only implements gRPC clients")
}

// GRPCClient returns a new GRPC client for interacting with the cloud plugin server.
func (p *GRPCStacksPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	ctx = metadata.NewOutgoingContext(ctx, p.Metadata)
	return &rpcapi.GRPCStacksClient{
		Client:     stacksproto1.NewCommandServiceClient(c),
		Broker:     broker,
		Services:   p.Services,
		Context:    ctx,
		ShutdownCh: p.ShutdownCh,
	}, nil
}

// GRPCServer always returns an error; we're only implementing the client
// interface, not the server.
func (p *GRPCStacksPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return errors.ErrUnsupported
}
