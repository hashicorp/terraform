//go:generate go run ./scripts/generate-plugins.go
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/hashicorp/hcl"

	"github.com/hashicorp/terraform/command"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/tfdiags"
)

const pluginCacheDirEnvVar = "TF_PLUGIN_CACHE_DIR"

// Config is the structure of the configuration for the Terraform CLI.
//
// This is not the configuration for Terraform itself. That is in the
// "config" package.
type Config struct {
	Providers    map[string]string
	Provisioners map[string]string

	DisableCheckpoint          bool `hcl:"disable_checkpoint"`
	DisableCheckpointSignature bool `hcl:"disable_checkpoint_signature"`

	// If set, enables local caching of plugins in this directory to
	// avoid repeatedly re-downloading over the Internet.
	PluginCacheDir string `hcl:"plugin_cache_dir"`

	Credentials        map[string]map[string]interface{}   `hcl:"credentials"`
	CredentialsHelpers map[string]*ConfigCredentialsHelper `hcl:"credentials_helper"`
}

// ConfigCredentialsHelper is the structure of the "credentials_helper"
// nested block within the CLI configuration.
type ConfigCredentialsHelper struct {
	Args []string `hcl:"args"`
}

// BuiltinConfig is the built-in defaults for the configuration. These
// can be overridden by user configurations.
var BuiltinConfig Config

// PluginOverrides are paths that override discovered plugins, set from
// the config file.
var PluginOverrides command.PluginOverrides

// ConfigFile returns the default path to the configuration file.
//
// On Unix-like systems this is the ".terraformrc" file in the home directory.
// On Windows, this is the "terraform.rc" file in the application data
// directory.
func ConfigFile() (string, error) {
	return configFile()
}

// ConfigDir returns the configuration directory for Terraform.
func ConfigDir() (string, error) {
	return configDir()
}

// LoadConfig loads the CLI configuration from ".terraformrc" files.
func LoadConfig(path string) (*Config, error) {
	// Read the HCL file and prepare for parsing
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"Error reading %s: %s", path, err)
	}

	// Parse it
	obj, err := hcl.Parse(string(d))
	if err != nil {
		return nil, fmt.Errorf(
			"Error parsing %s: %s", path, err)
	}

	// Build up the result
	var result Config
	if err := hcl.DecodeObject(&result, obj); err != nil {
		return nil, err
	}

	// Replace all env vars
	for k, v := range result.Providers {
		result.Providers[k] = os.ExpandEnv(v)
	}
	for k, v := range result.Provisioners {
		result.Provisioners[k] = os.ExpandEnv(v)
	}

	if result.PluginCacheDir != "" {
		result.PluginCacheDir = os.ExpandEnv(result.PluginCacheDir)
	}

	return &result, nil
}

// EnvConfig returns a Config populated from environment variables.
//
// Any values specified in this config should override those set in the
// configuration file.
func EnvConfig() *Config {
	config := &Config{}

	if envPluginCacheDir := os.Getenv(pluginCacheDirEnvVar); envPluginCacheDir != "" {
		// No Expandenv here, because expanding environment variables inside
		// an environment variable would be strange and seems unnecessary.
		// (User can expand variables into the value while setting it using
		// standard shell features.)
		config.PluginCacheDir = envPluginCacheDir
	}

	return config
}

// Validate checks for errors in the configuration that cannot be detected
// just by HCL decoding, returning any problems as diagnostics.
//
// On success, the returned diagnostics will return false from the HasErrors
// method. A non-nil diagnostics is not necessarily an error, since it may
// contain just warnings.
func (c *Config) Validate() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if c == nil {
		return diags
	}

	// FIXME: Right now our config parsing doesn't retain enough information
	// to give proper source references to any errors. We should improve
	// on this when we change the CLI config parser to use HCL2.

	// Check that all "credentials" blocks have valid hostnames.
	for givenHost := range c.Credentials {
		_, err := svchost.ForComparison(givenHost)
		if err != nil {
			diags = diags.Append(
				fmt.Errorf("The credentials %q block has an invalid hostname: %s", givenHost, err),
			)
		}
	}

	// Should have zero or one "credentials_helper" blocks
	if len(c.CredentialsHelpers) > 1 {
		diags = diags.Append(
			fmt.Errorf("No more than one credentials_helper block may be specified"),
		)
	}

	return diags
}

// Merge merges two configurations and returns a third entirely
// new configuration with the two merged.
func (c1 *Config) Merge(c2 *Config) *Config {
	var result Config
	result.Providers = make(map[string]string)
	result.Provisioners = make(map[string]string)
	for k, v := range c1.Providers {
		result.Providers[k] = v
	}
	for k, v := range c2.Providers {
		if v1, ok := c1.Providers[k]; ok {
			log.Printf("[INFO] Local %s provider configuration '%s' overrides '%s'", k, v, v1)
		}
		result.Providers[k] = v
	}
	for k, v := range c1.Provisioners {
		result.Provisioners[k] = v
	}
	for k, v := range c2.Provisioners {
		if v1, ok := c1.Provisioners[k]; ok {
			log.Printf("[INFO] Local %s provisioner configuration '%s' overrides '%s'", k, v, v1)
		}
		result.Provisioners[k] = v
	}
	result.DisableCheckpoint = c1.DisableCheckpoint || c2.DisableCheckpoint
	result.DisableCheckpointSignature = c1.DisableCheckpointSignature || c2.DisableCheckpointSignature

	result.PluginCacheDir = c1.PluginCacheDir
	if result.PluginCacheDir == "" {
		result.PluginCacheDir = c2.PluginCacheDir
	}

	if (len(c1.Credentials) + len(c2.Credentials)) > 0 {
		result.Credentials = make(map[string]map[string]interface{})
		for host, creds := range c1.Credentials {
			result.Credentials[host] = creds
		}
		for host, creds := range c2.Credentials {
			// We just clobber an entry from the other file right now. Will
			// improve on this later using the more-robust merging behavior
			// built in to HCL2.
			result.Credentials[host] = creds
		}
	}

	if (len(c1.CredentialsHelpers) + len(c2.CredentialsHelpers)) > 0 {
		result.CredentialsHelpers = make(map[string]*ConfigCredentialsHelper)
		for name, helper := range c1.CredentialsHelpers {
			result.CredentialsHelpers[name] = helper
		}
		for name, helper := range c2.CredentialsHelpers {
			result.CredentialsHelpers[name] = helper
		}
	}

	return &result
}
