// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

func TestContext2Apply_deferSample(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
		resource "aws_instance" "first" {}
		resource "aws_instance" "foo" {
			depends_on = [aws_instance.first]
		}
		resource "aws_instance" "bar" {}
		resource "aws_instance" "baz" {
			for_each = toset(["a", "b", "c"])
			foo = each.key == "a" ? aws_instance.foo.id : each.key
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
		Defer: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "first",
			),
		},
	})
	assertNoErrors(t, diags)
	rs := collectResourceNames(plan.Changes.Resources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.bar"}, "expected all 1 instances to be in the plan")

	rs = collectResourceNames(plan.DeferredResources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.first",
		"aws_instance.foo",
		"aws_instance.baz[\"a\"]", "aws_instance.baz[\"b\"]",
		"aws_instance.baz[\"c\"]"}, "excluded resources should have been deferred")

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
	`)
}

func TestContext2Apply_defer(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
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
		Defer: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "bar",
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

func TestContext2Apply_deferInstance(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
		resource "aws_instance" "foo" {
			count = 2
			foo = "${count.index}"
		}

		resource "aws_instance" "bar" {
			foo = "bar"
		}

		// should be deferred
		resource "aws_instance" "baz" {
			foo = "${aws_instance.foo[0].foo}"
		}

		// should not be deferred (for now it is, because we only track the
		// dependency on the resource, not the instance) (TODO(sams))
		resource "aws_instance" "noz" {
			foo = "${aws_instance.foo[1].foo}"
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
		Defer: []addrs.Targetable{
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "foo", addrs.IntKey(0),
			),
		},
	})
	assertNoErrors(t, diags)
	rs := collectResourceNames(plan.Changes.Resources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.foo[1]", "aws_instance.bar"}, "expected all 3 instances to be in the plan")

	rs = collectResourceNames(plan.DeferredResources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.foo[0]", "aws_instance.baz", "aws_instance.noz"}, "excluded resources should have been deferred")

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 2 {
		t.Fatalf("expected 2 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = 1
  type = aws_instance
`)
}

func TestContext2Apply_deferInstance2(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
		resource "aws_instance" "foo" {
			count = 2
			foo = "${count.index}"
		}

		resource "aws_instance" "bar" {
			foo = "bar"
		}

		// should be deferred
		resource "aws_instance" "baz" {
			foo = "${aws_instance.foo[0].foo}"
		}

		// should be deferred
		resource "aws_instance" "noz" {
			foo = "${aws_instance.foo[1].foo}"
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
		Defer: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})
	assertNoErrors(t, diags)
	rs := collectResourceNames(plan.Changes.Resources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.bar"}, "unexpected resources in plan")

	rs = collectResourceNames(plan.DeferredResources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.foo[0]", "aws_instance.foo[1]", "aws_instance.baz", "aws_instance.noz"}, "excluded resources should have been deferred")

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
`)
}

func TestContext2Apply_deferInstanceForEach(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
		resource "aws_instance" "pre_foo" {
			count = 2
			foo = "${count.index}"
		}

		resource "aws_instance" "foo" {
			for_each = {
				for k, v in aws_instance.pre_foo : k => v.id
			}
			foo = each.value
		}

		resource "aws_instance" "bar" {
			foo = "bar"
		}

		// should be deferred
		resource "aws_instance" "baz" {
			foo = "${aws_instance.foo["0"].foo}"
		}

		// should be deferred
		resource "aws_instance" "noz" {
			foo = "${aws_instance.foo["1"].foo}"
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
		Defer: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})
	assertNoErrors(t, diags)
	rs := collectResourceNames(plan.Changes.Resources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.bar", "aws_instance.pre_foo[0]", "aws_instance.pre_foo[1]"}, "unexpected resources in plan")

	rs = collectResourceNames(plan.DeferredResources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.foo[\"0\"]", "aws_instance.foo[\"1\"]", "aws_instance.baz", "aws_instance.noz"}, "excluded resources should have been deferred")

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(state.AllResourceInstanceObjectAddrs()) != 3 {
		t.Fatalf("expected 3 resources, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.pre_foo.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = 0
  type = aws_instance
aws_instance.pre_foo.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = 1
  type = aws_instance
`)
}

func TestContext2Apply_deferNestedModule(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			module "nested" {
				source = "./nested"
			}

			// should be deferred because it depends on deferred module
			resource "aws_instance" "foo" {
				foo = module.nested.bar
			}

			// should NOT be deferred
			resource "aws_instance" "bar" {
				foo = "bar"
			}

			`,
		// Everything in the nested module is deferred
		"nested/main.tf": `
			resource "aws_instance" "bar" {
				foo = "bar"
			}

			output "bar" {
				value = aws_instance.bar.foo
			}
`,
	})

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
		Defer: []addrs.Targetable{
			addrs.RootModuleInstance.Child("nested", addrs.NoKey),
		},
	})
	assertNoErrors(t, diags)

	rs := collectResourceNames(plan.Changes.Resources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.bar"}, "expected all 1 instances to be in the plan")

	rs = collectResourceNames(plan.DeferredResources)
	equalIgnoreOrder(t, rs, []string{"module.nested.aws_instance.bar", "aws_instance.foo"}, "expected nested module resources to be deferred")

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
	`)
}

func TestContext2Apply_deferNestedModuleResource(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			module "nested" {
				source = "./nested"
			}

			// should be deferred because it depends on deferred resource
			// in nested module
			resource "aws_instance" "foo" {
				foo = module.nested.bar
			}

			// should NOT be deferred
			resource "aws_instance" "bar" {
				foo = module.nested.baz
			}

			`,
		"nested/main.tf": `
			resource "aws_instance" "bar" {
				foo = "bar"
			}

			output "bar" {
				value = aws_instance.bar.foo
			}

			output "baz" {
				value = "static"
			}
`,
	})

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
		Defer: []addrs.Targetable{
			addrs.RootModuleInstance.Child("nested", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "aws_instance", "bar",
			),
		},
	})
	assertNoErrors(t, diags)

	rs := collectResourceNames(plan.Changes.Resources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.bar"}, "expected all 1 instances to be in the plan")

	rs = collectResourceNames(plan.DeferredResources)
	equalIgnoreOrder(t, rs, []string{"module.nested.aws_instance.bar", "aws_instance.foo"}, "expected nested module resources to be deferred")

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = static
  type = aws_instance
	`)
}

func TestContext2Apply_deferNestedPartialOutput(t *testing.T) {
	t.Skip(`This test is currently failing because we don't store the state for
		deferred resources, so any object that references a deferred resource
		simply gets unknown. If the resource is partially known from the config,
		e.g foo = "bar", then we should be able to use that value in the output.
	`)
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			module "nested" {
				source = "./nested"
			}

			resource "aws_instance" "foo" {
				foo = module.nested.bar
			}

			`,
		"nested/main.tf": `
			resource "aws_instance" "bar" {
				foo = "bar"
			}

			output "bar" {
				value = {
					foo = aws_instance.bar.foo
					static = "static"
				}
			}
`,
	})

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
		Defer: []addrs.Targetable{
			addrs.RootModuleInstance.Child("nested", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "aws_instance", "bar",
			),
		},
	})
	assertNoErrors(t, diags)

	rs := collectResourceNames(plan.Changes.Resources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.bar"}, "expected all 1 instances to be in the plan")

	rs = collectResourceNames(plan.DeferredResources)
	equalIgnoreOrder(t, rs, []string{"module.nested.aws_instance.bar", "aws_instance.foo"}, "expected nested module resources to be deferred")

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = static
  type = aws_instance
	`)
}

func TestContext2Apply_deferResourceWithExternalVariable(t *testing.T) {
	t.Skip(`This test is currently failing because required root variables
		cannot be bypassed by deferring the resource. At the time that the
		variable is evaluated, it is not aware that it is only dependent on
		deferred resources. This is a limitation of the current implementation.
	`)
	m := testModuleInline(t, map[string]string{
		"main.tf": `
		variable "external_var" {}

		resource "aws_instance" "foo" {
			foo = var.external_var
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
		Defer: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})
	assertNoErrors(t, diags)
	rs := collectResourceNames(plan.Changes.Resources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.bar"}, "expected all 1 instances to be in the plan")

	rs = collectResourceNames(plan.DeferredResources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.foo"}, "excluded resources should have been deferred")

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
	`)
}

func TestContext2Apply_deferDependsOn(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "aws_instance" "foo" {
				foo = "foo"
			}

			resource "aws_instance" "bar" {
				foo = "bar"
				depends_on = [aws_instance.foo]
			}

			resource "aws_instance" "baz" {
				foo = "baz"
			}
		`,
	})

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
		Defer: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})
	assertNoErrors(t, diags)
	rs := collectResourceNames(plan.Changes.Resources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.baz"}, "unexpected resources in plan")

	rs = collectResourceNames(plan.DeferredResources)
	equalIgnoreOrder(t, rs, []string{"aws_instance.foo", "aws_instance.bar"}, "expected resources to be deferred")

	state, diags := ctx.Apply(plan, m, nil)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.baz:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = baz
  type = aws_instance
	`)
}

func equalIgnoreOrder(t *testing.T, x, y []string, msg string) {
	t.Helper()
	less := func(a, b string) bool { return a < b }
	if diff := cmp.Diff(x, y, cmpopts.SortSlices(less)); diff != "" {
		t.Fatalf("%s\n%s", msg, diff)
	}
}

func collectResourceNames(resources interface{}) []string {
	return slices.Collect(func(yield func(string) bool) {
		switch res := resources.(type) {
		case []*plans.ResourceInstanceChangeSrc:
			for _, ch := range res {
				str := ch.Addr.Resource.String()
				if ch.Addr.Module != nil {
					str = fmt.Sprintf("%s.%s", ch.Addr.Module.String(), str)
				}
				if !yield(str) {
					return
				}
			}
		case []*plans.DeferredResourceInstanceChangeSrc:
			for _, ch := range res {
				str := ch.ChangeSrc.Addr.Resource.String()
				if ch.ChangeSrc.Addr.Module != nil {
					str = fmt.Sprintf("%s.%s", ch.ChangeSrc.Addr.Module.String(), str)
				}
				if !yield(str) {
					return
				}
			}
		case map[string]*states.Resource:
			for _, ch := range res {
				for key := range ch.Instances {
					str := ch.Addr.Instance(key).Resource.String()
					if ch.Addr.Module != nil {
						str = fmt.Sprintf("%s.%s", ch.Addr.Module.String(), str)
					}
					if !yield(str) {
						return
					}
				}
			}
		default:
			panic(fmt.Sprintf("unexpected type %T", resources))
		}
	})
}
