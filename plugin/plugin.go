package plugin

import (
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/plugin6"
)

var VersionedPlugins = map[int]plugin.PluginSet{
	5: {
		"provider":    &GRPCProviderPlugin{},
		"provisioner": &GRPCProvisionerPlugin{},
	},
	6: {
		"provider": &plugin6.GRPCProviderPlugin{},
	},
}
