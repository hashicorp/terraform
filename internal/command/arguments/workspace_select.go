// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// WorkspaceSelect represents the command-line arguments for the workspace
// select command.
type WorkspaceSelect struct {
	OrCreate bool
	Name     string
	Args     []string
}

// ParseWorkspaceSelect processes CLI arguments, returning a WorkspaceSelect
// value and errors. If errors are encountered, a WorkspaceSelect value is still
// returned representing the best effort interpretation of the arguments.
func ParseWorkspaceSelect(args []string) (*WorkspaceSelect, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ws := &WorkspaceSelect{}

	cmdFlags := defaultFlagSet("workspace select")
	cmdFlags.BoolVar(&ws.OrCreate, "or-create", false, "create workspace if it does not exist")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid arguments",
			"Expected a single argument: NAME.",
		))
		return ws, diags
	}

	ws.Name = args[0]
	ws.Args = args[1:]

	return ws, diags
}
