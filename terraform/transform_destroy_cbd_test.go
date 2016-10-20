package terraform

import (
	"strings"
	"testing"
)

func TestCBDEdgeTransformer(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeCreatorTest{AddrString: "test.A"})
	g.Add(&graphNodeCreatorTest{AddrString: "test.B"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.A", CBD: true})

	module := testModule(t, "transform-destroy-edge-basic")

	{
		tf := &DestroyEdgeTransformer{
			Module: module,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CBDEdgeTransformer{Module: module}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformCBDEdgeBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformCBDEdgeBasicStr = `
test.A
test.A (destroy)
  test.A
  test.B
test.B
`
