package discovery

import (
	"github.com/blang/semver"
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
