package rpc

import (
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/terraform"
)

type ProviderFunc func() terraform.ResourceProvider
type ProvisionerFunc func() terraform.ResourceProvisioner

type Config struct {
	Provider    ProviderFunc
	Provisioner ProvisionerFunc
}

const (
	ProviderPluginName    = "provider"
	ProvisionerPluginName = "provisioner"
)

// Map returns the map[string]plugin.Plugin to use for configuring a plugin
// server or client.
func Map(c *Config) map[string]plugin.Plugin {
	return map[string]plugin.Plugin{
		"provider":    &ResourceProviderPlugin{F: c.Provider},
		"provisioner": &ResourceProvisionerPlugin{F: c.Provisioner},
	}
}
