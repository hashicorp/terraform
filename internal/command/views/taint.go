// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Taint is the view interface for the "taint" command.
type Taint interface {
	// Success renders the message confirming a resource instance was tainted.
	Success(addr addrs.AbsResourceInstance)

	// AllowMissingWarning renders a warning when the resource was not found
	// but -allow-missing was set.
	AllowMissingWarning(addr addrs.AbsResourceInstance)

	// Diagnostics renders a set of warnings and errors.
	Diagnostics(diags tfdiags.Diagnostics)

	// HelpPrompt renders a prompt directing users to help output.
	HelpPrompt()
}

// NewTaint returns an initialized Taint implementation for the human view.
func NewTaint(view *View) Taint {
	return &TaintHuman{view: view}
}

// TaintHuman is the human-readable implementation of the Taint view.
type TaintHuman struct {
	view *View
}

var _ Taint = (*TaintHuman)(nil)

func (v *TaintHuman) Success(addr addrs.AbsResourceInstance) {
	v.view.streams.Println(fmt.Sprintf("Resource instance %s has been marked as tainted.", addr))
}

func (v *TaintHuman) AllowMissingWarning(addr addrs.AbsResourceInstance) {
	v.view.Diagnostics(tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Warning,
			"No such resource instance",
			fmt.Sprintf("Resource instance %s was not found, but this is not an error because -allow-missing was set.", addr),
		),
	})
}

func (v *TaintHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *TaintHuman) HelpPrompt() {
	v.view.HelpPrompt("taint")
}
