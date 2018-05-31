package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestDestroyEdgeTransformer_basic(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.A"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.B"})
	tf := &DestroyEdgeTransformer{
		Config:  testModule(t, "transform-destroy-edge-basic"),
		Schemas: simpleTestSchemas(),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyEdgeBasicStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestDestroyEdgeTransformer_create(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.A"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.B"})
	g.Add(&graphNodeCreatorTest{AddrString: "test_object.A"})
	tf := &DestroyEdgeTransformer{
		Config:  testModule(t, "transform-destroy-edge-basic"),
		Schemas: simpleTestSchemas(),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyEdgeCreatorStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestDestroyEdgeTransformer_multi(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.A"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.B"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.C"})
	tf := &DestroyEdgeTransformer{
		Config:  testModule(t, "transform-destroy-edge-multi"),
		Schemas: simpleTestSchemas(),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyEdgeMultiStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestDestroyEdgeTransformer_selfRef(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.A"})
	tf := &DestroyEdgeTransformer{
		Config:  testModule(t, "transform-destroy-edge-self-ref"),
		Schemas: simpleTestSchemas(),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyEdgeSelfRefStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestDestroyEdgeTransformer_module(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeDestroyerTest{AddrString: "module.child.test_object.b"})
	g.Add(&graphNodeDestroyerTest{AddrString: "test_object.a"})
	tf := &DestroyEdgeTransformer{
		Config:  testModule(t, "transform-destroy-edge-module"),
		Schemas: simpleTestSchemas(),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyEdgeModuleStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestDestroyEdgeTransformer_moduleOnly(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeDestroyerTest{AddrString: "module.child.test_object.a"})
	g.Add(&graphNodeDestroyerTest{AddrString: "module.child.test_object.b"})
	g.Add(&graphNodeDestroyerTest{AddrString: "module.child.test_object.c"})
	tf := &DestroyEdgeTransformer{
		Config:  testModule(t, "transform-destroy-edge-module-only"),
		Schemas: simpleTestSchemas(),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
module.child.test_object.a (destroy)
  module.child.test_object.b (destroy)
  module.child.test_object.c (destroy)
module.child.test_object.b (destroy)
  module.child.test_object.c (destroy)
module.child.test_object.c (destroy)
`)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

type graphNodeCreatorTest struct {
	AddrString string
	Refs       []string
}

var (
	_ GraphNodeCreator    = (*graphNodeCreatorTest)(nil)
	_ GraphNodeReferencer = (*graphNodeCreatorTest)(nil)
)

func (n *graphNodeCreatorTest) Name() string {
	return n.CreateAddr().String()
}

func (n *graphNodeCreatorTest) mustAddr() addrs.AbsResourceInstance {
	addr, diags := addrs.ParseAbsResourceInstanceStr(n.AddrString)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr
}

func (n *graphNodeCreatorTest) Path() addrs.ModuleInstance {
	return n.mustAddr().Module
}

func (n *graphNodeCreatorTest) CreateAddr() *addrs.AbsResourceInstance {
	addr := n.mustAddr()
	return &addr
}

func (n *graphNodeCreatorTest) References() []*addrs.Reference {
	ret := make([]*addrs.Reference, len(n.Refs))
	for i, str := range n.Refs {
		ref, diags := addrs.ParseRefStr(str)
		if diags.HasErrors() {
			panic(diags.Err())
		}
		ret[i] = ref
	}
	return ret
}

type graphNodeDestroyerTest struct {
	AddrString string
	CBD        bool
	Modified   bool
}

var _ GraphNodeDestroyer = (*graphNodeDestroyerTest)(nil)

func (n *graphNodeDestroyerTest) Name() string {
	result := n.DestroyAddr().String() + " (destroy)"
	if n.Modified {
		result += " (modified)"
	}

	return result
}

func (n *graphNodeDestroyerTest) mustAddr() addrs.AbsResourceInstance {
	addr, diags := addrs.ParseAbsResourceInstanceStr(n.AddrString)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr
}

func (n *graphNodeDestroyerTest) CreateBeforeDestroy() bool {
	return n.CBD
}

func (n *graphNodeDestroyerTest) ModifyCreateBeforeDestroy(v bool) error {
	n.Modified = true
	return nil
}

func (n *graphNodeDestroyerTest) DestroyAddr() *addrs.AbsResourceInstance {
	addr := n.mustAddr()
	return &addr
}

const testTransformDestroyEdgeBasicStr = `
test_object.A (destroy)
  test_object.B (destroy)
test_object.B (destroy)
`

const testTransformDestroyEdgeCreatorStr = `
test_object.A
  test_object.A (destroy)
test_object.A (destroy)
  test_object.B (destroy)
test_object.B (destroy)
`

const testTransformDestroyEdgeMultiStr = `
test_object.A (destroy)
  test_object.B (destroy)
  test_object.C (destroy)
test_object.B (destroy)
  test_object.C (destroy)
test_object.C (destroy)
`

const testTransformDestroyEdgeSelfRefStr = `
test_object.A (destroy)
`

const testTransformDestroyEdgeModuleStr = `
module.child.test_object.b (destroy)
  test_object.a (destroy)
test_object.a (destroy)
`
