//go:generate go run ./scripts/generate-plugins.go
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/terraform/command"
)

// Config is the structure of the configuration for the Terraform CLI.
//
// This is not the configuration for Terraform itself. That is in the
// "config" package.
type Config struct {
	Providers    map[string]string
	Provisioners map[string]string

	DisableCheckpoint          bool `hcl:"disable_checkpoint"`
	DisableCheckpointSignature bool `hcl:"disable_checkpoint_signature"`
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

	return &result, nil
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

	return &result
}
