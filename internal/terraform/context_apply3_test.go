// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Apply_included(t *testing.T) {
	// 	variable "foo" {}

	// resource "test_instance" "foo" {
	//     value = var.foo
	// }

	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "foo" {}

resource "aws_instance" "foo" {
    num = "2"
}

resource "aws_instance" "bar" {
    foo = "bar"
}
`})

	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	p.ApplyResourceChangeFn = testApplyFn
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})
	assertNoErrors(t, diags)
	rs := collectResourceNames(plan.Changes.Resources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.foo"}, "expected all 1 instances to be in the plan")

	rs = collectResourceNames(plan.DeferredResources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.bar"}, "excluded resources should have been deferred")

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance
	`)
}

func TestContext2Apply_includedCount(t *testing.T) {
	m := testModule(t, "apply-targeted-count")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	p.ApplyResourceChangeFn = testApplyFn
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})
	assertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo.2:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
	`)
}

func TestContext2Apply_includedCountIndex(t *testing.T) {
	m := testModule(t, "apply-targeted-count")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	p.ApplyResourceChangeFn = testApplyFn
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "foo", addrs.IntKey(1),
			),
		},
	})
	assertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
	`)
}

func TestContext2Apply_includedDestroy(t *testing.T) {
	m := testModule(t, "destroy-targeted")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	state.SetOutputValue(
		addrs.OutputValue{Name: "out"}.Absolute(addrs.RootModuleInstance),
		cty.StringVal("bar"), false,
	)

	child := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.b").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-bcd345"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	if diags := ctx.Validate(m, nil); diags.HasErrors() {
		t.Fatalf("validate errors: %s", diags.Err())
	}

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "a",
			),
		},
	})
	assertNoErrors(t, diags)

	state, diags = ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 0 {
		t.Fatalf("expected 0 resources, got: %#v", mod.Resources)
	}

	// the root output should not get removed; only the targeted resource.
	//
	// Note: earlier versions of this test expected 0 outputs, but it turns out
	// that was because Validate - not apply or destroy - removed the output
	// (which depends on the targeted resource) from state. That version of this
	// test did not match actual terraform behavior: the output remains in
	// state.
	//
	// The reason it remains in the state is that we prune out the root module
	// output values from the destroy graph as part of pruning out the "update"
	// nodes for the resources, because otherwise the root module output values
	// force the resources to stay in the graph and can therefore cause
	// unwanted dependency cycles.
	//
	// TODO: Future refactoring may enable us to remove the output from state in
	// this case, and that would be Just Fine - this test can be modified to
	// expect 0 outputs.
	// UPDATE: We no longer have to remove nodes when including, thus
	// no longer have to prune the output values from the destroy
	// graph, and the resources that stay in the graph while not being targeted
	//  would only end up being validated, not applied.
	// In this case, the output values can be appropriately destroyed from state.
	if len(state.RootOutputValues) != 0 {
		t.Fatalf("expected 0 output, got: %#v", state.RootOutputValues)
	}

	// the module instance should remain
	mod = state.Module(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resources, got: %#v", mod.Resources)
	}
}

func TestContext2Apply_includedDestroyCountDeps(t *testing.T) {
	m := testModule(t, "apply-destroy-targeted-count")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-bcd345"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"i-abc123"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("aws_instance.foo")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})
	assertNoErrors(t, diags)

	state, diags = ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `<no state>`)
}

// https://github.com/hashicorp/terraform/issues/4462
func TestContext2Apply_includedDestroyModule(t *testing.T) {
	m := testModule(t, "apply-targeted-module")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-bcd345"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc123"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	child := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-bcd345"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc123"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})
	assertNoErrors(t, diags)

	state, diags = ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = i-abc123
  provider = provider["registry.terraform.io/hashicorp/aws"]
aws_instance.foo:
  ID = i-bcd345
  provider = provider["registry.terraform.io/hashicorp/aws"]

module.child:
  aws_instance.bar:
    ID = i-abc123
    provider = provider["registry.terraform.io/hashicorp/aws"]
	`)
}

func TestContext2Apply_includedDestroyCountIndex(t *testing.T) {
	m := testModule(t, "apply-targeted-count")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	foo := &states.ResourceInstanceObjectSrc{
		Status:    states.ObjectReady,
		AttrsJSON: []byte(`{"id":"i-bcd345"}`),
	}
	bar := &states.ResourceInstanceObjectSrc{
		Status:    states.ObjectReady,
		AttrsJSON: []byte(`{"id":"i-abc123"}`),
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		foo,
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[1]").Resource,
		foo,
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[2]").Resource,
		foo,
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar[0]").Resource,
		bar,
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar[1]").Resource,
		bar,
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar[2]").Resource,
		bar,
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "foo", addrs.IntKey(2),
			),
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "bar", addrs.IntKey(1),
			),
		},
	})
	assertNoErrors(t, diags)

	state, diags = ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.bar.0:
  ID = i-abc123
  provider = provider["registry.terraform.io/hashicorp/aws"]
aws_instance.bar.2:
  ID = i-abc123
  provider = provider["registry.terraform.io/hashicorp/aws"]
aws_instance.foo.0:
  ID = i-bcd345
  provider = provider["registry.terraform.io/hashicorp/aws"]
aws_instance.foo.1:
  ID = i-bcd345
  provider = provider["registry.terraform.io/hashicorp/aws"]
	`)
}

func TestContext2Apply_includedModule(t *testing.T) {
	m := testModule(t, "apply-targeted-module")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	p.ApplyResourceChangeFn = testApplyFn
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child", addrs.NoKey),
		},
	})
	assertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.Module(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	if mod == nil {
		t.Fatalf("no child module found in the state!\n\n%#v", state)
	}
	if len(mod.Resources) != 2 {
		t.Fatalf("expected 2 resources, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
<no state>
module.child:
  aws_instance.bar:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    num = 2
    type = aws_instance
  aws_instance.foo:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    num = 2
    type = aws_instance
	`)
}

// GH-1858
func TestContext2Apply_includedModuleDep(t *testing.T) {
	m := testModule(t, "apply-targeted-module-dep")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	p.ApplyResourceChangeFn = testApplyFn
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf("Diff: %s", legacyDiffComparisonString(plan.Changes))
	}

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// The output from module.child is copied into aws_instance.foo.foo
	checkStateString(t, state, `
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance

  Dependencies:
    module.child.aws_instance.mod

module.child:
  aws_instance.mod:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    type = aws_instance
`)
}

// GH-10911 untargeted outputs should not be in the graph, and therefore
// not execute.
func TestContext2Apply_includedModuleUnrelatedOutputs(t *testing.T) {
	m := testModule(t, "apply-targeted-module-unrelated-outputs")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	p.ApplyResourceChangeFn = testApplyFn

	state := states.NewState()
	_ = state.EnsureModule(addrs.RootModuleInstance.Child("child2", addrs.NoKey))

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child2", addrs.NoKey),
		},
	})
	assertNoErrors(t, diags)

	s, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// - module.child1's instance_id output is dropped because we don't preserve
	//   non-root module outputs between runs (they can be recalculated from config)
	// - module.child2's instance_id is updated because its dependency is updated
	// - child2_id is updated because if its transitive dependency via module.child2
	checkStateString(t, s, `
<no state>
Outputs:

child2_id = foo

module.child2:
  aws_instance.foo:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    type = aws_instance
`)
}

func TestContext2Apply_includedModuleResource(t *testing.T) {
	m := testModule(t, "apply-targeted-module-resource")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	p.ApplyResourceChangeFn = testApplyFn
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})
	assertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.Module(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	if mod == nil || len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod)
	}

	checkStateString(t, state, `
<no state>
module.child:
  aws_instance.foo:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    num = 2
    type = aws_instance
	`)
}

func TestContext2Apply_includedResourceOrphanModule(t *testing.T) {
	m := testModule(t, "apply-targeted-resource-orphan-module")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	child := state.EnsureModule(addrs.RootModuleInstance.Child("parent", addrs.NoKey))
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		Included: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})
	assertNoErrors(t, diags)

	if _, diags := ctx.Apply(plan, m, nil); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}
}

func equalIgnoreOrder(t *testing.T, x, y []string, msg string) {
	t.Helper()
	less := func(a, b string) bool { return a < b }
	if diff := cmp.Diff(x, y, cmpopts.SortSlices(less)); diff != "" {
		t.Fatalf("%s\n%s", msg, diff)
	}
}
