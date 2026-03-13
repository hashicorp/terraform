// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Get represents the command-line arguments for the get command.
type Get struct {
	// Vars are the variable-related flags (-var, -var-file).
	Vars *Vars

	// Update, if true, checks already-downloaded modules for available
	// updates and installs the newest versions available.
	Update bool

	// TestDirectory is the Terraform test directory.
	TestDirectory string
}

// ParseGet processes CLI arguments, returning a Get value and diagnostics.
// If errors are encountered, a Get value is still returned representing
// the best effort interpretation of the arguments.
func ParseGet(args []string) (*Get, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	get := &Get{
		Vars: &Vars{},
	}

	cmdFlags := extendedFlagSet("get", nil, nil, get.Vars)
	cmdFlags.BoolVar(&get.Update, "update", false, "update")
	cmdFlags.StringVar(&get.TestDirectory, "test-directory", "tests", "test-directory")

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
			"Expected no positional arguments. Did you mean to use -chdir?",
		))
	}

	return get, diags
}
