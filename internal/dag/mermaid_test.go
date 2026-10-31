// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"strings"
	"testing"
)

type testVertex struct{ n string }

func (t *testVertex) Name() string { return t.n }

func (t *testVertex) DotNode(title string, opts *DotOpts) *DotNode {
	return &DotNode{Name: title, Attrs: map[string]string{"label": t.n}}
}

func TestMermaidBasic(t *testing.T) {
	g := &Graph{}

	a := &testVertex{n: "a"}
	b := &testVertex{n: "b"}
	c := &testVertex{n: "c"}

	g.Add(a)
	g.Add(b)
	g.Add(c)

	g.Connect(a, b)
	g.Connect(b, c)

	mg := newMarshalGraph("root", g)
	out := string(mg.Mermaid(&DotOpts{DrawCycles: true, MaxDepth: -1, Verbose: true}))

	if !strings.Contains(out, "a") || !strings.Contains(out, "b") || !strings.Contains(out, "c") {
		t.Fatalf("expected nodes a, b, c in output, got: %s", out)
	}

	if !strings.Contains(out, "-->") {
		t.Fatalf("expected edges in mermaid output, got: %s", out)
	}
}

func TestGraphMermaid(t *testing.T) {
	cases := []struct {
		Name   string
		Graph  func() *Graph
		Opts   DotOpts
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

				levelOne := []Vertex{stringV("foo"), stringV("bar")}
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
			Opts: DotOpts{
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
A --> C:::cycle
A --> root
B --> A:::cycle
C --> B:::cycle
classDef cycle stroke:#ff0000,stroke-width:2px;
`,
		},
		{
			Name: "cycle not drawn",
			Opts: DotOpts{
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
