// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"errors"
	"os"

	"github.com/hashicorp/go-plugin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// ServePlugin attempts to complete the go-plugin protocol handshake, and then
// if successful starts the plugin server and blocks until the given context
// is cancelled.
//
// Returns [ErrNotPluginClient] if this program doesn't seem to be running as
// the child of a plugin client, which is detected based on a magic environment
// variable that the client ought to have set.
func ServePlugin(ctx context.Context, opts ServerOpts) error {
	// go-plugin has its own check for the environment variable magic cookie
	// but it returns a generic error message. We'll pre-check it out here
	// instead so we can return a more specific error message.
	if os.Getenv(handshake.MagicCookieKey) != handshake.MagicCookieValue {
		return ErrNotPluginClient
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshake,
		VersionedPlugins: map[int]plugin.PluginSet{
			1: {
				"tfcore": &corePlugin{
					experimentsAllowed: opts.ExperimentsAllowed,
				},
			},
		},
		GRPCServer: func(opts []grpc.ServerOption) *grpc.Server {
			fullOpts := []grpc.ServerOption{
				grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
				grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
			}
			fullOpts = append(fullOpts, opts...)
			server := grpc.NewServer(fullOpts...)
			// We'll also monitor the given context for cancellation
			// and terminate the server gracefully if we get cancelled.
			go func() {
				<-ctx.Done()
				server.GracefulStop()
				// The above will block until all of the pending RPCs have
				// finished.
				os.Exit(0)
			}()
			return server
		},
	})
	return nil
}

var ErrNotPluginClient = errors.New("caller is not a plugin client")

type ServerOpts struct {
	ExperimentsAllowed bool
}

// handshake is the HandshakeConfig used to begin negotiation between client
// and server.
var handshake = plugin.HandshakeConfig{
	// The ProtocolVersion is the version that must match between TF core
	// and TF plugins.
	ProtocolVersion: 1,

	// The magic cookie values should NEVER be changed.
	MagicCookieKey:   "TERRAFORM_RPCAPI_COOKIE",
	MagicCookieValue: "fba0991c9bcd453982f0d88e2da95940",
}
