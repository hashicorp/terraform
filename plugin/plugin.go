package rpc

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
