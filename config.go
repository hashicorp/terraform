//go:generate go run ./scripts/generate-plugins.go
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/terraform/command"
	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
	"github.com/kardianos/osext"
	"github.com/mitchellh/cli"
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

// ContextOpts are the global ContextOpts we use to initialize the CLI.
var ContextOpts terraform.ContextOpts

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

// Discover plugins located on disk, and fall back on plugins baked into the
// Terraform binary.
//
// We look in the following places for plugins:
//
// 1. Terraform configuration path
// 2. Path where Terraform is installed
// 3. Path where Terraform is invoked
//
// Whichever file is discoverd LAST wins.
//
// Finally, we look at the list of plugins compiled into Terraform. If any of
// them has not been found on disk we use the internal version. This allows
// users to add / replace plugins without recompiling the main binary.
func (c *Config) Discover(ui cli.Ui) error {
	// Look in ~/.terraform.d/plugins/
	dir, err := ConfigDir()
	if err != nil {
		log.Printf("[ERR] Error loading config directory: %s", err)
	} else {
		if err := c.discover(filepath.Join(dir, "plugins")); err != nil {
			return err
		}
	}

	// Next, look in the same directory as the Terraform executable, usually
	// /usr/local/bin. If found, this replaces what we found in the config path.
	exePath, err := osext.Executable()
	if err != nil {
		log.Printf("[ERR] Error loading exe directory: %s", err)
	} else {
		if err := c.discover(filepath.Dir(exePath)); err != nil {
			return err
		}
	}

	// Finally look in the cwd (where we are invoke Terraform). If found, this
	// replaces anything we found in the config / install paths.
	if err := c.discover("."); err != nil {
		return err
	}

	// Finally, if we have a plugin compiled into Terraform and we didn't find
	// a replacement on disk, we'll just use the internal version. Only do this
	// from the main process, or the log output will break the plugin handshake.
	if os.Getenv("TF_PLUGIN_MAGIC_COOKIE") == "" {
		for name, _ := range command.InternalProviders {
			if path, found := c.Providers[name]; found {
				// Allow these warnings to be suppressed via TF_PLUGIN_DEV=1 or similar
				if os.Getenv("TF_PLUGIN_DEV") == "" {
					ui.Warn(fmt.Sprintf("[WARN] %s overrides an internal plugin for %s-provider.\n"+
						"  If you did not expect to see this message you will need to remove the old plugin.\n"+
						"  See https://www.terraform.io/docs/internals/internal-plugins.html", path, name))
				}
			} else {
				cmd, err := command.BuildPluginCommandString("provider", name)
				if err != nil {
					return err
				}
				c.Providers[name] = cmd
			}
		}
		for name, _ := range command.InternalProvisioners {
			if path, found := c.Provisioners[name]; found {
				if os.Getenv("TF_PLUGIN_DEV") == "" {
					ui.Warn(fmt.Sprintf("[WARN] %s overrides an internal plugin for %s-provisioner.\n"+
						"  If you did not expect to see this message you will need to remove the old plugin.\n"+
						"  See https://www.terraform.io/docs/internals/internal-plugins.html", path, name))
				}
			} else {
				cmd, err := command.BuildPluginCommandString("provisioner", name)
				if err != nil {
					return err
				}
				c.Provisioners[name] = cmd
			}
		}
	}

	return nil
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

func (c *Config) discover(path string) error {
	var err error

	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
	}

	err = c.discoverSingle(
		filepath.Join(path, "terraform-provider-*"), &c.Providers)
	if err != nil {
		return err
	}

	err = c.discoverSingle(
		filepath.Join(path, "terraform-provisioner-*"), &c.Provisioners)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) discoverSingle(glob string, m *map[string]string) error {
	matches, err := filepath.Glob(glob)
	if err != nil {
		return err
	}

	if *m == nil {
		*m = make(map[string]string)
	}

	for _, match := range matches {
		file := filepath.Base(match)

		// If the filename has a ".", trim up to there
		if idx := strings.Index(file, "."); idx >= 0 {
			file = file[:idx]
		}

		// Look for foo-bar-baz. The plugin name is "baz"
		parts := strings.SplitN(file, "-", 3)
		if len(parts) != 3 {
			continue
		}

		log.Printf("[DEBUG] Discovered plugin: %s = %s", parts[2], match)
		(*m)[parts[2]] = match
	}

	return nil
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
	// Build the plugin client configuration and init the plugin
	var config plugin.ClientConfig
	config.Cmd = pluginCmd(path)
	config.HandshakeConfig = tfplugin.Handshake
	config.Managed = true
	config.Plugins = tfplugin.PluginMap
	client := plugin.NewClient(&config)

	return func() (terraform.ResourceProvider, error) {
		// Request the RPC client so we can get the provider
		// so we can build the actual RPC-implemented provider.
		rpcClient, err := client.Client()
		if err != nil {
			return nil, err
		}

		raw, err := rpcClient.Dispense(tfplugin.ProviderPluginName)
		if err != nil {
			return nil, err
		}

		return raw.(terraform.ResourceProvider), nil
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
	// Build the plugin client configuration and init the plugin
	var config plugin.ClientConfig
	config.HandshakeConfig = tfplugin.Handshake
	config.Cmd = pluginCmd(path)
	config.Managed = true
	config.Plugins = tfplugin.PluginMap
	client := plugin.NewClient(&config)

	return func() (terraform.ResourceProvisioner, error) {
		rpcClient, err := client.Client()
		if err != nil {
			return nil, err
		}

		raw, err := rpcClient.Dispense(tfplugin.ProvisionerPluginName)
		if err != nil {
			return nil, err
		}

		return raw.(terraform.ResourceProvisioner), nil
	}
}

func pluginCmd(path string) *exec.Cmd {
	cmdPath := ""

	// If the path doesn't contain a separator, look in the same
	// directory as the Terraform executable first.
	if !strings.ContainsRune(path, os.PathSeparator) {
		exePath, err := osext.Executable()
		if err == nil {
			temp := filepath.Join(
				filepath.Dir(exePath),
				filepath.Base(path))

			if _, err := os.Stat(temp); err == nil {
				cmdPath = temp
			}
		}

		// If we still haven't found the executable, look for it
		// in the PATH.
		if v, err := exec.LookPath(path); err == nil {
			cmdPath = v
		}
	}

	// No plugin binary found, so try to use an internal plugin.
	if strings.Contains(path, command.TFSPACE) {
		parts := strings.Split(path, command.TFSPACE)
		return exec.Command(parts[0], parts[1:]...)
	}

	// If we still don't have a path, then just set it to the original
	// given path.
	if cmdPath == "" {
		cmdPath = path
	}

	// Build the command to execute the plugin
	return exec.Command(cmdPath)
}
