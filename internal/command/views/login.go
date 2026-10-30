// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)
type Login interface {
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewLogin(viewType arguments.ViewType, view *View) Login {
	switch viewType {
	case arguments.ViewHuman:
		return &LoginHuman{
			view: view,
		}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

type LoginHuman struct {
	view *View
}

func (v *LoginHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
