// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type StatePush interface {
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewStatePush(vt arguments.ViewType, view *View) StatePush {
	switch vt {
	case arguments.ViewHuman:
		return &StatePushHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type StatePushHuman struct {
	view *View
}

var _ StatePush = (*StatePushHuman)(nil)

func (v *StatePushHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
