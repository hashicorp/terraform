// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type StateMv interface {
	Move(moves []StateMvPair, dryRun bool, diags tfdiags.Diagnostics)

	Diagnostics(diags tfdiags.Diagnostics)
}

type StateMvPair struct {
	To   string
	From string
}

func NewStateMv(viewType arguments.ViewType, view *View) StateMv {
	switch viewType {
	case arguments.ViewHuman:
		return &StateMvHuman{view: view}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

type StateMvHuman struct {
	view *View
}

func (v *StateMvHuman) Move(moves []StateMvPair, dryRun bool, diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)

	if len(moves) == 0 {
		if dryRun {
			v.view.streams.Println("Would have moved nothing.")
		} else {
			v.view.streams.Println("No matching objects found.")
		}
		return
	}

	// There are moves to report, maybe hypothetically
	prefix := "Move"
	if dryRun {
		prefix = "Would move"
	}
	sb := &strings.Builder{}
	for _, m := range moves {
		sb.WriteString(fmt.Sprintf("%s %q to %q", prefix, m.From, m.To))
		sb.WriteString("\n")

	}
	v.view.streams.Print(sb.String())

	if !dryRun {
		v.view.streams.Println(fmt.Sprintf("Successfully moved %d object(s).", len(moves)))
	}
}

func (v *StateMvHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
