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

func TestFlattenTransformer_withProxy(t *testing.T) {
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
			&ProxyTransformer{},
		},
	}

	g, err := b.Build(rootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformFlattenProxyStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformFlattenStr = `
aws_instance.parent
aws_instance.parent-output
  module.child.output.output
module.child.aws_instance.child
  module.child.var.var
module.child.output.output
  module.child.aws_instance.child
module.child.plan-destroy
module.child.var.var
  aws_instance.parent
`

const testTransformFlattenProxyStr = `
aws_instance.parent
aws_instance.parent-output
  module.child.aws_instance.child
  module.child.output.output
module.child.aws_instance.child
  aws_instance.parent
  module.child.var.var
module.child.output.output
  module.child.aws_instance.child
module.child.plan-destroy
module.child.var.var
  aws_instance.parent
`
