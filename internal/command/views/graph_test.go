// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// This test verifies that the Graph view invokes logic to generate a DOT graph representation of a simple graph.
// Full testing of that logic is performed in the `dag` package tests.
func TestGraphDot_Graph(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewGraph(arguments.ViewDotGraph, view)

	// Construct a simple graph
	g := &terraform.Graph{}
	g.Add(stringV("A"))
	g.Add(stringV("B"))
	g.Add(stringV("C"))
	g.Connect(stringV("A"), stringV("B"))
	g.Connect(stringV("A"), stringV("C"))

	expected := `
digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] A"
		"[root] B"
		"[root] C"
		"[root] A" -> "[root] B"
		"[root] A" -> "[root] C"
	}
}`

	opts := &dag.DotOpts{
		DrawCycles: true,
		MaxDepth:   -1, // deprecated
		Verbose:    true,
	}

	v.Graph(g, opts)

	output := done(t)

	// Assert contents
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := strings.TrimSpace(expected)
	if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}

	gotStderr := strings.TrimSpace(output.Stderr())
	wantStderr := ""
	if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestGraphDot_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		diags      tfdiags.Diagnostics
		wantStdout string
		wantStderr string
	}{
		"warning": {
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Example warning",
					"This is an example warning message.",
				),
			},
			"Warning: Example warning\n\nThis is an example warning message.",
			"",
		},
		"error": {
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Example error",
					"This is an example error message.",
				),
			},
			"",
			"Error: Example error\n\nThis is an example error message.",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewGraph(arguments.ViewDotGraph, view)

			v.Diagnostics(tc.diags)

			// Assert contents
			output := done(t)
			gotStdout := strings.TrimSpace(output.Stdout())
			if diff := cmp.Diff(tc.wantStdout, gotStdout); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
			gotStderr := strings.TrimSpace(output.Stderr())
			if diff := cmp.Diff(tc.wantStderr, gotStderr); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
		})
	}
}

// This test verifies that the Graph view invokes logic to generate a Mermaid graph representation of a simple graph.
// Full testing of that logic is performed in the `dag` package tests.
func TestGraphMermaid_Graph(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewGraph(arguments.ViewMermaidGraph, view)

	// Construct a simple graph
	g := &terraform.Graph{}
	g.Add(stringV("A"))
	g.Add(stringV("B"))
	g.Add(stringV("C"))
	g.Connect(stringV("A"), stringV("B"))
	g.Connect(stringV("A"), stringV("C"))

	expected := `
flowchart LR
A["A"]
B["B"]
C["C"]
A --> B
A --> C
`

	opts := &dag.DotOpts{
		DrawCycles: true,
		MaxDepth:   -1, // deprecated
		Verbose:    true,
	}

	v.Graph(g, opts)

	output := done(t)

	// Assert contents
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := strings.TrimSpace(expected)
	if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}

	gotStderr := strings.TrimSpace(output.Stderr())
	wantStderr := ""
	if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestGraphMermaid_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		diags      tfdiags.Diagnostics
		wantStdout string
		wantStderr string
	}{
		"warning": {
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Example warning",
					"This is an example warning message.",
				),
			},
			"Warning: Example warning\n\nThis is an example warning message.",
			"",
		},
		"error": {
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Example error",
					"This is an example error message.",
				),
			},
			"",
			"Error: Example error\n\nThis is an example error message.",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewGraph(arguments.ViewMermaidGraph, view)

			v.Diagnostics(tc.diags)

			// Assert contents
			output := done(t)
			gotStdout := strings.TrimSpace(output.Stdout())
			if diff := cmp.Diff(tc.wantStdout, gotStdout); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
			gotStderr := strings.TrimSpace(output.Stderr())
			if diff := cmp.Diff(tc.wantStderr, gotStderr); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
		})
	}
}

// String vertex implementation for testing purposes.
// TODO: pull into a testing package or testing.go file in `dag` package for reuse.
type stringV string

var (
	_ dag.Vertex          = stringV("")
	_ dag.GraphNodeDotter = stringV("")
)

func (v stringV) Name() string {
	return string(v)
}

func (v stringV) DotNode(n string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{Name: n, Attrs: map[string]string{}}
}
