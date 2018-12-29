package plugin

import (
	"github.com/hashicorp/go-plugin"
)

// See serve.go for serving plugins

var VersionedPlugins = map[int]plugin.PluginSet{
	5: {
		"provider":    &GRPCProviderPlugin{},
		"provisioner": &GRPCProvisionerPlugin{},
	},
}
