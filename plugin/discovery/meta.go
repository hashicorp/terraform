package discovery

import (
	"crypto/sha256"
	"io"
	"os"
	"os/exec"

	"github.com/blang/semver"
	plugin "github.com/hashicorp/go-plugin"
	tfplugin "github.com/hashicorp/terraform/plugin"
)

// PluginMeta is metadata about a plugin, useful for launching the plugin
// and for understanding which plugins are available.
type PluginMeta struct {
	// Name is the name of the plugin, e.g. as inferred from the plugin
	// binary's filename, or by explicit configuration.
	Name string

	// Version is the semver version of the plugin, expressed as a string
	// that might not be semver-valid. (Call VersionObj to attempt to
	// parse it and thus detect if it is invalid.)
	Version string

	// Path is the absolute path of the executable that can be launched
	// to provide the RPC server for this plugin.
	Path string
}

// VersionObj returns the semver version of the plugin as an object, or
// an error if the version string is not semver-syntax-compliant.
func (m PluginMeta) VersionObj() (semver.Version, error) {
	return semver.Make(m.Version)
}

// SHA256 returns a SHA256 hash of the content of the referenced executable
// file, or an error if the file's contents cannot be read.
func (m PluginMeta) SHA256() ([]byte, error) {
	f, err := os.Open(m.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// ClientConfig returns a configuration object that can be used to instantiate
// a client for the referenced plugin.
func (m PluginMeta) ClientConfig() *plugin.ClientConfig {
	return &plugin.ClientConfig{
		Cmd:             exec.Command(m.Path),
		HandshakeConfig: tfplugin.Handshake,
		Managed:         true,
		Plugins:         tfplugin.PluginMap,
	}
}

// Client returns a plugin client for the referenced plugin.
func (m PluginMeta) Client() *plugin.Client {
	return plugin.NewClient(m.ClientConfig())
}
