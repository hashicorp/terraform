package main

import (
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/rpc"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-libucl"
	"github.com/mitchellh/osext"
)

// TFConfig is the global base configuration that has the
// basic providers registered. Users of this configuration
// should copy it (call the Copy method) before using it so
// that it isn't corrupted.
var TFConfig terraform.Config

// Config is the structure of the configuration for the Terraform CLI.
//
// This is not the configuration for Terraform itself. That is in the
// "config" package.
type Config struct {
	Providers map[string]string
}

// BuiltinConfig is the built-in defaults for the configuration. These
// can be overridden by user configurations.
var BuiltinConfig Config

// Put the parse flags we use for libucl in a constant so we can get
// equally behaving parsing everywhere.
const libuclParseFlags = libucl.ParserKeyLowercase

func init() {
	BuiltinConfig.Providers = map[string]string{
		"aws": "terraform-provider-aws",
	}
}

// LoadConfig loads the CLI configuration from ".terraformrc" files.
func LoadConfig(path string) (*Config, error) {
	var obj *libucl.Object

	// Parse the file and get the root object.
	parser := libucl.NewParser(libuclParseFlags)
	err := parser.AddFile(path)
	if err == nil {
		obj = parser.Object()
		defer obj.Close()
	}
	defer parser.Close()

	// If there was an error parsing, return now.
	if err != nil {
		return nil, err
	}

	// Build up the result
	var result Config

	if err := obj.Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Merge merges two configurations and returns a third entirely
// new configuration with the two merged.
func (c1 *Config) Merge(c2 *Config) *Config {
	var result Config
	result.Providers = make(map[string]string)
	for k, v := range c1.Providers {
		result.Providers[k] = v
	}
	for k, v := range c2.Providers {
		result.Providers[k] = v
	}

	return &result
}

// ProviderFactories returns the mapping of prefixes to
// ResourceProviderFactory that can be used to instantiate a
// binary-based plugin.
func (c *Config) ProviderFactories() map[string]terraform.ResourceProviderFactory {
	result := make(map[string]terraform.ResourceProviderFactory)
	for k, v := range c.Providers {
		result[k] = c.providerFactory(v)
	}

	return result
}

func (c *Config) providerFactory(path string) terraform.ResourceProviderFactory {
	originalPath := path

	return func() (terraform.ResourceProvider, error) {
		// First look for the provider on the PATH.
		path, err := exec.LookPath(path)
		if err != nil {
			// If that doesn't work, look for it in the same directory
			// as the executable that is running.
			exePath, err := osext.Executable()
			if err == nil {
				path = filepath.Join(
					filepath.Dir(exePath),
					filepath.Base(originalPath))
			}
		}

		// If we still don't have a path set, then set it to the
		// original path and let any errors that happen bubble out.
		if path == "" {
			path = originalPath
		}

		// Build the plugin client configuration and init the plugin
		var config plugin.ClientConfig
		config.Cmd = exec.Command(path)
		config.Managed = true
		client := plugin.NewClient(&config)

		// Request the RPC client and service name from the client
		// so we can build the actual RPC-implemented provider.
		rpcClient, err := client.Client()
		if err != nil {
			return nil, err
		}

		service, err := client.Service()
		if err != nil {
			return nil, err
		}

		return &rpc.ResourceProvider{
			Client: rpcClient,
			Name:   service,
		}, nil
	}
}
