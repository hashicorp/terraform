package terraform

import (
	"strings"
	"testing"
)

func TestResourceCountTransformer(t *testing.T) {
	cfg := testModule(t, "transform-resource-count-basic").Config()
	resource := cfg.Resources[0]

	g := Graph{Path: RootModulePath}
	{
		tf := &ResourceCountTransformer{Resource: resource}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testResourceCountTransformStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestResourceCountTransformer_countNegative(t *testing.T) {
	cfg := testModule(t, "transform-resource-count-negative").Config()
	resource := cfg.Resources[0]

	g := Graph{Path: RootModulePath}
	{
		tf := &ResourceCountTransformer{Resource: resource}
		if err := tf.Transform(&g); err == nil {
			t.Fatal("should error")
		}
	}
}

const testResourceCountTransformStr = `
aws_instance.foo #0
  aws_instance.foo #2
aws_instance.foo #1
  aws_instance.foo #2
aws_instance.foo #2
  aws_instance.foo #2
`
