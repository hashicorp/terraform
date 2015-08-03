package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestPruneNoopTransformer(t *testing.T) {
	g := Graph{Path: RootModulePath}

	a := &testGraphNodeNoop{NameValue: "A"}
	b := &testGraphNodeNoop{NameValue: "B", Value: true}
	c := &testGraphNodeNoop{NameValue: "C"}

	g.Add(a)
	g.Add(b)
	g.Add(c)
	g.Connect(dag.BasicEdge(a, b))
	g.Connect(dag.BasicEdge(b, c))

	{
		tf := &PruneNoopTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformPruneNoopStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformPruneNoopStr = `
A
  C
C
`

type testGraphNodeNoop struct {
	NameValue string
	Value     bool
}

func (v *testGraphNodeNoop) Name() string {
	return v.NameValue
}

func (v *testGraphNodeNoop) Noop(*NoopOpts) bool {
	return v.Value
}
