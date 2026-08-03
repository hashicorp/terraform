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
)

// TODO:@austinvalle: This shows an example of all resource instances of a provider being excluded, thus leaving
// a hanging NodeApplyableProvider instance (and close node). The test doesn't show visible side-effects
// from this but it seems like undesired behavior that should be cleaned up in a full implementation.
func TestContext2Plan_exclude(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "aws_instance" "foo" {}

resource "aws_instance" "bar" {}
`,
	})
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Excludes: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "bar",
			),
		},
	})
	tfdiags.AssertNoErrors(t, diags)

	if len(plan.Changes.Resources) != 0 {
		t.Errorf("expected no changes, got: %d", len(plan.Changes.Resources))
	}
}

func TestContext2Plan_excludeCount(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "aws_instance" "foo" {
  count = 3
}

resource "aws_instance" "bar" {
  count = 3
}

resource "aws_instance" "baz" {
  count = 3
}`,
	})
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Excludes: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "baz", addrs.IntKey(1),
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
		case "aws_instance.bar[0]":
		case "aws_instance.bar[1]":
		case "aws_instance.bar[2]":
		case "aws_instance.baz[0]":
		case "aws_instance.baz[2]":
			continue
		default:
			t.Errorf("unexpected resource change in plan after exclusions: %s %s", res.Action, resAddr)
		}
	}
}

func TestContext2Plan_excludeForEach(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "aws_instance" "foo" {
  for_each = toset(["key1","key2","key3"])
}

resource "aws_instance" "bar" {
  for_each = toset(["key4","key5","key6"])
}

resource "aws_instance" "baz" {
  for_each = toset(["key7","key8","key9"])
}`,
	})
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Excludes: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "bar",
			),
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "baz", addrs.StringKey("key8"),
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
		case `aws_instance.foo["key1"]`:
		case `aws_instance.foo["key2"]`:
		case `aws_instance.foo["key3"]`:
		case `aws_instance.baz["key7"]`:
		case `aws_instance.baz["key9"]`:
			continue
		default:
			t.Errorf("unexpected resource change in plan after exclusions: %s %s", res.Action, resAddr)
		}
	}
}

func TestContext2Plan_excludeModule(t *testing.T) {
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
resource "aws_instance" "foo" {}
resource "aws_instance" "bar" {}
`,
	})
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		Excludes: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child1", addrs.NoKey),
			addrs.RootModuleInstance.Child("child2", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "aws_instance", "bar",
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
		case `module.child2.aws_instance.foo`:
		case `module.child3.aws_instance.foo`:
		case `module.child3.aws_instance.bar`:
			continue
		default:
			t.Errorf("unexpected resource change in plan after exclusions: %s %s", res.Action, resAddr)
		}
	}
}

func TestContext2Plan_excludeModuleExpansion(t *testing.T) {
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
resource "aws_instance" "foo" {}
resource "aws_instance" "bar" {}
resource "aws_instance" "baz" {
  count = 2
}
resource "aws_instance" "quux" {
  for_each = toset(["key1","key2"])
}
`,
	})
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
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
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(0)).ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "baz", addrs.IntKey(1),
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(0)).ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "quux", addrs.StringKey("key1"),
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(1)).Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(1)).Resource(
				addrs.ManagedResourceMode, "aws_instance", "bar",
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(1)).Resource(
				addrs.ManagedResourceMode, "aws_instance", "baz",
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(1)).Resource(
				addrs.ManagedResourceMode, "aws_instance", "quux",
			),
			addrs.RootModuleInstance.Child("child_count2", addrs.IntKey(2)).Resource(
				addrs.ManagedResourceMode, "aws_instance", "bar",
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key1")).Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key1")).ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "baz", addrs.IntKey(0),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key1")).ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "quux", addrs.StringKey("key1"),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).Resource(
				addrs.ManagedResourceMode, "aws_instance", "bar",
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "baz", addrs.IntKey(0),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "baz", addrs.IntKey(1),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "quux", addrs.StringKey("key1"),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key2")).ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "quux", addrs.StringKey("key2"),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key3")).Resource(
				addrs.ManagedResourceMode, "aws_instance", "bar",
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key3")).ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "baz", addrs.IntKey(1),
			),
			addrs.RootModuleInstance.Child("child_foreach1", addrs.StringKey("key3")).ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "quux", addrs.StringKey("key2"),
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
		case `module.child_count2[0].aws_instance.bar`:
		case `module.child_count2[0].aws_instance.baz[0]`:
		case `module.child_count2[0].aws_instance.quux["key2"]`:
		case `module.child_count2[2].aws_instance.foo`:
		case `module.child_count2[2].aws_instance.baz[0]`:
		case `module.child_count2[2].aws_instance.baz[1]`:
		case `module.child_count2[2].aws_instance.quux["key1"]`:
		case `module.child_count2[2].aws_instance.quux["key2"]`:
		case `module.child_foreach1["key1"].aws_instance.bar`:
		case `module.child_foreach1["key1"].aws_instance.baz[1]`:
		case `module.child_foreach1["key1"].aws_instance.quux["key2"]`:
		case `module.child_foreach1["key3"].aws_instance.foo`:
		case `module.child_foreach1["key3"].aws_instance.baz[0]`:
		case `module.child_foreach1["key3"].aws_instance.quux["key1"]`:
			continue
		default:
			t.Errorf("unexpected resource change in plan after exclusions: %s %s", res.Action, resAddr)
		}
	}
}
