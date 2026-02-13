// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// ProvidersLock represents the command-line arguments for the providers lock
// command.
type ProvidersLock struct {
	Platforms         FlagStringSlice
	FSMirrorDir       string
	NetMirrorURL      string
	TestsDirectory    string
	EnablePluginCache bool
	Providers         []string
}

// ParseProvidersLock processes CLI arguments, returning a ProvidersLock value
// and errors. If errors are encountered, a ProvidersLock value is still
// returned representing the best effort interpretation of the arguments.
func ParseProvidersLock(args []string) (*ProvidersLock, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	providersLock := &ProvidersLock{}

	cmdFlags := defaultFlagSet("providers lock")
	cmdFlags.Var(&providersLock.Platforms, "platform", "target platform")
	cmdFlags.StringVar(&providersLock.FSMirrorDir, "fs-mirror", "", "filesystem mirror directory")
	cmdFlags.StringVar(&providersLock.NetMirrorURL, "net-mirror", "", "network mirror base URL")
	cmdFlags.StringVar(&providersLock.TestsDirectory, "test-directory", "tests", "test-directory")
	cmdFlags.BoolVar(&providersLock.EnablePluginCache, "enable-plugin-cache", false, "")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	if providersLock.FSMirrorDir != "" && providersLock.NetMirrorURL != "" {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid installation method options",
			"The -fs-mirror and -net-mirror command line options are mutually-exclusive.",
		))
	}

	providersLock.Providers = cmdFlags.Args()

	return providersLock, diags
}
