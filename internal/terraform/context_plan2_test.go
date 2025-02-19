// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContext2Plan_removedDuringRefresh(t *testing.T) {
	// This tests the situation where an object tracked in the previous run
	// state has been deleted outside of Terraform, which we should detect
	// during the refresh step and thus ultimately produce a plan to recreate
	// the object, since it's still present in the configuration.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
}
`,
	})

	p := simpleMockProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{Block: simpleTestSchema()},
		ResourceTypes: map[string]providers.Schema{
			"test_object": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"arg": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
		resp.NewState = cty.NullVal(req.PriorState.Type())
		return resp
	}
	p.UpgradeResourceStateFn = func(req providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
		// We should've been given the prior state JSON as our input to upgrade.
		if !bytes.Contains(req.RawStateJSON, []byte("previous_run")) {
			t.Fatalf("UpgradeResourceState request doesn't contain the previous run object\n%s", req.RawStateJSON)
		}

		// We'll put something different in "arg" as part of upgrading, just
		// so that we can verify below that PrevRunState contains the upgraded
		// (but NOT refreshed) version of the object.
		resp.UpgradedState = cty.ObjectVal(map[string]cty.Value{
			"arg": cty.StringVal("upgraded"),
		})
		return resp
	}

	addr := mustResourceInstanceAddr("test_object.a")
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addr, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"arg":"previous_run"}`),
			Status:    states.ObjectTainted,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)

	if !p.UpgradeResourceStateCalled {
		t.Errorf("Provider's UpgradeResourceState wasn't called; should've been")
	}
	if !p.ReadResourceCalled {
		t.Errorf("Provider's ReadResource wasn't called; should've been")
	}

	// The object should be absent from the plan's prior state, because that
	// records the result of refreshing.
	if got := plan.PriorState.ResourceInstance(addr); got != nil {
		t.Errorf(
			"instance %s is in the prior state after planning; should've been removed\n%s",
			addr, spew.Sdump(got),
		)
	}

	// However, the object should still be in the PrevRunState, because
	// that reflects what we believed to exist before refreshing.
	if got := plan.PrevRunState.ResourceInstance(addr); got == nil {
		t.Errorf(
			"instance %s is missing from the previous run state after planning; should've been preserved",
			addr,
		)
	} else {
		if !bytes.Contains(got.Current.AttrsJSON, []byte("upgraded")) {
			t.Fatalf("previous run state has non-upgraded object\n%s", got.Current.AttrsJSON)
		}
	}

	// This situation should result in a drifted resource change.
	var drifted *plans.ResourceInstanceChangeSrc
	for _, dr := range plan.DriftedResources {
		if dr.Addr.Equal(addr) {
			drifted = dr
			break
		}
	}

	if drifted == nil {
		t.Errorf("instance %s is missing from the drifted resource changes", addr)
	} else {
		if got, want := drifted.Action, plans.Delete; got != want {
			t.Errorf("unexpected instance %s drifted resource change action. got: %s, want: %s", addr, got, want)
		}
	}

	// Because the configuration still mentions test_object.a, we should've
	// planned to recreate it in order to fix the drift.
	for _, c := range plan.Changes.Resources {
		if c.Action != plans.Create {
			t.Fatalf("expected Create action for missing %s, got %s", c.Addr, c.Action)
		}
	}
}

func TestContext2Plan_noChangeDataSourceSensitiveNestedSet(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "bar" {
  sensitive = true
  default   = "baz"
}

data "test_data_source" "foo" {
  foo {
    bar = var.bar
  }
}
`,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		DataSources: map[string]*configschema.Block{
			"test_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {Type: cty.String, Optional: true},
							},
						},
						Nesting: configschema.NestingSet,
					},
				},
			},
		},
	})

	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"id":  cty.StringVal("data_id"),
			"foo": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{"bar": cty.StringVal("baz")})}),
		}),
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("data.test_data_source.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"data_id", "foo":[{"bar":"baz"}]}`),
			AttrSensitivePaths: []cty.Path{
				cty.GetAttrPath("foo"),
			},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables)))
	assertNoErrors(t, diags)

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.NoOp {
			t.Fatalf("expected NoOp, got: %q %s", res.Addr, res.Action)
		}
	}
}

func TestContext2Plan_orphanDataInstance(t *testing.T) {
	// ensure the planned replacement of the data source is evaluated properly
	m := testModuleInline(t, map[string]string{
		"main.tf": `
data "test_object" "a" {
  for_each = { new = "ok" }
}

output "out" {
  value = [ for k, _ in data.test_object.a: k ]
}
`,
	})

	p := simpleMockProvider()
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
		resp.State = req.Config
		return resp
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(mustResourceInstanceAddr(`data.test_object.a["old"]`), &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"test_string":"foo"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)

	change, err := plan.Changes.Outputs[0].Decode()
	if err != nil {
		t.Fatal(err)
	}

	expected := cty.TupleVal([]cty.Value{cty.StringVal("new")})

	if change.After.Equals(expected).False() {
		t.Fatalf("expected %#v, got %#v\n", expected, change.After)
	}
}

func TestContext2Plan_basicConfigurationAliases(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
provider "test" {
  alias = "z"
  test_string = "config"
}

module "mod" {
  source = "./mod"
  providers = {
    test.x = test.z
  }
}
`,

		"mod/main.tf": `
terraform {
  required_providers {
    test = {
      source = "registry.terraform.io/hashicorp/test"
      configuration_aliases = [ test.x ]
	}
  }
}

resource "test_object" "a" {
  provider = test.x
}

`,
	})

	p := simpleMockProvider()

	// The resource within the module should be using the provider configured
	// from the root module. We should never see an empty configuration.
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		if req.Config.GetAttr("test_string").IsNull() {
			resp.Diagnostics = resp.Diagnostics.Append(errors.New("missing test_string value"))
		}
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	assertNoErrors(t, diags)
}

func TestContext2Plan_dataReferencesResourceInModules(t *testing.T) {
	p := testProvider("test")
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
		cfg := req.Config.AsValueMap()
		cfg["id"] = cty.StringVal("d")
		resp.State = cty.ObjectVal(cfg)
		return resp
	}

	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  things = {
    old = "first"
    new = "second"
  }
}

module "mod" {
  source = "./mod"
  for_each = local.things
}
`,

		"./mod/main.tf": `
resource "test_resource" "a" {
}

data "test_data_source" "d" {
  depends_on = [test_resource.a]
}

resource "test_resource" "b" {
  value = data.test_data_source.d.id
}
`})

	oldDataAddr := mustResourceInstanceAddr(`module.mod["old"].data.test_data_source.d`)

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr(`module.mod["old"].test_resource.a`),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"a"}`),
				Status:    states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr(`module.mod["old"].test_resource.b`),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"b","value":"d"}`),
				Status:    states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			oldDataAddr,
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"d"}`),
				Status:    states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)

	oldMod := oldDataAddr.Module

	for _, c := range plan.Changes.Resources {
		// there should be no changes from the old module instance
		if c.Addr.Module.Equal(oldMod) && c.Action != plans.NoOp {
			t.Errorf("unexpected change %s for %s\n", c.Action, c.Addr)
		}
	}
}

func TestContext2Plan_resourceChecksInExpandedModule(t *testing.T) {
	// When a resource is in a nested module we have two levels of expansion
	// to do: first expand the module the resource is declared in, and then
	// expand the resource itself.
	//
	// In earlier versions of Terraform we did that expansion as two levels
	// of DynamicExpand, which led to a bug where we didn't have any central
	// location from which to register all of the instances of a checkable
	// resource.
	//
	// We now handle the full expansion all in one graph node and one dynamic
	// subgraph, which avoids the problem. This is a regression test for the
	// earlier bug. If this test is panicking with "duplicate checkable objects
	// report" then that suggests the bug is reintroduced and we're now back
	// to reporting each module instance separately again, which is incorrect.

	p := testProvider("test")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{},
		},
		ResourceTypes: map[string]providers.Schema{
			"test": {
				Block: &configschema.Block{},
			},
		},
	}
	p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
		resp.NewState = req.PriorState
		return resp
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp.PlannedState = cty.EmptyObjectVal
		return resp
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		resp.NewState = req.PlannedState
		return resp
	}

	m := testModuleInline(t, map[string]string{
		"main.tf": `
			module "child" {
				source = "./child"
				count = 2 # must be at least 2 for this test to be valid
			}
		`,
		"child/child.tf": `
			locals {
				a = "a"
			}

			resource "test" "test1" {
				lifecycle {
					postcondition {
						# It doesn't matter what this checks as long as it
						# passes, because if we don't handle expansion properly
						# then we'll crash before we even get to evaluating this.
						condition = local.a == local.a
						error_message = "Postcondition failed."
					}
				}
			}

			resource "test" "test2" {
				count = 2

				lifecycle {
					postcondition {
						# It doesn't matter what this checks as long as it
						# passes, because if we don't handle expansion properly
						# then we'll crash before we even get to evaluating this.
						condition = local.a == local.a
						error_message = "Postcondition failed."
					}
				}
			}
		`,
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	priorState := states.NewState()
	plan, diags := ctx.Plan(m, priorState, DefaultPlanOpts)
	assertNoErrors(t, diags)

	resourceInsts := []addrs.AbsResourceInstance{
		mustResourceInstanceAddr("module.child[0].test.test1"),
		mustResourceInstanceAddr("module.child[0].test.test2[0]"),
		mustResourceInstanceAddr("module.child[0].test.test2[1]"),
		mustResourceInstanceAddr("module.child[1].test.test1"),
		mustResourceInstanceAddr("module.child[1].test.test2[0]"),
		mustResourceInstanceAddr("module.child[1].test.test2[1]"),
	}

	for _, instAddr := range resourceInsts {
		t.Run(fmt.Sprintf("results for %s", instAddr), func(t *testing.T) {
			if rc := plan.Changes.ResourceInstance(instAddr); rc != nil {
				if got, want := rc.Action, plans.Create; got != want {
					t.Errorf("wrong action for %s\ngot:  %s\nwant: %s", instAddr, got, want)
				}
				if got, want := rc.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
					t.Errorf("wrong action reason for %s\ngot:  %s\nwant: %s", instAddr, got, want)
				}
			} else {
				t.Errorf("no planned change for %s", instAddr)
			}

			if checkResult := plan.Checks.GetObjectResult(instAddr); checkResult != nil {
				if got, want := checkResult.Status, checks.StatusPass; got != want {
					t.Errorf("wrong check status for %s\ngot:  %s\nwant: %s", instAddr, got, want)
				}
			} else {
				t.Errorf("no check result for %s", instAddr)
			}
		})
	}
}

func TestContext2Plan_dataResourceChecksManagedResourceChange(t *testing.T) {
	// This tests the situation where the remote system contains data that
	// isn't valid per a data resource postcondition, but that the
	// configuration is destined to make the remote system valid during apply
	// and so we must defer reading the data resource and checking its
	// conditions until the apply step.
	//
	// This is an exception to the rule tested in
	// TestContext2Plan_dataReferencesResourceIndirectly which is relevant
	// whenever there's at least one precondition or postcondition attached
	// to a data resource.
	//
	// See TestContext2Plan_managedResourceChecksOtherManagedResourceChange for
	// an incorrect situation where a data resource is used only indirectly
	// to drive a precondition elsewhere, which therefore doesn't achieve this
	// special exception.

	p := testProvider("test")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_resource": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
						"valid": {
							Type:     cty.Bool,
							Required: true,
						},
					},
				},
			},
		},
		DataSources: map[string]providers.Schema{
			"test_data_source": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
						"valid": {
							Type:     cty.Bool,
							Computed: true,
						},
					},
				},
			},
		},
	}
	var mu sync.Mutex
	validVal := cty.False
	p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
		// NOTE: This assumes that the prior state declared below will have
		// "valid" set to false already, and thus will match validVal above.
		resp.NewState = req.PriorState
		return resp
	}
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
		cfg := req.Config.AsValueMap()
		mu.Lock()
		cfg["valid"] = validVal
		mu.Unlock()
		resp.State = cty.ObjectVal(cfg)
		return resp
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		cfg := req.Config.AsValueMap()
		prior := req.PriorState.AsValueMap()
		resp.PlannedState = cty.ObjectVal(map[string]cty.Value{
			"id":    prior["id"],
			"valid": cfg["valid"],
		})
		return resp
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		planned := req.PlannedState.AsValueMap()

		mu.Lock()
		validVal = planned["valid"]
		mu.Unlock()

		resp.NewState = req.PlannedState
		return resp
	}

	m := testModuleInline(t, map[string]string{
		"main.tf": `

resource "test_resource" "a" {
	valid = true
}

locals {
	# NOTE: We intentionally read through a local value here to make sure
	# that this behavior still works even if there isn't a direct dependency
	# between the data resource and the managed resource.
	object_id = test_resource.a.id
}

data "test_data_source" "a" {
	id = local.object_id

	lifecycle {
		postcondition {
			condition     = self.valid
			error_message = "Not valid!"
		}
	}
}
`})

	managedAddr := mustResourceInstanceAddr(`test_resource.a`)
	dataAddr := mustResourceInstanceAddr(`data.test_data_source.a`)

	// This state is intended to represent the outcome of a previous apply that
	// failed due to postcondition failure but had already updated the
	// relevant object to be invalid.
	//
	// It could also potentially represent a similar situation where the
	// previous apply succeeded but there has been a change outside of
	// Terraform that made it invalid, although technically in that scenario
	// the state data would become invalid only during the planning step. For
	// our purposes here that's close enough because we don't have a real
	// remote system in place anyway.
	priorState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			managedAddr,
			&states.ResourceInstanceObjectSrc{
				// NOTE: "valid" is false here but is true in the configuration
				// above, which is intended to represent that applying the
				// configuration change would make this object become valid.
				AttrsJSON: []byte(`{"id":"boop","valid":false}`),
				Status:    states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, priorState, DefaultPlanOpts)
	assertNoErrors(t, diags)

	if rc := plan.Changes.ResourceInstance(dataAddr); rc != nil {
		if got, want := rc.Action, plans.Read; got != want {
			t.Errorf("wrong action for %s\ngot:  %s\nwant: %s", dataAddr, got, want)
		}
		if got, want := rc.ActionReason, plans.ResourceInstanceReadBecauseDependencyPending; got != want {
			t.Errorf("wrong action reason for %s\ngot:  %s\nwant: %s", dataAddr, got, want)
		}
	} else {
		t.Fatalf("no planned change for %s", dataAddr)
	}

	if rc := plan.Changes.ResourceInstance(managedAddr); rc != nil {
		if got, want := rc.Action, plans.Update; got != want {
			t.Errorf("wrong action for %s\ngot:  %s\nwant: %s", managedAddr, got, want)
		}
		if got, want := rc.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason for %s\ngot:  %s\nwant: %s", managedAddr, got, want)
		}
	} else {
		t.Fatalf("no planned change for %s", managedAddr)
	}

	// This is primarily a plan-time test, since the special handling of
	// data resources is a plan-time concern, but we'll still try applying the
	// plan here just to make sure it's valid.
	newState, diags := ctx.Apply(plan, m, nil)
	assertNoErrors(t, diags)

	if rs := newState.ResourceInstance(dataAddr); rs != nil {
		if !rs.HasCurrent() {
			t.Errorf("no final state for %s", dataAddr)
		}
	} else {
		t.Errorf("no final state for %s", dataAddr)
	}

	if rs := newState.ResourceInstance(managedAddr); rs != nil {
		if !rs.HasCurrent() {
			t.Errorf("no final state for %s", managedAddr)
		}
	} else {
		t.Errorf("no final state for %s", managedAddr)
	}

	if got, want := validVal, cty.True; got != want {
		t.Errorf("wrong final valid value\ngot:  %#v\nwant: %#v", got, want)
	}

}

func TestContext2Plan_managedResourceChecksOtherManagedResourceChange(t *testing.T) {
	// This tests the incorrect situation where a managed resource checks
	// another managed resource indirectly via a data resource.
	// This doesn't work because Terraform can't tell that the data resource
	// outcome will be updated by a separate managed resource change and so
	// we expect it to fail.
	// This would ideally have worked except that we previously included a
	// special case in the rules for data resources where they only consider
	// direct dependencies when deciding whether to defer (except when the
	// data resource itself has conditions) and so they can potentially
	// read "too early" if the user creates the explicitly-not-recommended
	// situation of a data resource and a managed resource in the same
	// configuration both representing the same remote object.

	p := testProvider("test")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_resource": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
						"valid": {
							Type:     cty.Bool,
							Required: true,
						},
					},
				},
			},
		},
		DataSources: map[string]providers.Schema{
			"test_data_source": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
						"valid": {
							Type:     cty.Bool,
							Computed: true,
						},
					},
				},
			},
		},
	}
	var mu sync.Mutex
	validVal := cty.False
	p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
		// NOTE: This assumes that the prior state declared below will have
		// "valid" set to false already, and thus will match validVal above.
		resp.NewState = req.PriorState
		return resp
	}
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
		cfg := req.Config.AsValueMap()
		if cfg["id"].AsString() == "main" {
			mu.Lock()
			cfg["valid"] = validVal
			mu.Unlock()
		}
		resp.State = cty.ObjectVal(cfg)
		return resp
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		cfg := req.Config.AsValueMap()
		prior := req.PriorState.AsValueMap()
		resp.PlannedState = cty.ObjectVal(map[string]cty.Value{
			"id":    prior["id"],
			"valid": cfg["valid"],
		})
		return resp
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		planned := req.PlannedState.AsValueMap()

		if planned["id"].AsString() == "main" {
			mu.Lock()
			validVal = planned["valid"]
			mu.Unlock()
		}

		resp.NewState = req.PlannedState
		return resp
	}

	m := testModuleInline(t, map[string]string{
		"main.tf": `

resource "test_resource" "a" {
  valid = true
}

locals {
	# NOTE: We intentionally read through a local value here because a
	# direct reference from data.test_data_source.a to test_resource.a would
	# cause Terraform to defer the data resource to the apply phase due to
	# there being a pending change for the managed resource. We're explicitly
	# testing the failure case where the data resource read happens too
	# eagerly, which is what results from the reference being only indirect
	# so Terraform can't "see" that the data resource result might be affected
	# by changes to the managed resource.
	object_id = test_resource.a.id
}

data "test_data_source" "a" {
	id = local.object_id
}

resource "test_resource" "b" {
	valid = true

	lifecycle {
		precondition {
			condition     = data.test_data_source.a.valid
			error_message = "Not valid!"
		}
	}
}
`})

	managedAddrA := mustResourceInstanceAddr(`test_resource.a`)
	managedAddrB := mustResourceInstanceAddr(`test_resource.b`)

	// This state is intended to represent the outcome of a previous apply that
	// failed due to postcondition failure but had already updated the
	// relevant object to be invalid.
	//
	// It could also potentially represent a similar situation where the
	// previous apply succeeded but there has been a change outside of
	// Terraform that made it invalid, although technically in that scenario
	// the state data would become invalid only during the planning step. For
	// our purposes here that's close enough because we don't have a real
	// remote system in place anyway.
	priorState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			managedAddrA,
			&states.ResourceInstanceObjectSrc{
				// NOTE: "valid" is false here but is true in the configuration
				// above, which is intended to represent that applying the
				// configuration change would make this object become valid.
				AttrsJSON: []byte(`{"id":"main","valid":false}`),
				Status:    states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			managedAddrB,
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"checker","valid":true}`),
				Status:    states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, priorState, DefaultPlanOpts)
	if !diags.HasErrors() {
		t.Fatalf("unexpected successful plan; should've failed with non-passing precondition")
	}

	if got, want := diags.Err().Error(), "Resource precondition failed: Not valid!"; !strings.Contains(got, want) {
		t.Errorf("Missing expected error message\ngot: %s\nwant substring: %s", got, want)
	}
}

func TestContext2Plan_destroyWithRefresh(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
}
`,
	})

	p := simpleMockProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{Block: simpleTestSchema()},
		ResourceTypes: map[string]providers.Schema{
			"test_object": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"arg": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	// This is called from the first instance of this provider, so we can't
	// check p.ReadResourceCalled after plan.
	readResourceCalled := false
	p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
		readResourceCalled = true
		newVal, err := cty.Transform(req.PriorState, func(path cty.Path, v cty.Value) (cty.Value, error) {
			if len(path) == 1 && path[0] == (cty.GetAttrStep{Name: "arg"}) {
				return cty.StringVal("current"), nil
			}
			return v, nil
		})
		if err != nil {
			// shouldn't get here
			t.Fatalf("ReadResourceFn transform failed")
			return providers.ReadResourceResponse{}
		}
		return providers.ReadResourceResponse{
			NewState: newVal,
		}
	}

	upgradeResourceStateCalled := false
	p.UpgradeResourceStateFn = func(req providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
		upgradeResourceStateCalled = true
		t.Logf("UpgradeResourceState %s", req.RawStateJSON)

		// In the destroy-with-refresh codepath we end up calling
		// UpgradeResourceState twice, because we do so once during refreshing
		// (as part making a normal plan) and then again during the plan-destroy
		// walk. The second call recieves the result of the earlier refresh,
		// so we need to tolerate both "before" and "current" as possible
		// inputs here.
		if !bytes.Contains(req.RawStateJSON, []byte("before")) {
			if !bytes.Contains(req.RawStateJSON, []byte("current")) {
				t.Fatalf("UpgradeResourceState request doesn't contain the 'before' object or the 'current' object\n%s", req.RawStateJSON)
			}
		}

		// We'll put something different in "arg" as part of upgrading, just
		// so that we can verify below that PrevRunState contains the upgraded
		// (but NOT refreshed) version of the object.
		resp.UpgradedState = cty.ObjectVal(map[string]cty.Value{
			"arg": cty.StringVal("upgraded"),
		})
		return resp
	}

	addr := mustResourceInstanceAddr("test_object.a")
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addr, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"arg":"before"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:        plans.DestroyMode,
		SkipRefresh: false, // the default
	})
	assertNoErrors(t, diags)

	if !upgradeResourceStateCalled {
		t.Errorf("Provider's UpgradeResourceState wasn't called; should've been")
	}
	if !readResourceCalled {
		t.Errorf("Provider's ReadResource wasn't called; should've been")
	}

	if plan.PriorState == nil {
		t.Fatal("missing plan state")
	}

	for _, c := range plan.Changes.Resources {
		if c.Action != plans.Delete {
			t.Errorf("unexpected %s change for %s", c.Action, c.Addr)
		}
	}

	if instState := plan.PrevRunState.ResourceInstance(addr); instState == nil {
		t.Errorf("%s has no previous run state at all after plan", addr)
	} else {
		if instState.Current == nil {
			t.Errorf("%s has no current object in the previous run state", addr)
		} else if got, want := instState.Current.AttrsJSON, `"upgraded"`; !bytes.Contains(got, []byte(want)) {
			t.Errorf("%s has wrong previous run state after plan\ngot:\n%s\n\nwant substring: %s", addr, got, want)
		}
	}
	if instState := plan.PriorState.ResourceInstance(addr); instState == nil {
		t.Errorf("%s has no prior state at all after plan", addr)
	} else {
		if instState.Current == nil {
			t.Errorf("%s has no current object in the prior state", addr)
		} else if got, want := instState.Current.AttrsJSON, `"current"`; !bytes.Contains(got, []byte(want)) {
			t.Errorf("%s has wrong prior state after plan\ngot:\n%s\n\nwant substring: %s", addr, got, want)
		}
	}
}

func TestContext2Plan_destroySkipRefresh(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
}
`,
	})

	p := simpleMockProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{Block: simpleTestSchema()},
		ResourceTypes: map[string]providers.Schema{
			"test_object": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"arg": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
		t.Helper()
		t.Errorf("unexpected call to ReadResource")
		resp.NewState = req.PriorState
		return resp
	}
	p.UpgradeResourceStateFn = func(req providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
		t.Logf("UpgradeResourceState %s", req.RawStateJSON)
		// We should've been given the prior state JSON as our input to upgrade.
		if !bytes.Contains(req.RawStateJSON, []byte("before")) {
			t.Fatalf("UpgradeResourceState request doesn't contain the 'before' object\n%s", req.RawStateJSON)
		}

		// We'll put something different in "arg" as part of upgrading, just
		// so that we can verify below that PrevRunState contains the upgraded
		// (but NOT refreshed) version of the object.
		resp.UpgradedState = cty.ObjectVal(map[string]cty.Value{
			"arg": cty.StringVal("upgraded"),
		})
		return resp
	}

	addr := mustResourceInstanceAddr("test_object.a")
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addr, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"arg":"before"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:        plans.DestroyMode,
		SkipRefresh: true,
	})
	assertNoErrors(t, diags)

	if !p.UpgradeResourceStateCalled {
		t.Errorf("Provider's UpgradeResourceState wasn't called; should've been")
	}
	if p.ReadResourceCalled {
		t.Errorf("Provider's ReadResource was called; shouldn't have been")
	}

	if plan.PriorState == nil {
		t.Fatal("missing plan state")
	}

	for _, c := range plan.Changes.Resources {
		if c.Action != plans.Delete {
			t.Errorf("unexpected %s change for %s", c.Action, c.Addr)
		}
	}

	if instState := plan.PrevRunState.ResourceInstance(addr); instState == nil {
		t.Errorf("%s has no previous run state at all after plan", addr)
	} else {
		if instState.Current == nil {
			t.Errorf("%s has no current object in the previous run state", addr)
		} else if got, want := instState.Current.AttrsJSON, `"upgraded"`; !bytes.Contains(got, []byte(want)) {
			t.Errorf("%s has wrong previous run state after plan\ngot:\n%s\n\nwant substring: %s", addr, got, want)
		}
	}
	if instState := plan.PriorState.ResourceInstance(addr); instState == nil {
		t.Errorf("%s has no prior state at all after plan", addr)
	} else {
		if instState.Current == nil {
			t.Errorf("%s has no current object in the prior state", addr)
		} else if got, want := instState.Current.AttrsJSON, `"upgraded"`; !bytes.Contains(got, []byte(want)) {
			// NOTE: The prior state should still have been _upgraded_, even
			// though we skipped running refresh after upgrading it.
			t.Errorf("%s has wrong prior state after plan\ngot:\n%s\n\nwant substring: %s", addr, got, want)
		}
	}
}

func TestContext2Plan_unmarkingSensitiveAttributeForOutput(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_resource" "foo" {
}

output "result" {
  value = nonsensitive(test_resource.foo.sensitive_attr)
}
`,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"sensitive_attr": {
						Type:      cty.String,
						Computed:  true,
						Sensitive: true,
					},
				},
			},
		},
	})

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.UnknownVal(cty.Object(map[string]cty.Type{
				"id":             cty.String,
				"sensitive_attr": cty.String,
			})),
		}
	}

	state := states.NewState()

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected create, got: %q %s", res.Addr, res.Action)
		}
	}
}

func TestContext2Plan_destroyNoProviderConfig(t *testing.T) {
	// providers do not need to be configured during a destroy plan
	p := simpleMockProvider()
	p.ValidateProviderConfigFn = func(req providers.ValidateProviderConfigRequest) (resp providers.ValidateProviderConfigResponse) {
		v := req.Config.GetAttr("test_string")
		if v.IsNull() || !v.IsKnown() || v.AsString() != "ok" {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("invalid provider configuration: %#v", req.Config))
		}
		return resp
	}

	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  value = "ok"
}

provider "test" {
  test_string = local.value
}
`,
	})

	addr := mustResourceInstanceAddr("test_object.a")
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addr, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"test_string":"foo"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
	})
	assertNoErrors(t, diags)
}

func TestContext2Plan_movedResourceBasic(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object.a")
	addrB := mustResourceInstanceAddr("test_object.b")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_object" "b" {
			}

			moved {
				from = test_object.a
				to   = test_object.b
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		// The prior state tracks test_object.a, which we should treat as
		// test_object.b because of the "moved" block in the config.
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addrA,
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addrA.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrA)
		if instPlan != nil {
			t.Fatalf("unexpected plan for %s; should've moved to %s", addrA, addrB)
		}
	})
	t.Run(addrB.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrB)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrB)
		}

		if got, want := instPlan.Addr, addrB; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrA; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_movedResourceMissingModule(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object.a")
	addrB := mustResourceInstanceAddr("module.gone.test_object.b")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			moved {
				from = test_object.a
				to   = module.gone.test_object.b
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		// The prior state tracks test_object.a, which we should treat as
		// test_object.b because of the "moved" block in the config.
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addrA,
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addrA.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrA)
		if instPlan != nil {
			t.Fatalf("unexpected plan for %s; should've moved to %s", addrA, addrB)
		}
	})
	t.Run(addrB.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrB)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrB)
		}

		if got, want := instPlan.Addr, addrB; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrA; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Delete; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceDeleteBecauseNoMoveTarget; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_movedResourceCollision(t *testing.T) {
	addrNoKey := mustResourceInstanceAddr("test_object.a")
	addrZeroKey := mustResourceInstanceAddr("test_object.a[0]")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_object" "a" {
				# No "count" set, so test_object.a[0] will want
				# to implicitly move to test_object.a, but will get
				# blocked by the existing object at that address.
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addrNoKey, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
		s.SetResourceInstanceCurrent(addrZeroKey, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	// We should have a warning, though! We'll lightly abuse the "for RPC"
	// feature of diagnostics to get some more-readily-comparable diagnostic
	// values.
	gotDiags := diags.ForRPC()
	wantDiags := tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Warning,
			"Unresolved resource instance address changes",
			`Terraform tried to adjust resource instance addresses in the prior state based on change information recorded in the configuration, but some adjustments did not succeed due to existing objects already at the intended addresses:
  - test_object.a[0] could not move to test_object.a

Terraform has planned to destroy these objects. If Terraform's proposed changes aren't appropriate, you must first resolve the conflicts using the "terraform state" subcommands and then create a new plan.`,
		),
	}.ForRPC()
	if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
		t.Errorf("wrong diagnostics\n%s", diff)
	}

	t.Run(addrNoKey.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrNoKey)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrNoKey)
		}

		if got, want := instPlan.Addr, addrNoKey; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrNoKey; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run(addrZeroKey.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrZeroKey)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrZeroKey)
		}

		if got, want := instPlan.Addr, addrZeroKey; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrZeroKey; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Delete; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceDeleteBecauseWrongRepetition; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_movedResourceCollisionDestroy(t *testing.T) {
	// This is like TestContext2Plan_movedResourceCollision but intended to
	// ensure we still produce the expected warning (and produce it only once)
	// when we're creating a destroy plan, rather than a normal plan.
	// (This case is interesting at the time of writing because we happen to
	// use a normal plan as a trick to refresh before creating a destroy plan.
	// This test will probably become uninteresting if a future change to
	// the destroy-time planning behavior handles refreshing in a different
	// way, which avoids this pre-processing step of running a normal plan
	// first.)

	addrNoKey := mustResourceInstanceAddr("test_object.a")
	addrZeroKey := mustResourceInstanceAddr("test_object.a[0]")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_object" "a" {
				# No "count" set, so test_object.a[0] will want
				# to implicitly move to test_object.a, but will get
				# blocked by the existing object at that address.
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addrNoKey, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
		s.SetResourceInstanceCurrent(addrZeroKey, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	// We should have a warning, though! We'll lightly abuse the "for RPC"
	// feature of diagnostics to get some more-readily-comparable diagnostic
	// values.
	gotDiags := diags.ForRPC()
	wantDiags := tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Warning,
			"Unresolved resource instance address changes",
			// NOTE: This message is _lightly_ confusing in the destroy case,
			// because it says "Terraform has planned to destroy these objects"
			// but this is a plan to destroy all objects, anyway. We expect the
			// conflict situation to be pretty rare though, and even rarer in
			// a "terraform destroy", so we'll just live with that for now
			// unless we see evidence that lots of folks are being confused by
			// it in practice.
			`Terraform tried to adjust resource instance addresses in the prior state based on change information recorded in the configuration, but some adjustments did not succeed due to existing objects already at the intended addresses:
  - test_object.a[0] could not move to test_object.a

Terraform has planned to destroy these objects. If Terraform's proposed changes aren't appropriate, you must first resolve the conflicts using the "terraform state" subcommands and then create a new plan.`,
		),
	}.ForRPC()
	if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
		// If we get here with a diff that makes it seem like the above warning
		// is being reported twice, the likely cause is not correctly handling
		// the warnings from the hidden normal plan we run as part of preparing
		// for a destroy plan, unless that strategy has changed in the meantime
		// since we originally wrote this test.
		t.Errorf("wrong diagnostics\n%s", diff)
	}

	t.Run(addrNoKey.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrNoKey)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrNoKey)
		}

		if got, want := instPlan.Addr, addrNoKey; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrNoKey; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Delete; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run(addrZeroKey.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrZeroKey)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrZeroKey)
		}

		if got, want := instPlan.Addr, addrZeroKey; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrZeroKey; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Delete; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_movedResourceUntargeted(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object.a")
	addrB := mustResourceInstanceAddr("test_object.b")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_object" "b" {
			}

			moved {
				from = test_object.a
				to   = test_object.b
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		// The prior state tracks test_object.a, which we should treat as
		// test_object.b because of the "moved" block in the config.
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	t.Run("without targeting instance A", func(t *testing.T) {
		_, diags := ctx.Plan(m, state, &PlanOpts{
			Mode: plans.NormalMode,
			Targets: []addrs.Targetable{
				// NOTE: addrA isn't included here, but it's pending move to addrB
				// and so this plan request is invalid.
				addrB,
			},
		})
		diags.Sort()

		// We're semi-abusing "ForRPC" here just to get diagnostics that are
		// more easily comparable than the various different diagnostics types
		// tfdiags uses internally. The RPC-friendly diagnostics are also
		// comparison-friendly, by discarding all of the dynamic type information.
		gotDiags := diags.ForRPC()
		wantDiags := tfdiags.Diagnostics{
			tfdiags.Sourceless(
				tfdiags.Warning,
				"Resource targeting is in effect",
				`You are creating a plan with the -target option, which means that the result of this plan may not represent all of the changes requested by the current configuration.

The -target option is not for routine use, and is provided only for exceptional situations such as recovering from errors or mistakes, or when Terraform specifically suggests to use it as part of an error message.`,
			),
			tfdiags.Sourceless(
				tfdiags.Error,
				"Moved resource instances excluded by targeting",
				`Resource instances in your current state have moved to new addresses in the latest configuration. Terraform must include those resource instances while planning in order to ensure a correct result, but your -target=... options do not fully cover all of those resource instances.

To create a valid plan, either remove your -target=... options altogether or add the following additional target options:
  -target="test_object.a"

Note that adding these options may include further additional resource instances in your plan, in order to respect object dependencies.`,
			),
		}.ForRPC()

		if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
			t.Errorf("wrong diagnostics\n%s", diff)
		}
	})
	t.Run("without targeting instance B", func(t *testing.T) {
		_, diags := ctx.Plan(m, state, &PlanOpts{
			Mode: plans.NormalMode,
			Targets: []addrs.Targetable{
				addrA,
				// NOTE: addrB isn't included here, but it's pending move from
				// addrA and so this plan request is invalid.
			},
		})
		diags.Sort()

		// We're semi-abusing "ForRPC" here just to get diagnostics that are
		// more easily comparable than the various different diagnostics types
		// tfdiags uses internally. The RPC-friendly diagnostics are also
		// comparison-friendly, by discarding all of the dynamic type information.
		gotDiags := diags.ForRPC()
		wantDiags := tfdiags.Diagnostics{
			tfdiags.Sourceless(
				tfdiags.Warning,
				"Resource targeting is in effect",
				`You are creating a plan with the -target option, which means that the result of this plan may not represent all of the changes requested by the current configuration.

The -target option is not for routine use, and is provided only for exceptional situations such as recovering from errors or mistakes, or when Terraform specifically suggests to use it as part of an error message.`,
			),
			tfdiags.Sourceless(
				tfdiags.Error,
				"Moved resource instances excluded by targeting",
				`Resource instances in your current state have moved to new addresses in the latest configuration. Terraform must include those resource instances while planning in order to ensure a correct result, but your -target=... options do not fully cover all of those resource instances.

To create a valid plan, either remove your -target=... options altogether or add the following additional target options:
  -target="test_object.b"

Note that adding these options may include further additional resource instances in your plan, in order to respect object dependencies.`,
			),
		}.ForRPC()

		if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
			t.Errorf("wrong diagnostics\n%s", diff)
		}
	})
	t.Run("without targeting either instance", func(t *testing.T) {
		_, diags := ctx.Plan(m, state, &PlanOpts{
			Mode: plans.NormalMode,
			Targets: []addrs.Targetable{
				mustResourceInstanceAddr("test_object.unrelated"),
				// NOTE: neither addrA nor addrB are included here, but there's
				// a pending move between them and so this is invalid.
			},
		})
		diags.Sort()

		// We're semi-abusing "ForRPC" here just to get diagnostics that are
		// more easily comparable than the various different diagnostics types
		// tfdiags uses internally. The RPC-friendly diagnostics are also
		// comparison-friendly, by discarding all of the dynamic type information.
		gotDiags := diags.ForRPC()
		wantDiags := tfdiags.Diagnostics{
			tfdiags.Sourceless(
				tfdiags.Warning,
				"Resource targeting is in effect",
				`You are creating a plan with the -target option, which means that the result of this plan may not represent all of the changes requested by the current configuration.

The -target option is not for routine use, and is provided only for exceptional situations such as recovering from errors or mistakes, or when Terraform specifically suggests to use it as part of an error message.`,
			),
			tfdiags.Sourceless(
				tfdiags.Error,
				"Moved resource instances excluded by targeting",
				`Resource instances in your current state have moved to new addresses in the latest configuration. Terraform must include those resource instances while planning in order to ensure a correct result, but your -target=... options do not fully cover all of those resource instances.

To create a valid plan, either remove your -target=... options altogether or add the following additional target options:
  -target="test_object.a"
  -target="test_object.b"

Note that adding these options may include further additional resource instances in your plan, in order to respect object dependencies.`,
			),
		}.ForRPC()

		if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
			t.Errorf("wrong diagnostics\n%s", diff)
		}
	})
	t.Run("with both addresses in the target set", func(t *testing.T) {
		// The error messages in the other subtests above suggest adding
		// addresses to the set of targets. This additional test makes sure that
		// following that advice actually leads to a valid result.

		_, diags := ctx.Plan(m, state, &PlanOpts{
			Mode: plans.NormalMode,
			Targets: []addrs.Targetable{
				// This time we're including both addresses in the target,
				// to get the same effect an end-user would get if following
				// the advice in our error message in the other subtests.
				addrA,
				addrB,
			},
		})
		diags.Sort()

		wantDiags := tfdiags.Diagnostics{
			// Still get the warning about the -target option...
			tfdiags.Sourceless(
				tfdiags.Warning,
				"Resource targeting is in effect",
				`You are creating a plan with the -target option, which means that the result of this plan may not represent all of the changes requested by the current configuration.

The -target option is not for routine use, and is provided only for exceptional situations such as recovering from errors or mistakes, or when Terraform specifically suggests to use it as part of an error message.`,
			),
			// ...but now we have no error about test_object.a
		}

		tfdiags.AssertDiagnosticsMatch(t, wantDiags, diags)
	})
}

func TestContext2Plan_crossResourceMoveBasic(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object_one.a")
	addrB := mustResourceInstanceAddr("test_object_two.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_object_two" "a" {
			}

			moved {
				from = test_object_one.a
				to   = test_object_two.a
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		// The prior state tracks test_object.a, which we should treat as
		// test_object.b because of the "moved" block in the config.
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"value":"before"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := &testing_provider.MockProvider{}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_object_one": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
			"test_object_two": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
		ServerCapabilities: providers.ServerCapabilities{
			MoveResourceState: true,
		},
	}
	p.MoveResourceStateResponse = &providers.MoveResourceStateResponse{
		TargetState: cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal("after"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addrA.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrA)
		if instPlan != nil {
			t.Fatalf("unexpected plan for %s; should've moved to %s", addrA, addrB)
		}
	})
	t.Run(addrB.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrB)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrB)
		}

		if got, want := instPlan.Addr, addrB; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrA; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Update; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_crossProviderMove(t *testing.T) {
	addrA := mustResourceInstanceAddr("one_object.a")
	addrB := mustResourceInstanceAddr("two_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "two_object" "a" {
			}

			moved {
				from = one_object.a
				to   = two_object.a
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		// The prior state tracks test_object.a, which we should treat as
		// test_object.b because of the "moved" block in the config.
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"value":"before"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/one"]`))
	})

	one := &testing_provider.MockProvider{}
	one.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"one_object": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	two := &testing_provider.MockProvider{}
	two.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"two_object": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
		ServerCapabilities: providers.ServerCapabilities{
			MoveResourceState: true,
		},
	}
	two.MoveResourceStateResponse = &providers.MoveResourceStateResponse{
		TargetState: cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal("after"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("one"): testProviderFuncFixed(one),
			addrs.NewDefaultProvider("two"): testProviderFuncFixed(two),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addrA.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrA)
		if instPlan != nil {
			t.Fatalf("unexpected plan for %s; should've moved to %s", addrA, addrB)
		}
	})
	t.Run(addrB.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrB)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrB)
		}

		if got, want := instPlan.Addr, addrB; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrA; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Update; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_crossResourceMoveMissingConfig(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object_one.a")
	addrB := mustResourceInstanceAddr("test_object_two.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			moved {
				from = test_object_one.a
				to   = test_object_two.a
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		// The prior state tracks test_object.a, which we should treat as
		// test_object.b because of the "moved" block in the config.
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"value":"before"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := &testing_provider.MockProvider{}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{},
		ResourceTypes: map[string]providers.Schema{
			"test_object_one": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
			"test_object_two": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}
	p.MoveResourceStateResponse = &providers.MoveResourceStateResponse{
		TargetState: cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal("after"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addrA.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrA)
		if instPlan != nil {
			t.Fatalf("unexpected plan for %s; should've moved to %s", addrA, addrB)
		}
	})
	t.Run(addrB.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrB)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrB)
		}

		if got, want := instPlan.Addr, addrB; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrA; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Delete; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceDeleteBecauseNoMoveTarget; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_untargetedResourceSchemaChange(t *testing.T) {
	// an untargeted resource which requires a schema migration should not
	// block planning due external changes in the plan.
	addrA := mustResourceInstanceAddr("test_object.a")
	addrB := mustResourceInstanceAddr("test_object.b")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
}
resource "test_object" "b" {
}`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
		s.SetResourceInstanceCurrent(addrB, &states.ResourceInstanceObjectSrc{
			// old_list is no longer in the schema
			AttrsJSON: []byte(`{"old_list":["used to be","a list here"]}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()

	// external changes trigger a "drift report", but because test_object.b was
	// not targeted, the state was not fixed to match the schema and cannot be
	// deocded for the report.
	p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
		obj := req.PriorState.AsValueMap()
		// test_number changed externally
		obj["test_number"] = cty.NumberIntVal(1)
		resp.NewState = cty.ObjectVal(obj)
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		Targets: []addrs.Targetable{
			addrA,
		},
	})
	//
	assertNoErrors(t, diags)
}

func TestContext2Plan_movedResourceRefreshOnly(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object.a")
	addrB := mustResourceInstanceAddr("test_object.b")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_object" "b" {
			}

			moved {
				from = test_object.a
				to   = test_object.b
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		// The prior state tracks test_object.a, which we should treat as
		// test_object.b because of the "moved" block in the config.
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.RefreshOnlyMode,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addrA.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrA)
		if instPlan != nil {
			t.Fatalf("unexpected plan for %s; should've moved to %s", addrA, addrB)
		}
	})
	t.Run(addrB.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrB)
		if instPlan != nil {
			t.Fatalf("unexpected plan for %s", addrB)
		}
	})
	t.Run("drift", func(t *testing.T) {
		var drifted *plans.ResourceInstanceChangeSrc
		for _, dr := range plan.DriftedResources {
			if dr.Addr.Equal(addrB) {
				drifted = dr
				break
			}
		}

		if drifted == nil {
			t.Fatalf("instance %s is missing from the drifted resource changes", addrB)
		}

		if got, want := drifted.PrevRunAddr, addrA; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := drifted.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_refreshOnlyMode(t *testing.T) {
	addr := mustResourceInstanceAddr("test_object.a")

	// The configuration, the prior state, and the refresh result intentionally
	// have different values for "test_string" so we can observe that the
	// refresh took effect but the configuration change wasn't considered.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_object" "a" {
				arg = "after"
			}

			output "out" {
				value = test_object.a.arg
			}
		`,
	})
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addr, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"arg":"before"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{Block: simpleTestSchema()},
		ResourceTypes: map[string]providers.Schema{
			"test_object": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"arg": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		newVal, err := cty.Transform(req.PriorState, func(path cty.Path, v cty.Value) (cty.Value, error) {
			if len(path) == 1 && path[0] == (cty.GetAttrStep{Name: "arg"}) {
				return cty.StringVal("current"), nil
			}
			return v, nil
		})
		if err != nil {
			// shouldn't get here
			t.Fatalf("ReadResourceFn transform failed")
			return providers.ReadResourceResponse{}
		}
		return providers.ReadResourceResponse{
			NewState: newVal,
		}
	}
	p.UpgradeResourceStateFn = func(req providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
		// We should've been given the prior state JSON as our input to upgrade.
		if !bytes.Contains(req.RawStateJSON, []byte("before")) {
			t.Fatalf("UpgradeResourceState request doesn't contain the 'before' object\n%s", req.RawStateJSON)
		}

		// We'll put something different in "arg" as part of upgrading, just
		// so that we can verify below that PrevRunState contains the upgraded
		// (but NOT refreshed) version of the object.
		resp.UpgradedState = cty.ObjectVal(map[string]cty.Value{
			"arg": cty.StringVal("upgraded"),
		})
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.RefreshOnlyMode,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}
	checkPlanCompleteAndApplyable(t, plan)

	if !p.UpgradeResourceStateCalled {
		t.Errorf("Provider's UpgradeResourceState wasn't called; should've been")
	}
	if !p.ReadResourceCalled {
		t.Errorf("Provider's ReadResource wasn't called; should've been")
	}

	if got, want := len(plan.Changes.Resources), 0; got != want {
		t.Errorf("plan contains resource changes; want none\n%s", spew.Sdump(plan.Changes.Resources))
	}

	if instState := plan.PriorState.ResourceInstance(addr); instState == nil {
		t.Errorf("%s has no prior state at all after plan", addr)
	} else {
		if instState.Current == nil {
			t.Errorf("%s has no current object after plan", addr)
		} else if got, want := instState.Current.AttrsJSON, `"current"`; !bytes.Contains(got, []byte(want)) {
			// Should've saved the result of refreshing
			t.Errorf("%s has wrong prior state after plan\ngot:\n%s\n\nwant substring: %s", addr, got, want)
		}
	}
	if instState := plan.PrevRunState.ResourceInstance(addr); instState == nil {
		t.Errorf("%s has no previous run state at all after plan", addr)
	} else {
		if instState.Current == nil {
			t.Errorf("%s has no current object in the previous run state", addr)
		} else if got, want := instState.Current.AttrsJSON, `"upgraded"`; !bytes.Contains(got, []byte(want)) {
			// Should've saved the result of upgrading
			t.Errorf("%s has wrong previous run state after plan\ngot:\n%s\n\nwant substring: %s", addr, got, want)
		}
	}

	// The output value should also have updated. If not, it's likely that we
	// skipped updating the working state to match the refreshed state when we
	// were evaluating the resource.
	if outChangeSrc := plan.Changes.OutputValue(addrs.RootModuleInstance.OutputValue("out")); outChangeSrc == nil {
		t.Errorf("no change planned for output value 'out'")
	} else {
		outChange, err := outChangeSrc.Decode()
		if err != nil {
			t.Fatalf("failed to decode output value 'out': %s", err)
		}
		got := outChange.After
		want := cty.StringVal("current")
		if !want.RawEquals(got) {
			t.Errorf("wrong value for output value 'out'\ngot:  %#v\nwant: %#v", got, want)
		}
	}
}

func TestContext2Plan_refreshOnlyMode_deposed(t *testing.T) {
	addr := mustResourceInstanceAddr("test_object.a")
	deposedKey := states.DeposedKey("byebye")

	// The configuration, the prior state, and the refresh result intentionally
	// have different values for "test_string" so we can observe that the
	// refresh took effect but the configuration change wasn't considered.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_object" "a" {
				arg = "after"
			}

			output "out" {
				value = test_object.a.arg
			}
		`,
	})
	state := states.BuildState(func(s *states.SyncState) {
		// Note that we're intentionally recording a _deposed_ object here,
		// and not including a current object, so a normal (non-refresh)
		// plan would normally plan to create a new object _and_ destroy
		// the deposed one, but refresh-only mode should prevent that.
		s.SetResourceInstanceDeposed(addr, deposedKey, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"arg":"before"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{Block: simpleTestSchema()},
		ResourceTypes: map[string]providers.Schema{
			"test_object": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"arg": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		newVal, err := cty.Transform(req.PriorState, func(path cty.Path, v cty.Value) (cty.Value, error) {
			if len(path) == 1 && path[0] == (cty.GetAttrStep{Name: "arg"}) {
				return cty.StringVal("current"), nil
			}
			return v, nil
		})
		if err != nil {
			// shouldn't get here
			t.Fatalf("ReadResourceFn transform failed")
			return providers.ReadResourceResponse{}
		}
		return providers.ReadResourceResponse{
			NewState: newVal,
		}
	}
	p.UpgradeResourceStateFn = func(req providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
		// We should've been given the prior state JSON as our input to upgrade.
		if !bytes.Contains(req.RawStateJSON, []byte("before")) {
			t.Fatalf("UpgradeResourceState request doesn't contain the 'before' object\n%s", req.RawStateJSON)
		}

		// We'll put something different in "arg" as part of upgrading, just
		// so that we can verify below that PrevRunState contains the upgraded
		// (but NOT refreshed) version of the object.
		resp.UpgradedState = cty.ObjectVal(map[string]cty.Value{
			"arg": cty.StringVal("upgraded"),
		})
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.RefreshOnlyMode,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	if !p.UpgradeResourceStateCalled {
		t.Errorf("Provider's UpgradeResourceState wasn't called; should've been")
	}
	if !p.ReadResourceCalled {
		t.Errorf("Provider's ReadResource wasn't called; should've been")
	}

	if got, want := len(plan.Changes.Resources), 0; got != want {
		t.Errorf("plan contains resource changes; want none\n%s", spew.Sdump(plan.Changes.Resources))
	}

	if instState := plan.PriorState.ResourceInstance(addr); instState == nil {
		t.Errorf("%s has no prior state at all after plan", addr)
	} else {
		if obj := instState.Deposed[deposedKey]; obj == nil {
			t.Errorf("%s has no deposed object after plan", addr)
		} else if got, want := obj.AttrsJSON, `"current"`; !bytes.Contains(got, []byte(want)) {
			// Should've saved the result of refreshing
			t.Errorf("%s has wrong prior state after plan\ngot:\n%s\n\nwant substring: %s", addr, got, want)
		}
	}
	if instState := plan.PrevRunState.ResourceInstance(addr); instState == nil {
		t.Errorf("%s has no previous run state at all after plan", addr)
	} else {
		if obj := instState.Deposed[deposedKey]; obj == nil {
			t.Errorf("%s has no deposed object in the previous run state", addr)
		} else if got, want := obj.AttrsJSON, `"upgraded"`; !bytes.Contains(got, []byte(want)) {
			// Should've saved the result of upgrading
			t.Errorf("%s has wrong previous run state after plan\ngot:\n%s\n\nwant substring: %s", addr, got, want)
		}
	}

	// The output value should also have updated. If not, it's likely that we
	// skipped updating the working state to match the refreshed state when we
	// were evaluating the resource.
	if outChangeSrc := plan.Changes.OutputValue(addrs.RootModuleInstance.OutputValue("out")); outChangeSrc == nil {
		t.Errorf("no change planned for output value 'out'")
	} else {
		outChange, err := outChangeSrc.Decode()
		if err != nil {
			t.Fatalf("failed to decode output value 'out': %s", err)
		}
		got := outChange.After
		want := cty.UnknownVal(cty.String)
		if !want.RawEquals(got) {
			t.Errorf("wrong value for output value 'out'\ngot:  %#v\nwant: %#v", got, want)
		}
	}

	// Deposed objects should not be represented in drift.
	if len(plan.DriftedResources) > 0 {
		t.Errorf("unexpected drifted resources (%d)", len(plan.DriftedResources))
	}
}

func TestContext2Plan_refreshOnlyMode_orphan(t *testing.T) {
	addr := mustAbsResourceAddr("test_object.a")

	// The configuration, the prior state, and the refresh result intentionally
	// have different values for "test_string" so we can observe that the
	// refresh took effect but the configuration change wasn't considered.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_object" "a" {
				arg = "after"
				count = 1
			}

			output "out" {
				value = test_object.a.*.arg
			}
		`,
	})
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addr.Instance(addrs.IntKey(0)), &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"arg":"before"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
		s.SetResourceInstanceCurrent(addr.Instance(addrs.IntKey(1)), &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"arg":"before"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{Block: simpleTestSchema()},
		ResourceTypes: map[string]providers.Schema{
			"test_object": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"arg": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		newVal, err := cty.Transform(req.PriorState, func(path cty.Path, v cty.Value) (cty.Value, error) {
			if len(path) == 1 && path[0] == (cty.GetAttrStep{Name: "arg"}) {
				return cty.StringVal("current"), nil
			}
			return v, nil
		})
		if err != nil {
			// shouldn't get here
			t.Fatalf("ReadResourceFn transform failed")
			return providers.ReadResourceResponse{}
		}
		return providers.ReadResourceResponse{
			NewState: newVal,
		}
	}
	p.UpgradeResourceStateFn = func(req providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
		// We should've been given the prior state JSON as our input to upgrade.
		if !bytes.Contains(req.RawStateJSON, []byte("before")) {
			t.Fatalf("UpgradeResourceState request doesn't contain the 'before' object\n%s", req.RawStateJSON)
		}

		// We'll put something different in "arg" as part of upgrading, just
		// so that we can verify below that PrevRunState contains the upgraded
		// (but NOT refreshed) version of the object.
		resp.UpgradedState = cty.ObjectVal(map[string]cty.Value{
			"arg": cty.StringVal("upgraded"),
		})
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.RefreshOnlyMode,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	if !p.UpgradeResourceStateCalled {
		t.Errorf("Provider's UpgradeResourceState wasn't called; should've been")
	}
	if !p.ReadResourceCalled {
		t.Errorf("Provider's ReadResource wasn't called; should've been")
	}

	if got, want := len(plan.Changes.Resources), 0; got != want {
		t.Errorf("plan contains resource changes; want none\n%s", spew.Sdump(plan.Changes.Resources))
	}

	if rState := plan.PriorState.Resource(addr); rState == nil {
		t.Errorf("%s has no prior state at all after plan", addr)
	} else {
		for i := 0; i < 2; i++ {
			instKey := addrs.IntKey(i)
			if obj := rState.Instance(instKey).Current; obj == nil {
				t.Errorf("%s%s has no object after plan", addr, instKey)
			} else if got, want := obj.AttrsJSON, `"current"`; !bytes.Contains(got, []byte(want)) {
				// Should've saved the result of refreshing
				t.Errorf("%s%s has wrong prior state after plan\ngot:\n%s\n\nwant substring: %s", addr, instKey, got, want)
			}
		}
	}
	if rState := plan.PrevRunState.Resource(addr); rState == nil {
		t.Errorf("%s has no prior state at all after plan", addr)
	} else {
		for i := 0; i < 2; i++ {
			instKey := addrs.IntKey(i)
			if obj := rState.Instance(instKey).Current; obj == nil {
				t.Errorf("%s%s has no object after plan", addr, instKey)
			} else if got, want := obj.AttrsJSON, `"upgraded"`; !bytes.Contains(got, []byte(want)) {
				// Should've saved the result of upgrading
				t.Errorf("%s%s has wrong prior state after plan\ngot:\n%s\n\nwant substring: %s", addr, instKey, got, want)
			}
		}
	}

	// The output value should also have updated. If not, it's likely that we
	// skipped updating the working state to match the refreshed state when we
	// were evaluating the resource.
	if outChangeSrc := plan.Changes.OutputValue(addrs.RootModuleInstance.OutputValue("out")); outChangeSrc == nil {
		t.Errorf("no change planned for output value 'out'")
	} else {
		outChange, err := outChangeSrc.Decode()
		if err != nil {
			t.Fatalf("failed to decode output value 'out': %s", err)
		}
		got := outChange.After
		want := cty.TupleVal([]cty.Value{cty.StringVal("current"), cty.StringVal("current")})
		if !want.RawEquals(got) {
			t.Errorf("wrong value for output value 'out'\ngot:  %#v\nwant: %#v", got, want)
		}
	}
}

func TestContext2Plan_invalidSensitiveModuleOutput(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"child/main.tf": `
output "out" {
  value = sensitive("xyz")
}`,
		"main.tf": `
module "child" {
  source = "./child"
}

output "root" {
  value = module.child.out
}`,
	})

	ctx := testContext2(t, &ContextOpts{})

	_, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	if got, want := diags.Err().Error(), "Output refers to sensitive values"; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Plan_planDataSourceSensitiveNested(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "bar" {
}

data "test_data_source" "foo" {
  foo {
    bar = test_instance.bar.sensitive
  }
}
`,
	})

	p := new(testing_provider.MockProvider)
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp.PlannedState = cty.ObjectVal(map[string]cty.Value{
			"sensitive": cty.UnknownVal(cty.String),
		})
		return resp
	}
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"sensitive": {
						Type:      cty.String,
						Computed:  true,
						Sensitive: true,
					},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"test_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {Type: cty.String, Optional: true},
							},
						},
						Nesting: configschema.NestingSet,
					},
				},
			},
		},
	})

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("data.test_data_source.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"string":"data_id", "foo":[{"bar":"old"}]}`),
			AttrSensitivePaths: []cty.Path{
				cty.GetAttrPath("foo"),
			},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_instance.bar").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"sensitive":"old"}`),
			AttrSensitivePaths: []cty.Path{
				cty.GetAttrPath("sensitive"),
			},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)

	for _, res := range plan.Changes.Resources {
		switch res.Addr.String() {
		case "test_instance.bar":
			if res.Action != plans.Update {
				t.Fatalf("unexpected %s change for %s", res.Action, res.Addr)
			}
		case "data.test_data_source.foo":
			if res.Action != plans.Read {
				t.Fatalf("unexpected %s change for %s", res.Action, res.Addr)
			}
		default:
			t.Fatalf("unexpected %s change for %s", res.Action, res.Addr)
		}
	}
}

func TestContext2Plan_forceReplace(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object.a")
	addrB := mustResourceInstanceAddr("test_object.b")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_object" "a" {
			}
			resource "test_object" "b" {
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
		s.SetResourceInstanceCurrent(addrB, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addrA,
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addrA.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrA)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrA)
		}

		if got, want := instPlan.Action, plans.DeleteThenCreate; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceReplaceByRequest; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run(addrB.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrB)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrB)
		}

		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_forceReplaceIncompleteAddr(t *testing.T) {
	addr0 := mustResourceInstanceAddr("test_object.a[0]")
	addr1 := mustResourceInstanceAddr("test_object.a[1]")
	addrBare := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_object" "a" {
				count = 2
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addr0, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
		s.SetResourceInstanceCurrent(addr1, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addrBare,
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}
	diagsErr := diags.ErrWithWarnings()
	if diagsErr == nil {
		t.Fatalf("no warnings were returned")
	}
	if got, want := diagsErr.Error(), "Incompletely-matched force-replace resource instance"; !strings.Contains(got, want) {
		t.Errorf("missing expected warning\ngot:\n%s\n\nwant substring: %s", got, want)
	}

	t.Run(addr0.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addr0)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addr0)
		}

		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run(addr1.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addr1)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addr1)
		}

		if got, want := instPlan.Action, plans.NoOp; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

// Verify that adding a module instance does force existing module data sources
// to be deferred
func TestContext2Plan_noChangeDataSourceAddingModuleInstance(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  data = {
    a = "a"
    b = "b"
  }
}

module "one" {
  source   = "./mod"
  for_each = local.data
  input = each.value
}

module "two" {
  source   = "./mod"
  for_each = module.one
  input = each.value.output
}
`,
		"mod/main.tf": `
variable "input" {
}

resource "test_resource" "x" {
  value = var.input
}

data "test_data_source" "d" {
  foo = test_resource.x.id
}

output "output" {
  value = test_resource.x.id
}
`,
	})

	p := testProvider("test")
	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"id":  cty.StringVal("data"),
			"foo": cty.StringVal("foo"),
		}),
	}
	state := states.NewState()
	modOne := addrs.RootModuleInstance.Child("one", addrs.StringKey("a"))
	modTwo := addrs.RootModuleInstance.Child("two", addrs.StringKey("a"))
	one := state.EnsureModule(modOne)
	two := state.EnsureModule(modTwo)
	one.SetResourceInstanceCurrent(
		mustResourceInstanceAddr(`test_resource.x`).Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo","value":"a"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	one.SetResourceInstanceCurrent(
		mustResourceInstanceAddr(`data.test_data_source.d`).Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"data"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	two.SetResourceInstanceCurrent(
		mustResourceInstanceAddr(`test_resource.x`).Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo","value":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	two.SetResourceInstanceCurrent(
		mustResourceInstanceAddr(`data.test_data_source.d`).Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"data"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)

	for _, res := range plan.Changes.Resources {
		// both existing data sources should be read during plan
		if res.Addr.Module[0].InstanceKey == addrs.StringKey("b") {
			continue
		}

		if res.Addr.Resource.Resource.Mode == addrs.DataResourceMode && res.Action != plans.NoOp {
			t.Errorf("unexpected %s/%s plan for %s", res.Action, res.ActionReason, res.Addr)
		}
	}
}

func TestContext2Plan_moduleExpandOrphansResourceInstance(t *testing.T) {
	// This test deals with the situation where a user has changed the
	// repetition/expansion mode for a module call while there are already
	// resource instances from the previous declaration in the state.
	//
	// This is conceptually just the same as removing the resources
	// from the module configuration only for that instance, but the
	// implementation of it ends up a little different because it's
	// an entry in the resource address's _module path_ that we'll find
	// missing, rather than the resource's own instance key, and so
	// our analyses need to handle that situation by indicating that all
	// of the resources under the missing module instance have zero
	// instances, regardless of which resource in that module we might
	// be asking about, and do so without tripping over any missing
	// registrations in the instance expander that might lead to panics
	// if we aren't careful.
	//
	// (For some history here, see https://github.com/hashicorp/terraform/issues/30110 )

	addrNoKey := mustResourceInstanceAddr("module.child.test_object.a[0]")
	addrZeroKey := mustResourceInstanceAddr("module.child[0].test_object.a[0]")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			module "child" {
				source = "./child"
				count = 1
			}
		`,
		"child/main.tf": `
			resource "test_object" "a" {
				count = 1
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		// Notice that addrNoKey is the address which lacks any instance key
		// for module.child, and so that module instance doesn't match the
		// call declared above with count = 1, and therefore the resource
		// inside is "orphaned" even though the resource block actually
		// still exists there.
		s.SetResourceInstanceCurrent(addrNoKey, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	t.Run(addrNoKey.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrNoKey)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrNoKey)
		}

		if got, want := instPlan.Addr, addrNoKey; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrNoKey; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Delete; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceDeleteBecauseNoModule; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})

	t.Run(addrZeroKey.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrZeroKey)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrZeroKey)
		}

		if got, want := instPlan.Addr, addrZeroKey; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrZeroKey; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Create; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_resourcePreconditionPostcondition(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "boop" {
  type = string
}

resource "test_resource" "a" {
  value = var.boop
  lifecycle {
    precondition {
      condition     = var.boop == "boop"
      error_message = "Wrong boop."
    }
    postcondition {
      condition     = self.output != ""
      error_message = "Output must not be blank."
    }
  }
}

`,
	})

	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"value": {
						Type:     cty.String,
						Required: true,
					},
					"output": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	})

	t.Run("conditions pass", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
			m := req.ProposedNewState.AsValueMap()
			m["output"] = cty.StringVal("bar")

			resp.PlannedState = cty.ObjectVal(m)
			resp.LegacyTypeSystem = true
			return resp
		}
		plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("boop"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		for _, res := range plan.Changes.Resources {
			switch res.Addr.String() {
			case "test_resource.a":
				if res.Action != plans.Create {
					t.Fatalf("unexpected %s change for %s", res.Action, res.Addr)
				}
			default:
				t.Fatalf("unexpected %s change for %s", res.Action, res.Addr)
			}
		}
	})

	t.Run("precondition fail", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("nope"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		if !diags.HasErrors() {
			t.Fatal("succeeded; want errors")
		}
		if got, want := diags.Err().Error(), "Resource precondition failed: Wrong boop."; got != want {
			t.Fatalf("wrong error:\ngot:  %s\nwant: %q", got, want)
		}
		if p.PlanResourceChangeCalled {
			t.Errorf("Provider's PlanResourceChange was called; should'nt've been")
		}
	})

	t.Run("precondition fail refresh-only", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		state := states.BuildState(func(s *states.SyncState) {
			s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_resource.a"), &states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"value":"boop","output":"blorp"}`),
				Status:    states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
		})
		_, diags := ctx.Plan(m, state, &PlanOpts{
			Mode: plans.RefreshOnlyMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("nope"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		if len(diags) == 0 {
			t.Fatalf("no diags, but should have warnings")
		}
		if got, want := diags.ErrWithWarnings().Error(), "Resource precondition failed: Wrong boop."; got != want {
			t.Fatalf("wrong warning:\ngot:  %s\nwant: %q", got, want)
		}
		if !p.ReadResourceCalled {
			t.Errorf("Provider's ReadResource wasn't called; should've been")
		}
	})

	t.Run("postcondition fail", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
			m := req.ProposedNewState.AsValueMap()
			m["output"] = cty.StringVal("")

			resp.PlannedState = cty.ObjectVal(m)
			resp.LegacyTypeSystem = true
			return resp
		}
		plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("boop"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		if !diags.HasErrors() {
			t.Fatal("succeeded; want errors")
		}
		if got, want := diags.Err().Error(), "Resource postcondition failed: Output must not be blank."; got != want {
			t.Fatalf("wrong error:\ngot:  %s\nwant: %q", got, want)
		}
		if !p.PlanResourceChangeCalled {
			t.Errorf("Provider's PlanResourceChange wasn't called; should've been")
		}

		if !plan.Errored {
			t.Errorf("plan is not marked as errored")
		}

		// The plan should still include a proposed change for the resource
		// instance whose postcondition failed, since its plan is valid in
		// isolation even though the postcondition prevents any further
		// planning downstream. This gives the UI the option of describing
		// the planned change as additional context alongside the error
		// message, although it's up to the UI to decide whether and how to
		// do that.
		changes := plan.Changes
		changeSrc := changes.ResourceInstance(mustResourceInstanceAddr("test_resource.a"))
		if changeSrc != nil {
			if got, want := changeSrc.Action, plans.Create; got != want {
				t.Errorf("wrong proposed change action\ngot:  %s\nwant: %s", got, want)
			}
		} else {
			t.Errorf("no planned change for test_resource.a")
		}
	})

	t.Run("postcondition fail refresh-only", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		state := states.BuildState(func(s *states.SyncState) {
			s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_resource.a"), &states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"value":"boop","output":"blorp"}`),
				Status:    states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
		})
		p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
			newVal, err := cty.Transform(req.PriorState, func(path cty.Path, v cty.Value) (cty.Value, error) {
				if len(path) == 1 && path[0] == (cty.GetAttrStep{Name: "output"}) {
					return cty.StringVal(""), nil
				}
				return v, nil
			})
			if err != nil {
				// shouldn't get here
				t.Fatalf("ReadResourceFn transform failed")
				return providers.ReadResourceResponse{}
			}
			return providers.ReadResourceResponse{
				NewState: newVal,
			}
		}
		_, diags := ctx.Plan(m, state, &PlanOpts{
			Mode: plans.RefreshOnlyMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("boop"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		if len(diags) == 0 {
			t.Fatalf("no diags, but should have warnings")
		}
		if got, want := diags.ErrWithWarnings().Error(), "Resource postcondition failed: Output must not be blank."; got != want {
			t.Fatalf("wrong warning:\ngot:  %s\nwant: %q", got, want)
		}
		if !p.ReadResourceCalled {
			t.Errorf("Provider's ReadResource wasn't called; should've been")
		}
		if p.PlanResourceChangeCalled {
			t.Errorf("Provider's PlanResourceChange was called; should'nt've been")
		}
	})

	t.Run("precondition and postcondition fail refresh-only", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		state := states.BuildState(func(s *states.SyncState) {
			s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_resource.a"), &states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"value":"boop","output":"blorp"}`),
				Status:    states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
		})
		p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
			newVal, err := cty.Transform(req.PriorState, func(path cty.Path, v cty.Value) (cty.Value, error) {
				if len(path) == 1 && path[0] == (cty.GetAttrStep{Name: "output"}) {
					return cty.StringVal(""), nil
				}
				return v, nil
			})
			if err != nil {
				// shouldn't get here
				t.Fatalf("ReadResourceFn transform failed")
				return providers.ReadResourceResponse{}
			}
			return providers.ReadResourceResponse{
				NewState: newVal,
			}
		}
		_, diags := ctx.Plan(m, state, &PlanOpts{
			Mode: plans.RefreshOnlyMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("nope"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		if got, want := len(diags), 2; got != want {
			t.Errorf("wrong number of warnings, got %d, want %d", got, want)
		}
		warnings := diags.ErrWithWarnings().Error()
		wantWarnings := []string{
			"Resource precondition failed: Wrong boop.",
			"Resource postcondition failed: Output must not be blank.",
		}
		for _, want := range wantWarnings {
			if !strings.Contains(warnings, want) {
				t.Errorf("missing warning:\ngot:  %s\nwant to contain: %q", warnings, want)
			}
		}
		if !p.ReadResourceCalled {
			t.Errorf("Provider's ReadResource wasn't called; should've been")
		}
		if p.PlanResourceChangeCalled {
			t.Errorf("Provider's PlanResourceChange was called; should'nt've been")
		}
	})
}

func TestContext2Plan_dataSourcePreconditionPostcondition(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "boop" {
  type = string
}

data "test_data_source" "a" {
  foo = var.boop
  lifecycle {
    precondition {
      condition     = var.boop == "boop"
      error_message = "Wrong boop."
    }
    postcondition {
      condition     = length(self.results) > 0
      error_message = "Results cannot be empty."
    }
  }
}

resource "test_resource" "a" {
  value    = data.test_data_source.a.results[0]
}
`,
	})

	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"value": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"test_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Required: true,
					},
					"results": {
						Type:     cty.List(cty.String),
						Computed: true,
					},
				},
			},
		},
	})

	t.Run("conditions pass", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})
		p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
			State: cty.ObjectVal(map[string]cty.Value{
				"foo":     cty.StringVal("boop"),
				"results": cty.ListVal([]cty.Value{cty.StringVal("boop")}),
			}),
		}
		plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("boop"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		for _, res := range plan.Changes.Resources {
			switch res.Addr.String() {
			case "test_resource.a":
				if res.Action != plans.Create {
					t.Fatalf("unexpected %s change for %s", res.Action, res.Addr)
				}
			case "data.test_data_source.a":
				if res.Action != plans.Read {
					t.Fatalf("unexpected %s change for %s", res.Action, res.Addr)
				}
			default:
				t.Fatalf("unexpected %s change for %s", res.Action, res.Addr)
			}
		}

		addr := mustResourceInstanceAddr("data.test_data_source.a")
		if gotResult := plan.Checks.GetObjectResult(addr); gotResult == nil {
			t.Errorf("no check result for %s", addr)
		} else {
			wantResult := &states.CheckResultObject{
				Status: checks.StatusPass,
			}
			if diff := cmp.Diff(wantResult, gotResult, valueComparer); diff != "" {
				t.Errorf("wrong check result for %s\n%s", addr, diff)
			}
		}
	})

	t.Run("precondition fail", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})
		_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("nope"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		if !diags.HasErrors() {
			t.Fatal("succeeded; want errors")
		}
		if got, want := diags.Err().Error(), "Resource precondition failed: Wrong boop."; got != want {
			t.Fatalf("wrong error:\ngot:  %s\nwant: %q", got, want)
		}
		if p.ReadDataSourceCalled {
			t.Errorf("Provider's ReadResource was called; should'nt've been")
		}
	})

	t.Run("precondition fail refresh-only", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})
		plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.RefreshOnlyMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("nope"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		if len(diags) == 0 {
			t.Fatalf("no diags, but should have warnings")
		}
		if got, want := diags.ErrWithWarnings().Error(), "Resource precondition failed: Wrong boop."; got != want {
			t.Fatalf("wrong warning:\ngot:  %s\nwant: %q", got, want)
		}
		for _, res := range plan.Changes.Resources {
			switch res.Addr.String() {
			case "test_resource.a":
				if res.Action != plans.Create {
					t.Fatalf("unexpected %s change for %s", res.Action, res.Addr)
				}
			case "data.test_data_source.a":
				if res.Action != plans.Read {
					t.Fatalf("unexpected %s change for %s", res.Action, res.Addr)
				}
			default:
				t.Fatalf("unexpected %s change for %s", res.Action, res.Addr)
			}
		}
	})

	t.Run("postcondition fail", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})
		p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
			State: cty.ObjectVal(map[string]cty.Value{
				"foo":     cty.StringVal("boop"),
				"results": cty.ListValEmpty(cty.String),
			}),
		}
		_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("boop"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		if !diags.HasErrors() {
			t.Fatal("succeeded; want errors")
		}
		if got, want := diags.Err().Error(), "Resource postcondition failed: Results cannot be empty."; got != want {
			t.Fatalf("wrong error:\ngot:  %s\nwant: %q", got, want)
		}
		if !p.ReadDataSourceCalled {
			t.Errorf("Provider's ReadDataSource wasn't called; should've been")
		}
	})

	t.Run("postcondition fail refresh-only", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})
		p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
			State: cty.ObjectVal(map[string]cty.Value{
				"foo":     cty.StringVal("boop"),
				"results": cty.ListValEmpty(cty.String),
			}),
		}
		plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.RefreshOnlyMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("boop"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		if got, want := diags.ErrWithWarnings().Error(), "Resource postcondition failed: Results cannot be empty."; got != want {
			t.Fatalf("wrong error:\ngot:  %s\nwant: %q", got, want)
		}
		addr := mustResourceInstanceAddr("data.test_data_source.a")
		if gotResult := plan.Checks.GetObjectResult(addr); gotResult == nil {
			t.Errorf("no check result for %s", addr)
		} else {
			wantResult := &states.CheckResultObject{
				Status: checks.StatusFail,
				FailureMessages: []string{
					"Results cannot be empty.",
				},
			}
			if diff := cmp.Diff(wantResult, gotResult, valueComparer); diff != "" {
				t.Errorf("wrong check result\n%s", diff)
			}
		}
	})

	t.Run("precondition and postcondition fail refresh-only", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})
		p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
			State: cty.ObjectVal(map[string]cty.Value{
				"foo":     cty.StringVal("nope"),
				"results": cty.ListValEmpty(cty.String),
			}),
		}
		_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.RefreshOnlyMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("nope"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		if got, want := len(diags), 2; got != want {
			t.Errorf("wrong number of warnings, got %d, want %d", got, want)
		}
		warnings := diags.ErrWithWarnings().Error()
		wantWarnings := []string{
			"Resource precondition failed: Wrong boop.",
			"Resource postcondition failed: Results cannot be empty.",
		}
		for _, want := range wantWarnings {
			if !strings.Contains(warnings, want) {
				t.Errorf("missing warning:\ngot:  %s\nwant to contain: %q", warnings, want)
			}
		}
	})
}

func TestContext2Plan_outputPrecondition(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "boop" {
  type = string
}

output "a" {
  value = var.boop
  precondition {
    condition     = var.boop == "boop"
    error_message = "Wrong boop."
  }
}
`,
	})

	p := testProvider("test")

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	t.Run("condition pass", func(t *testing.T) {
		plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("boop"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		addr := addrs.RootModuleInstance.OutputValue("a")
		outputPlan := plan.Changes.OutputValue(addr)
		if outputPlan == nil {
			t.Fatalf("no plan for %s at all", addr)
		}
		if got, want := outputPlan.Addr, addr; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := outputPlan.Action, plans.Create; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if gotResult := plan.Checks.GetObjectResult(addr); gotResult == nil {
			t.Errorf("no check result for %s", addr)
		} else {
			wantResult := &states.CheckResultObject{
				Status: checks.StatusPass,
			}
			if diff := cmp.Diff(wantResult, gotResult, valueComparer); diff != "" {
				t.Errorf("wrong check result\n%s", diff)
			}
		}
	})

	t.Run("condition fail", func(t *testing.T) {
		_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("nope"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		if !diags.HasErrors() {
			t.Fatal("succeeded; want errors")
		}
		if got, want := diags.Err().Error(), "Module output value precondition failed: Wrong boop."; got != want {
			t.Fatalf("wrong error:\ngot:  %s\nwant: %q", got, want)
		}
	})

	t.Run("condition fail refresh-only", func(t *testing.T) {
		plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.RefreshOnlyMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("nope"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		if len(diags) == 0 {
			t.Fatalf("no diags, but should have warnings")
		}
		if got, want := diags.ErrWithWarnings().Error(), "Module output value precondition failed: Wrong boop."; got != want {
			t.Errorf("wrong warning:\ngot:  %s\nwant: %q", got, want)
		}
		addr := addrs.RootModuleInstance.OutputValue("a")
		outputPlan := plan.Changes.OutputValue(addr)
		if outputPlan == nil {
			t.Fatalf("no plan for %s at all", addr)
		}
		if got, want := outputPlan.Addr, addr; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := outputPlan.Action, plans.Create; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if gotResult := plan.Checks.GetObjectResult(addr); gotResult == nil {
			t.Errorf("no condition result for %s", addr)
		} else {
			wantResult := &states.CheckResultObject{
				Status:          checks.StatusFail,
				FailureMessages: []string{"Wrong boop."},
			}
			if diff := cmp.Diff(wantResult, gotResult, valueComparer); diff != "" {
				t.Errorf("wrong condition result\n%s", diff)
			}
		}
	})
}

func TestContext2Plan_preconditionErrors(t *testing.T) {
	testCases := []struct {
		condition   string
		wantSummary string
		wantDetail  string
	}{
		{
			"data.test_data_source",
			"Invalid reference",
			`The "data" object must be followed by two attribute names`,
		},
		{
			"self.value",
			`Invalid "self" reference`,
			"only in resource provisioner, connection, and postcondition blocks",
		},
		{
			"data.foo.bar",
			"Reference to undeclared resource",
			`A data resource "foo" "bar" has not been declared in the root module`,
		},
		{
			"test_resource.b.value",
			"Invalid condition result",
			"Condition expression must return either true or false",
		},
		{
			"test_resource.c.value",
			"Invalid condition result",
			"Invalid condition result value: a bool is required",
		},
	}

	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	for _, tc := range testCases {
		t.Run(tc.condition, func(t *testing.T) {
			main := fmt.Sprintf(`
			resource "test_resource" "a" {
				value = var.boop
				lifecycle {
					precondition {
						condition     = %s
						error_message = "Not relevant."
					}
				}
			}

			resource "test_resource" "b" {
				value = null
			}

			resource "test_resource" "c" {
				value = "bar"
			}
			`, tc.condition)
			m := testModuleInline(t, map[string]string{"main.tf": main})

			plan, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
			if !diags.HasErrors() {
				t.Fatal("succeeded; want errors")
			}

			if !plan.Errored {
				t.Fatal("plan failed to record error")
			}

			diag := diags[0]
			if got, want := diag.Description().Summary, tc.wantSummary; got != want {
				t.Errorf("unexpected summary\n got: %s\nwant: %s", got, want)
			}
			if got, want := diag.Description().Detail, tc.wantDetail; !strings.Contains(got, want) {
				t.Errorf("unexpected summary\ngot: %s\nwant to contain %q", got, want)
			}

			for _, kv := range plan.Checks.ConfigResults.Elements() {
				// All these are configuration or evaluation errors
				if kv.Value.Status != checks.StatusError {
					t.Errorf("incorrect status, got %s", kv.Value.Status)
				}
			}
		})
	}
}

func TestContext2Plan_preconditionSensitiveValues(t *testing.T) {
	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "boop" {
  sensitive = true
  type      = string
}

output "a" {
  sensitive = true
  value     = var.boop

  precondition {
    condition     = length(var.boop) <= 4
    error_message = "Boop is too long, ${length(var.boop)} > 4"
  }
}
`,
	})

	_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"boop": &InputValue{
				Value:      cty.StringVal("bleep"),
				SourceType: ValueFromCLIArg,
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	if got, want := len(diags), 2; got != want {
		t.Errorf("wrong number of diags, got %d, want %d", got, want)
	}
	for _, diag := range diags {
		desc := diag.Description()
		if desc.Summary == "Module output value precondition failed" {
			if got, want := desc.Detail, "This check failed, but has an invalid error message as described in the other accompanying messages."; !strings.Contains(got, want) {
				t.Errorf("unexpected detail\ngot: %s\nwant to contain %q", got, want)
			}
		} else if desc.Summary == "Error message refers to sensitive values" {
			if got, want := desc.Detail, "The error expression used to explain this condition refers to sensitive values, so Terraform will not display the resulting message."; !strings.Contains(got, want) {
				t.Errorf("unexpected detail\ngot: %s\nwant to contain %q", got, want)
			}
		} else {
			t.Errorf("unexpected summary\ngot: %s", desc.Summary)
		}
	}
}

func TestContext2Plan_triggeredBy(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  count = 1
  test_string = "new"
}
resource "test_object" "b" {
  count = 1
  test_string = test_object.a[count.index].test_string
  lifecycle {
    # the change to test_string in the other resource should trigger replacement
    replace_triggered_by = [ test_object.a[count.index].test_string ]
  }
}
`,
	})

	p := simpleMockProvider()

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.a[0]"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"test_string":"old"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.b[0]"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}
	for _, c := range plan.Changes.Resources {
		switch c.Addr.String() {
		case "test_object.a[0]":
			if c.Action != plans.Update {
				t.Fatalf("unexpected %s change for %s\n", c.Action, c.Addr)
			}
		case "test_object.b[0]":
			if c.Action != plans.DeleteThenCreate {
				t.Fatalf("unexpected %s change for %s\n", c.Action, c.Addr)
			}
			if c.ActionReason != plans.ResourceInstanceReplaceByTriggers {
				t.Fatalf("incorrect reason for change: %s\n", c.ActionReason)
			}
		default:
			t.Fatal("unexpected change", c.Addr, c.Action)
		}
	}
}

func TestContext2Plan_dataSchemaChange(t *testing.T) {
	// We can't decode the prior state when a data source upgrades the schema
	// in an incompatible way. Since prior state for data sources is purely
	// informational, decoding should be skipped altogether.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
data "test_object" "a" {
  obj {
    # args changes from a list to a map
    args = {
      val = "string"
	}
  }
}
`,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		DataSources: map[string]*configschema.Block{
			"test_object": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"obj": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"args": {Type: cty.Map(cty.String), Optional: true},
							},
						},
						Nesting: configschema.NestingSet,
					},
				},
			},
		},
	})

	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
		resp.State = req.Config
		return resp
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(mustResourceInstanceAddr(`data.test_object.a`), &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"id":"old","obj":[{"args":["string"]}]}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)
}

func TestContext2Plan_applyGraphError(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
}
resource "test_object" "b" {
	depends_on = [test_object.a]
}
`,
	})

	p := simpleMockProvider()

	// Here we introduce a cycle via state which only shows up in the apply
	// graph where the actual destroy instances are connected in the graph.
	// This could happen for example when a user has an existing state with
	// stored dependencies, and changes the config in such a way that
	// contradicts the stored dependencies.
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectTainted,
			AttrsJSON:    []byte(`{"test_string":"a"}`),
			Dependencies: []addrs.ConfigResource{mustResourceInstanceAddr("test_object.b").ContainingResource().Config()},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.b").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsJSON: []byte(`{"test_string":"b"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
	})
	if !diags.HasErrors() {
		t.Fatal("cycle error not detected")
	}

	msg := diags.ErrWithWarnings().Error()
	if !strings.Contains(msg, "Cycle") {
		t.Fatalf("no cycle error found:\n got: %s\n", msg)
	}
}

// plan a destroy with no state where configuration could fail to evaluate
// expansion indexes.
func TestContext2Plan_emptyDestroy(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  enable = true
  value  = local.enable ? module.example[0].out : null
}

module "example" {
  count  = local.enable ? 1 : 0
  source = "./example"
}
`,
		"example/main.tf": `
resource "test_resource" "x" {
}

output "out" {
  value = test_resource.x
}
`,
	})

	p := testProvider("test")
	state := states.NewState()

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
	})

	assertNoErrors(t, diags)

	// ensure that the given states are valid and can be serialized
	if plan.PrevRunState == nil {
		t.Fatal("nil plan.PrevRunState")
	}
	if plan.PriorState == nil {
		t.Fatal("nil plan.PriorState")
	}
}

// A deposed instances which no longer exists during ReadResource creates NoOp
// change, which should not affect the plan.
func TestContext2Plan_deposedNoLongerExists(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "b" {
  count = 1
  test_string = "updated"
  lifecycle {
    create_before_destroy = true
  }
}
`,
	})

	p := simpleMockProvider()
	p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
		s := req.PriorState.GetAttr("test_string").AsString()
		if s == "current" {
			resp.NewState = req.PriorState
			return resp
		}
		// pretend the non-current instance has been deleted already
		resp.NewState = cty.NullVal(req.PriorState.Type())
		return resp
	}

	// Here we introduce a cycle via state which only shows up in the apply
	// graph where the actual destroy instances are connected in the graph.
	// This could happen for example when a user has an existing state with
	// stored dependencies, and changes the config in such a way that
	// contradicts the stored dependencies.
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceDeposed(
		mustResourceInstanceAddr("test_object.a[0]").Resource,
		states.DeposedKey("deposed"),
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectTainted,
			AttrsJSON:    []byte(`{"test_string":"old"}`),
			Dependencies: []addrs.ConfigResource{},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.a[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectTainted,
			AttrsJSON:    []byte(`{"test_string":"current"}`),
			Dependencies: []addrs.ConfigResource{},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
	})
	assertNoErrors(t, diags)
}

// make sure there are no cycles with changes around a provider configured via
// managed resources.
func TestContext2Plan_destroyWithResourceConfiguredProvider(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  in = "a"
}

provider "test" {
  alias = "other"
  in = test_object.a.out
}

resource "test_object" "b" {
  provider = test.other
  in = "a"
}
`})

	testProvider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"in": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
			ResourceTypes: map[string]providers.Schema{
				"test_object": providers.Schema{
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"in": {
								Type:     cty.String,
								Optional: true,
							},
							"out": {
								Type:     cty.Number,
								Computed: true,
							},
						},
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(testProvider),
		},
	})

	// plan+apply to create the initial state
	opts := SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables))
	plan, diags := ctx.Plan(m, states.NewState(), opts)
	assertNoErrors(t, diags)
	state, diags := ctx.Apply(plan, m, nil)
	assertNoErrors(t, diags)

	// Resource changes which have dependencies across providers which
	// themselves depend on resources can result in cycles.
	// Because other_object transitively depends on the module resources
	// through its provider, we trigger changes on both sides of this boundary
	// to ensure we can create a valid plan.
	//
	// Try to replace both instances
	addrA := mustResourceInstanceAddr("test_object.a")
	addrB := mustResourceInstanceAddr(`test_object.b`)
	opts.ForceReplace = []addrs.AbsResourceInstance{addrA, addrB}

	_, diags = ctx.Plan(m, state, opts)
	assertNoErrors(t, diags)
}

func TestContext2Plan_destroyPartialState(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
}

output "out" {
  value = module.mod.out
}

module "mod" {
  source = "./mod"
}
`,

		"./mod/main.tf": `
resource "test_object" "a" {
  count = 2

  lifecycle {
    precondition {
	  # test_object_b has already been destroyed, so referencing the first
      # instance must not fail during a destroy plan.
      condition = test_object.b[0].test_string == "invalid"
      error_message = "should not block destroy"
    }
    precondition {
      # this failing condition should bot block a destroy plan
      condition = !local.continue
      error_message = "should not block destroy"
    }
  }
}

resource "test_object" "b" {
  count = 2
}

locals {
  continue = true
}

output "out" {
  # the reference to test_object.b[0] may not be valid during a destroy plan,
  # but should not fail.
  value = local.continue ? test_object.a[1].test_string != "invalid"  && test_object.b[0].test_string != "invalid" : false

  precondition {
    # test_object_b has already been destroyed, so referencing the first
    # instance must not fail during a destroy plan.
    condition = test_object.b[0].test_string == "invalid"
    error_message = "should not block destroy"
  }
  precondition {
    # this failing condition should bot block a destroy plan
    condition = test_object.a[0].test_string == "invalid"
    error_message = "should not block destroy"
  }
}
`})

	p := simpleMockProvider()

	// This state could be the result of a failed destroy, leaving only 2
	// remaining instances. We want to be able to continue the destroy to
	// remove everything without blocking on invalid references or failing
	// conditions.
	state := states.NewState()
	mod := state.EnsureModule(addrs.RootModuleInstance.Child("mod", addrs.NoKey))
	mod.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.a[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectTainted,
			AttrsJSON:    []byte(`{"test_string":"current"}`),
			Dependencies: []addrs.ConfigResource{},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	mod.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.a[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectTainted,
			AttrsJSON:    []byte(`{"test_string":"current"}`),
			Dependencies: []addrs.ConfigResource{},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
	})
	assertNoErrors(t, diags)
}

func TestContext2Plan_destroyPartialStateLocalRef(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "already_destroyed" {
  count = 1
  source = "./mod"
}

locals {
  eval_error = module.already_destroyed[0].out
}

output "already_destroyed" {
  value = local.eval_error
}

`,

		"./mod/main.tf": `
resource "test_object" "a" {
}

output "out" {
  value = test_object.a.test_string
}
`})

	p := simpleMockProvider()

	state := states.NewState()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
	})
	assertNoErrors(t, diags)
}

// Make sure the data sources in the prior state are serializeable even if
// there were an error in the plan.
func TestContext2Plan_dataSourceReadPlanError(t *testing.T) {
	m, snap := testModuleWithSnapshot(t, "data-source-read-with-plan-error")
	awsProvider := testProvider("aws")
	testProvider := testProvider("test")

	testProvider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp.PlannedState = req.ProposedNewState
		resp.Diagnostics = resp.Diagnostics.Append(errors.New("oops"))
		return resp
	}

	state := states.NewState()

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"):  testProviderFuncFixed(awsProvider),
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(testProvider),
		},
	})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	if !diags.HasErrors() {
		t.Fatalf("expected plan error")
	}

	// make sure we can serialize the plan even if there were an error
	_, _, _, err := contextOptsForPlanViaFile(t, snap, plan)
	if err != nil {
		t.Fatalf("failed to round-trip through planfile: %s", err)
	}
}

func TestContext2Plan_ignoredMarkedValue(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  map = {
    prior = "value"
    new   = sensitive("ignored")
  }
}
`})

	testProvider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"test_object": providers.Schema{
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"map": {
								Type:     cty.Map(cty.String),
								Optional: true,
							},
						},
					},
				},
			},
		},
	}

	testProvider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		// We're going to ignore any changes here and return the prior state.
		resp.PlannedState = req.PriorState
		return resp
	}

	state := states.NewState()
	root := state.RootModule()
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"map":{"prior":"value"}}`),
			Dependencies: []addrs.ConfigResource{},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(testProvider),
		},
	})

	// plan+apply to create the initial state
	opts := SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables))
	plan, diags := ctx.Plan(m, state, opts)
	assertNoErrors(t, diags)

	for _, c := range plan.Changes.Resources {
		if c.Action != plans.NoOp {
			t.Errorf("unexpected %s change for %s", c.Action, c.Addr)
		}
	}
}

func TestContext2Plan_externalProvidersWithState(t *testing.T) {
	// In this test we're going to use an external provider for a resource
	// that is already in the state. Terraform should allow this, even though
	// the provider isn't defined in the configuration.

	m := testModuleInline(t, map[string]string{
		"main.tf": ``, // no resources
	})

	state := states.BuildState(func(state *states.SyncState) {
		state.SetResourceInstanceCurrent(mustResourceInstanceAddr("foo.a"), &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["hashicorp/foo"]`))
	})

	fooProvider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"foo": {
					Block: &configschema.Block{},
				},
			},
		},
		ConfigureProviderFn: func(providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
			return providers.ConfigureProviderResponse{
				Diagnostics: tfdiags.Diagnostics{
					tfdiags.Sourceless(
						tfdiags.Error,
						"Pre-configured provider was reconfigured by the modules runtime",
						"An externally-configured provider should not have its ConfigureProvider function called during planning.",
					),
				},
			}
		},
	}

	ctx, diags := NewContext(&ContextOpts{
		PreloadedProviderSchemas: map[addrs.Provider]providers.GetProviderSchemaResponse{
			addrs.MustParseProviderSourceString("hashicorp/foo"): *fooProvider.GetProviderSchemaResponse,
		},
	})
	assertNoDiagnostics(t, diags)

	fooProvider.ConfigureProviderCalled = true
	_, diags = ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		ExternalProviders: map[addrs.RootProviderConfig]providers.Interface{
			addrs.RootProviderConfig{
				Provider: addrs.MustParseProviderSourceString("hashicorp/foo"),
			}: fooProvider,
		},
	})
	assertNoDiagnostics(t, diags)
}

func TestContext2Plan_externalProviders(t *testing.T) {
	// This test exercises the option for callers to pass in their own
	// already-configured provider instances, instead of the modules runtime
	// configuring the providers itself. This is for situations where the root
	// module is actually a shared module and some external system is acting
	// as the true root, such as in Stacks where the stack configuration is
	// the root and each component is a separate module tree with external
	// providers passed into it similar to what we're doing in this test.

	m := testModuleInline(t, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					bar = {
						source = "terraform.io/builtin/bar"
					}
					baz = {
						configuration_aliases = [ baz.beep ]
					}
				}
			}

			resource "foo" "a" {}
			resource "bar" "b" {}
			resource "baz" "c" {
				provider = baz.beep
			}
		`,
	})

	mustNotConfigure := func(providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
		return providers.ConfigureProviderResponse{
			Diagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Pre-configured provider was reconfigured by the modules runtime",
					"An externally-configured provider should not have its ConfigureProvider function called during planning.",
				),
			},
		}
	}

	fooAddr := addrs.MustParseProviderSourceString("hashicorp/foo")
	fooConfigAddr := addrs.RootProviderConfig{
		Provider: fooAddr,
	}
	fooProvider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						// We have a required argument here just so that the
						// plan will fail if the runtime erroneously tries
						// to prepare a configuration for this provider;
						// in the absense of an external provider instance
						// the runtime would behave as if there were an empty
						// provider "foo" block and try to decode its synthetic
						// empty body against this schema, but that should not
						// happen if we're assuming the provider is already
						// configured by an external caller.
						"reqd": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
			ResourceTypes: map[string]providers.Schema{
				"foo": {
					Block: &configschema.Block{},
				},
			},
		},
		ConfigureProviderFn: mustNotConfigure,
	}
	barAddr := addrs.MustParseProviderSourceString("terraform.io/builtin/bar")
	barConfigAddr := addrs.RootProviderConfig{
		Provider: barAddr,
	}
	barProvider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"bar": {
					Block: &configschema.Block{},
				},
			},
		},
		ConfigureProviderFn: mustNotConfigure,
	}
	bazAddr := addrs.MustParseProviderSourceString("hashicorp/baz")
	bazConfigAddr := addrs.RootProviderConfig{
		Provider: bazAddr,
		Alias:    "beep",
	}
	bazProvider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"baz": {
					Block: &configschema.Block{},
				},
			},
		},
		ConfigureProviderFn: mustNotConfigure,
	}

	ctx, diags := NewContext(&ContextOpts{
		PreloadedProviderSchemas: map[addrs.Provider]providers.GetProviderSchemaResponse{
			fooAddr: *fooProvider.GetProviderSchemaResponse,
			barAddr: *barProvider.GetProviderSchemaResponse,
			bazAddr: *bazProvider.GetProviderSchemaResponse,
		},
	})
	assertNoDiagnostics(t, diags)

	// Many of the MockProvider methods check whether Configure was called and
	// return an error if not, and so we'll set that flag ahead of time to
	// simulate the effect of the provider being pre-configured by an external
	// caller.
	fooProvider.ConfigureProviderCalled = true
	barProvider.ConfigureProviderCalled = true
	bazProvider.ConfigureProviderCalled = true

	_, diags = ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		ExternalProviders: map[addrs.RootProviderConfig]providers.Interface{
			fooConfigAddr: fooProvider,
			barConfigAddr: barProvider,
			bazConfigAddr: bazProvider,
		},
	})
	assertNoDiagnostics(t, diags)

	// If everything has worked as intended, the Plan call should've skipped
	// configuring and closing all of the providers, because that's the
	// caller's job to deal with.
	if fooProvider.GetProviderSchemaCalled {
		t.Errorf("called GetProviderSchema for %s", fooAddr)
	}
	if fooProvider.CloseCalled {
		t.Errorf("called Close for %s", fooAddr)
	}
	if barProvider.GetProviderSchemaCalled {
		t.Errorf("called GetProviderSchema for %s", barAddr)
	}
	if barProvider.CloseCalled {
		t.Errorf("called Close for %s", barAddr)
	}
	if bazProvider.GetProviderSchemaCalled {
		t.Errorf("called GetProviderSchema for %s", bazAddr)
	}
	if bazProvider.CloseCalled {
		t.Errorf("called Close for %s", bazAddr)
	}

	// However, the Plan call _should_ have asked each of the providers to
	// plan a resource change.
	if !fooProvider.PlanResourceChangeCalled {
		t.Errorf("did not call PlanResourceChange for %s", fooConfigAddr)
	}
	if !barProvider.PlanResourceChangeCalled {
		t.Errorf("did not call PlanResourceChange for %s", barConfigAddr)
	}
	if !bazProvider.PlanResourceChangeCalled {
		t.Errorf("did not call PlanResourceChange for %s", bazConfigAddr)
	}
}

func TestContext2Apply_externalDependencyDeferred(t *testing.T) {
	// This test deals with the situation where the stacks runtime knows
	// that an upstream component already has deferred actions and so
	// it's telling us that we need to artifically treat everything in
	// the current configuration as deferred.

	cfg := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test" "a" {
				name = "a"
			}

			resource "test" "b" {
				name           = "b"
				upstream_names = [test.a.name]
			}

			resource "test" "c" {
				name = "c"
				upstream_names = toset([
					test.a.name,
					test.b.name,
				])
			}
		`,
	})

	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"test": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"name": {
								Type:     cty.String,
								Required: true,
							},
							"upstream_names": {
								Type:     cty.Set(cty.String),
								Optional: true,
							},
						},
					},
				},
			},
		},
		PlanResourceChangeFn: func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			return providers.PlanResourceChangeResponse{
				PlannedState: req.ProposedNewState,
			}
		},
	}
	resourceInstancesActionsInPlan := func(p *plans.Plan) map[string]plans.Action {
		ret := make(map[string]plans.Action)
		for _, cs := range p.Changes.Resources {
			// Anything that was deferred will not appear in the result at
			// all. Non-deferred actions that don't actually need to do anything
			// _will_ appear, but with action set to [plans.NoOp].
			ret[cs.Addr.String()] = cs.Action
		}
		return ret
	}
	cmpOpts := cmp.Options{
		ctydebug.CmpOptions,
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(cfg, states.NewState(), &PlanOpts{
		Mode:                       plans.NormalMode,
		DeferralAllowed:            true,
		ExternalDependencyDeferred: true,
	})
	assertNoDiagnostics(t, diags)
	if plan.Applyable {
		t.Fatal("plan is applyable; should not be, because there's nothing to do yet")
	}
	if plan.Complete {
		t.Fatal("plan is complete; should have deferred actions")
	}

	gotActions := resourceInstancesActionsInPlan(plan)
	wantActions := map[string]plans.Action{
		// No actions at all, because everything was deferred!
	}
	if diff := cmp.Diff(wantActions, gotActions, cmpOpts); diff != "" {
		t.Fatalf("wrong actions in plan\n%s", diff)
	}

	if len(plan.DeferredResources) != 3 {
		t.Fatalf("expected exactly 3 deferred resources, got %d", len(plan.DeferredResources))
	}

	for _, res := range plan.DeferredResources {
		if res.DeferredReason != providers.DeferredReasonDeferredPrereq {
			t.Fatalf("expected all resources to be deferred due to deferred prerequisites, but %s was not", res.ChangeSrc.Addr)
		}
	}
}

func TestContext2Plan_removedResourceForgetBasic(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
removed {
  from = test_object.a
  lifecycle {
    destroy = false
  }
}
`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"foo":"bar"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addrA,
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	// We should have a warning, though! We'll lightly abuse the "for RPC"
	// feature of diagnostics to get some more-readily-comparable diagnostic
	// values.
	gotDiags := diags.ForRPC()
	wantDiags := tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Warning,
			"Some objects will no longer be managed by Terraform",
			`If you apply this plan, Terraform will discard its tracking information for the following objects, but it will not delete them:
 - test_object.a

After applying this plan, Terraform will no longer manage these objects. You will need to import them into Terraform to manage them again.`,
		),
	}.ForRPC()
	if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
		t.Errorf("wrong diagnostics\n%s", diff)
	}

	t.Run(addrA.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrA)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrA)
		}

		if got, want := instPlan.Addr, addrA; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrA; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Forget; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceDeleteBecauseNoResourceConfig; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_removedResourceDestroyBasic(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		// block is effectively a no-op, because the resource has already been
		// removed from config.
		"main.tf": `
			removed {
				from = test_object.a
                lifecycle {
                  destroy = true
                }
			}
		`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"foo":"bar"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addrA,
		},
	})
	if len(diags) > 0 {
		t.Fatalf("unexpected diags\n%s", diags.Err().Error())
	}

	t.Run(addrA.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstance(addrA)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrA)
		}

		if got, want := instPlan.Addr, addrA; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrA; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Delete; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceDeleteBecauseNoResourceConfig; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

// This test case imagines a situation in which a create_before_destroy resource
// was replaced during the last apply. The replace partially failed: the create
// step succeeded while the destroy step failed, so a deposed resource
// instance remains in state.
// Now the user has removed the "resource" block from config and replaced it
// with a "removed" block with destroy = false.
// In this case, the user signals their intent that any instances of the
// resource be forgotten, not destroyed, so the correct action is Forget.
func TestContext2Plan_removedResourceForgetDeposed(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object.a")
	deposedKey := states.DeposedKey("gone")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
removed {
  from = test_object.a
  lifecycle {
    destroy = false
  }
}
`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceDeposed(addrA, deposedKey, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"foo":"bar"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addrA,
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	// We should have a warning, though! We'll lightly abuse the "for RPC"
	// feature of diagnostics to get some more-readily-comparable diagnostic
	// values.
	gotDiags := diags.ForRPC()
	wantDiags := tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Warning,
			"Some objects will no longer be managed by Terraform",
			`If you apply this plan, Terraform will discard its tracking information for the following objects, but it will not delete them:
 - test_object.a

After applying this plan, Terraform will no longer manage these objects. You will need to import them into Terraform to manage them again.`,
		),
	}.ForRPC()
	if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
		t.Errorf("wrong diagnostics\n%s", diff)
	}

	t.Run(addrA.String(), func(t *testing.T) {
		instPlan := plan.Changes.ResourceInstanceDeposed(addrA, deposedKey)
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addrA)
		}

		if got, want := instPlan.Addr, addrA; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addrA; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Forget; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_removedResourceErrorStillInConfig(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {}

removed {
  from = test_object.a
  lifecycle {
    destroy = false
  }
}
`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"foo":"bar"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addrA,
		},
	})
	if !diags.HasErrors() {
		t.Fatalf("unexpected success; want error")
	}
	if got, want := diags.Err().Error(), "Removed resource still exists"; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot: %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Plan_removedModuleRemovesDeeplyNestedResources(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
removed {
  from = module.child
  lifecycle {
	destroy = false
  }
}
`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(mustResourceInstanceAddr("module.child.test_object.a"), &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"foo":"bar"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
		s.SetResourceInstanceCurrent(mustResourceInstanceAddr("module.child.module.grandchild.test_object.a"), &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"foo":"bar"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))

		// We'll also check it works for deeply nested deposed objects too.
		s.SetResourceInstanceDeposed(mustResourceInstanceAddr("module.child.module.grandchild.test_object.a"), "gone", &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"foo":"bar"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
	})
	if diags.HasErrors() {
		t.Errorf("expected no errors but got %s", diags.Err())
	}

	resources := []string{"module.child.test_object.a", "module.child.module.grandchild.test_object.a"}
	for _, resource := range resources {
		t.Run(resource, func(t *testing.T) {
			addr := mustResourceInstanceAddr(resource)

			instPlan := plan.Changes.ResourceInstance(addr)
			if instPlan == nil {
				t.Fatalf("no plan for %s at all", addr)
			}

			if got, want := instPlan.Addr, addr; !got.Equal(want) {
				t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := instPlan.PrevRunAddr, addr; !got.Equal(want) {
				t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := instPlan.Action, plans.Forget; got != want {
				t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := instPlan.ActionReason, plans.ResourceInstanceDeleteBecauseNoResourceConfig; got != want {
				t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
			}
		})
	}

	t.Run("module.child.module.grandchild.test_object.a (deposed)", func(t *testing.T) {
		addr := mustResourceInstanceAddr("module.child.module.grandchild.test_object.a")

		instPlan := plan.Changes.ResourceInstanceDeposed(addr, "gone")
		if instPlan == nil {
			t.Fatalf("no plan for %s at all", addr)
		}

		if got, want := instPlan.Addr, addr; !got.Equal(want) {
			t.Errorf("wrong current address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.PrevRunAddr, addr; !got.Equal(want) {
			t.Errorf("wrong previous run address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.Action, plans.Forget; got != want {
			t.Errorf("wrong planned action\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := instPlan.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
			t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestContext2Plan_removedModuleErrorStillInConfigNested(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "a" {
  source = "./mod"
}

removed {
  from = module.a.test_object.a
  lifecycle {
    destroy = false
  }
}
`,

		"mod/main.tf": `
resource "test_object" "a" {}
`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"foo":"bar"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addrA,
		},
	})
	if !diags.HasErrors() {
		t.Fatalf("unexpected success; want error")
	}
	if got, want := diags.Err().Error(), "Removed resource still exists"; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot: %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Plan_removedModuleErrorStillInConfig(t *testing.T) {
	addrA := mustResourceInstanceAddr("test_object.a")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "a" {
  source = "./mod"
}

removed {
  from = module.a
  lifecycle {
    destroy = false
  }
}
`,

		"mod/main.tf": `
resource "test_object" "a" {}
`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"foo":"bar"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		ForceReplace: []addrs.AbsResourceInstance{
			addrA,
		},
	})
	if !diags.HasErrors() {
		t.Fatalf("unexpected success; want error")
	}
	if got, want := diags.Err().Error(), "Removed module still exists"; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot: %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Plan_sensitiveInputVariableValue(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "boop" {
  type = string
  # this variable is not marked sensitive
}

resource "test_resource" "a" {
  value = var.boop
}

`,
	})

	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"value": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	// Build state with sensitive value in resource object
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_resource.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"value":"secret"}]}`),
			AttrSensitivePaths: []cty.Path{
				cty.GetAttrPath("value"),
			},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	// Create a sensitive-marked value for the input variable. This is not
	// possible through the normal CLI path, but is possible when the plan is
	// created and modified by the stacks runtime.
	secret := cty.StringVal("secret").Mark(marks.Sensitive)
	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"boop": &InputValue{
				Value:      secret,
				SourceType: ValueFromUnknown,
			},
		},
	})
	assertNoErrors(t, diags)
	for _, res := range plan.Changes.Resources {
		switch res.Addr.String() {
		case "test_resource.a":
			if res.Action != plans.NoOp {
				t.Errorf("unexpected %s change for %s", res.Action, res.Addr)
			}
		default:
			t.Errorf("unexpected %s change for %s", res.Action, res.Addr)
		}
	}
}
func TestContext2Plan_dataSourceSensitiveRead(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
data "test_data_source" "foo" {
}

resource "test_object" "obj" {
	test_string = data.test_data_source.foo.bar
}
`,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		DataSources: map[string]*configschema.Block{
			"test_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"bar": {
						Type:      cty.String,
						Computed:  true,
						Sensitive: true,
					},
				},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"test_object": simpleTestSchema(),
		},
	})

	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"bar": cty.StringVal("data_id"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables)))
	assertNoErrors(t, diags)

	ch := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.obj"))
	if len(ch.AfterSensitivePaths) == 0 {
		t.Fatal("expected marked values in test_object.obj")
	}
}

// When a schema declares that attributes nested within sets are sensitive, the
// resulting cty values will transfer those marks to the containing set. Verify
// that this does not present a change in the plan.
func TestContext2Plan_nestedSensitiveMarks(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "obj" {
  set_block {
    foo = "bar"
  }
}
`,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_object": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"set_block": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"foo": {
									Type:      cty.String,
									Sensitive: true,
									Optional:  true,
								},
							},
						},
					},
				},
			},
		},
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.obj"), &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"id":"z","set_block":[{"foo":"bar"}]}`),
			AttrSensitivePaths: []cty.Path{
				cty.GetAttrPath("set_block"),
			},
			Status: states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables)))
	assertNoErrors(t, diags)

	ch := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.obj"))
	if ch.Action != plans.NoOp {
		t.Fatal("expected no change in plan")
	}
}

func TestContext2Plan_ephemeralInResource(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					beep = {
						source = "terraform.io/builtin/beep"
					}
				}
			}

			variable "in" {
				type      = string
				ephemeral = true
			}

			resource "beep" "boop" {
				in = var.in
			}

			data "beep" "boop" {
				in = var.in
			}
		`,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"beep": {
				Attributes: map[string]*configschema.Attribute{
					"in": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
	})
	p.GetProviderSchemaResponse.DataSources = p.GetProviderSchemaResponse.ResourceTypes

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("beep"): testProviderFuncFixed(p),
		},
	})

	var wantDiags tfdiags.Diagnostics
	wantDiags = wantDiags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Invalid use of ephemeral value",
		Detail:   `Ephemeral values are not valid for "in", because it is not a write-only attribute and must be persisted to state.`,
		Subject: &hcl.Range{
			Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
			Start: hcl.Pos{
				Line: 16, Column: 10, Byte: 223,
			},
			End: hcl.Pos{
				Line: 16, Column: 16, Byte: 229,
			},
		},
	})
	wantDiags = wantDiags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Invalid use of ephemeral value",
		Detail:   `Ephemeral values are not valid for "in", because it is not a write-only attribute and must be persisted to state.`,
		Subject: &hcl.Range{
			Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
			Start: hcl.Pos{
				Line: 20, Column: 10, Byte: 269,
			},
			End: hcl.Pos{
				Line: 20, Column: 16, Byte: 275,
			},
		},
	})
	_, gotDiags := ctx.Plan(
		m, states.NewState(),
		SimplePlanOpts(plans.NormalMode, InputValues{
			"in": {
				Value:      cty.StringVal("hello"),
				SourceType: ValueFromCaller,
			},
		}),
	)
	// We'll use the "for RPC" representation just as a convenient shortcut
	// to not worry about exactly which diagnostic type Terraform Core chose
	// to return here.
	gotDiags = gotDiags.ForRPC()
	gotDiags.Sort()
	wantDiags = wantDiags.ForRPC()
	wantDiags.Sort()
	if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
		t.Errorf("wrong diagnostics\n%s", diff)
	}

}

func TestContext2Plan_ephemeralInProviderConfig(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					beep = {
						source = "terraform.io/builtin/beep"
					}
				}
			}

			variable "in" {
				type      = string
				ephemeral = true
			}

			provider "beep" {
				in = var.in
			}

			data "beep" "boop" {
			}
		`,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"in": {
					Type:     cty.String,
					Optional: true,
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"beep": {},
		},
	})
	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.EmptyObjectVal,
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("beep"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(
		m, states.NewState(),
		SimplePlanOpts(plans.NormalMode, InputValues{
			"in": {
				Value:      cty.StringVal("hello"),
				SourceType: ValueFromCaller,
			},
		}),
	)
	assertNoDiagnostics(t, diags)

	if !p.ConfigureProviderCalled {
		t.Fatal("ConfigureProvider was not called")
	}
	got := p.ConfigureProviderRequest.Config
	want := cty.ObjectVal(map[string]cty.Value{
		// The value is not marked here, because Terraform Core unmarks it
		// before calling the provider; ephemerality is Terraform Core's
		// concern to deal with, not the provider's.
		"in": cty.StringVal("hello"),
	})
	if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong provider configuration\n%s", diff)
	}
}

// This test explicitly reproduces the issue described in #34976.
func TestContext2Plan_34976(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
data "test_object" "obj" {}

module "a" {
  depends_on = [data.test_object.obj]
  source = "./mod"
}

output "value" {
  value = try(module.a.notreal, null)
}
`,
		"mod/main.tf": ``,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		DataSources: map[string]*configschema.Block{
			"test_object": {},
		},
	})
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {

		// Just very small delay that means the output will be processed before
		// the module.
		time.Sleep(50 * time.Millisecond)

		return providers.ReadDataSourceResponse{
			State: req.Config,
		}
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	// Just shouldn't crash.
	_, diags := ctx.Plan(m, states.NewState(), SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables)))
	assertNoErrors(t, diags)
}

func TestContext2Plan_sensitiveRequiredReplace(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "obj" {
	value = "changed"
}
`,
	})
	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_object": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:      cty.String,
							Required:  true,
							Sensitive: true,
						},
					},
				},
			},
		},
	}
	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal("changed"),
		}),
		RequiresReplace: []cty.Path{
			cty.GetAttrPath("value"),
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.obj"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: mustParseJson(map[string]any{
					"value": "original",
				}),
				AttrSensitivePaths: []cty.Path{
					cty.GetAttrPath("value"),
				},
				Status: states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	_, diags := ctx.Plan(m, state, nil)
	if len(diags) > 0 {
		t.Errorf("unexpected diags\n%s", diags)
	}
}

func TestContext2Plan_writeOnlyRequiredReplace(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "obj" {
	value = "changed"
}
`,
	})
	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_object": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:      cty.String,
							Required:  true,
							WriteOnly: true,
						},
					},
				},
			},
		},
	}
	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"value": cty.NullVal(cty.String),
		}),
		RequiresReplace: []cty.Path{
			cty.GetAttrPath("value"),
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	rAddr := mustResourceInstanceAddr("test_object.obj")
	pAddr := mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`)
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			rAddr,
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: mustParseJson(map[string]any{
					"value": nil,
				}),
				Status: states.ObjectReady,
			},
			pAddr)
	})

	plan, diags := ctx.Plan(m, state, nil)
	assertNoErrors(t, diags)

	if plan.Changes.Empty() {
		t.Fatal("unexpected empty plan")
	}
	expectedChanges := &plans.Changes{
		Resources: []*plans.ResourceInstanceChange{
			{
				Addr:         rAddr,
				PrevRunAddr:  rAddr,
				ProviderAddr: pAddr,
				Change: plans.Change{
					Action: plans.DeleteThenCreate,
					Before: cty.ObjectVal(map[string]cty.Value{
						"value": cty.NullVal(cty.String),
					}),
					After: cty.ObjectVal(map[string]cty.Value{
						"value": cty.NullVal(cty.String),
					}),
				},
				ActionReason:    plans.ResourceInstanceReplaceBecauseCannotUpdate,
				RequiredReplace: cty.NewPathSet(cty.GetAttrPath("value")),
			},
		},
	}

	schemas, schemaDiags := ctx.Schemas(m, plan.PriorState)
	assertNoDiagnostics(t, schemaDiags)

	changes, err := plan.Changes.Decode(schemas)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectedChanges, changes, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("unexpected changes: %s", diff)
	}
}

func TestContext2Plan_writeOnlyRequiredReplace_createBeforeDestroy(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "obj" {
	value = "changed"
	lifecycle {
		create_before_destroy = true
	}
}
`,
	})
	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_object": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:      cty.String,
							Required:  true,
							WriteOnly: true,
						},
					},
				},
			},
		},
	}
	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"value": cty.NullVal(cty.String),
		}),
		RequiresReplace: []cty.Path{
			cty.GetAttrPath("value"),
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	rAddr := mustResourceInstanceAddr("test_object.obj")
	pAddr := mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`)
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			rAddr,
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: mustParseJson(map[string]any{
					"value": nil,
				}),
				Status: states.ObjectReady,
			},
			pAddr)
	})

	plan, diags := ctx.Plan(m, state, nil)
	assertNoErrors(t, diags)

	if plan.Changes.Empty() {
		t.Fatal("unexpected empty plan")
	}
	expectedChanges := &plans.Changes{
		Resources: []*plans.ResourceInstanceChange{
			{
				Addr:         rAddr,
				PrevRunAddr:  rAddr,
				ProviderAddr: pAddr,
				Change: plans.Change{
					Action: plans.CreateThenDelete,
					Before: cty.ObjectVal(map[string]cty.Value{
						"value": cty.NullVal(cty.String),
					}),
					After: cty.ObjectVal(map[string]cty.Value{
						"value": cty.NullVal(cty.String),
					}),
				},
				ActionReason:    plans.ResourceInstanceReplaceBecauseCannotUpdate,
				RequiredReplace: cty.NewPathSet(cty.GetAttrPath("value")),
			},
		},
	}

	schemas, schemaDiags := ctx.Schemas(m, plan.PriorState)
	assertNoDiagnostics(t, schemaDiags)

	changes, err := plan.Changes.Decode(schemas)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectedChanges, changes, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("unexpected changes: %s", diff)
	}
}

func TestContext2Plan_selfReferences(t *testing.T) {
	tcs := []struct {
		attribute string
	}{
		// Note here, the type returned by the lookup doesn't really matter as
		// we should safely fail before we even get to type checking.
		{
			attribute: "count = test_object.a[0].test_string",
		},
		{
			attribute: "count = test_object.a[*].test_string",
		},
		{
			attribute: "for_each = test_object.a[0].test_string",
		},
		{
			attribute: "for_each = test_object.a[*].test_string",
		},
		// Even though the can and try functions might normally allow some
		// fairly crazy things, we're still going to put a stop to a self
		// reference since it is more akin to a compilation error than some kind
		// of dynamic exception.
		{
			attribute: "for_each = can(test_object.a[0].test_string) ? 0 : 1",
		},
		{
			attribute: "count = try(test_object.a[0].test_string, 0)",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.attribute, func(t *testing.T) {
			tmpl := `
resource "test_object" "a" {
  %%attribute%%
}
`
			module := strings.ReplaceAll(tmpl, "%%attribute%%", tc.attribute)
			m := testModuleInline(t, map[string]string{
				"main.tf": module,
			})

			p := simpleMockProvider()
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					// The providers never actually going to get called here, we should
					// catch the error long before anything happens.
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
				},
			})

			_, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
			if len(diags) != 1 {
				t.Fatalf("expected one diag, got %d: %s", len(diags), diags.ErrWithWarnings())
			}

			got, want := diags.Err().Error(), "Self-referential block: Configuration for test_object.a may not refer to itself."
			if cmp.Diff(want, got) != "" {
				t.Fatalf("unexpected error\n%s", cmp.Diff(want, got))
			}
		})
	}

}

func TestContext2Plan_destroySkipsVariableValidations(t *testing.T) {
	// this validation cannot block destroy, because we can't be sure arbitrary
	// expressions can be evaluated at all during destroy.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "input" {
  type = string

  validation {
    condition = var.input == "foo"
    error_message = "bad input"
  }
}

resource "test_object" "a" {
  test_string = var.input
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.BuildState(func(state *states.SyncState) {
		state.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.a"),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"test_string":"foo"}`),
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	}), &PlanOpts{
		Mode: plans.DestroyMode,
		SetVariables: InputValues{
			"input": {
				Value:       cty.StringVal("foo"),
				SourceType:  ValueFromCLIArg,
				SourceRange: tfdiags.SourceRange{},
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("expected no errors, but got %s", diags.ErrWithWarnings())
	}

	planResult := plan.Checks.GetObjectResult(addrs.AbsInputVariableInstance{
		Variable: addrs.InputVariable{
			Name: "input",
		},
		Module: addrs.RootModuleInstance,
	})

	if planResult != nil && planResult.Status != checks.StatusUnknown {
		// checks should not have been evaluated, because the variable is not required for destroy.
		t.Errorf("expected checks to be pass but was %s", planResult.Status)
	}
}

func TestContext2Plan_orphanOutput(t *testing.T) {
	// ensure the planned replacement of the data source is evaluated properly
	m := testModuleInline(t, map[string]string{
		"main.tf": `
output "staying" {
  value = "foo"
}
`,
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(mustAbsOutputValue("output.old"), cty.StringVal("old_value"), false)
		s.SetOutputValue(mustAbsOutputValue("output.staying"), cty.StringVal("foo"), false)
	})

	ctx := testContext2(t, &ContextOpts{})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)

	expectedChanges := &plans.Changes{
		Outputs: []*plans.OutputChange{
			{
				Addr: mustAbsOutputValue("output.old"),
				Change: plans.Change{
					Action: plans.Delete,
					Before: cty.StringVal("old_value"),
					After:  cty.NullVal(cty.DynamicPseudoType),
				},
			},
			{
				Addr: mustAbsOutputValue("output.staying"),
				Change: plans.Change{
					Action: plans.NoOp,
					Before: cty.StringVal("foo"),
					After:  cty.StringVal("foo"),
				},
			},
		},
	}
	changes, err := plan.Changes.Decode(nil)
	if err != nil {
		t.Fatal(err)
	}

	sort.SliceStable(changes.Outputs, func(i, j int) bool {
		return changes.Outputs[i].Addr.String() < changes.Outputs[j].Addr.String()
	})

	if diff := cmp.Diff(expectedChanges, changes, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("unexpected changes: %s", diff)
	}
}

func TestContext2Plan_multiInstanceSelfRef(t *testing.T) {
	// The postcondition here references self, but because instances are
	// processed concurrently some instances may not be registered yet during
	// evaluation. This should still evaluate without error, because we know our
	// self value exists.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_resource" "test" {
}

data "test_data_source" "foo" {
  count = 100
  lifecycle {
    postcondition {
      condition = self.attr == null
      error_message = "error"
    }
  }
  depends_on = [test_resource.test]
}
`,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		DataSources: map[string]*configschema.Block{
			"test_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, states.NewState(), SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables)))
	assertNoErrors(t, diags)
}

func TestContext2Plan_upgradeState_WriteOnlyAttribute(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "aws_instance" "foo" {
	wo_attr = "value"
}
`,
	})

	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"wo_attr": {
						Type:      cty.String,
						Optional:  true,
						WriteOnly: true,
					},
				},
			},
		},
	})
	p.UpgradeResourceStateFn = func(r providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
		return providers.UpgradeResourceStateResponse{
			UpgradedState: cty.ObjectVal(map[string]cty.Value{
				"wo_attr": cty.StringVal("not-empty"),
			}),
		}
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	priorState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("aws_instance.foo"),
			&states.ResourceInstanceObjectSrc{
				// The UpgradeResourceStateFn above does not care about specific prior
				// state but it must not be empty for the function to be actually called
				AttrsJSON: []byte(`{"wo_attr":null}`),
				Status:    states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		)
	})

	// Plan should invoke state upgrade logic and trigger validation, given the mocks above
	_, diags := ctx.Plan(m, priorState, DefaultPlanOpts)
	if !diags.HasErrors() {
		t.Fatalf("expected errors but got none")
	}

	if got, want := diags.Err().Error(), "Invalid resource state upgrade"; !strings.Contains(got, want) {
		t.Errorf("unexpected error message\ngot: %s\nwant substring: %s", got, want)
	}
}

func TestContext2Plan_orphanUpdateInstance(t *testing.T) {
	// ean orphaned instance should still reflect the refreshed state in the plan
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  for_each = {}
}
`,
	})

	p := simpleMockProvider()
	p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
		state := req.PriorState.AsValueMap()
		state["test_string"] = cty.StringVal("new")
		resp.NewState = cty.ObjectVal(state)
		return resp
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(mustResourceInstanceAddr(`test_object.a["old"]`), &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"test_string":"old"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)

	resourceType := p.GetProviderSchemaResponse.ResourceTypes["test_object"].Block.ImpliedType()
	change, err := plan.Changes.ResourceInstance(mustResourceInstanceAddr(`test_object.a["old"]`)).Decode(resourceType)
	if err != nil {
		t.Fatal(err)
	}
	if change.Before.GetAttr("test_string").AsString() != "new" {
		t.Fatalf("resource before value not refreshed in plan: %#v\n", change.Before)
	}
}
