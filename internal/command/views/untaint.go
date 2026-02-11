// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Untaint is the view interface for the "untaint" command.
type Untaint interface {
	// Success renders the message confirming a resource instance was untainted.
	Success(addr addrs.AbsResourceInstance)

	// AllowMissingWarning renders a warning when the resource was not found
	// but -allow-missing was set.
	AllowMissingWarning(addr addrs.AbsResourceInstance)

	// Diagnostics renders a set of warnings and errors.
	Diagnostics(diags tfdiags.Diagnostics)

	// HelpPrompt renders a prompt directing users to help output.
	HelpPrompt()
}

// NewUntaint returns an initialized Untaint implementation for the human view.
func NewUntaint(view *View) Untaint {
	return &UntaintHuman{view: view}
}

// UntaintHuman is the human-readable implementation of the Untaint view.
type UntaintHuman struct {
	view *View
}

var _ Untaint = (*UntaintHuman)(nil)

func (v *UntaintHuman) Success(addr addrs.AbsResourceInstance) {
	v.view.streams.Println(fmt.Sprintf("Resource instance %s has been successfully untainted.", addr))
}

func (v *UntaintHuman) AllowMissingWarning(addr addrs.AbsResourceInstance) {
	v.view.Diagnostics(tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Warning,
			"No such resource instance",
			fmt.Sprintf("Resource instance %s was not found, but this is not an error because -allow-missing was set.", addr),
		),
	})
}

func (v *UntaintHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *UntaintHuman) HelpPrompt() {
	v.view.HelpPrompt("untaint")
}
