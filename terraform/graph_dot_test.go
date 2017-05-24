package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

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

				levelOne := []interface{}{"foo", "bar"}
				for i, s := range levelOne {
					levelOne[i] = &testDrawable{
						VertexName: s.(string),
					}
					v := levelOne[i]

					g.Add(v)
					g.Connect(dag.BasicEdge(v, root))
				}

				levelTwo := []string{"baz", "qux"}
				for i, s := range levelTwo {
					v := &testDrawable{
						VertexName: s,
					}

					g.Add(v)
					g.Connect(dag.BasicEdge(v, levelOne[i]))
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

				g.Connect(dag.BasicEdge(vA, root))
				g.Connect(dag.BasicEdge(vA, vC))
				g.Connect(dag.BasicEdge(vB, vA))
				g.Connect(dag.BasicEdge(vC, vB))

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
			Name: "subgraphs, no depth restriction",
			Opts: dag.DotOpts{
				MaxDepth: -1,
			},
			Graph: func() *Graph {
				var g Graph
				root := &testDrawableOrigin{"root"}
				g.Add(root)

				var sub Graph
				vSubRoot := sub.Add(&testDrawableOrigin{"sub_root"})

				var subsub Graph
				subsub.Add(&testDrawableOrigin{"subsub_root"})
				vSubV := sub.Add(&testDrawableSubgraph{
					VertexName:   "subsub",
					SubgraphMock: &subsub,
				})

				vSub := g.Add(&testDrawableSubgraph{
					VertexName:   "sub",
					SubgraphMock: &sub,
				})

				g.Connect(dag.BasicEdge(vSub, root))
				sub.Connect(dag.BasicEdge(vSubV, vSubRoot))

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

		{
			Name: "subgraphs, with depth restriction",
			Opts: dag.DotOpts{
				MaxDepth: 1,
			},
			Graph: func() *Graph {
				var g Graph
				root := &testDrawableOrigin{"root"}
				g.Add(root)

				var sub Graph
				rootSub := sub.Add(&testDrawableOrigin{"sub_root"})

				var subsub Graph
				subsub.Add(&testDrawableOrigin{"subsub_root"})

				subV := sub.Add(&testDrawableSubgraph{
					VertexName:   "subsub",
					SubgraphMock: &subsub,
				})
				vSub := g.Add(&testDrawableSubgraph{
					VertexName:   "sub",
					SubgraphMock: &sub,
				})

				g.Connect(dag.BasicEdge(vSub, root))
				sub.Connect(dag.BasicEdge(subV, rootSub))
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

type testDrawableSubgraph struct {
	VertexName      string
	SubgraphMock    *Graph
	DependentOnMock []string
}

func (node *testDrawableSubgraph) Name() string {
	return node.VertexName
}
func (node *testDrawableSubgraph) Subgraph() dag.Grapher {
	return node.SubgraphMock
}
func (node *testDrawableSubgraph) DotNode(n string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{Name: n, Attrs: map[string]string{}}
}
func (node *testDrawableSubgraph) DependentOn() []string {
	return node.DependentOnMock
}
