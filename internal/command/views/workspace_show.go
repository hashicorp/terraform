// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The WorkspaceShow view is used for the `workspace show` subcommand.
type WorkspaceShow interface {
	Show(workspace string, diags tfdiags.Diagnostics)
}

func NewWorkspaceShow(viewType arguments.ViewType, view *View) WorkspaceShow {
	switch viewType {
	case arguments.ViewHuman:
		return &WorkspaceShowHuman{view: view}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

// The WorkspaceShowHuman implementation renders human-readable output.
type WorkspaceShowHuman struct {
	view *View
}

var _ WorkspaceShow = (*WorkspaceShowHuman)(nil)

func (v *WorkspaceShowHuman) Show(workspace string, diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
	if workspace != "" {
		v.view.streams.Println(workspace)
	}
}
