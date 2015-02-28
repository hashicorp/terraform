package terraform

import (
	"strings"
	"testing"
)

func TestTransitiveReductionTransformer(t *testing.T) {
	mod := testModule(t, "transform-trans-reduce-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &TransitiveReductionTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformTransReduceBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformTransReduceBasicStr = `
aws_instance.A
aws_instance.B
  aws_instance.A
aws_instance.C
  aws_instance.B
`
