// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateMv is the view interface for the "state mv" command.
type StateMv interface {
	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)

	// HelpPrompt directs the user to the full help output for the command.
	HelpPrompt()
}

// NewStateMv returns an initialized StateMv implementation for the human view.
func NewStateMv(view *View) StateMv {
	return &StateMvHuman{view: view}
}

// StateMvHuman is the human-readable implementation of the StateMv view.
type StateMvHuman struct {
	view *View
}

var _ StateMv = (*StateMvHuman)(nil)

func (v *StateMvHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *StateMvHuman) HelpPrompt() {
	v.view.HelpPrompt("state mv")
}
