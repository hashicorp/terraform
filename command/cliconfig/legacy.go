package cliconfig

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/tfdiags"
)

// The legacy.go file and the other files with a "legacy_" prefix together deal
// with the old-style CLI config language that is implemented with HCL v1.0.
//
// We continue to support this old form of CLI config so that users can
// upgrade from older Terraform versions without losing all of their existing
// settings, but we won't be adding any new features to the old format and
// so users will need to migrate to the new structure in order to use those.
//
// Terraform distinguishes between the old and new CLI configuration based on
// where the settings are loaded from. The legacy search locations are:
// ~/.terraformrc (or %APPDATA%/terraform.rc on Windows)
// ~/.terraform/*.tfrc or (%APPDATA%/terraform.d/*.tfrc on Windows)
// Files matching these are considered legacy and Terraform loads them using
// these legacy codepaths. New-style CLI configuration lives in an OS-specific
// standard configuration location that cannot overlap with the legacy
// locations unless the user does something unusual, like intentionally setting
// up confusing symlinks.
//
// The code in these files was largely copied verbatim from the old CLI config
// implementation and just lightly renamed with "Legacy..." prefixes on the
// global symbols, to minimize the risk of inadvertently changing the behavior.

// LegacyConfig is the structure of the configuration for the Terraform CLI.
type LegacyConfig struct {
	Providers    map[string]string
	Provisioners map[string]string

	DisableCheckpoint          bool `hcl:"disable_checkpoint"`
	DisableCheckpointSignature bool `hcl:"disable_checkpoint_signature"`

	// If set, enables local caching of plugins in this directory to
	// avoid repeatedly re-downloading over the Internet.
	PluginCacheDir string `hcl:"plugin_cache_dir"`

	Hosts map[string]*LegacyConfigHost `hcl:"host"`

	Credentials        map[string]map[string]interface{}         `hcl:"credentials"`
	CredentialsHelpers map[string]*LegacyConfigCredentialsHelper `hcl:"credentials_helper"`

	// ProviderInstallation represents any provider_installation blocks
	// in the configuration. Only one of these is allowed across the whole
	// configuration, but we decode into a slice here so that we can handle
	// that validation at validation time rather than initial decode time.
	ProviderInstallation []*LegacyProviderInstallation
}

// LegacyConfigHost is the structure of the "host" nested block within the CLI
// configuration, which can be used to override the default service host
// discovery behavior for a particular hostname.
type LegacyConfigHost struct {
	Services map[string]interface{} `hcl:"services"`
}

// LegacyConfigCredentialsHelper is the structure of the "credentials_helper"
// nested block within the CLI configuration.
type LegacyConfigCredentialsHelper struct {
	Args []string `hcl:"args"`
}

// LegacyBuiltinConfig is the built-in defaults for the configuration. These
// can be overridden by user configurations.
var LegacyBuiltinConfig LegacyConfig

// LegacyConfigFile returns the default path to the configuration file.
//
// On Unix-like systems this is the ".terraformrc" file in the home directory.
// On Windows, this is the "terraform.rc" file in the application data
// directory.
func LegacyConfigFile() (string, error) {
	return legacyConfigFile()
}

// LegacyConfigDir returns the configuration directory for Terraform.
func LegacyConfigDir() (string, error) {
	return legacyConfigDir()
}

// LegacyLoadConfig reads the CLI configuration from the various filesystem locations
// and from the environment, returning a merged configuration along with any
// diagnostics (errors and warnings) encountered along the way.
func LegacyLoadConfig() (*LegacyConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	configVal := LegacyBuiltinConfig // copy
	config := &configVal

	if mainFilename, err := legacyCLIConfigFile(); err == nil {
		if _, err := os.Stat(mainFilename); err == nil {
			mainConfig, mainDiags := legacyLoadConfigFile(mainFilename)
			diags = diags.Append(mainDiags)
			config = config.Merge(mainConfig)
		}
	}

	// Unless the user has specifically overridden the configuration file
	// location using an environment variable, we'll also load what we find
	// in the config directory. We skip the config directory when source
	// file override is set because we interpret the environment variable
	// being set as an intention to ignore the default set of CLI config
	// files because we're doing something special, like running Terraform
	// in automation with a locally-customized configuration.
	if legacyCLIConfigFileOverride() == "" {
		if configDir, err := LegacyConfigDir(); err == nil {
			if info, err := os.Stat(configDir); err == nil && info.IsDir() {
				dirConfig, dirDiags := legacyLoadConfigDir(configDir)
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

// legacyLoadConfigFile loads the CLI configuration from ".terraformrc" files.
func legacyLoadConfigFile(path string) (*LegacyConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result := &LegacyConfig{}

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
	providerInstBlocks, moreDiags := legacyDecodeProviderInstallationFromConfig(obj)
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

func legacyLoadConfigDir(path string) (*LegacyConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result := &LegacyConfig{}

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
		fileConfig, fileDiags := legacyLoadConfigFile(filePath)
		diags = diags.Append(fileDiags)
		result = result.Merge(fileConfig)
	}

	return result, diags
}

// Validate checks for errors in the configuration that cannot be detected
// just by HCL decoding, returning any problems as diagnostics.
//
// On success, the returned diagnostics will return false from the HasErrors
// method. A non-nil diagnostics is not necessarily an error, since it may
// contain just warnings.
func (c *LegacyConfig) Validate() tfdiags.Diagnostics {
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
func (c1 *LegacyConfig) Merge(c2 *LegacyConfig) *LegacyConfig {
	var result LegacyConfig
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

	if (len(c1.Hosts) + len(c2.Hosts)) > 0 {
		result.Hosts = make(map[string]*LegacyConfigHost)
		for name, host := range c1.Hosts {
			result.Hosts[name] = host
		}
		for name, host := range c2.Hosts {
			result.Hosts[name] = host
		}
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
		result.CredentialsHelpers = make(map[string]*LegacyConfigCredentialsHelper)
		for name, helper := range c1.CredentialsHelpers {
			result.CredentialsHelpers[name] = helper
		}
		for name, helper := range c2.CredentialsHelpers {
			result.CredentialsHelpers[name] = helper
		}
	}

	if (len(c1.ProviderInstallation) + len(c2.ProviderInstallation)) > 0 {
		result.ProviderInstallation = append(result.ProviderInstallation, c1.ProviderInstallation...)
		result.ProviderInstallation = append(result.ProviderInstallation, c2.ProviderInstallation...)
	}

	return &result
}

func legacyCLIConfigFile() (string, error) {
	mustExist := true

	configFilePath := legacyCLIConfigFileOverride()
	if configFilePath == "" {
		var err error
		configFilePath, err = LegacyConfigFile()
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
		return configFilePath, nil
	}

	if mustExist || !os.IsNotExist(err) {
		return "", err
	}

	log.Println("[DEBUG] File doesn't exist, but doesn't need to. Ignoring.")
	return "", nil
}

func legacyCLIConfigFileOverride() string {
	configFilePath := os.Getenv("TF_CLI_CONFIG_FILE")
	if configFilePath == "" {
		configFilePath = os.Getenv("TERRAFORM_CONFIG")
	}
	return configFilePath
}
