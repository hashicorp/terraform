// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Providers represents the command-line arguments for the providers command.
type Providers struct {
	// Path is the directory containing the configuration to be inspected. If
	// unspecified, providers will use the current directory.
	Path string

	// TestDirectory is the directory containing any test files that should be
	// inspected alongside the main configuration. Should be relative to the
	// Path.
	TestDirectory string
}

// ParseProviders processes CLI arguments, returning a Providers value and errors.
// If errors are encountered, a Providers value is still returned representing
// the best effort interpretation of the arguments.
func ParseProviders(args []string) (*Providers, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	providers := &Providers{
		Path: ".",
	}

	cmdFlags := defaultFlagSet("providers")
	cmdFlags.StringVar(&providers.TestDirectory, "test-directory", "tests", "test-directory")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"Expected at most one positional argument.",
		))
	}

	if len(args) > 0 {
		providers.Path = args[0]
	}

	return providers, diags
}
