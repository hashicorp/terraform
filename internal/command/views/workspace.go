// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"errors"
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
		return &WorkspaceJSON{
			view: NewJSONView(view),
		}
	case arguments.ViewHuman:
		// TODO: Allow use of WorkspaceHuman here when we remove use of cli.Ui from workspace commands.
		panic("human readable output via Views is a breaking change, so this code path shouldn't be used until that's possible.")
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The WorkspaceJSON implementation renders machine-readable logs, suitable for
// integrating with other software.
type WorkspaceJSON struct {
	view *JSONView
}

var _ Workspace = (*WorkspaceJSON)(nil)

// Diagnostics renders a list of diagnostics, including the option for compact warnings.
func (v *WorkspaceJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// Output is used to render data in the terminal, e.g. the workspaces returned from `workspace list`
func (v *WorkspaceJSON) Output(msg string) {
	v.view.Log(msg)
}

// Error
//
// This method is a temporary measure while the workspace subcommands contain both
// use of cli.Ui for human output and view.View for machine-readable output.
// In future calling code should use Diagnostics directly.
//
// If a message is being logged as an error we can create a native error (which can be made from a string),
// use existing logic in (tfdiags.Diagnostics) Append to create an error diagnostic from a native error,
// and then log that single error diagnostic.
func (v *WorkspaceJSON) Error(msg string) {
	var diags tfdiags.Diagnostics
	err := errors.New(msg)
	diags = diags.Append(err)

	v.Diagnostics(diags)
}

// Warn
//
// This method is a temporary measure while the workspace subcommands contain both
// use of cli.Ui for human output and view.View for machine-readable output.
// In future calling code should use Diagnostics directly.
//
// This method takes inspiration from how native errors are converted into error diagnostics;
// the Details value is left empty and the provided string is used only in the Summary.
// See : https://github.com/hashicorp/terraform/blob/v1.14.4/internal/tfdiags/error.go
func (v *WorkspaceJSON) Warn(msg string) {
	var diags tfdiags.Diagnostics
	warn := tfdiags.Sourceless(
		tfdiags.Warning,
		msg,
		"",
	)
	diags = diags.Append(warn)

	v.Diagnostics(diags)
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
