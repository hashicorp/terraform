package terraform

import (
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

func TestGraphConnectTo(t *testing.T) {
	var g Graph
	g.Add(&testGraphDependable{VertexName: "a", Mock: []string{"a"}})
	b := g.Add(&testGraphDependable{VertexName: "b"})

	if missing := g.ConnectTo(b, []string{"a"}); len(missing) > 0 {
		t.Fatalf("bad: %#v", missing)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphConnectDepsStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

type testGraphDependable struct {
	VertexName string
	Mock       []string
}

func (v *testGraphDependable) Name() string {
	return v.VertexName
}

func (v *testGraphDependable) DependableName() []string {
	return v.Mock
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
