// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Untaint is used for the untaint command.
type Untaint interface {
	Untaint(name addrs.AbsResourceInstance, diags tfdiags.Diagnostics)

	Diagnostics(diags tfdiags.Diagnostics)
}

// NewUntaint returns an initialized Untaint implementation for the given ViewType.
func NewUntaint(vt arguments.ViewType, view *View) Untaint {
	switch vt {
	case arguments.ViewHuman:
		return &UntaintHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The UntaintHuman implementation renders diagnostics in a human-readable form.
type UntaintHuman struct {
	view *View
}

var _ Untaint = (*UntaintHuman)(nil)

func (v *UntaintHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *UntaintHuman) Untaint(name addrs.AbsResourceInstance, diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)

	v.view.streams.Println(
		fmt.Sprintf("Resource instance %s has been successfully untainted.", name),
	)
}
