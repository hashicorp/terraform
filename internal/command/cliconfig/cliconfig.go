// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const pluginCacheDirEnvVar = "TF_PLUGIN_CACHE_DIR"
const pluginCacheMayBreakLockFileEnvVar = "TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE"

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

	// PluginCacheMayBreakDependencyLockFile is an interim accommodation for
	// those who wish to use the Plugin Cache Dir even in cases where doing so
	// will cause the dependency lock file to be incomplete.
	//
	// This is likely to become a silent no-op in future Terraform versions but
	// is here in recognition of the fact that the dependency lock file is not
	// yet a good fit for all Terraform workflows and folks in that category
	// would prefer to have the plugin cache dir's behavior to take priority
	// over the requirements of the dependency lock file.
	PluginCacheMayBreakDependencyLockFile bool `hcl:"plugin_cache_may_break_dependency_lock_file"`

	Hosts map[string]*ConfigHost `hcl:"host"`

	Credentials        map[string]map[string]interface{}   `hcl:"credentials"`
	CredentialsHelpers map[string]*ConfigCredentialsHelper `hcl:"credentials_helper"`

	// ProviderInstallation represents any provider_installation blocks
	// in the configuration. Only one of these is allowed across the whole
	// configuration, but we decode into a slice here so that we can handle
	// that validation at validation time rather than initial decode time.
	ProviderInstallation []*ProviderInstallation
}

// ConfigHost is the structure of the "host" nested block within the CLI
// configuration, which can be used to override the default service host
// discovery behavior for a particular hostname.
type ConfigHost struct {
	Services map[string]interface{} `hcl:"services"`
}

// ConfigCredentialsHelper is the structure of the "credentials_helper"
// nested block within the CLI configuration.
type ConfigCredentialsHelper struct {
	Args []string `hcl:"args"`
}

// BuiltinConfig is the built-in defaults for the configuration. These
// can be overridden by user configurations.
var BuiltinConfig Config

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

// LoadConfig reads the CLI configuration from the various filesystem locations
// and from the environment, returning a merged configuration along with any
// diagnostics (errors and warnings) encountered along the way.
func LoadConfig() (*Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	configVal := BuiltinConfig // copy
	config := &configVal

	if mainFilename, mainFileDiags := cliConfigFile(); len(mainFileDiags) == 0 {
		if _, err := os.Stat(mainFilename); err == nil {
			mainConfig, mainDiags := loadConfigFile(mainFilename)
			diags = diags.Append(mainDiags)
			config = config.Merge(mainConfig)
		}
	} else {
		diags = diags.Append(mainFileDiags)
	}

	// Unless the user has specifically overridden the configuration file
	// location using an environment variable, we'll also load what we find
	// in the config directory. We skip the config directory when source
	// file override is set because we interpret the environment variable
	// being set as an intention to ignore the default set of CLI config
	// files because we're doing something special, like running Terraform
	// in automation with a locally-customized configuration.
	if cliConfigFileOverride() == "" {
		if configDir, err := ConfigDir(); err == nil {
			if info, err := os.Stat(configDir); err == nil && info.IsDir() {
				dirConfig, dirDiags := loadConfigDir(configDir)
				diags = diags.Append(dirDiags)
				config = config.Merge(dirConfig)
			}
		}
	} else {
		log.Printf("[DEBUG] Not reading CLI config directory because config location is overridden by environment variable")
	}

	if envConfig := EnvConfig(); envConfig != nil {
		// envConfig takes precedence
		config = envConfig.Merge(config)
	}

	diags = diags.Append(config.Validate())

	return config, diags
}

// loadConfigFile loads the CLI configuration from ".terraformrc" files.
func loadConfigFile(path string) (*Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result := &Config{}

	log.Printf("Loading CLI configuration from %s", path)

	// Read the HCL file and prepare for parsing
	d, err := ioutil.ReadFile(path)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error reading %s: %s", path, err))
		return result, diags
	}

	// Parse it
	obj, err := hcl.Parse(string(d))
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error parsing %s: %s", path, err))
		return result, diags
	}

	// Build up the result
	if err := hcl.DecodeObject(&result, obj); err != nil {
		diags = diags.Append(fmt.Errorf("Error parsing %s: %s", path, err))
		return result, diags
	}

	// Deal with the provider_installation block, which is not handled using
	// DecodeObject because its structure is not compatible with the
	// limitations of that function.
	providerInstBlocks, moreDiags := decodeProviderInstallationFromConfig(obj)
	diags = diags.Append(moreDiags)
	result.ProviderInstallation = providerInstBlocks

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

	return result, diags
}

func loadConfigDir(path string) (*Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result := &Config{}

	entries, err := ioutil.ReadDir(path)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error reading %s: %s", path, err))
		return result, diags
	}

	for _, entry := range entries {
		name := entry.Name()
		// Ignoring errors here because it is used only to indicate pattern
		// syntax errors, and our patterns are hard-coded here.
		hclMatched, _ := filepath.Match("*.tfrc", name)
		jsonMatched, _ := filepath.Match("*.tfrc.json", name)
		if !(hclMatched || jsonMatched) {
			continue
		}

		filePath := filepath.Join(path, name)
		fileConfig, fileDiags := loadConfigFile(filePath)
		diags = diags.Append(fileDiags)
		result = result.Merge(fileConfig)
	}

	return result, diags
}

// EnvConfig returns a Config populated from environment variables.
//
// Any values specified in this config should override those set in the
// configuration file.
func EnvConfig() *Config {
	env := makeEnvMap(os.Environ())
	return envConfig(env)
}

func envConfig(env map[string]string) *Config {
	config := &Config{}

	if envPluginCacheDir := env[pluginCacheDirEnvVar]; envPluginCacheDir != "" {
		// No Expandenv here, because expanding environment variables inside
		// an environment variable would be strange and seems unnecessary.
		// (User can expand variables into the value while setting it using
		// standard shell features.)
		config.PluginCacheDir = envPluginCacheDir
	}

	if envMayBreak := env[pluginCacheMayBreakLockFileEnvVar]; envMayBreak != "" && envMayBreak != "0" {
		// This is an environment variable analog to the
		// plugin_cache_may_break_dependency_lock_file setting. If either this
		// or the config file setting are enabled then it's enabled; there is
		// no way to override back to false if either location sets this to
		// true.
		config.PluginCacheMayBreakDependencyLockFile = true
	}

	return config
}

func makeEnvMap(environ []string) map[string]string {
	if len(environ) == 0 {
		return nil
	}

	ret := make(map[string]string, len(environ))
	for _, entry := range environ {
		eq := strings.IndexByte(entry, '=')
		if eq == -1 {
			continue
		}
		ret[entry[:eq]] = entry[eq+1:]
	}
	return ret
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

	// Check that all "host" blocks have valid hostnames.
	for givenHost := range c.Hosts {
		_, err := svchost.ForComparison(givenHost)
		if err != nil {
			diags = diags.Append(
				fmt.Errorf("The host %q block has an invalid hostname: %s", givenHost, err),
			)
		}
	}

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

	// Should have zero or one "provider_installation" blocks
	if len(c.ProviderInstallation) > 1 {
		diags = diags.Append(
			fmt.Errorf("No more than one provider_installation block may be specified"),
		)
	}

	if c.PluginCacheDir != "" {
		_, err := os.Stat(c.PluginCacheDir)
		if err != nil {
			diags = diags.Append(
				fmt.Errorf("The specified plugin cache dir %s cannot be opened: %s", c.PluginCacheDir, err),
			)
		}
	}

	return diags
}

// Merge merges two configurations and returns a third entirely
// new configuration with the two merged.
func (c *Config) Merge(c2 *Config) *Config {
	var result Config
	result.Providers = make(map[string]string)
	result.Provisioners = make(map[string]string)
	maps.Copy(result.Providers, c.Providers)
	maps.Copy(result.Provisioners, c.Provisioners)
	for k, v := range c2.Providers {
		if v1, ok := c.Providers[k]; ok {
			log.Printf("[INFO] Local %s provider configuration '%s' overrides '%s'", k, v, v1)
		}
		result.Providers[k] = v
	}
	for k, v := range c2.Provisioners {
		if v1, ok := c.Provisioners[k]; ok {
			log.Printf("[INFO] Local %s provisioner configuration '%s' overrides '%s'", k, v, v1)
		}
		result.Provisioners[k] = v
	}
	result.DisableCheckpoint = c.DisableCheckpoint || c2.DisableCheckpoint
	result.DisableCheckpointSignature = c.DisableCheckpointSignature || c2.DisableCheckpointSignature

	result.PluginCacheDir = c.PluginCacheDir
	if result.PluginCacheDir == "" {
		result.PluginCacheDir = c2.PluginCacheDir
	}

	if c.PluginCacheMayBreakDependencyLockFile || c2.PluginCacheMayBreakDependencyLockFile {
		// This setting saturates to "on"; once either configuration sets it,
		// there is no way to override it back to off again.
		result.PluginCacheMayBreakDependencyLockFile = true
	}

	if (len(c.Hosts) + len(c2.Hosts)) > 0 {
		result.Hosts = make(map[string]*ConfigHost)
		maps.Copy(result.Hosts, c.Hosts)
		maps.Copy(result.Hosts, c2.Hosts)
	}

	if (len(c.Credentials) + len(c2.Credentials)) > 0 {
		result.Credentials = make(map[string]map[string]interface{})
		maps.Copy(result.Credentials, c.Credentials)
		// We just clobber an entry from the other file right now. Will
		// improve on this later using the more-robust merging behavior
		// built in to HCL2.
		maps.Copy(result.Credentials, c2.Credentials)
	}

	if (len(c.CredentialsHelpers) + len(c2.CredentialsHelpers)) > 0 {
		result.CredentialsHelpers = make(map[string]*ConfigCredentialsHelper)
		maps.Copy(result.CredentialsHelpers, c.CredentialsHelpers)
		maps.Copy(result.CredentialsHelpers, c2.CredentialsHelpers)
	}

	if (len(c.ProviderInstallation) + len(c2.ProviderInstallation)) > 0 {
		result.ProviderInstallation = append(result.ProviderInstallation, c.ProviderInstallation...)
		result.ProviderInstallation = append(result.ProviderInstallation, c2.ProviderInstallation...)
	}

	return &result
}

func cliConfigFile() (string, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	mustExist := true

	configFilePath := cliConfigFileOverride()
	if configFilePath == "" {
		var err error
		configFilePath, err = ConfigFile()
		mustExist = false

		if err != nil {
			log.Printf(
				"[ERROR] Error detecting default CLI config file path: %s",
				err)
		}
	}

	log.Printf("[DEBUG] Attempting to open CLI config file: %s", configFilePath)
	f, err := os.Open(configFilePath)
	if err == nil {
		f.Close()
		return configFilePath, diags
	}

	if mustExist || !errors.Is(err, fs.ErrNotExist) {
		diags = append(diags, tfdiags.Sourceless(
			tfdiags.Warning,
			"Unable to open CLI configuration file",
			fmt.Sprintf("The CLI configuration file at %q does not exist.", configFilePath),
		))
	}

	log.Println("[DEBUG] File doesn't exist, but doesn't need to. Ignoring.")
	return "", diags
}

func cliConfigFileOverride() string {
	configFilePath := os.Getenv("TF_CLI_CONFIG_FILE")
	if configFilePath == "" {
		configFilePath = os.Getenv("TERRAFORM_CONFIG")
	}
	return configFilePath
}
