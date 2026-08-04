// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// TODO:@austinvalle: This shows an example of all resource instances of a provider being excluded, thus leaving
// a hanging NodeApplyableProvider instance (and close node). The test doesn't show visible side-effects
// from this but it seems like undesired behavior that should be cleaned up in a full implementation.
func TestContext2Plan_exclude(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "foo" {}

resource "test_instance" "bar" {}
`,
	})
	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Excludes: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "test_instance", "foo",
			),
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "test_instance", "bar",
			),
		},
	})
	tfdiags.AssertNoErrors(t, diags)

	if len(plan.Changes.Resources) != 0 {
		t.Errorf("expected no changes, got: %d", len(plan.Changes.Resources))
	}
}

func TestContext2Plan_exclude_destroy(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "foo" {}

resource "test_instance" "bar" {}
`,
	})
	p := testProvider("test")
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_instance.foo"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"value1"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_instance.bar"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"value2"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
		Excludes: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "test_instance", "bar",
			),
		},
	})
	tfdiags.AssertNoErrors(t, diags)

	if len(plan.Changes.Resources) != 1 {
		t.Errorf("expected 1 change, got: %d", len(plan.Changes.Resources))
	}
	for _, res := range plan.Changes.Resources {
		switch resAddr := res.Addr.String(); resAddr {
		// Expected resources after exclusions
		case `test_instance.foo`:
			continue
		default:
			t.Errorf("unexpected resource change in plan after exclusions: %s %s", res.Action, resAddr)
		}
	}
}

func TestContext2Plan_exclude_count(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "foo" {
  count = 3
}

resource "test_instance" "bar" {
  count = 3
}

resource "test_instance" "baz" {
  count = 3
}`,
	})
	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Excludes: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "test_instance", "foo",
			),
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "baz", addrs.IntKey(1),
			),
		},
	})
	tfdiags.AssertNoErrors(t, diags)

	if len(plan.Changes.Resources) != 5 {
		t.Errorf("expected 5 changes, got: %d", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		switch resAddr := res.Addr.String(); resAddr {
		// Expected resources after exclusions
		case "test_instance.bar[0]":
		case "test_instance.bar[1]":
		case "test_instance.bar[2]":
		case "test_instance.baz[0]":
		case "test_instance.baz[2]":
			continue
		default:
			t.Errorf("unexpected resource change in plan after exclusions: %s %s", res.Action, resAddr)
		}
	}
}

func TestContext2Plan_exclude_forEach(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "foo" {
  for_each = toset(["key1","key2","key3"])
}

resource "test_instance" "bar" {
  for_each = toset(["key4","key5","key6"])
}

resource "test_instance" "baz" {
  for_each = toset(["key7","key8","key9"])
}`,
	})
	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Excludes: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "test_instance", "bar",
			),
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "baz", addrs.StringKey("key8"),
			),
		},
	})
	tfdiags.AssertNoErrors(t, diags)

	if len(plan.Changes.Resources) != 5 {
		t.Errorf("expected 3 changes, got: %d", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		switch resAddr := res.Addr.String(); resAddr {
		// Expected resources after exclusions
		case `test_instance.foo["key1"]`:
		case `test_instance.foo["key2"]`:
		case `test_instance.foo["key3"]`:
		case `test_instance.baz["key7"]`:
		case `test_instance.baz["key9"]`:
			continue
		default:
			t.Errorf("unexpected resource change in plan after exclusions: %s %s", res.Action, resAddr)
		}
	}
}

func TestContext2Plan_exclude_module(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "child1" {
  source = "./child"
}
module "child2" {
  source = "./child"
}
module "child3" {
  source = "./child"
}
`,
		"child/main.tf": `
resource "test_instance" "foo" {}
resource "test_instance" "bar" {}
`,
	})
	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Excludes: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child1", addrs.NoKey),
			addrs.RootModuleInstance.Child("child2", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "test_instance", "bar",
			),
		},
	})
	tfdiags.AssertNoErrors(t, diags)

	if len(plan.Changes.Resources) != 3 {
		t.Errorf("expected 3 changes, got: %d", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		switch resAddr := res.Addr.String(); resAddr {
		// Expected resources after exclusions
		case `module.child2.test_instance.foo`:
		case `module.child3.test_instance.foo`:
		case `module.child3.test_instance.bar`:
			continue
		default:
			t.Errorf("unexpected resource change in plan after exclusions: %s %s", res.Action, resAddr)
		}
	}
}

func TestContext2Plan_exclude_moduleExpansion(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "child_count1" {
  source = "./child"
  count = 3
}
module "child_count2" {
  source = "./child"
  count = 3
}
module "child_foreach1" {
  source = "./child"
  for_each = toset(["key1","key2","key3"])
}
module "child_foreach2" {
  source = "./child"
  for_each = toset(["key3","key4","key5"])
}
`,
		"child/main.tf": `
resource "test_instance" "foo" {}
resource "test_instance" "bar" {}
resource "test_instance" "baz" {
  count = 2
}
resource "test_instance" "quux" {
  for_each = toset(["key1","key2"])
}
`,
	})
	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		// Excludes contains various permutations of excluding:
		//  - module instances
		//  - resources in module instances
		//  - expanded resource instances in module instances
		Excludes: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child_count1", addrs.NoKey),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(0)).Resource(
				addrs.ManagedResourceMode, "test_instance", "foo",
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(0)).ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "baz", addrs.IntKey(1),
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(0)).ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "quux", addrs.StringKey("key1"),
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(1)).Resource(
				addrs.ManagedResourceMode, "test_instance", "foo",
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(1)).Resource(
				addrs.ManagedResourceMode, "test_instance", "bar",
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(1)).Resource(
				addrs.ManagedResourceMode, "test_instance", "baz",
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(1)).Resource(
				addrs.ManagedResourceMode, "test_instance", "quux",
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(2)).Resource(
				addrs.ManagedResourceMode, "test_instance", "bar",
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key1")).Resource(
				addrs.ManagedResourceMode, "test_instance", "foo",
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key1")).ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "baz", addrs.IntKey(0),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key1")).ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "quux", addrs.StringKey("key1"),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).Resource(
				addrs.ManagedResourceMode, "test_instance", "foo",
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).Resource(
				addrs.ManagedResourceMode, "test_instance", "bar",
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "baz", addrs.IntKey(0),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "baz", addrs.IntKey(1),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "quux", addrs.StringKey("key1"),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "quux", addrs.StringKey("key2"),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key3")).Resource(
				addrs.ManagedResourceMode, "test_instance", "bar",
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key3")).ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "baz", addrs.IntKey(1),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key3")).ResourceInstance(
				addrs.ManagedResourceMode, "test_instance", "quux", addrs.StringKey("key2"),
			),
			addrs.RootModuleInstance.Child("child_foreach2", addrs.NoKey),
		},
	})
	tfdiags.AssertNoErrors(t, diags)

	if len(plan.Changes.Resources) != 14 {
		t.Errorf("expected 14 changes, got: %d", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		switch resAddr := res.Addr.String(); resAddr {
		// Expected resources after exclusions
		case `module.child_count2[0].test_instance.bar`:
		case `module.child_count2[0].test_instance.baz[0]`:
		case `module.child_count2[0].test_instance.quux["key2"]`:
		case `module.child_count2[2].test_instance.foo`:
		case `module.child_count2[2].test_instance.baz[0]`:
		case `module.child_count2[2].test_instance.baz[1]`:
		case `module.child_count2[2].test_instance.quux["key1"]`:
		case `module.child_count2[2].test_instance.quux["key2"]`:
		case `module.child_foreach1["key1"].test_instance.bar`:
		case `module.child_foreach1["key1"].test_instance.baz[1]`:
		case `module.child_foreach1["key1"].test_instance.quux["key2"]`:
		case `module.child_foreach1["key3"].test_instance.foo`:
		case `module.child_foreach1["key3"].test_instance.baz[0]`:
		case `module.child_foreach1["key3"].test_instance.quux["key1"]`:
			continue
		default:
			t.Errorf("unexpected resource change in plan after exclusions: %s %s", res.Action, resAddr)
		}
	}
}

func TestContext2Plan_exclude_dependencies(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "foo" {
  lifecycle {
    action_trigger {
	  events = [after_create]
	  actions = [action.test_action.do_the_foo]
	}
  }
}

resource "test_instance" "bar" {
  value = test_instance.foo.id
}

module "child" {
  source = "./child"
  count = 2
  input = test_instance.bar.id
}

action "test_action" "do_the_foo" {
  config {
    input = test_instance.foo.value
  }
}

output "foo_out" {
  value = test_instance.foo.id
}

output "quux_out" {
  value = module.child[0].quux_out
}
`,
		"child/main.tf": `
variable "input" {
  type = string
}
resource "test_instance" "baz" {
  value = var.input
  lifecycle {
    action_trigger {
	  events = [after_create]
	  actions = [action.test_action.do_the_baz]
	}
  }
}

ephemeral "test_data" "baz_eph" {
  value = "this ephemeral resource should be excluded!"
  depends_on = [test_instance.baz]
}

resource "test_instance" "quux" {
  count = 5
  depends_on = [test_instance.baz]
}

action "test_action" "do_the_baz" {
  config {
    input = test_instance.baz.value
  }
}

output "quux_out" {
  value = test_instance.quux.*
}
`,
	})
	p := testProvider("test")
	p.OpenEphemeralResourceResponse = &providers.OpenEphemeralResourceResponse{
		Result: cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal("this ephemeral resource should be excluded!"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Excludes: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "test_instance", "bar",
			),
		},
	})
	tfdiags.AssertNoErrors(t, diags)

	if len(plan.Changes.Resources) != 1 {
		t.Errorf("expected 1 resource change, got: %d", len(plan.Changes.Resources))
	}

	if len(plan.Changes.Outputs) != 1 {
		t.Errorf("expected 1 output change, got: %d", len(plan.Changes.Outputs))
	}

	if len(plan.Changes.ActionInvocations) != 1 {
		t.Errorf("expected 1 action invocation, got: %d", len(plan.Changes.ActionInvocations))
	}

	for _, res := range plan.Changes.Resources {
		switch resAddr := res.Addr.String(); resAddr {
		// Expected resources after exclusions
		case `test_instance.foo`:
			continue
		default:
			t.Errorf("unexpected resource change in plan after exclusions: %s %s", res.Action, resAddr)
		}
	}

	for _, out := range plan.Changes.Outputs {
		switch outAddr := out.Addr.String(); outAddr {
		// Expected outputs after exclusions
		case `output.foo_out`:
			continue
		default:
			t.Errorf("unexpected output change in plan after exclusions: %s %s", out.Action, outAddr)
		}
	}

	for _, action := range plan.Changes.ActionInvocations {
		switch actionAddr := action.Addr.String(); actionAddr {
		// Expected action invocations after exclusions
		case `action.test_action.do_the_foo`:
			continue
		default:
			t.Errorf("unexpected action invocation in plan after exclusions: %s", actionAddr)
		}
	}

	if p.OpenEphemeralResourceCalled {
		t.Error("unexpected call to OpenEphemeralResource: ephemeral resource in this test configuration should be excluded")
	}
}
