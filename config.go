package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/rpc"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/osext"
)

// Config is the structure of the configuration for the Terraform CLI.
//
// This is not the configuration for Terraform itself. That is in the
// "config" package.
type Config struct {
	Providers    map[string]string
	Provisioners map[string]string
}

// BuiltinConfig is the built-in defaults for the configuration. These
// can be overridden by user configurations.
var BuiltinConfig Config

// ContextOpts are the global ContextOpts we use to initialize the CLI.
var ContextOpts terraform.ContextOpts

func init() {
	BuiltinConfig.Providers = map[string]string{
		"aws":          "terraform-provider-aws",
		"digitalocean": "terraform-provider-digitalocean",
		"heroku":       "terraform-provider-heroku",
		"dnsimple":     "terraform-provider-dnsimple",
		"consul":       "terraform-provider-consul",
		"cloudflare":   "terraform-provider-cloudflare",
	}
	BuiltinConfig.Provisioners = map[string]string{
		"local-exec":  "terraform-provisioner-local-exec",
		"remote-exec": "terraform-provisioner-remote-exec",
		"file":        "terraform-provisioner-file",
	}
}

// ConfigFile returns the default path to the configuration file.
//
// On Unix-like systems this is the ".terraformrc" file in the home directory.
// On Windows, this is the "terraform.rc" file in the application data
// directory.
func ConfigFile() (string, error) {
    return configFile()
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
		result.Providers[k] = v
	}
	for k, v := range c1.Provisioners {
		result.Provisioners[k] = v
	}
	for k, v := range c2.Provisioners {
		result.Provisioners[k] = v
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
	return func() (terraform.ResourceProvider, error) {
		// Build the plugin client configuration and init the plugin
		var config plugin.ClientConfig
		config.Cmd = pluginCmd(path)
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

// ProvisionerFactories returns the mapping of prefixes to
// ResourceProvisionerFactory that can be used to instantiate a
// binary-based plugin.
func (c *Config) ProvisionerFactories() map[string]terraform.ResourceProvisionerFactory {
	result := make(map[string]terraform.ResourceProvisionerFactory)
	for k, v := range c.Provisioners {
		result[k] = c.provisionerFactory(v)
	}

	return result
}

func (c *Config) provisionerFactory(path string) terraform.ResourceProvisionerFactory {
	return func() (terraform.ResourceProvisioner, error) {
		// Build the plugin client configuration and init the plugin
		var config plugin.ClientConfig
		config.Cmd = pluginCmd(path)
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

		return &rpc.ResourceProvisioner{
			Client: rpcClient,
			Name:   service,
		}, nil
	}
}

func pluginCmd(path string) *exec.Cmd {
	originalPath := path

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

	// Build the command to execute the plugin
	return exec.Command(path)
}
