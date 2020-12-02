// Package cliconfig has the types representing and the logic to load CLI-level
// configuration settings.
//
// The CLI config is a small collection of settings that a user can override via
// some files in their home directory or, in some cases, via environment
// variables. The CLI config is not the same thing as a Terraform configuration
// written in the Terraform language; the logic for those lives in the top-level
// directory "configs".
package cliconfig

import (
	"os"
)

const pluginCacheDirEnvVar = "TF_PLUGIN_CACHE_DIR"

// EnvConfig returns a Config populated from environment variables.
//
// Any values specified in this config should override those set in the
// configuration file.
func EnvConfig() *LegacyConfig {
	config := &LegacyConfig{}

	if envPluginCacheDir := os.Getenv(pluginCacheDirEnvVar); envPluginCacheDir != "" {
		// No Expandenv here, because expanding environment variables inside
		// an environment variable would be strange and seems unnecessary.
		// (User can expand variables into the value while setting it using
		// standard shell features.)
		config.PluginCacheDir = envPluginCacheDir
	}

	return config
}
