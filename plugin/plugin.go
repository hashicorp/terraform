package plugin

import (
	"github.com/hashicorp/go-plugin"
)

// See serve.go for serving plugins

// PluginMap should be used by clients for the map of plugins.
var PluginMap = map[string]plugin.Plugin{
	"provider":    &ResourceProviderPlugin{},
	"provisioner": &ResourceProvisionerPlugin{},
}
