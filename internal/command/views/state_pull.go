// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The StatePull is used for the `state pull` command.
type StatePull interface {
	// The pulled state is printed to stdout
	State(state string)

	Diagnostics(diags tfdiags.Diagnostics)
}

// NewStatePull returns an initialized StatePull implementation for the given ViewType.
func NewStatePull(vt arguments.ViewType, view *View) StatePull {
	switch vt {
	case arguments.ViewHuman:
		return &StatePullHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The StatePullHuman implementation renders diagnostics in a human-readable form.
type StatePullHuman struct {
	view *View
}

var _ StatePull = (*StatePullHuman)(nil)

func (v *StatePullHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *StatePullHuman) State(state string) {
	v.view.streams.Println(state)
}
