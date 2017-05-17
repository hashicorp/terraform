package terraform

import (
	"strings"
	"testing"
)

func TestFlatConfigTransformer_nilModule(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &FlatConfigTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(g.Vertices()) > 0 {
		t.Fatal("graph should be empty")
	}
}

func TestFlatConfigTransformer(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &FlatConfigTransformer{
		Module: testModule(t, "transform-flat-config-basic"),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformFlatConfigBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformFlatConfigBasicStr = `
aws_instance.bar
aws_instance.foo
module.child.aws_instance.baz
`
