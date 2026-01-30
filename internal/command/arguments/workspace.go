// Copyright (c) HashiCorp, Inc.
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

// ParseWorkspaceList processes CLI arguments, returning a Workspace value and errors.
// If errors are encountered, an Workspace value is still returned representing
// the best effort interpretation of the arguments.
func ParseWorkspaceList(args []string) (*Workspace, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var jsonOutput bool
	cmdFlags := defaultFlagSet("workspace list")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")
	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	workspace := &Workspace{}

	switch {
	case jsonOutput:
		workspace.ViewType = ViewJSON
	default:
		workspace.ViewType = ViewHuman
	}

	return workspace, diags
}
