package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestCBDEdgeTransformer(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.A"})
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.B"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.A", CBD: true})

	module := testModule(t, "transform-destroy-edge-basic")

	{
		tf := &DestroyEdgeTransformer{
			Config:     module,
			Components: simpleMockComponentFactory(),
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CBDEdgeTransformer{
			Config:     module,
			Components: simpleMockComponentFactory(),
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformCBDEdgeBasicStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestCBDEdgeTransformer_depNonCBD(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.A"})
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.B"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.A"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.B", CBD: true})

	module := testModule(t, "transform-destroy-edge-basic")

	{
		tf := &DestroyEdgeTransformer{
			Config:     module,
			Components: simpleMockComponentFactory(),
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CBDEdgeTransformer{
			Config:     module,
			Components: simpleMockComponentFactory(),
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformCBDEdgeDepNonCBDStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestCBDEdgeTransformer_depNonCBDCount(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.A"})
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.B[0]"})
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.B[1]"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.A", CBD: true})

	module := testModule(t, "transform-destroy-edge-splat")

	{
		tf := &DestroyEdgeTransformer{
			Config:     module,
			Components: simpleMockComponentFactory(),
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CBDEdgeTransformer{
			Config:     module,
			Components: simpleMockComponentFactory(),
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
test_object.A
test_object.A (destroy)
  test_object.A
  test_object.B[0]
  test_object.B[1]
test_object.B[0]
test_object.B[1]
	`)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestCBDEdgeTransformer_depNonCBDCountBoth(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.A[0]"})
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.A[1]"})
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.B[0]"})
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.B[1]"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.A[0]", CBD: true})
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.A[1]", CBD: true})

	module := testModule(t, "transform-destroy-edge-splat")

	{
		tf := &DestroyEdgeTransformer{
			Config:     module,
			Components: simpleMockComponentFactory(),
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CBDEdgeTransformer{
			Config:     module,
			Components: simpleMockComponentFactory(),
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
test_object.A[0]
test_object.A[0] (destroy)
  test_object.A[0]
  test_object.B[0]
  test_object.B[1]
test_object.A[1]
test_object.A[1] (destroy)
  test_object.A[1]
  test_object.B[0]
  test_object.B[1]
test_object.B[0]
test_object.B[1]
	`)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

const testTransformCBDEdgeBasicStr = `
test_object.A
test_object.A (destroy)
  test_object.A
  test_object.B
test_object.B
`

const testTransformCBDEdgeDepNonCBDStr = `
test_object.A
test_object.A (destroy) (modified)
  test_object.A
  test_object.B
  test_object.B (destroy)
test_object.B
test_object.B (destroy)
  test_object.B
`
