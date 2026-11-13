// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type StateReplaceProvider interface {
	// DescribeChanges reports the changes that will be made to the state.
	// This could be zero changes or a list of resources.
	DescribeChanges(from, to tfaddr.Provider, resources []*states.Resource)

	// Success prints a summary after the state is successfully updated.
	Success(replacedCount int)

	// The user cancelled the operation.
	Cancelled()

	Diagnostics(diags tfdiags.Diagnostics)
}

func NewStateReplaceProvider(vt arguments.ViewType, view *View) StateReplaceProvider {
	switch vt {
	case arguments.ViewHuman:
		return &StateReplaceProviderHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type StateReplaceProviderHuman struct {
	view *View
}

var _ StateReplaceProvider = (*StateReplaceProviderHuman)(nil)

func (v *StateReplaceProviderHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *StateReplaceProviderHuman) DescribeChanges(from, to tfaddr.Provider, resources []*states.Resource) {
	if len(resources) == 0 {
		v.view.streams.Println("No matching resources found.")
		return
	}

	// Explain the changes present
	sb := strings.Builder{}
	colorize := v.view.colorize
	sb.WriteString("Terraform will perform the following actions:\n\n")
	sb.WriteString(colorize.Color("  [yellow]~[reset] Updating provider:\n"))
	sb.WriteString(colorize.Color(fmt.Sprintf("    [red]-[reset] %s\n", from)))
	sb.WriteString(colorize.Color(fmt.Sprintf("    [green]+[reset] %s\n\n", to)))

	sb.WriteString(colorize.Color(fmt.Sprintf("[bold]Changing[reset] %d resources:\n\n", len(resources))))
	for _, resource := range resources {
		sb.WriteString(colorize.Color(fmt.Sprintf("  %s\n", resource.Addr)))
	}

	v.view.streams.Print(sb.String())
}

func (v *StateReplaceProviderHuman) Success(replacedCount int) {
	v.view.streams.Println(
		fmt.Sprintf("\nSuccessfully replaced provider for %d resources.", replacedCount),
	)
}

func (v *StateReplaceProviderHuman) Cancelled() {
	v.view.streams.Println("Cancelled replacing providers.")
}
