// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type StateRm interface {
	Remove(removals []string, dryRun bool, diags tfdiags.Diagnostics)

	Diagnostics(diags tfdiags.Diagnostics)
}

func NewStateRm(viewType arguments.ViewType, view *View) StateRm {
	switch viewType {
	case arguments.ViewHuman:
		return &StateRmHuman{view: view}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

type StateRmHuman struct {
	view *View
}

func (v *StateRmHuman) Remove(removals []string, dryRun bool, diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)

	if len(removals) == 0 {
		if dryRun {
			v.view.streams.Println("Would have removed nothing.")
			return
		}

		// Note: Zero removals in non-dryrun mode is handled as an error in the command;
		// that scenario should not invoke this view method.
		panic("Remove: method should not be used when -dryrun=false and there are no removals")
	}

	// There are removals to display, maybe speculatively
	prefix := "Removed "
	if dryRun {
		prefix = "Would remove "
	}

	for _, removal := range removals {
		v.view.streams.Println(prefix + removal)
	}

	if !dryRun {
		v.view.streams.Println(
			fmt.Sprintf("Successfully removed %d resource instance(s).", len(removals)),
		)
		return
	}
}

func (v *StateRmHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
