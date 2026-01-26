// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Workspace view is used for workspace subcommands.
type Workspace interface {
	Diagnostics(diags tfdiags.Diagnostics)
	Output(message string)
}

// NewInit returns Init implementation for the given ViewType.
func NewWorkspace(vt arguments.ViewType, view *View) Workspace {
	switch vt {
	case arguments.ViewJSON:
		panic("machine readable output not implemented for workspace subcommands")
	case arguments.ViewHuman:
		return &WorkspaceHuman{
			view: view,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The WorkspaceHuman implementation renders human-readable text logs, suitable for
// a scrolling terminal.
type WorkspaceHuman struct {
	view *View
}

var _ Workspace = (*WorkspaceHuman)(nil)

// Diagnostics renders a list of diagnostics, including the option for compact warnings.
func (v *WorkspaceHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// Output is used to render text in the terminal, such as data returned from a command.
func (v *WorkspaceHuman) Output(msg string) {
	v.view.streams.Println(v.prepareMessage(msg))
}

func (v *WorkspaceHuman) prepareMessage(msg string) string {
	return v.view.colorize.Color(strings.TrimSpace(msg))
}
