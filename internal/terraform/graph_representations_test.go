// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/dag"
)

// Minimal vertex implementation for testing purposes
//
// NOTE: the receiver for this type is a value, not a pointer.
// This means the string will be used as the unique identifier of the vertex (see `marshalVertexID`).
// If the receiver was a pointer then the memory address would be used as the identifier,
// which would not be stable for testing purposes. The Mermaid formatting tests are sensitive to this.
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

func TestGraphDot(t *testing.T) {
	cases := []struct {
		Name   string
		Graph  func() *Graph
		Opts   dag.DotOpts
		Expect string
	}{
		{
			Name:  "empty",
			Graph: func() *Graph { return &Graph{} },
			Expect: `
digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
	}
}`,
		},
		{
			Name: "three-level",
			Graph: func() *Graph {
				var g Graph
				root := stringV("root")
				g.Add(root)

				levelOne := []dag.Vertex{stringV("foo"), stringV("bar")}
				for i, s := range levelOne {
					levelOne[i] = stringV(
						s.Name(),
					)
					v := levelOne[i]

					g.Add(v)
					g.Connect(v, root)
				}

				levelTwo := []string{"baz", "qux"}
				for i, s := range levelTwo {
					v := stringV(s)

					g.Add(v)
					g.Connect(v, levelOne[i])
				}

				return &g
			},
			Expect: `
digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] bar"
		"[root] baz"
		"[root] foo"
		"[root] qux"
		"[root] root"
		"[root] bar" -> "[root] root"
		"[root] baz" -> "[root] foo"
		"[root] foo" -> "[root] root"
		"[root] qux" -> "[root] bar"
	}
}
			`,
		},

		{
			Name: "cycle",
			Opts: dag.DotOpts{
				DrawCycles: true,
			},
			Graph: func() *Graph {
				var g Graph
				root := stringV("root")
				g.Add(root)

				vA := g.Add(stringV("A"))

				vB := g.Add(stringV("B"))

				vC := g.Add(stringV("C"))

				g.Connect(vA, root)
				g.Connect(vA, vC)
				g.Connect(vB, vA)
				g.Connect(vC, vB)

				return &g
			},
			Expect: `
digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] A"
		"[root] B"
		"[root] C"
		"[root] root"
		"[root] A" -> "[root] B" [color = "red", penwidth = "2.0"]
		"[root] A" -> "[root] C"
		"[root] A" -> "[root] root"
		"[root] B" -> "[root] A"
		"[root] B" -> "[root] C" [color = "red", penwidth = "2.0"]
		"[root] C" -> "[root] A" [color = "red", penwidth = "2.0"]
		"[root] C" -> "[root] B"
	}
}
					`,
		},
		{
			Name: "cycle not drawn",
			Opts: dag.DotOpts{
				DrawCycles: false, // Do not draw cycles
			},
			Graph: func() *Graph {
				var g Graph
				root := stringV("root")
				g.Add(root)

				vA := g.Add(stringV("A"))

				vB := g.Add(stringV("B"))

				vC := g.Add(stringV("C"))

				g.Connect(vA, root)
				g.Connect(vA, vC)
				g.Connect(vB, vA)
				g.Connect(vC, vB)

				return &g
			},
			Expect: `
digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] A"
		"[root] B"
		"[root] C"
		"[root] root"
		"[root] A" -> "[root] C"
		"[root] A" -> "[root] root"
		"[root] B" -> "[root] A"
		"[root] C" -> "[root] B"
	}
}
					`,
		},
	}
	for _, tc := range cases {
		tn := tc.Name
		t.Run(tn, func(t *testing.T) {
			g := tc.Graph()
			actual := string(g.Dot(&tc.Opts))

			expected := strings.TrimSpace(tc.Expect) + "\n"
			if actual != expected {
				t.Fatalf("%s:\n\nexpected:\n%s\n\ngot:\n%s", tn, expected, actual)
			}
		})
	}
}

func TestGraphMermaid(t *testing.T) {
	cases := []struct {
		Name   string
		Graph  func() *Graph
		Opts   dag.DotOpts
		Expect string
	}{
		{
			Name:  "empty",
			Graph: func() *Graph { return &Graph{} },
			Expect: `
flowchart LR`,
		},
		{
			Name: "three-level",
			Graph: func() *Graph {
				var g Graph
				root := stringV("root")
				g.Add(root)

				levelOne := []dag.Vertex{stringV("foo"), stringV("bar")}
				for i, s := range levelOne {
					levelOne[i] = stringV(s.Name())
					v := levelOne[i]

					g.Add(v)
					g.Connect(v, root)
				}

				levelTwo := []string{"baz", "qux"}
				for i, s := range levelTwo {
					v := stringV(s)

					g.Add(v)
					g.Connect(v, levelOne[i])
				}

				return &g
			},
			Expect: `
flowchart LR
bar["bar"]
baz["baz"]
foo["foo"]
qux["qux"]
root["root"]
bar --> root
baz --> foo
foo --> root
qux --> bar
`,
		},

		{
			Name: "cycle",
			Opts: dag.DotOpts{
				DrawCycles: true,
			},
			Graph: func() *Graph {
				var g Graph
				root := stringV("root")
				g.Add(root)

				vA := g.Add(stringV("A"))

				vB := g.Add(stringV("B"))

				vC := g.Add(stringV("C"))

				g.Connect(vA, root)
				g.Connect(vA, vC)
				g.Connect(vB, vA)
				g.Connect(vC, vB)

				return &g
			},
			Expect: `
flowchart LR
A["A"]
B["B"]
C["C"]
root["root"]
A --> C
A --> root
B --> A
C --> B
classDef cycle stroke:#ff0000,stroke-width:2px;
`,
		},
		{
			Name: "cycle not drawn",
			Opts: dag.DotOpts{
				DrawCycles: false, // Do not draw cycles
			},
			Graph: func() *Graph {
				var g Graph
				root := stringV("root")
				g.Add(root)

				vA := g.Add(stringV("A"))

				vB := g.Add(stringV("B"))

				vC := g.Add(stringV("C"))

				g.Connect(vA, root)
				g.Connect(vA, vC)
				g.Connect(vB, vA)
				g.Connect(vC, vB)

				return &g
			},
			Expect: `
flowchart LR
A["A"]
B["B"]
C["C"]
root["root"]
A --> C
A --> root
B --> A
C --> B
`,
		},
	}
	for _, tc := range cases {
		tn := tc.Name
		t.Run(tn, func(t *testing.T) {
			g := tc.Graph()
			actual := string(g.Mermaid(&tc.Opts))

			expected := strings.TrimSpace(tc.Expect) + "\n"
			if actual != expected {
				t.Fatalf("%s:\n\nexpected:\n%s\n\ngot:\n%s", tn, expected, actual)
			}
		})
	}
}
