package terraform

import (
	"strings"
	"testing"
)

func TestFlattenTransformer(t *testing.T) {
	mod := testModule(t, "transform-flatten")

	var b BasicGraphBuilder
	b = BasicGraphBuilder{
		Steps: []GraphTransformer{
			&ConfigTransformer{Module: mod},
			&VertexTransformer{
				Transforms: []GraphVertexTransformer{
					&ExpandTransform{
						Builder: &b,
					},
				},
			},
			&FlattenTransformer{},
		},
	}

	g, err := b.Build(rootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformFlattenStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformFlattenStr = `
aws_instance.parent
module.child.aws_instance.child
  module.child.var.var
module.child.var.var
  aws_instance.parent
`
