package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/dag"
)

// testGraphContains is an assertion helper that tests that a node is
// contained in the graph.
func testGraphContains(t *testing.T, g *Graph, name string) {
	for _, v := range g.Vertices() {
		if dag.VertexName(v) == name {
			return
		}
	}

	t.Fatalf(
		"Expected %q in:\n\n%s",
		name, g.String())
}

// testGraphnotContains is an assertion helper that tests that a node is
// NOT contained in the graph.
func testGraphNotContains(t *testing.T, g *Graph, name string) {
	for _, v := range g.Vertices() {
		if dag.VertexName(v) == name {
			t.Fatalf(
				"Expected %q to NOT be in:\n\n%s",
				name, g.String())
		}
	}
}

// testGraphHappensBefore is an assertion helper that tests that node
// A (dag.VertexName value) happens before node B.
func testGraphHappensBefore(t *testing.T, g *Graph, A, B string) {
	t.Helper()
	// Find the B vertex
	var vertexB dag.Vertex
	for _, v := range g.Vertices() {
		if dag.VertexName(v) == B {
			vertexB = v
			break
		}
	}
	if vertexB == nil {
		t.Fatalf(
			"Expected %q before %q. Couldn't find %q in:\n\n%s",
			A, B, B, g.String())
	}

	// Look at ancestors
	deps, err := g.Ancestors(vertexB)
	if err != nil {
		t.Fatalf("Error: %s in graph:\n\n%s", err, g.String())
	}

	// Make sure B is in there
	for _, v := range deps.List() {
		if dag.VertexName(v) == A {
			// Success
			return
		}
	}

	t.Fatalf(
		"Expected %q before %q in:\n\n%s",
		A, B, g.String())
}

type testGraphSubPath struct {
	PathFn func() []string
}

func (v *testGraphSubPath) Path() []string { return v.PathFn() }

type testGraphDependable struct {
	VertexName      string
	DependentOnMock []string
}

func (v *testGraphDependable) Name() string {
	return v.VertexName
}

func (v *testGraphDependable) DependableName() []string {
	return []string{v.VertexName}
}

func (v *testGraphDependable) DependentOn() []string {
	return v.DependentOnMock
}

const testGraphAddStr = `
42
84
`

const testGraphConnectDepsStr = `
a
b
  a
`
