package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
)

func TestApplyGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(ApplyGraphBuilder)
}

func TestApplyGraphBuilder(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.create"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Create,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.other"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
				},
			},
			{
				Addr: mustResourceInstanceAddr("module.child.test_object.create"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Create,
				},
			},
			{
				Addr: mustResourceInstanceAddr("module.child.test_object.other"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Create,
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:     testModule(t, "graph-builder-apply-basic"),
		Changes:    changes,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong path %q", g.Path.String())
	}

	actual := strings.TrimSpace(g.String())

	expected := strings.TrimSpace(testApplyGraphBuilderStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// This tests the ordering of two resources where a non-CBD depends
// on a CBD. GH-11349.
func TestApplyGraphBuilder_depCbd(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.A"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.CreateThenDelete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.B"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:     testModule(t, "graph-builder-apply-dep-cbd"),
		Changes:    changes,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong path %q", g.Path.String())
	}

	// We're going to go hunting for our deposed instance node here, so we
	// can find out its key to use in the assertions below.
	var dk states.DeposedKey
	for _, v := range g.Vertices() {
		tv, ok := v.(*NodeDestroyDeposedResourceInstanceObject)
		if !ok {
			continue
		}
		if dk != states.NotDeposed {
			t.Fatalf("more than one deposed instance node in the graph; want only one")
		}
		dk = tv.DeposedKey
	}
	if dk == states.NotDeposed {
		t.Fatalf("no deposed instance node in the graph; want one")
	}

	destroyName := fmt.Sprintf("test_object.A (destroy deposed %s)", dk)

	// Create A, Modify B, Destroy A
	testGraphHappensBefore(
		t, g,
		"test_object.A",
		destroyName,
	)
	testGraphHappensBefore(
		t, g,
		"test_object.A",
		"test_object.B",
	)
	testGraphHappensBefore(
		t, g,
		"test_object.B",
		destroyName,
	)
}

// This tests the ordering of two resources that are both CBD that
// require destroy/create.
func TestApplyGraphBuilder_doubleCBD(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.A"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.CreateThenDelete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.B"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.CreateThenDelete,
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:        testModule(t, "graph-builder-apply-double-cbd"),
		Changes:       changes,
		Components:    simpleMockComponentFactory(),
		Schemas:       simpleTestSchemas(),
		DisableReduce: true,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong path %q", g.Path.String())
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testApplyGraphBuilderDoubleCBDStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// This tests the ordering of two resources being destroyed that depend
// on each other from only state. GH-11749
func TestApplyGraphBuilder_destroyStateOnly(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("module.child.test_object.A"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Delete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("module.child.test_object.B"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Delete,
				},
			},
		},
	}

	state := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"test_object.A": &ResourceState{
						Type: "test_object",
						Primary: &InstanceState{
							ID:         "foo",
							Attributes: map[string]string{},
						},
						Provider: "provider.test",
					},

					"test_object.B": &ResourceState{
						Type: "test_object",
						Primary: &InstanceState{
							ID:         "bar",
							Attributes: map[string]string{},
						},
						Dependencies: []string{"test_object.A"},
						Provider:     "provider.test",
					},
				},
			},
		},
	})

	b := &ApplyGraphBuilder{
		Config:        testModule(t, "empty"),
		Changes:       changes,
		State:         state,
		Components:    simpleMockComponentFactory(),
		Schemas:       simpleTestSchemas(),
		DisableReduce: true,
	}

	g, diags := b.Build(addrs.RootModuleInstance)
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}
	t.Logf("Graph:\n%s", g.String())

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong path %q", g.Path.String())
	}

	testGraphHappensBefore(
		t, g,
		"module.child.test_object.B (destroy)",
		"module.child.test_object.A (destroy)")
}

// This tests the ordering of destroying a single count of a resource.
func TestApplyGraphBuilder_destroyCount(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.A[1]"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Delete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.B"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:        testModule(t, "graph-builder-apply-count"),
		Changes:       changes,
		Components:    simpleMockComponentFactory(),
		Schemas:       simpleTestSchemas(),
		DisableReduce: true,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong module path %q", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testApplyGraphBuilderDestroyCountStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestApplyGraphBuilder_moduleDestroy(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("module.A.test_object.foo"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Delete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("module.B.test_object.foo"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Delete,
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:     testModule(t, "graph-builder-apply-module-destroy"),
		Changes:    changes,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphHappensBefore(
		t, g,
		"module.B.test_object.foo (destroy)",
		"module.A.test_object.foo (destroy)",
	)
}

func TestApplyGraphBuilder_provisioner(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.foo"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Create,
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:     testModule(t, "graph-builder-apply-provisioner"),
		Changes:    changes,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphContains(t, g, "provisioner.test")
	testGraphHappensBefore(
		t, g,
		"provisioner.test",
		"test_object.foo",
	)
}

func TestApplyGraphBuilder_provisionerDestroy(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.foo"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Delete,
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Destroy:    true,
		Config:     testModule(t, "graph-builder-apply-provisioner"),
		Changes:    changes,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphContains(t, g, "provisioner.test")
	testGraphHappensBefore(
		t, g,
		"provisioner.test",
		"test_object.foo (destroy)",
	)
}

func TestApplyGraphBuilder_targetModule(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.foo"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
				},
			},
			{
				Addr: mustResourceInstanceAddr("module.child2.test_object.foo"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:     testModule(t, "graph-builder-apply-target-module"),
		Changes:    changes,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child2", addrs.NoKey),
		},
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphNotContains(t, g, "module.child1.output.instance_id")
}

const testApplyGraphBuilderStr = `
meta.count-boundary (EachMode fixup)
  module.child.test_object.other
  test_object.other
module.child.provisioner.test
module.child.test_object.create
  module.child.test_object.create (prepare state)
module.child.test_object.create (prepare state)
  module.child.provisioner.test
  provider.test
module.child.test_object.other
  module.child.test_object.create
  module.child.test_object.other (prepare state)
module.child.test_object.other (prepare state)
  provider.test
provider.test
provider.test (close)
  module.child.test_object.other
  test_object.other
provisioner.test (close)
  module.child.test_object.create
root
  meta.count-boundary (EachMode fixup)
  provider.test (close)
  provisioner.test (close)
test_object.create
  test_object.create (prepare state)
test_object.create (prepare state)
  provider.test
test_object.other
  test_object.create
  test_object.other (prepare state)
test_object.other (prepare state)
  provider.test
`

const testApplyGraphBuilderDoubleCBDStr = `
meta.count-boundary (EachMode fixup)
  provider.test
  test_object.A
  test_object.A (destroy)
  test_object.B
  test_object.B (destroy)
provider.test
provider.test (close)
  provider.test
  test_object.A
  test_object.A (destroy)
  test_object.B
  test_object.B (destroy)
root
  meta.count-boundary (EachMode fixup)
  provider.test (close)
test_object.A
  provider.test
test_object.A (destroy)
  provider.test
  test_object.A
  test_object.B
  test_object.B (destroy)
test_object.B
  provider.test
  test_object.A
test_object.B (destroy)
  provider.test
  test_object.B
`

const testApplyGraphBuilderDestroyCountStr = `
meta.count-boundary (EachMode fixup)
  provider.test
  test_object.A[1] (destroy)
  test_object.B
provider.test
provider.test (close)
  provider.test
  test_object.A[1] (destroy)
  test_object.B
root
  meta.count-boundary (EachMode fixup)
  provider.test (close)
test_object.A[1] (destroy)
  provider.test
test_object.B
  provider.test
  test_object.A[1] (destroy)
`
