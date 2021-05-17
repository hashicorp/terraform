package plugin6

import (
	"github.com/hashicorp/go-plugin"
	proto "github.com/hashicorp/terraform/internal/tfplugin6"
)

const (
	// The constants below are the names of the plugins that can be dispensed
	// from the plugin server.
	ProviderPluginName = "provider"

	// DefaultProtocolVersion is the protocol version assumed for legacy clients
	// that don't specify a particular version during their handshake. Since we
	// explicitly set VersionedPlugins in Serve, this number does not need to
	// change with the protocol version and can effectively stay 4 forever
	// (unless we need the "biggest hammer" approach to break all provider
	// compatibility).
	DefaultProtocolVersion = 4
)

// Handshake is the HandshakeConfig used to configure clients and servers.
var Handshake = plugin.HandshakeConfig{
	// The ProtocolVersion is the version that must match between TF core
	// and TF plugins.
	ProtocolVersion: DefaultProtocolVersion,

	// The magic cookie values should NEVER be changed.
	MagicCookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
}

type GRPCProviderFunc func() proto.ProviderServer

// ServeOpts are the configurations to serve a plugin.
type ServeOpts struct {
	GRPCProviderFunc GRPCProviderFunc
}

// Serve serves a plugin. This function never returns and should be the final
// function called in the main function of the plugin.
func Serve(opts *ServeOpts) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  Handshake,
		VersionedPlugins: pluginSet(opts),
		GRPCServer:       plugin.DefaultGRPCServer,
	})
}

func pluginSet(opts *ServeOpts) map[int]plugin.PluginSet {
	plugins := map[int]plugin.PluginSet{}

	// add the new protocol versions if they're configured
	if opts.GRPCProviderFunc != nil {
		plugins[6] = plugin.PluginSet{}
		if opts.GRPCProviderFunc != nil {
			plugins[6]["provider"] = &GRPCProviderPlugin{
				GRPCProvider: opts.GRPCProviderFunc,
			}
		}
	}
	return plugins
}
