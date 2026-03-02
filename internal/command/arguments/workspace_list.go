// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// WorkspaceList represents the command-line arguments for the workspace list
// command.
type WorkspaceList struct {
	Args []string
}

// ParseWorkspaceList processes CLI arguments, returning a WorkspaceList value
// and errors. If errors are encountered, a WorkspaceList value is still
// returned representing the best effort interpretation of the arguments.
func ParseWorkspaceList(args []string) (*WorkspaceList, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	wl := &WorkspaceList{}

	cmdFlags := defaultFlagSet("workspace list")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	wl.Args = cmdFlags.Args()

	return wl, diags
}
