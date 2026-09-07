// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/dag"
)

type stringV string

func (v stringV) Name() string {
	return string(v)
}

func TestGraphDot(t *testing.T) {
	cases := []struct {
		Name   string
		Graph  testGraphFunc
		Opts   dag.DotOpts
		Expect string
		Error  string
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
				root := &testDrawableOrigin{"root"}
				g.Add(root)

				levelOne := []dag.Vertex{stringV("foo"), stringV("bar")}
				for i, s := range levelOne {
					levelOne[i] = &testDrawable{
						VertexName: s.Name(),
					}
					v := levelOne[i]

					g.Add(v)
					g.Connect(v, root)
				}

				levelTwo := []string{"baz", "qux"}
				for i, s := range levelTwo {
					v := &testDrawable{
						VertexName: s,
					}

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
				root := &testDrawableOrigin{"root"}
				g.Add(root)

				vA := g.Add(&testDrawable{
					VertexName: "A",
				})

				vB := g.Add(&testDrawable{
					VertexName: "B",
				})

				vC := g.Add(&testDrawable{
					VertexName: "C",
				})

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
	}
	for _, tc := range cases {
		tn := tc.Name
		t.Run(tn, func(t *testing.T) {
			g := tc.Graph()
			var err error
			//actual, err := GraphDot(g, &tc.Opts)
			actual := string(g.Dot(&tc.Opts))

			if err == nil && tc.Error != "" {
				t.Fatalf("%s: expected err: %s, got none", tn, tc.Error)
			}
			if err != nil && tc.Error == "" {
				t.Fatalf("%s: unexpected err: %s", tn, err)
			}
			if err != nil && tc.Error != "" {
				if !strings.Contains(err.Error(), tc.Error) {
					t.Fatalf("%s: expected err: %s\nto contain: %s", tn, err, tc.Error)
				}
				return
			}

			expected := strings.TrimSpace(tc.Expect) + "\n"
			if actual != expected {
				t.Fatalf("%s:\n\nexpected:\n%s\n\ngot:\n%s", tn, expected, actual)
			}
		})
	}
}

type testGraphFunc func() *Graph

type testDrawable struct {
	VertexName      string
	DependentOnMock []string
}

func (node *testDrawable) Name() string {
	return node.VertexName
}
func (node *testDrawable) DotNode(n string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{Name: n, Attrs: map[string]string{}}
}
func (node *testDrawable) DependableName() []string {
	return []string{node.VertexName}
}
func (node *testDrawable) DependentOn() []string {
	return node.DependentOnMock
}

type testDrawableOrigin struct {
	VertexName string
}

func (node *testDrawableOrigin) Name() string {
	return node.VertexName
}
func (node *testDrawableOrigin) DotNode(n string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{Name: n, Attrs: map[string]string{}}
}
func (node *testDrawableOrigin) DotOrigin() bool {
	return true
}
func (node *testDrawableOrigin) DependableName() []string {
	return []string{node.VertexName}
}
