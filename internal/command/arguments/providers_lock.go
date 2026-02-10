// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProvidersLock represents the command-line arguments for the providers lock command.
type ProvidersLock struct {
	// Platforms is the list of target platforms to request package checksums for.
	Platforms FlagStringSlice

	// FSMirrorDir is the filesystem mirror directory to consult instead of the
	// origin registry.
	FSMirrorDir string

	// NetMirrorURL is the network mirror base URL to consult instead of the
	// origin registry.
	NetMirrorURL string

	// TestDirectory is the directory containing test files, defaults to "tests".
	TestDirectory string

	// EnablePluginCache enables the usage of the globally configured plugin cache.
	EnablePluginCache bool

	// Providers is the list of provider source addresses given as positional arguments.
	Providers []string
}

// ParseProvidersLock processes CLI arguments, returning a ProvidersLock value and error
// diagnostics. If there are any diagnostics present, a ProvidersLock value is still
// returned representing the best effort interpretation of the arguments.
func ParseProvidersLock(args []string) (*ProvidersLock, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result := &ProvidersLock{
		TestDirectory: "tests",
	}

	cmdFlags := defaultFlagSet("providers lock")
	cmdFlags.Var(&result.Platforms, "platform", "target platform")
	cmdFlags.StringVar(&result.FSMirrorDir, "fs-mirror", "", "filesystem mirror directory")
	cmdFlags.StringVar(&result.NetMirrorURL, "net-mirror", "", "network mirror base URL")
	cmdFlags.StringVar(&result.TestDirectory, "test-directory", "tests", "test-directory")
	cmdFlags.BoolVar(&result.EnablePluginCache, "enable-plugin-cache", false, "enable plugin cache")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	if remaining := cmdFlags.Args(); len(remaining) > 0 {
		result.Providers = remaining
	}

	if result.FSMirrorDir != "" && result.NetMirrorURL != "" {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid installation method options",
			"The -fs-mirror and -net-mirror command line options are mutually-exclusive.",
		))
	}

	return result, diags
}
