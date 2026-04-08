// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Workspace represents the command-line arguments common between all workspace subcommands.
//
// Subcommands that accept additional arguments should have a specific struct that embeds this struct.
type Workspace struct {
	// ViewType specifies which output format to use
	ViewType ViewType
}

// ParseWorkspace processes CLI arguments, returning a Workspace value and errors.
// If errors are encountered, an Workspace value is still returned representing
// the best effort interpretation of the arguments.
func ParseWorkspace(args []string) (*Workspace, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	cmdFlags := defaultFlagSet("workspace list")
	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	// There should not be any non-flag arguments for the workspace list command.
	// In future, when other workspace subcommands start using the arguments package,
	// this code will need to change.
	args = cmdFlags.Args()
	if len(args) != 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"Expected no positional arguments.",
		))
	}

	return &Workspace{ViewType: ViewHuman}, diags
}
