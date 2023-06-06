package rpcapi

import (
	"context"

	"github.com/hashicorp/go-plugin"
)

// ServePlugin attempts to complete the go-plugin protocol handshake, and then
// if successful starts the plugin server and blocks until externally
// terminated.
func ServePlugin(ctx context.Context) error {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshake,
		VersionedPlugins: map[int]plugin.PluginSet{
			1: plugin.PluginSet{
				"tfcore": &corePlugin{},
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
	return nil
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
