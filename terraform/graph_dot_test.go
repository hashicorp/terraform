package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dot"
)

func TestGraphDot(t *testing.T) {
	cases := map[string]struct {
		Graph  testGraphFunc
		Opts   GraphDotOpts
		Expect string
		Error  string
	}{
		"empty": {
			Graph: func() *Graph { return &Graph{} },
			Error: "No DOT origin nodes found",
		},
		"three-level": {
			Graph: func() *Graph {
				var g Graph
				root := &testDrawableOrigin{"root"}
				g.Add(root)

				levelOne := []string{"foo", "bar"}
				for _, s := range levelOne {
					g.Add(&testDrawable{
						VertexName:      s,
						DependentOnMock: []string{"root"},
					})
				}

				levelTwo := []string{"baz", "qux"}
				for i, s := range levelTwo {
					g.Add(&testDrawable{
						VertexName:      s,
						DependentOnMock: levelOne[i : i+1],
					})
				}

				g.ConnectDependents()
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
		"cycle": {
			Opts: GraphDotOpts{
				DrawCycles: true,
			},
			Graph: func() *Graph {
				var g Graph
				root := &testDrawableOrigin{"root"}
				g.Add(root)

				g.Add(&testDrawable{
					VertexName:      "A",
					DependentOnMock: []string{"root", "C"},
				})

				g.Add(&testDrawable{
					VertexName:      "B",
					DependentOnMock: []string{"A"},
				})

				g.Add(&testDrawable{
					VertexName:      "C",
					DependentOnMock: []string{"B"},
				})

				g.ConnectDependents()
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
		"subgraphs, no depth restriction": {
			Opts: GraphDotOpts{
				MaxDepth: -1,
			},
			Graph: func() *Graph {
				var g Graph
				root := &testDrawableOrigin{"root"}
				g.Add(root)

				var sub Graph
				sub.Add(&testDrawableOrigin{"sub_root"})

				var subsub Graph
				subsub.Add(&testDrawableOrigin{"subsub_root"})
				sub.Add(&testDrawableSubgraph{
					VertexName:      "subsub",
					SubgraphMock:    &subsub,
					DependentOnMock: []string{"sub_root"},
				})
				g.Add(&testDrawableSubgraph{
					VertexName:      "sub",
					SubgraphMock:    &sub,
					DependentOnMock: []string{"root"},
				})

				g.ConnectDependents()
				sub.ConnectDependents()
				return &g
			},
			Expect: `
digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] root"
		"[root] sub"
		"[root] sub" -> "[root] root"
	}
	subgraph "cluster_sub" {
		label = "sub"
		"[sub] sub_root"
		"[sub] subsub"
		"[sub] subsub" -> "[sub] sub_root"
	}
	subgraph "cluster_subsub" {
		label = "subsub"
		"[subsub] subsub_root"
	}
}
			`,
		},
		"subgraphs, with depth restriction": {
			Opts: GraphDotOpts{
				MaxDepth: 1,
			},
			Graph: func() *Graph {
				var g Graph
				root := &testDrawableOrigin{"root"}
				g.Add(root)

				var sub Graph
				sub.Add(&testDrawableOrigin{"sub_root"})

				var subsub Graph
				subsub.Add(&testDrawableOrigin{"subsub_root"})
				sub.Add(&testDrawableSubgraph{
					VertexName:      "subsub",
					SubgraphMock:    &subsub,
					DependentOnMock: []string{"sub_root"},
				})
				g.Add(&testDrawableSubgraph{
					VertexName:      "sub",
					SubgraphMock:    &sub,
					DependentOnMock: []string{"root"},
				})

				g.ConnectDependents()
				sub.ConnectDependents()
				return &g
			},
			Expect: `
digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] root"
		"[root] sub"
		"[root] sub" -> "[root] root"
	}
	subgraph "cluster_sub" {
		label = "sub"
		"[sub] sub_root"
		"[sub] subsub"
		"[sub] subsub" -> "[sub] sub_root"
	}
}
			`,
		},
	}

	for tn, tc := range cases {
		actual, err := GraphDot(tc.Graph(), &tc.Opts)
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
			continue
		}

		expected := strings.TrimSpace(tc.Expect) + "\n"
		if actual != expected {
			t.Fatalf("%s:\n\nexpected:\n%s\n\ngot:\n%s", tn, expected, actual)
		}
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
func (node *testDrawable) DotNode(n string, opts *GraphDotOpts) *dot.Node {
	return dot.NewNode(n, map[string]string{})
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
func (node *testDrawableOrigin) DotNode(n string, opts *GraphDotOpts) *dot.Node {
	return dot.NewNode(n, map[string]string{})
}
func (node *testDrawableOrigin) DotOrigin() bool {
	return true
}
func (node *testDrawableOrigin) DependableName() []string {
	return []string{node.VertexName}
}

type testDrawableSubgraph struct {
	VertexName      string
	SubgraphMock    *Graph
	DependentOnMock []string
}

func (node *testDrawableSubgraph) Name() string {
	return node.VertexName
}
func (node *testDrawableSubgraph) Subgraph() *Graph {
	return node.SubgraphMock
}
func (node *testDrawableSubgraph) DotNode(n string, opts *GraphDotOpts) *dot.Node {
	return dot.NewNode(n, map[string]string{})
}
func (node *testDrawableSubgraph) DependentOn() []string {
	return node.DependentOnMock
}
