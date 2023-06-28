// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
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
		Config:  testModule(t, "graph-builder-apply-basic"),
		Changes: changes,
		Plugins: simpleMockPluginLibrary(),
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong path %q", g.Path.String())
	}

	got := strings.TrimSpace(g.String())
	want := strings.TrimSpace(testApplyGraphBuilderStr)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("wrong result\n%s", diff)
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
			AttrsJSON:    []byte(`{"id":"B","test_list":["x"]}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_object.A")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	b := &ApplyGraphBuilder{
		Config:  testModule(t, "graph-builder-apply-dep-cbd"),
		Changes: changes,
		Plugins: simpleMockPluginLibrary(),
		State:   state,
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
		Config:  testModule(t, "graph-builder-apply-double-cbd"),
		Changes: changes,
		Plugins: simpleMockPluginLibrary(),
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
	var destroyA, destroyB string
	for _, v := range g.Vertices() {
		tv, ok := v.(*NodeDestroyDeposedResourceInstanceObject)
		if !ok {
			continue
		}

		switch tv.Addr.Resource.Resource.Name {
		case "A":
			destroyA = fmt.Sprintf("test_object.A (destroy deposed %s)", tv.DeposedKey)
		case "B":
			destroyB = fmt.Sprintf("test_object.B (destroy deposed %s)", tv.DeposedKey)
		default:
			t.Fatalf("unknown instance: %s", tv.Addr)
		}
	}

	// Create A, Modify B, Destroy A
	testGraphHappensBefore(
		t, g,
		"test_object.A",
		destroyA,
	)
	testGraphHappensBefore(
		t, g,
		"test_object.A",
		"test_object.B",
	)
	testGraphHappensBefore(
		t, g,
		"test_object.B",
		destroyB,
	)
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

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	child := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.A").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.B").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"bar"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("module.child.test_object.A")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	b := &ApplyGraphBuilder{
		Config:  testModule(t, "empty"),
		Changes: changes,
		State:   state,
		Plugins: simpleMockPluginLibrary(),
	}

	g, diags := b.Build(addrs.RootModuleInstance)
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

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

	state := states.NewState()
	root := state.RootModule()
	addrA := mustResourceInstanceAddr("test_object.A[1]")
	root.SetResourceInstanceCurrent(
		addrA.Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"B"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.B").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"B"}`),
			Dependencies: []addrs.ConfigResource{addrA.ContainingResource().Config()},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	b := &ApplyGraphBuilder{
		Config:  testModule(t, "graph-builder-apply-count"),
		Changes: changes,
		Plugins: simpleMockPluginLibrary(),
		State:   state,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong module path %q", g.Path)
	}

	got := strings.TrimSpace(g.String())
	want := strings.TrimSpace(testApplyGraphBuilderDestroyCountStr)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("wrong result\n%s", diff)
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

	state := states.NewState()
	modA := state.EnsureModule(addrs.RootModuleInstance.Child("A", addrs.NoKey))
	modA.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	modB := state.EnsureModule(addrs.RootModuleInstance.Child("B", addrs.NoKey))
	modB.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"foo","value":"foo"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("module.A.test_object.foo")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	b := &ApplyGraphBuilder{
		Config:  testModule(t, "graph-builder-apply-module-destroy"),
		Changes: changes,
		Plugins: simpleMockPluginLibrary(),
		State:   state,
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
		Config:  testModule(t, "graph-builder-apply-target-module"),
		Changes: changes,
		Plugins: simpleMockPluginLibrary(),
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

// Ensure that an update resulting from the removal of a resource happens after
// that resource is destroyed.
func TestApplyGraphBuilder_updateFromOrphan(t *testing.T) {
	schemas := simpleTestSchemas()
	instanceSchema := schemas.Providers[addrs.NewDefaultProvider("test")].ResourceTypes["test_object"]

	bBefore, _ := plans.NewDynamicValue(
		cty.ObjectVal(map[string]cty.Value{
			"id":          cty.StringVal("b_id"),
			"test_string": cty.StringVal("a_id"),
		}), instanceSchema.ImpliedType())
	bAfter, _ := plans.NewDynamicValue(
		cty.ObjectVal(map[string]cty.Value{
			"id":          cty.StringVal("b_id"),
			"test_string": cty.StringVal("changed"),
		}), instanceSchema.ImpliedType())

	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.a"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Delete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.b"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
					Before: bBefore,
					After:  bAfter,
				},
			},
		},
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_object",
			Name: "a",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"a_id"}`),
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	root.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_object",
			Name: "b",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"b_id","test_string":"a_id"}`),
			Dependencies: []addrs.ConfigResource{
				{
					Resource: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_object",
						Name: "a",
					},
					Module: root.Addr.Module(),
				},
			},
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)

	b := &ApplyGraphBuilder{
		Config:  testModule(t, "graph-builder-apply-orphan-update"),
		Changes: changes,
		Plugins: simpleMockPluginLibrary(),
		State:   state,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := strings.TrimSpace(`
test_object.a (destroy)
test_object.b
  test_object.a (destroy)
`)

	instanceGraph := filterInstances(g)
	got := strings.TrimSpace(instanceGraph.String())

	if got != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, got)
	}
}

// Ensure that an update resulting from the removal of a resource happens before
// a CBD resource is destroyed.
func TestApplyGraphBuilder_updateFromCBDOrphan(t *testing.T) {
	schemas := simpleTestSchemas()
	instanceSchema := schemas.Providers[addrs.NewDefaultProvider("test")].ResourceTypes["test_object"]

	bBefore, _ := plans.NewDynamicValue(
		cty.ObjectVal(map[string]cty.Value{
			"id":          cty.StringVal("b_id"),
			"test_string": cty.StringVal("a_id"),
		}), instanceSchema.ImpliedType())
	bAfter, _ := plans.NewDynamicValue(
		cty.ObjectVal(map[string]cty.Value{
			"id":          cty.StringVal("b_id"),
			"test_string": cty.StringVal("changed"),
		}), instanceSchema.ImpliedType())

	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.a"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Delete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.b"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
					Before: bBefore,
					After:  bAfter,
				},
			},
		},
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_object",
			Name: "a",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			Status:              states.ObjectReady,
			AttrsJSON:           []byte(`{"id":"a_id"}`),
			CreateBeforeDestroy: true,
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_object",
			Name: "b",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"b_id","test_string":"a_id"}`),
			Dependencies: []addrs.ConfigResource{
				{
					Resource: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_object",
						Name: "a",
					},
					Module: root.Addr.Module(),
				},
			},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	b := &ApplyGraphBuilder{
		Config:  testModule(t, "graph-builder-apply-orphan-update"),
		Changes: changes,
		Plugins: simpleMockPluginLibrary(),
		State:   state,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := strings.TrimSpace(`
test_object.a (destroy)
  test_object.b
test_object.b
`)

	instanceGraph := filterInstances(g)
	got := strings.TrimSpace(instanceGraph.String())

	if got != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, got)
	}
}

// The orphan clean up node should not be connected to a provider
func TestApplyGraphBuilder_orphanedWithProvider(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.A"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Delete,
				},
			},
		},
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.A").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"A"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"].foo`),
	)

	b := &ApplyGraphBuilder{
		Config:  testModule(t, "graph-builder-orphan-alias"),
		Changes: changes,
		Plugins: simpleMockPluginLibrary(),
		State:   state,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatal(err)
	}

	// The cleanup node has no state or config of its own, so would create a
	// default provider which we don't want.
	testGraphNotContains(t, g, "provider.test")
}

func TestApplyGraphBuilder_withChecks(t *testing.T) {
	awsProvider := mockProviderWithResourceTypeSchema("aws_instance", simpleTestSchema())

	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("aws_instance.foo"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Create,
				},
			},
			{
				Addr: mustResourceInstanceAddr("aws_instance.baz"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Create,
				},
			},
			{
				Addr: mustResourceInstanceAddr("data.aws_data_source.bar"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Read,
				},
				ActionReason: plans.ResourceInstanceReadBecauseCheckNested,
			},
		},
	}

	plugins := newContextPlugins(map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("aws"): providers.FactoryFixed(awsProvider),
	}, nil)

	b := &ApplyGraphBuilder{
		Config:    testModule(t, "apply-with-checks"),
		Changes:   changes,
		Plugins:   plugins,
		State:     states.NewState(),
		Operation: walkApply,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong path %q", g.Path.String())
	}

	got := strings.TrimSpace(g.String())
	// We're especially looking for the edge here, where aws_instance.bat
	// has a dependency on aws_instance.boo
	want := strings.TrimSpace(testPlanWithCheckGraphBuilderStr)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("\ngot:\n%s\n\nwant:\n%s\n\ndiff:\n%s", got, want, diff)
	}

}

const testPlanWithCheckGraphBuilderStr = `
(execute checks)
  aws_instance.baz
aws_instance.baz
  aws_instance.baz (expand)
aws_instance.baz (expand)
  aws_instance.foo
aws_instance.foo
  aws_instance.foo (expand)
aws_instance.foo (expand)
  provider["registry.terraform.io/hashicorp/aws"]
check.my_check (expand)
  data.aws_data_source.bar
data.aws_data_source.bar
  (execute checks)
  data.aws_data_source.bar (expand)
data.aws_data_source.bar (expand)
  provider["registry.terraform.io/hashicorp/aws"]
provider["registry.terraform.io/hashicorp/aws"]
provider["registry.terraform.io/hashicorp/aws"] (close)
  data.aws_data_source.bar
root
  check.my_check (expand)
  provider["registry.terraform.io/hashicorp/aws"] (close)
`

const testApplyGraphBuilderStr = `
module.child (close)
  module.child.test_object.other
module.child (expand)
module.child.test_object.create
  module.child.test_object.create (expand)
module.child.test_object.create (expand)
  module.child (expand)
  provider["registry.terraform.io/hashicorp/test"]
module.child.test_object.other
  module.child.test_object.other (expand)
module.child.test_object.other (expand)
  module.child.test_object.create
provider["registry.terraform.io/hashicorp/test"]
provider["registry.terraform.io/hashicorp/test"] (close)
  module.child.test_object.other
  test_object.other
root
  module.child (close)
  provider["registry.terraform.io/hashicorp/test"] (close)
test_object.create
  test_object.create (expand)
test_object.create (expand)
  provider["registry.terraform.io/hashicorp/test"]
test_object.other
  test_object.other (expand)
test_object.other (expand)
  test_object.create
`

const testApplyGraphBuilderDestroyCountStr = `
provider["registry.terraform.io/hashicorp/test"]
provider["registry.terraform.io/hashicorp/test"] (close)
  test_object.B
root
  provider["registry.terraform.io/hashicorp/test"] (close)
test_object.A (expand)
  provider["registry.terraform.io/hashicorp/test"]
test_object.A[1] (destroy)
  provider["registry.terraform.io/hashicorp/test"]
test_object.B
  test_object.A[1] (destroy)
  test_object.B (expand)
test_object.B (expand)
  test_object.A (expand)
`
