package plugin

import (
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/terraform"
)

// The constants below are the names of the plugins that can be dispensed
// from the plugin server.
const (
	ProviderPluginName    = "provider"
	ProvisionerPluginName = "provisioner"
)

// Handshake is the HandshakeConfig used to configure clients and servers.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
}

type ProviderFunc func() terraform.ResourceProvider
type ProvisionerFunc func() terraform.ResourceProvisioner

// ServeOpts are the configurations to serve a plugin.
type ServeOpts struct {
	ProviderFunc    ProviderFunc
	ProvisionerFunc ProvisionerFunc
}

// Serve serves a plugin. This function never returns and should be the final
// function called in the main function of the plugin.
func Serve(opts *ServeOpts) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins:         pluginMap(opts),
	})
}

// pluginMap returns the map[string]plugin.Plugin to use for configuring a plugin
// server or client.
func pluginMap(opts *ServeOpts) map[string]plugin.Plugin {
	return map[string]plugin.Plugin{
		"provider":    &ResourceProviderPlugin{F: opts.ProviderFunc},
		"provisioner": &ResourceProvisionerPlugin{F: opts.ProvisionerFunc},
	}
}
