package terraform

import (
	"strings"
	"testing"
)

func TestDestroyEdgeTransformer(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeDestroyerTest{AddrString: "test.A"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.B"})
	tf := &DestroyEdgeTransformer{
		Module: testModule(t, "transform-destroy-edge-basic"),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyEdgeBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestDestroyEdgeTransformer_multi(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeDestroyerTest{AddrString: "test.A"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.B"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.C"})
	tf := &DestroyEdgeTransformer{
		Module: testModule(t, "transform-destroy-edge-multi"),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyEdgeMultiStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

type graphNodeDestroyerTest struct {
	AddrString string
}

func (n *graphNodeDestroyerTest) Name() string { return n.DestroyAddr().String() + " (destroy)" }
func (n *graphNodeDestroyerTest) DestroyAddr() *ResourceAddress {
	addr, err := ParseResourceAddress(n.AddrString)
	if err != nil {
		panic(err)
	}

	return addr
}

const testTransformDestroyEdgeBasicStr = `
test.A (destroy)
  test.B (destroy)
test.B (destroy)
`

const testTransformDestroyEdgeMultiStr = `
test.A (destroy)
  test.B (destroy)
test.B (destroy)
  test.C (destroy)
test.C (destroy)
`
