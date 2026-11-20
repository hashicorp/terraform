// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Graph interface {
	// Log a graph describing only the resources in the configuration.
	// There needs to be custom logic for displaying only the resources in the configuration,
	// and this is implemented in the views package.
	ResourceOnlyGraph(graph addrs.DirectedGraph[addrs.ConfigResource])

	// Unexported logic for rendering a resource-only graph.
	resourceOnlyGraph(graph addrs.DirectedGraph[addrs.ConfigResource]) string

	// Log a representation of the whole graph.
	// This uses logic from the `dag` package to generate the graph.
	Graph(g *terraform.Graph, opts *dag.DotOpts)

	// Log diagnostics raised when the command fails to prepare a graph.
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewGraph(viewType arguments.ViewType, view *View) Graph {
	switch viewType {
	case arguments.ViewDotGraph:
		return &GraphDot{view: view}
	case arguments.ViewMermaidGraph:
		return &GraphMermaid{view: view}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

type GraphDot struct {
	view *View
}

func (v *GraphDot) ResourceOnlyGraph(graph addrs.DirectedGraph[addrs.ConfigResource]) {
	// No diagnostics are logged here, as warning diagnostics going to stdout could
	// disrupt use of the graph output.

	v.view.streams.Println(v.resourceOnlyGraph(graph))
}

func (v *GraphDot) Graph(g *terraform.Graph, opts *dag.DotOpts) {
	// No diagnostics are logged here, as warning diagnostics going to stdout could
	// disrupt use of the graph output.

	v.view.streams.Println(string(g.Dot(opts)))
}

func (v *GraphDot) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *GraphDot) resourceOnlyGraph(graph addrs.DirectedGraph[addrs.ConfigResource]) string {
	out := &strings.Builder{}

	fmt.Fprintln(out, "digraph G {")
	// Horizontal presentation is easier to read because our nodes tend
	// to be much wider than they are tall. The leftmost nodes in the output
	// are those Terraform would visit first.
	fmt.Fprintln(out, "  rankdir = \"RL\";")
	fmt.Fprintln(out, "  node [shape = rect, fontname = \"sans-serif\"];")

	// To help relate the output back to the configuration it came from,
	// and to make the individual node labels more reasonably sized when
	// deeply nested inside modules, we'll cluster the nodes together by
	// the module they belong to and then show only the local resource
	// address in the individual nodes. We'll accomplish that by sorting
	// the nodes first by module, so we can then notice the transitions.
	allAddrs := graph.AllNodes()
	if len(allAddrs) == 0 {
		fmt.Fprintln(out, "  /* This configuration does not contain any resources.         */")
		fmt.Fprintln(out, "  /* For a more detailed graph, try: terraform graph -type=plan */")
	}
	addrsOrder := make([]addrs.ConfigResource, 0, len(allAddrs))
	for _, addr := range allAddrs {
		addrsOrder = append(addrsOrder, addr)
	}
	sort.Slice(addrsOrder, func(i, j int) bool {
		iAddr, jAddr := addrsOrder[i], addrsOrder[j]
		iModStr, jModStr := iAddr.Module.String(), jAddr.Module.String()
		switch {
		case iModStr != jModStr:
			return iModStr < jModStr
		default:
			iRes, jRes := iAddr.Resource, jAddr.Resource
			switch {
			case iRes.Mode != jRes.Mode:
				return iRes.Mode == addrs.DataResourceMode
			case iRes.Type != jRes.Type:
				return iRes.Type < jRes.Type
			default:
				return iRes.Name < jRes.Name
			}
		}
	})

	currentMod := addrs.RootModule
	for _, addr := range addrsOrder {
		if !addr.Module.Equal(currentMod) {
			// We need a new subgraph, then.
			// Experimentally it seems like nested clusters tend to make it
			// hard for dot to converge on a good layout, so we'll stick with
			// just one level of clusters for now but could revise later based
			// on feedback.
			if !currentMod.IsRoot() {
				fmt.Fprintln(out, "  }")
			}
			currentMod = addr.Module
			fmt.Fprintf(out, "  subgraph \"cluster_%s\" {\n", currentMod.String())
			fmt.Fprintf(out, "    label = %q\n", currentMod.String())
			fmt.Fprintf(out, "    fontname = %q\n", "sans-serif")
		}
		if currentMod.IsRoot() {
			fmt.Fprintf(out, "  %q [label=%q];\n", addr.String(), addr.Resource.String())
		} else {
			fmt.Fprintf(out, "    %q [label=%q];\n", addr.String(), addr.Resource.String())
		}
	}
	if !currentMod.IsRoot() {
		fmt.Fprintln(out, "  }")
	}

	// Now we'll emit all of the edges.
	// We use addrsOrder for both levels to ensure a consistent ordering between
	// runs without further sorting, which means we visit more nodes than we
	// really need to but this output format is only really useful for relatively
	// small graphs anyway, so this should be fine.
	for _, sourceAddr := range addrsOrder {
		deps := graph.DirectDependenciesOf(sourceAddr)
		for _, targetAddr := range addrsOrder {
			if !deps.Has(targetAddr) {
				continue
			}
			fmt.Fprintf(out, "  %q -> %q;\n", sourceAddr.String(), targetAddr.String())
		}
	}

	fmt.Fprintln(out, "}")

	return out.String()
}

type GraphMermaid struct {
	view *View
}

func (v *GraphMermaid) ResourceOnlyGraph(graph addrs.DirectedGraph[addrs.ConfigResource]) {
	// No diagnostics are logged here, as warning diagnostics going to stdout could
	// disrupt use of the graph output.

	v.view.streams.Println(v.resourceOnlyGraph(graph))
}

func (v *GraphMermaid) Graph(g *terraform.Graph, opts *dag.DotOpts) {
	// No diagnostics are logged here, as warning diagnostics going to stdout could
	// disrupt use of the graph output.

	v.view.streams.Println(string(g.Mermaid(opts)))
}

func (v *GraphMermaid) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *GraphMermaid) resourceOnlyGraph(graph addrs.DirectedGraph[addrs.ConfigResource]) string {
	out := &strings.Builder{}

	// use left-to-right layout by default
	fmt.Fprintln(out, "flowchart LR")

	// collect and sort addresses similar to resourceOnlyGraph for deterministic output
	allAddrs := graph.AllNodes()
	if len(allAddrs) == 0 {
		fmt.Fprintln(out, "  %% This configuration does not contain any resources.")
		fmt.Fprintln(out, "  %% For a more detailed graph, try: terraform graph -type=plan")
		return out.String()
	}

	addrsOrder := make([]addrs.ConfigResource, 0, len(allAddrs))
	for _, addr := range allAddrs {
		addrsOrder = append(addrsOrder, addr)
	}
	sort.Slice(addrsOrder, func(i, j int) bool {
		iAddr, jAddr := addrsOrder[i], addrsOrder[j]
		iModStr, jModStr := iAddr.Module.String(), jAddr.Module.String()
		switch {
		case iModStr != jModStr:
			return iModStr < jModStr
		default:
			iRes, jRes := iAddr.Resource, jAddr.Resource
			switch {
			case iRes.Mode != jRes.Mode:
				return iRes.Mode == addrs.DataResourceMode
			case iRes.Type != jRes.Type:
				return iRes.Type < jRes.Type
			default:
				return iRes.Name < jRes.Name
			}
		}
	})

	currentMod := addrs.RootModule
	for _, addr := range addrsOrder {
		if !addr.Module.Equal(currentMod) {
			if !currentMod.IsRoot() {
				fmt.Fprintln(out, "  end")
			}
			currentMod = addr.Module

			fmt.Fprintf(out, "  subgraph %s\n", currentMod.String())
		}
		id := addr.String()
		label := addr.Resource.String()
		if currentMod.IsRoot() {
			fmt.Fprintf(out, "  %s[%s]\n", id, dag.MermaidEscapeLabel(label))
		} else {
			fmt.Fprintf(out, "    %s[%s]\n", id, dag.MermaidEscapeLabel(label))
		}
	}
	if !currentMod.IsRoot() {
		fmt.Fprintln(out, "  end")
	}

	// emit edges
	for _, sourceAddr := range addrsOrder {
		deps := graph.DirectDependenciesOf(sourceAddr)
		srcID := sourceAddr.String()
		for _, targetAddr := range addrsOrder {
			if !deps.Has(targetAddr) {
				continue
			}
			tgtID := targetAddr.String()
			fmt.Fprintf(out, "  %s --> %s\n", srcID, tgtID)
		}
	}

	return out.String()
}
