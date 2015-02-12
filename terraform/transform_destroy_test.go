package terraform

import (
	"strings"
	"testing"
)

func TestDestroyTransformer(t *testing.T) {
	mod := testModule(t, "transform-destroy-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &DestroyTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformDestroyBasicStr = `
aws_instance.bar
  aws_instance.bar (destroy)
aws_instance.bar (destroy)
  aws_instance.foo
aws_instance.foo
  aws_instance.foo (destroy)
aws_instance.foo (destroy)
`
