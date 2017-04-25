package terraform

import (
	"strings"
	"testing"
)

func TestDestroyEdgeTransformer_basic(t *testing.T) {
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

func TestDestroyEdgeTransformer_create(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeDestroyerTest{AddrString: "test.A"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test.B"})
	g.Add(&graphNodeCreatorTest{AddrString: "test.A"})
	tf := &DestroyEdgeTransformer{
		Module: testModule(t, "transform-destroy-edge-basic"),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyEdgeCreatorStr)
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

func TestDestroyEdgeTransformer_selfRef(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeDestroyerTest{AddrString: "test.A"})
	tf := &DestroyEdgeTransformer{
		Module: testModule(t, "transform-destroy-edge-self-ref"),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyEdgeSelfRefStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestDestroyEdgeTransformer_module(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeDestroyerTest{AddrString: "module.child.aws_instance.b"})
	g.Add(&graphNodeDestroyerTest{AddrString: "aws_instance.a"})
	tf := &DestroyEdgeTransformer{
		Module: testModule(t, "transform-destroy-edge-module"),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyEdgeModuleStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestDestroyEdgeTransformer_moduleOnly(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeDestroyerTest{AddrString: "module.child.aws_instance.a"})
	g.Add(&graphNodeDestroyerTest{AddrString: "module.child.aws_instance.b"})
	g.Add(&graphNodeDestroyerTest{AddrString: "module.child.aws_instance.c"})
	tf := &DestroyEdgeTransformer{
		Module: testModule(t, "transform-destroy-edge-module-only"),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
module.child.aws_instance.a (destroy)
  module.child.aws_instance.b (destroy)
  module.child.aws_instance.c (destroy)
module.child.aws_instance.b (destroy)
  module.child.aws_instance.c (destroy)
module.child.aws_instance.c (destroy)
`)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

type graphNodeCreatorTest struct {
	AddrString string
	Refs       []string
}

func (n *graphNodeCreatorTest) Name() string { return n.CreateAddr().String() }
func (n *graphNodeCreatorTest) CreateAddr() *ResourceAddress {
	addr, err := ParseResourceAddress(n.AddrString)
	if err != nil {
		panic(err)
	}

	return addr
}

func (n *graphNodeCreatorTest) References() []string { return n.Refs }

type graphNodeDestroyerTest struct {
	AddrString string
	CBD        bool
	Modified   bool
}

func (n *graphNodeDestroyerTest) Name() string {
	result := n.DestroyAddr().String() + " (destroy)"
	if n.Modified {
		result += " (modified)"
	}

	return result
}

func (n *graphNodeDestroyerTest) CreateBeforeDestroy() bool { return n.CBD }

func (n *graphNodeDestroyerTest) ModifyCreateBeforeDestroy(v bool) error {
	n.Modified = true
	return nil
}

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

const testTransformDestroyEdgeCreatorStr = `
test.A
  test.A (destroy)
test.A (destroy)
  test.B (destroy)
test.B (destroy)
`

const testTransformDestroyEdgeMultiStr = `
test.A (destroy)
  test.B (destroy)
  test.C (destroy)
test.B (destroy)
  test.C (destroy)
test.C (destroy)
`

const testTransformDestroyEdgeSelfRefStr = `
test.A (destroy)
`

const testTransformDestroyEdgeModuleStr = `
aws_instance.a (destroy)
module.child.aws_instance.b (destroy)
  aws_instance.a (destroy)
`
