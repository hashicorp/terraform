// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateRm is the view interface for the "state rm" command.
type StateRm interface {
	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)

	// HelpPrompt directs the user to the full help output for the command.
	HelpPrompt()
}

// NewStateRm returns an initialized StateRm implementation for the human view.
func NewStateRm(view *View) StateRm {
	return &StateRmHuman{view: view}
}

// StateRmHuman is the human-readable implementation of the StateRm view.
type StateRmHuman struct {
	view *View
}

var _ StateRm = (*StateRmHuman)(nil)

func (v *StateRmHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *StateRmHuman) HelpPrompt() {
	v.view.HelpPrompt("state rm")
}
