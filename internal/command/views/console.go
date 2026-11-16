// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Console is the view used for the `console` command.
type Console interface {
	// Output is a generic method for outputting text to the console.
	Output(output string)

	Diagnostics(diags tfdiags.Diagnostics)
}

func NewConsole(viewType arguments.ViewType, view *View) Console {
	switch viewType {
	case arguments.ViewHuman:
		return &ConsoleHuman{view: view}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

type ConsoleHuman struct {
	view *View
}

var _ Console = (*ConsoleHuman)(nil)

func (v *ConsoleHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *ConsoleHuman) Output(output string) {
	v.view.streams.Println(output)
}
