package terraform

import (
	"reflect"
	"strings"
	"testing"
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
