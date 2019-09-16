package main

// This file has some compatibility aliases/wrappers for functionality that
// has now moved into command/cliconfig .
//
// Don't add anything new here! If new functionality is needed, better to just
// add it in command/cliconfig and then call there directly.

import (
	"github.com/hashicorp/terraform/command/cliconfig"
	"github.com/hashicorp/terraform/tfdiags"
)

//go:generate go run ./scripts/generate-plugins.go

// Config is the structure of the configuration for the Terraform CLI.
//
// This is not the configuration for Terraform itself. That is in the
// "configs" package.
type Config = cliconfig.Config

// ConfigHost is the structure of the "host" nested block within the CLI
// configuration, which can be used to override the default service host
// discovery behavior for a particular hostname.
type ConfigHost = cliconfig.ConfigHost

// ConfigCredentialsHelper is the structure of the "credentials_helper"
// nested block within the CLI configuration.
type ConfigCredentialsHelper = cliconfig.ConfigCredentialsHelper

// BuiltinConfig is the built-in defaults for the configuration. These
// can be overridden by user configurations.
var BuiltinConfig = cliconfig.BuiltinConfig

// ConfigFile returns the default path to the configuration file.
//
// On Unix-like systems this is the ".terraformrc" file in the home directory.
// On Windows, this is the "terraform.rc" file in the application data
// directory.
func ConfigFile() (string, error) {
	return cliconfig.ConfigFile()
}

// ConfigDir returns the configuration directory for Terraform.
func ConfigDir() (string, error) {
	return cliconfig.ConfigDir()
}

// LoadConfig reads the CLI configuration from the various filesystem locations
// and from the environment, returning a merged configuration along with any
// diagnostics (errors and warnings) encountered along the way.
func LoadConfig() (*Config, tfdiags.Diagnostics) {
	return cliconfig.LoadConfig()
}

// EnvConfig returns a Config populated from environment variables.
//
// Any values specified in this config should override those set in the
// configuration file.
func EnvConfig() *Config {
	return cliconfig.EnvConfig()
}
