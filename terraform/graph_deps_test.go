package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestGraphConnectDeps(t *testing.T) {
	var g dag.Graph
	g.Add(&testGraphDependable{VertexName: "a", Mock: []string{"a"}})
	b := g.Add(&testGraphDependable{VertexName: "b"})

	if n := GraphConnectDeps(&g, b, []string{"a"}); n != 1 {
		t.Fatalf("bad: %d", n)
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

const testGraphConnectDepsStr = `
a
b
  a
`
