// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type WorkspaceShow struct {
	Workspace
}

func ParseWorkspaceShow(args []string) (*WorkspaceShow, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	cmdFlags := defaultFlagSet("workspace show")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	// `workspace show` takes no positional arguments.
	// We could add validation here to return an error when unexpected arguments are present,
	// but this would be a breaking change as no validation was performed in this case before.

	return &WorkspaceShow{Workspace: Workspace{ViewType: ViewHuman}}, diags
}
