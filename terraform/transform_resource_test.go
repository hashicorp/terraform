package terraform

import (
	"strings"
	"testing"
)

func TestResourceCountTransformerOld(t *testing.T) {
	cfg := testModule(t, "transform-resource-count-basic").Config()
	resource := cfg.Resources[0]

	g := Graph{Path: RootModulePath}
	{
		tf := &ResourceCountTransformerOld{Resource: resource}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testResourceCountTransformOldStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestResourceCountTransformerOld_countNegative(t *testing.T) {
	cfg := testModule(t, "transform-resource-count-negative").Config()
	resource := cfg.Resources[0]

	g := Graph{Path: RootModulePath}
	{
		tf := &ResourceCountTransformerOld{Resource: resource}
		if err := tf.Transform(&g); err == nil {
			t.Fatal("should error")
		}
	}
}

func TestResourceCountTransformerOld_deps(t *testing.T) {
	cfg := testModule(t, "transform-resource-count-deps").Config()
	resource := cfg.Resources[0]

	g := Graph{Path: RootModulePath}
	{
		tf := &ResourceCountTransformerOld{Resource: resource}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testResourceCountTransformOldDepsStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testResourceCountTransformOldStr = `
aws_instance.foo #0
aws_instance.foo #1
aws_instance.foo #2
`

const testResourceCountTransformOldDepsStr = `
aws_instance.foo #0
aws_instance.foo #1
  aws_instance.foo #0
`
