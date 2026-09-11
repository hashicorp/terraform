// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The WorkspaceNew view is used for the `workspace new` subcommand.
type WorkspaceNew interface {
	// LogWorkspaceCreationSuccess is called when a new workspace has been successfully created
	LogWorkspaceCreationSuccess(workspaceName string, diags tfdiags.Diagnostics)

	Diagnostics(diags tfdiags.Diagnostics)
}

func NewWorkspaceNew(viewType arguments.ViewType, view *View) WorkspaceNew {
	switch viewType {
	case arguments.ViewHuman:
		return &WorkspaceNewHuman{
			view: view,
		}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

// The WorkspaceNewHuman implementation renders human-readable logs, suitable for direct consumption by a human user.
type WorkspaceNewHuman struct {
	view *View
}

var _ WorkspaceNew = (*WorkspaceNewHuman)(nil)

func (v *WorkspaceNewHuman) LogWorkspaceCreationSuccess(workspaceName string, diags tfdiags.Diagnostics) {
	// Print diags above output
	v.view.Diagnostics(diags)

	msg := fmt.Sprintf(
		strings.TrimSpace(EnvCreated), workspaceName)

	v.log(msg)
}

// Diagnostics is used to display diagnostic messages, but should only be used when the command
// is unsuccessful and returns early.
func (v *WorkspaceNewHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (s *WorkspaceNewHuman) log(preparedMessage string) {
	msg := s.view.colorize.Color(strings.TrimSpace(preparedMessage))
	s.view.streams.Println(msg)
}
