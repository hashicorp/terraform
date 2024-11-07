// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
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

	tf := &DestroyEdgeTransformer{}
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

	tf := &DestroyEdgeTransformer{}
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
	tf := &DestroyEdgeTransformer{}
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

	tf := &DestroyEdgeTransformer{}
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

	tf := &DestroyEdgeTransformer{}
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

	tf := &DestroyEdgeTransformer{}
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

func TestPruneUnusedNodesTransformer_rootModuleOutputValues(t *testing.T) {
	// This is a kinda-weird test case covering the very narrow situation
	// where a root module output value depends on a resource, where we
	// need to make sure that the output value doesn't block pruning of
	// the resource from the graph. This special case exists because although
	// root module objects are "expanders", they in practice always expand
	// to exactly one instance and so don't have the usual requirement of
	// needing to stick around in order to support downstream expanders
	// when there are e.g. nested expanding modules.

	// In order to keep this test focused on the pruneUnusedNodesTransformer
	// as much as possible we're using a minimal graph construction here which
	// is just enough to get the nodes we need, but this does mean that this
	// test might be invalidated by future changes to the apply graph builder,
	// and so if something seems off here it might help to compare the
	// following with the real apply graph transformer and verify whether
	// this smaller construction is still realistic enough to be a valid test.
	// It might be valid to change or remove this test to "make it work", as
	// long as you verify that there is still _something_ upholding the
	// invariant that a root module output value should not block a resource
	// node from being pruned from the graph.

	concreteResource := func(a *NodeAbstractResource) dag.Vertex {
		return &nodeExpandApplyableResource{
			NodeAbstractResource: a,
		}
	}

	concreteResourceInstance := func(a *NodeAbstractResourceInstance) dag.Vertex {
		return &NodeApplyableResourceInstance{
			NodeAbstractResourceInstance: a,
		}
	}

	resourceInstAddr := mustResourceInstanceAddr("test.a")
	providerCfgAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.MustParseProviderSourceString("foo/test"),
	}
	emptyObjDynamicVal, err := plans.NewDynamicValue(cty.EmptyObjectVal, cty.EmptyObject)
	if err != nil {
		t.Fatal(err)
	}
	nullObjDynamicVal, err := plans.NewDynamicValue(cty.NullVal(cty.EmptyObject), cty.EmptyObject)
	if err != nil {
		t.Fatal(err)
	}

	config := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test" "a" {
			}

			output "test" {
				value = test.a.foo
			}
		`,
	})
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			resourceInstAddr,
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{}`),
			},
			providerCfgAddr,
		)
	})
	changes := plans.NewChangesSrc()
	changes.AppendResourceInstanceChange(&plans.ResourceInstanceChangeSrc{
		Addr:         resourceInstAddr,
		PrevRunAddr:  resourceInstAddr,
		ProviderAddr: providerCfgAddr,
		ChangeSrc: plans.ChangeSrc{
			Action: plans.Delete,
			Before: emptyObjDynamicVal,
			After:  nullObjDynamicVal,
		},
	})

	builder := &BasicGraphBuilder{
		Steps: []GraphTransformer{
			&ConfigTransformer{
				Concrete: concreteResource,
				Config:   config,
			},
			&OutputTransformer{
				Config: config,
			},
			&DiffTransformer{
				Concrete: concreteResourceInstance,
				State:    state,
				Changes:  changes,
			},
			&ReferenceTransformer{},
			&AttachDependenciesTransformer{},
			&pruneUnusedNodesTransformer{},
			&CloseRootModuleTransformer{},
		},
	}
	graph, diags := builder.Build(addrs.RootModuleInstance)
	assertNoDiagnostics(t, diags)

	// At this point, thanks to pruneUnusedNodesTransformer, we should still
	// have the node for the output value, but the "test.a (expand)" node
	// should've been pruned in recognition of the fact that we're performing
	// a destroy and therefore we only need the "test.a (destroy)" node.

	nodesByName := make(map[string]dag.Vertex)
	nodesByResourceExpand := make(map[string]dag.Vertex)
	for _, n := range graph.Vertices() {
		name := dag.VertexName(n)
		if _, exists := nodesByName[name]; exists {
			t.Fatalf("multiple nodes have name %q", name)
		}
		nodesByName[name] = n

		if exp, ok := n.(*nodeExpandApplyableResource); ok {
			addr := exp.Addr
			if _, exists := nodesByResourceExpand[addr.String()]; exists {
				t.Fatalf("multiple nodes are expanders for %s", addr)
			}
			nodesByResourceExpand[addr.String()] = exp
		}
	}

	// NOTE: The following is sensitive to the current name string formats we
	// use for these particular node types. These names are not contractual
	// so if this breaks in future it is fine to update these names to the new
	// names as long as you verify first that the new names correspond to
	// the same meaning as what we're assuming below.
	if _, exists := nodesByName["test.a (destroy)"]; !exists {
		t.Errorf("missing destroy node for resource instance test.a")
	}
	if _, exists := nodesByName["output.test (expand)"]; !exists {
		t.Errorf("missing expand for output value 'test'")
	}

	// We _must not_ have any node that expands a resource.
	if len(nodesByResourceExpand) != 0 {
		t.Errorf("resource expand nodes remain the graph after transform; should've been pruned\n%s", spew.Sdump(nodesByResourceExpand))
	}
}

// NoOp changes should not be participating in the destroy sequence
func TestDestroyEdgeTransformer_noOp(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(testDestroyNode("test_object.A"))
	g.Add(testUpdateNode("test_object.B"))
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
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_object.A"),
				mustConfigResourceAddr("test_object.B")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	if err := (&AttachStateTransformer{State: state}).Transform(&g); err != nil {
		t.Fatal(err)
	}

	tf := &DestroyEdgeTransformer{
		// We only need a minimal object to indicate GraphNodeCreator change is
		// a NoOp here.
		Changes: &plans.ChangesSrc{
			Resources: []*plans.ResourceInstanceChangeSrc{
				{
					Addr:      mustResourceInstanceAddr("test_object.B"),
					ChangeSrc: plans.ChangeSrc{Action: plans.NoOp},
				},
			},
		},
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := strings.TrimSpace(`
test_object.A (destroy)
  test_object.C (destroy)
test_object.B
test_object.C (destroy)`)

	actual := strings.TrimSpace(g.String())
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestDestroyEdgeTransformer_dataDependsOn(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}

	addrA := mustResourceInstanceAddr("test_object.A")
	instA := NewNodeAbstractResourceInstance(addrA)
	a := &NodeDestroyResourceInstance{NodeAbstractResourceInstance: instA}
	g.Add(a)

	// B here represents a data sources, which is effectively an update during
	// apply, but won't have dependencies stored in the state.
	addrB := mustResourceInstanceAddr("test_object.B")
	instB := NewNodeAbstractResourceInstance(addrB)
	instB.Dependencies = append(instB.Dependencies, addrA.ConfigResource())
	b := &NodeApplyableResourceInstance{NodeAbstractResourceInstance: instB}

	g.Add(b)

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

	if err := (&AttachStateTransformer{State: state}).Transform(&g); err != nil {
		t.Fatal(err)
	}

	tf := &DestroyEdgeTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
test_object.A (destroy)
test_object.B
  test_object.A (destroy)
`)
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
