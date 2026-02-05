package terraform

import (
	"maps"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/addrs"
	provider "github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var slicesOpt = cmpopts.SortSlices(func(a, b string) bool {
	return a < b
})

func TestContextApply_TargetInstance(t *testing.T) {
	testConfig := testModuleInline(t, map[string]string{
		"main.tf": `
	locals {
		numbers = {
			index = sum([resource.test_object.foo[0].test_number, resource.test_object.baz.test_number]) + sum(resource.test_object.ben[*].test_number)
			unused = 0
		}
	}
	resource "test_object" "foo" {
		count = 2
		test_string = join("_", ["foo", count.index])
		test_number = count.index
	}

	resource "test_object" "baz" {
		test_number = 0
	}		
	
	resource "test_object" "ben" {
		count = 2
		test_number = 0
	}		

	resource "test_object" "bar" {
		count = 2
    	test_string = resource.test_object.foo[local.numbers.index].test_string
	}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(
		t,
		&ContextOpts{
			Providers: map[addrs.Provider]provider.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		},
	)

	testTarget := mustResourceInstanceAddr("test_object.bar[0]")
	planOptions := DefaultPlanOpts
	planOptions.Targets = []addrs.Targetable{
		testTarget,
	}
	planOptions.SafeTargeting = true
	pr, diags := ctx.Plan(testConfig, states.NewState(), planOptions)
	tfdiags.AssertNoErrors(t, diags)

	rs := pr.Changes.Resources
	if len(rs) != 5 {
		for _, r := range rs {
			t.Logf("resource: %s\n", r.Addr)
		}
		t.Fatalf("expected 5 resources, got %d", len(rs))
	}

	state, diags := ctx.Apply(pr, testConfig, nil)
	tfdiags.AssertNoErrors(t, diags)

	res := state.RootModule().Resources
	instances := map[string]*states.ResourceInstance{}
	for _, r := range res {
		for key, inst := range r.Instances {
			instances[r.Addr.Instance(key).String()] = inst
		}
	}
	instancesStrs := []string{"test_object.foo[0]",
		"test_object.bar[0]", "test_object.baz",
		"test_object.ben[0]", "test_object.ben[1]"}

	if diff := cmp.Diff(instancesStrs, slices.Collect(maps.Keys(instances)), slicesOpt); diff != "" {
		t.Fatalf("unexpected resource instances (-want +got):\n%s", diff)
	}

}
func TestContextApply_MarkPrecision(t *testing.T) {
	testConfig := testModuleInline(t, map[string]string{
		"main.tf": `
	
	locals {
		age = 10
		str = "hello"
	}
	
	resource "test_object" "me" {
		test_string = "me"
		test_number = 10
	}
	
	resource "test_object" "foo" {
		count = 2
		test_string = join("_", ["foo", local.age])
		test_number = count.index + test_object.me.test_number // "test_object.me" should be included in bar's deps, but it is not.
	}
	
	resource "test_object" "bar" {
		count = 2
		test_string = join("_", ["foo", test_object.foo[0].test_string])
		test_number = count.index
	}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(
		t,
		&ContextOpts{
			Providers: map[addrs.Provider]provider.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		},
	)

	testTarget := mustResourceInstanceAddr("test_object.bar[0]")
	planOptions := DefaultPlanOpts
	planOptions.Targets = []addrs.Targetable{
		testTarget,
	}
	planOptions.SafeTargeting = true
	pr, diags := ctx.Plan(testConfig, states.NewState(), planOptions)
	tfdiags.AssertNoErrors(t, diags)

	rs := pr.Changes.Resources
	if len(rs) != 3 {
		for _, r := range rs {
			t.Logf("resource: %s\n", r.Addr)
		}
		t.Fatalf("expected 3 resources, got %d", len(rs))
	}

	state, diags := ctx.Apply(pr, testConfig, nil)
	tfdiags.AssertNoErrors(t, diags)

	res := state.RootModule().Resources
	instances := map[string]*states.ResourceInstance{}
	for _, r := range res {
		for key, inst := range r.Instances {
			instances[r.Addr.Instance(key).String()] = inst
		}
	}
	instancesStrs := []string{"test_object.foo[0]",
		"test_object.bar[0]", "test_object.me"}

	if diff := cmp.Diff(instancesStrs, slices.Collect(maps.Keys(instances)), slicesOpt); diff != "" {
		t.Fatalf("unexpected resource instances (-want +got):\n%s", diff)
	}
}

func TestContextApply_TargetModuleInstance(t *testing.T) {
	testConfig := testModuleInline(t, map[string]string{
		"main.tf": `
	module "child" {
		source = "./child"
		count = 2
		child_var = resource.test_object.parent[0].test_number
	}
	
	resource "test_object" "parent" {
		count = 2
		test_number = 0
	}	
`,
		"child/main.tf": `
		
	variable "child_var" {
		type = number
		default = 0
	}
		
	locals {
		index = {
			used = sum([resource.test_object.foo[0].test_number, resource.test_object.baz.test_number, var.child_var]) + sum(resource.test_object.ben[*].test_number)
			unused = [resource.test_object.baz.test_number, resource.test_object.foo[1].test_number]
		}
	}
	resource "test_object" "foo" {
		count = 2
		test_string = join("_", ["foo", count.index])
		test_number = count.index
	}

	resource "test_object" "baz" {
		test_number = 0
	}		

	resource "test_object" "ben" {
		count = 2
		test_number = 0
	}		

	resource "test_object" "bar" {
		count = 2
		test_string = resource.test_object.foo[local.index.used].test_string
	}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(
		t,
		&ContextOpts{
			Providers: map[addrs.Provider]provider.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		},
	)

	testTarget := mustResourceInstanceAddr("module.child[0].test_object.bar[0]")
	planOptions := DefaultPlanOpts
	planOptions.Targets = []addrs.Targetable{
		testTarget,
	}
	planOptions.SafeTargeting = true
	planOptions.DeferralAllowed = true
	pr, diags := ctx.Plan(testConfig, states.NewState(), planOptions)
	tfdiags.AssertNoErrors(t, diags)

	rs := pr.Changes.Resources
	if len(rs) != 6 {
		for _, r := range rs {
			t.Logf("resource: %s\n", r.Addr)
		}
		t.Fatalf("expected 6 resources, got %d", len(rs))
	}

	state, diags := ctx.Apply(pr, testConfig, nil)
	tfdiags.AssertNoErrors(t, diags)

	res := state.AllResourceInstanceObjectAddrs()
	instances := []string{}
	for _, r := range res {
		instances = append(instances, r.String())
	}
	instancesStrs := []string{"module.child[0].test_object.foo[0]", "test_object.parent[0]",
		"module.child[0].test_object.bar[0]", "module.child[0].test_object.baz",
		"module.child[0].test_object.ben[0]", "module.child[0].test_object.ben[1]"}

	if diff := cmp.Diff(instancesStrs, instances, slicesOpt); diff != "" {
		t.Fatalf("unexpected resource instances (-want +got):\n%s", diff)
	}

}
