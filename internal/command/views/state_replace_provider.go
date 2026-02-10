// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateReplaceProvider is the view interface for the "state replace-provider" command.
type StateReplaceProvider interface {
	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)

	// HelpPrompt directs the user to the full help output for the command.
	HelpPrompt()
}

// NewStateReplaceProvider returns an initialized StateReplaceProvider
// implementation for the human view.
func NewStateReplaceProvider(view *View) StateReplaceProvider {
	return &StateReplaceProviderHuman{view: view}
}

// StateReplaceProviderHuman is the human-readable implementation of the
// StateReplaceProvider view.
type StateReplaceProviderHuman struct {
	view *View
}

var _ StateReplaceProvider = (*StateReplaceProviderHuman)(nil)

func (v *StateReplaceProviderHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *StateReplaceProviderHuman) HelpPrompt() {
	v.view.HelpPrompt("state replace-provider")
}
