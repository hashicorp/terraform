package plugin

import (
	"os"
	"os/exec"

	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
)

// The TF_DISABLE_PLUGIN_TLS environment variable is intended only for use by
// the plugin SDK test framework. We do not recommend Terraform CLI end-users
// set this variable.
var enableAutoMTLS = os.Getenv("TF_DISABLE_PLUGIN_TLS") == ""

// ClientConfig returns a configuration object that can be used to instantiate
// a client for the plugin described by the given metadata.
func ClientConfig(m discovery.PluginMeta) *plugin.ClientConfig {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Level:  hclog.Trace,
		Output: os.Stderr,
	})

	return &plugin.ClientConfig{
		Cmd:              exec.Command(m.Path),
		HandshakeConfig:  Handshake,
		VersionedPlugins: VersionedPlugins,
		Managed:          true,
		Logger:           logger,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		AutoMTLS:         enableAutoMTLS,
	}
}

// Client returns a plugin client for the plugin described by the given metadata.
func Client(m discovery.PluginMeta) *plugin.Client {
	return plugin.NewClient(ClientConfig(m))
}
