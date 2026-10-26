// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Taint is used for the taint command.
type Taint interface {
	Taint(name addrs.AbsResourceInstance, diags tfdiags.Diagnostics)

	Diagnostics(diags tfdiags.Diagnostics)
}

// NewTaint returns an initialized Taint implementation for the given ViewType.
func NewTaint(vt arguments.ViewType, view *View) Taint {
	switch vt {
	case arguments.ViewHuman:
		return &TaintHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The TaintHuman implementation renders diagnostics in a human-readable form.
type TaintHuman struct {
	view *View
}

var _ Taint = (*TaintHuman)(nil)

func (v *TaintHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *TaintHuman) Taint(name addrs.AbsResourceInstance, diags tfdiags.Diagnostics) {
	v.Diagnostics(diags)

	v.view.streams.Println(
		fmt.Sprintf("Resource instance %s has been marked as tainted.", name),
	)
}
