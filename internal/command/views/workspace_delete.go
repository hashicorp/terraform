// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The WorkspaceDelete view is used for the `workspace delete` subcommand.
type WorkspaceDelete interface {
	// LogWorkspaceDeletionSuccess is called when a workspace has been successfully deleted.
	// The state may have been empty or contained resources, the latter is communicated to the user through warning diagnostics.
	LogWorkspaceDeletionSuccess(workspaceName string, diags tfdiags.Diagnostics)

	Diagnostics(diags tfdiags.Diagnostics)
}

func NewWorkspaceDelete(viewType arguments.ViewType, view *View) WorkspaceDelete {
	switch viewType {
	case arguments.ViewHuman:
		return &WorkspaceDeleteHuman{
			view: view,
		}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

// The WorkspaceDeleteHuman implementation renders human-readable logs, suitable for direct consumption by a human user.
type WorkspaceDeleteHuman struct {
	view *View
}

var _ WorkspaceDelete = (*WorkspaceDeleteHuman)(nil)

func (v *WorkspaceDeleteHuman) LogWorkspaceDeletionSuccess(workspaceName string, diags tfdiags.Diagnostics) {
	// Print diags above output
	v.view.Diagnostics(diags)

	v.view.streams.Println(v.view.colorize.Color(fmt.Sprintf(`[reset][green]Deleted workspace %q!`, workspaceName)))
}

// Diagnostics is used to display diagnostic messages, but should only be used when the command
// is unsuccessful and returns early.
func (v *WorkspaceDeleteHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
