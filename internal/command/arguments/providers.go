// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// Providers represents the command-line arguments for the providers command.
type Providers struct {
	// TestsDirectory is the directory containing Terraform test files.
	TestsDirectory string
}

// ParseProviders processes CLI arguments, returning a Providers value and
// errors. If errors are encountered, a Providers value is still returned
// representing the best effort interpretation of the arguments.
func ParseProviders(args []string) (*Providers, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	providers := &Providers{}

	cmdFlags := defaultFlagSet("providers")
	cmdFlags.StringVar(&providers.TestsDirectory, "test-directory", "tests", "test-directory")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"Did you mean to use -chdir?",
		))
	}

	return providers, diags
}
