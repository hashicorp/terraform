package plugin

import (
	"github.com/hashicorp/go-plugin"
)

var VersionedPlugins = map[int]plugin.PluginSet{
	5: {
		"provider":    &GRPCProviderPlugin{},
		"provisioner": &GRPCProvisionerPlugin{},
	},
}
