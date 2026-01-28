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

	// These methods are present in the interface to allow
	// backwards compatibility while human-readable output is
	// fulfilled using the cli.Ui interface.
	Error(message string)
	Warn(message string)
}

// NewWorkspace returns the Workspace implementation for the given ViewType.
func NewWorkspace(vt arguments.ViewType, view *View) Workspace {
	switch vt {
	case arguments.ViewJSON:
		panic("machine readable output not implemented for workspace subcommands")
	case arguments.ViewHuman:
		// TODO: Allow use of WorkspaceHuman here when we remove use of cli.Ui from workspace commands.
		panic("human readable output via Views is a breaking change, so this code path shouldn't be used until that's possible.")
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

// Error is implemented to fulfil the Workspace interface
// Once we can make breaking changes that interface shouldn't have an
// Error method, so this method should be deleted in future.
func (v *WorkspaceHuman) Error(msg string) {
	panic("(WorkspaceHuman).Error should not be used")
}

// Warn is implemented to fulfil the Workspace interface
// Onc we can make breaking changes that interface shouldn't have an
// Warn method, so this method should be deleted in future.
func (v *WorkspaceHuman) Warn(msg string) {
	panic("(WorkspaceHuman).Warn should not be used")
}

func (v *WorkspaceHuman) prepareMessage(msg string) string {
	return v.view.colorize.Color(strings.TrimSpace(msg))
}
