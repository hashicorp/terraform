// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StatePush is the view interface for the "state push" command.
type StatePush interface {
	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)

	// HelpPrompt directs the user to the full help output for the command.
	HelpPrompt()
}

// NewStatePush returns an initialized StatePush implementation for the human view.
func NewStatePush(view *View) StatePush {
	return &StatePushHuman{view: view}
}

// StatePushHuman is the human-readable implementation of the StatePush view.
type StatePushHuman struct {
	view *View
}

var _ StatePush = (*StatePushHuman)(nil)

func (v *StatePushHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *StatePushHuman) HelpPrompt() {
	v.view.HelpPrompt("state push")
}
