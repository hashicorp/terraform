package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestExpandTransform_impl(t *testing.T) {
	var _ GraphVertexTransformer = new(ExpandTransform)
}

func TestExpandTransform(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Connect(dag.BasicEdge(1, 2))

	tf := &ExpandTransform{}
	out, err := tf.Transform(&testExpandable{
		Result: &g,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	sn, ok := out.(GraphNodeSubgraph)
	if !ok {
		t.Fatalf("not subgraph: %#v", out)
	}

	actual := strings.TrimSpace(sn.Subgraph().String())
	expected := strings.TrimSpace(testExpandTransformStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestExpandTransform_nonExpandable(t *testing.T) {
	tf := &ExpandTransform{}
	out, err := tf.Transform(42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if out != 42 {
		t.Fatalf("bad: %#v", out)
	}
}

type testExpandable struct {
	// Inputs
	Result      *Graph
	ResultError error

	// Outputs
	Builder GraphBuilder
}

func (n *testExpandable) Expand(b GraphBuilder) (GraphNodeSubgraph, error) {
	n.Builder = b
	return &testSubgraph{n.Result}, n.ResultError
}

type testSubgraph struct {
	Graph *Graph
}

func (n *testSubgraph) Subgraph() *Graph {
	return n.Graph
}

const testExpandTransformStr = `
1
  2
2
`
