// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type StateList interface {
	List(addresses []string, diags tfdiags.Diagnostics)

	Diagnostics(diags tfdiags.Diagnostics)
}

func NewStateList(viewType arguments.ViewType, view *View) StateList {
	switch viewType {
	case arguments.ViewHuman:
		return &StateListHuman{view: view}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

type StateListHuman struct {
	view *View
}

func (v *StateListHuman) List(addresses []string, diags tfdiags.Diagnostics) {
	sb := &strings.Builder{}
	for _, addr := range addresses {
		sb.WriteString(addr)
		sb.WriteString("\n")
	}
	v.view.streams.Print(sb.String())

	// The `state list` command outputs the list of resource addresses
	// before any diagnostics, and this is preserved here for now.
	v.view.Diagnostics(diags)
}

func (v *StateListHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
