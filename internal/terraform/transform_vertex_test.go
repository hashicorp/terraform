package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestVertexTransformer_impl(t *testing.T) {
	var _ GraphTransformer = new(VertexTransformer)
}

func TestVertexTransformer(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(dag.BasicEdge(1, 2))
	g.Connect(dag.BasicEdge(2, 3))

	{
		tf := &VertexTransformer{
			Transforms: []GraphVertexTransformer{
				&testVertexTransform{Source: 2, Target: 42},
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testVertexTransformerStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

type testVertexTransform struct {
	Source, Target dag.Vertex
}

func (t *testVertexTransform) Transform(v dag.Vertex) (dag.Vertex, error) {
	if t.Source == v {
		v = t.Target
	}

	return v, nil
}

const testVertexTransformerStr = `
1
  42
3
42
  3
`
