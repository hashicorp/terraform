package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestCBDEdgeTransformer(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeCreatorTest{AddrString: "test.A"})
	g.Add(&graphNodeCreatorTest{AddrString: "test.B"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.A", CBD: true})

	module := testModule(t, "transform-destroy-edge-basic")

	{
		tf := &DestroyEdgeTransformer{
			Config: module,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CBDEdgeTransformer{Config: module}
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

func TestCBDEdgeTransformer_depNonCBD(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeCreatorTest{AddrString: "test.A"})
	g.Add(&graphNodeCreatorTest{AddrString: "test.B"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.A"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.B", CBD: true})

	module := testModule(t, "transform-destroy-edge-basic")

	{
		tf := &DestroyEdgeTransformer{
			Config: module,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CBDEdgeTransformer{Config: module}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformCBDEdgeDepNonCBDStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestCBDEdgeTransformer_depNonCBDCount(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeCreatorTest{AddrString: "test.A"})
	g.Add(&graphNodeCreatorTest{AddrString: "test.B[0]"})
	g.Add(&graphNodeCreatorTest{AddrString: "test.B[1]"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.A", CBD: true})

	module := testModule(t, "transform-destroy-edge-splat")

	{
		tf := &DestroyEdgeTransformer{
			Config: module,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CBDEdgeTransformer{Config: module}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
test.A
test.A (destroy)
  test.A
  test.B[0]
  test.B[1]
test.B[0]
test.B[1]
	`)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestCBDEdgeTransformer_depNonCBDCountBoth(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeCreatorTest{AddrString: "test.A[0]"})
	g.Add(&graphNodeCreatorTest{AddrString: "test.A[1]"})
	g.Add(&graphNodeCreatorTest{AddrString: "test.B[0]"})
	g.Add(&graphNodeCreatorTest{AddrString: "test.B[1]"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.A[0]", CBD: true})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.A[1]", CBD: true})

	module := testModule(t, "transform-destroy-edge-splat")

	{
		tf := &DestroyEdgeTransformer{
			Config: module,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CBDEdgeTransformer{Config: module}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
test.A[0]
test.A[0] (destroy)
  test.A[0]
  test.B[0]
  test.B[1]
test.A[1]
test.A[1] (destroy)
  test.A[1]
  test.B[0]
  test.B[1]
test.B[0]
test.B[1]
	`)
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

const testTransformCBDEdgeDepNonCBDStr = `
test.A
test.A (destroy) (modified)
  test.A
  test.B
  test.B (destroy)
test.B
test.B (destroy)
  test.B
`
