package terraform

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestGraphAdd(t *testing.T) {
	// Test Add since we override it and want to make sure we don't break it.
	var g Graph
	g.Add(42)
	g.Add(84)

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphAddStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphConnectDependent(t *testing.T) {
	var g Graph
	g.Add(&testGraphDependable{VertexName: "a"})
	b := g.Add(&testGraphDependable{
		VertexName:      "b",
		DependentOnMock: []string{"a"},
	})

	if missing := g.ConnectDependent(b); len(missing) > 0 {
		t.Fatalf("bad: %#v", missing)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphConnectDepsStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphReplace_DependableWithNonDependable(t *testing.T) {
	var g Graph
	a := g.Add(&testGraphDependable{VertexName: "a"})
	b := g.Add(&testGraphDependable{
		VertexName:      "b",
		DependentOnMock: []string{"a"},
	})
	newA := "non-dependable-a"

	if missing := g.ConnectDependent(b); len(missing) > 0 {
		t.Fatalf("bad: %#v", missing)
	}

	if !g.Replace(a, newA) {
		t.Fatalf("failed to replace")
	}

	c := g.Add(&testGraphDependable{
		VertexName:      "c",
		DependentOnMock: []string{"a"},
	})

	// This should fail by reporting missing, since a node with dependable
	// name "a" is no longer in the graph.
	missing := g.ConnectDependent(c)
	expected := []string{"a"}
	if !reflect.DeepEqual(expected, missing) {
		t.Fatalf("expected: %#v, got: %#v", expected, missing)
	}
}

func TestGraphWalk_panicWrap(t *testing.T) {
	var g Graph

	// Add our crasher
	v := &testGraphSubPath{
		PathFn: func() []string {
			panic("yo")
		},
	}
	g.Add(v)

	err := g.Walk(GraphWalkerPanicwrap(new(NullGraphWalker)))
	if err == nil {
		t.Fatal("should error")
	}
}

// testGraphHappensBefore is an assertion helper that tests that node
// A (dag.VertexName value) happens before node B.
func testGraphHappensBefore(t *testing.T, g *Graph, A, B string) {
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
