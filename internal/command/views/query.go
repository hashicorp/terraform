// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Query renders outputs for query executions.
type Query interface {
	Operation() Operation
	Hooks() []terraform.Hook

	Diagnostics(diags tfdiags.Diagnostics)
	HelpPrompt()
}

func NewQuery(vt arguments.ViewType, view *View) Query {
	switch vt {
	case arguments.ViewJSON:
		return &QueryJSON{
			view: NewJSONView(view),
		}
	case arguments.ViewHuman:
		return &QueryHuman{
			view:         view,
			inAutomation: view.RunningInAutomation(),
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type QueryHuman struct {
	view *View

	inAutomation bool
}

var _ Query = (*QueryHuman)(nil)

func (v *QueryHuman) Operation() Operation {
	return NewQueryOperation(arguments.ViewHuman, v.inAutomation, v.view)
}

func (v *QueryHuman) Hooks() []terraform.Hook {
	return []terraform.Hook{
		NewUiHook(v.view),
	}
}

func (v *QueryHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
func (v *QueryHuman) HelpPrompt() {
	v.view.HelpPrompt("query")
}

type QueryJSON struct {
	view *JSONView
}

var _ Query = (*QueryJSON)(nil)

func (v *QueryJSON) Operation() Operation {
	return &QueryOperationJSON{view: v.view}
}

func (v *QueryJSON) Hooks() []terraform.Hook {
	return []terraform.Hook{
		newJSONHook(v.view),
	}
}

func (v *QueryJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *QueryJSON) HelpPrompt() {
}
