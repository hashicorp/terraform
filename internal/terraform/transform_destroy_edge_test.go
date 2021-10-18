package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

func TestDestroyEdgeTransformer_basic(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(testDestroyNode("test_object.A"))
	g.Add(testDestroyNode("test_object.B"))

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.A").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"A"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.B").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"B","test_string":"x"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_object.A")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	if err := (&AttachStateTransformer{State: state}).Transform(&g); err != nil {
		t.Fatal(err)
	}

	tf := &DestroyEdgeTransformer{
		Config: testModule(t, "transform-destroy-edge-basic"),
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

func TestDestroyEdgeTransformer_multi(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(testDestroyNode("test_object.A"))
	g.Add(testDestroyNode("test_object.B"))
	g.Add(testDestroyNode("test_object.C"))

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.A").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"A"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.B").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"B","test_string":"x"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_object.A")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.C").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"C","test_string":"x"}`),
			Dependencies: []addrs.ConfigResource{
				mustConfigResourceAddr("test_object.A"),
				mustConfigResourceAddr("test_object.B"),
			},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	if err := (&AttachStateTransformer{State: state}).Transform(&g); err != nil {
		t.Fatal(err)
	}

	tf := &DestroyEdgeTransformer{
		Config: testModule(t, "transform-destroy-edge-multi"),
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
	g.Add(testDestroyNode("test_object.A"))
	tf := &DestroyEdgeTransformer{
		Config: testModule(t, "transform-destroy-edge-self-ref"),
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
	g.Add(testDestroyNode("module.child.test_object.b"))
	g.Add(testDestroyNode("test_object.a"))
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	child := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"a"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("module.child.test_object.b")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.b").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"b","test_string":"x"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	if err := (&AttachStateTransformer{State: state}).Transform(&g); err != nil {
		t.Fatal(err)
	}

	tf := &DestroyEdgeTransformer{
		Config: testModule(t, "transform-destroy-edge-module"),
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

	state := states.NewState()
	for moduleIdx := 0; moduleIdx < 2; moduleIdx++ {
		g.Add(testDestroyNode(fmt.Sprintf("module.child[%d].test_object.a", moduleIdx)))
		g.Add(testDestroyNode(fmt.Sprintf("module.child[%d].test_object.b", moduleIdx)))
		g.Add(testDestroyNode(fmt.Sprintf("module.child[%d].test_object.c", moduleIdx)))

		child := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.IntKey(moduleIdx)))
		child.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.a").Resource,
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"a"}`),
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		child.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.b").Resource,
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"b","test_string":"x"}`),
				Dependencies: []addrs.ConfigResource{
					mustConfigResourceAddr("module.child.test_object.a"),
				},
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		child.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.c").Resource,
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"c","test_string":"x"}`),
				Dependencies: []addrs.ConfigResource{
					mustConfigResourceAddr("module.child.test_object.a"),
					mustConfigResourceAddr("module.child.test_object.b"),
				},
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	}

	if err := (&AttachStateTransformer{State: state}).Transform(&g); err != nil {
		t.Fatal(err)
	}

	tf := &DestroyEdgeTransformer{
		Config: testModule(t, "transform-destroy-edge-module-only"),
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	// The analyses done in the destroy edge transformer are between
	// not-yet-expanded objects, which is conservative and so it will generate
	// edges that aren't strictly necessary. As a special case we filter out
	// any edges that are between resources instances that are in different
	// instances of the same module, because those edges are never needed
	// (one instance of a module cannot depend on another instance of the
	// same module) and including them can, in complex cases, cause cycles due
	// to unnecessary interactions between destroyed and created module
	// instances in the same plan.
	//
	// Therefore below we expect to see the dependencies within each instance
	// of module.child reflected, but we should not see any dependencies
	// _between_ instances of module.child.

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
module.child[0].test_object.a (destroy)
  module.child[0].test_object.b (destroy)
  module.child[0].test_object.c (destroy)
module.child[0].test_object.b (destroy)
  module.child[0].test_object.c (destroy)
module.child[0].test_object.c (destroy)
module.child[1].test_object.a (destroy)
  module.child[1].test_object.b (destroy)
  module.child[1].test_object.c (destroy)
module.child[1].test_object.b (destroy)
  module.child[1].test_object.c (destroy)
module.child[1].test_object.c (destroy)
`)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestDestroyEdgeTransformer_destroyThenUpdate(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(testUpdateNode("test_object.A"))
	g.Add(testDestroyNode("test_object.B"))

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.A").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"A","test_string":"old"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.B").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"B","test_string":"x"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_object.A")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	if err := (&AttachStateTransformer{State: state}).Transform(&g); err != nil {
		t.Fatal(err)
	}

	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "a" {
	test_string = "udpated"
}
`,
	})
	tf := &DestroyEdgeTransformer{
		Config: m,
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := strings.TrimSpace(`
test_object.A
  test_object.B (destroy)
test_object.B (destroy)
`)
	actual := strings.TrimSpace(g.String())

	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func testDestroyNode(addrString string) GraphNodeDestroyer {
	instAddr := mustResourceInstanceAddr(addrString)
	inst := NewNodeAbstractResourceInstance(instAddr)
	return &NodeDestroyResourceInstance{NodeAbstractResourceInstance: inst}
}

func testUpdateNode(addrString string) GraphNodeCreator {
	instAddr := mustResourceInstanceAddr(addrString)
	inst := NewNodeAbstractResourceInstance(instAddr)
	return &NodeApplyableResourceInstance{NodeAbstractResourceInstance: inst}
}

const testTransformDestroyEdgeBasicStr = `
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
