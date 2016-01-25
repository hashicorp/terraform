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

// Config is used to configure Map to return the map necessary to configure
// clients and servers.
type Config struct {
	Provider    ProviderFunc
	Provisioner ProvisionerFunc
}

type ProviderFunc func() terraform.ResourceProvider
type ProvisionerFunc func() terraform.ResourceProvisioner

// Map returns the map[string]plugin.Plugin to use for configuring a plugin
// server or client.
func Map(c *Config) map[string]plugin.Plugin {
	return map[string]plugin.Plugin{
		"provider":    &ResourceProviderPlugin{F: c.Provider},
		"provisioner": &ResourceProvisionerPlugin{F: c.Provisioner},
	}
}

// Handshake is the HandshakeConfig used to configure clients and servers.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
}
