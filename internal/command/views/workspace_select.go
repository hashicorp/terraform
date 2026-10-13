// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The WorkspaceSelect view is used for the `workspace select` subcommand.
type WorkspaceSelect interface {
	Diagnostics(diags tfdiags.Diagnostics)

	// Log that a workspace has been selected.
	SelectedWorkspace(workspace string, diags tfdiags.Diagnostics)

	// Log that a workspace that didn't exist has been created and selected.
	CreatedWorkspace(workspace string, diags tfdiags.Diagnostics)
}

func NewWorkspaceSelect(viewType arguments.ViewType, view *View) WorkspaceSelect {
	switch viewType {
	case arguments.ViewHuman:
		return &WorkspaceSelectHuman{view: view}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

// The WorkspaceSelectHuman implementation renders human-readable output.
type WorkspaceSelectHuman struct {
	view *View
}

var _ WorkspaceSelect = (*WorkspaceSelectHuman)(nil)

func (v *WorkspaceSelectHuman) CreatedWorkspace(workspace string, diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
	v.print(fmt.Sprintf(EnvCreated, workspace))
}

func (v *WorkspaceSelectHuman) SelectedWorkspace(workspace string, diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
	v.print(fmt.Sprintf(`[reset][green]Switched to workspace %q.`, workspace))
}

func (v *WorkspaceSelectHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *WorkspaceSelectHuman) print(message string) {
	v.view.streams.Println(v.view.colorize.Color(strings.TrimSpace(message)))
}
