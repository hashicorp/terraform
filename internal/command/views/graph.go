// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Graph interface {
	// Log a pre-prepared graph representation along with any diagnostics.
	//
	// Calling code generates the string, so owns handling whether output is dot or mermaid format.
	// In future we could refactor that logic into a GraphMermaid and GraphDot views, with human-
	// readable diagnostics. This is similar to the 'raw' view for the `output` command.
	Graph(graph string, diags tfdiags.Diagnostics)

	// Log diagnostics raised when the command fails to prepare a graph.
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewGraph(viewType arguments.ViewType, view *View) Graph {
	switch viewType {
	case arguments.ViewHuman:
		return &GraphHuman{view: view}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

type GraphHuman struct {
	view *View
}

func (v *GraphHuman) Graph(graph string, diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)

	v.view.streams.Println(graph)
}

func (v *GraphHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
