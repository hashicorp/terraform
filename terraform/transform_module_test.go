package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestModuleInputTransformer(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(dag.BasicEdge(1, 2))
	g.Connect(dag.BasicEdge(1, 3))

	{
		tf := &ModuleInputTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testModuleInputTransformStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testModuleInputTransformStr = `
1
  2
  3
2
  module inputs
3
  module inputs
module inputs
`
