// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"errors"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// WorkspaceList represent arguments specific to the `terraform workspace list` command.
type WorkspaceList struct {
	Workspace
}

// ParseWorkspaceList processes CLI arguments, returning a WorkspaceList value and errors.
// If errors are encountered, an WorkspaceList value is still returned representing
// the best effort interpretation of the arguments.
func ParseWorkspaceList(args []string) (*WorkspaceList, tfdiags.Diagnostics) {
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

	// `workspace list` takes no positional arguments. Historically there was a DIR argument that was replaced with the -chdir flag.
	// Here we replicate the old behaviour of suggesting the user to use -chdir if they provide any positional arguments.
	args = cmdFlags.Args()
	if len(args) != 0 {
		diags = diags.Append(errors.New("Too many command line arguments. Did you mean to use -chdir?"))
	}

	switch {
	case jsonOutput:
		return &WorkspaceList{Workspace: Workspace{ViewType: ViewJSON}}, diags
	default:
		return &WorkspaceList{Workspace: Workspace{ViewType: ViewHuman}}, diags
	}
}
