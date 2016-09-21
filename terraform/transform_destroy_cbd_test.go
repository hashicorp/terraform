package terraform

import (
	"strings"
	"testing"
)

func TestCBDEdgeTransformer(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeCreatorTest{AddrString: "test.A"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.A", CBD: true})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.B"})

	{
		tf := &DestroyEdgeTransformer{
			Module: testModule(t, "transform-destroy-edge-basic"),
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CBDEdgeTransformer{}
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
  test.B (destroy)
test.B (destroy)
`
