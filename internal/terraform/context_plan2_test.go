package terraform

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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

	p := new(MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
			AttrSensitivePaths: []cty.PathValueMarks{
				{
					Path:  cty.GetAttrPath("foo"),
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
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

	p := new(MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
				`Resource instances in your current state have moved to new addresses in the latest configuration. Terraform must include those resource instances while planning in order to ensure a correct result, but your -target=... options to not fully cover all of those resource instances.

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
				`Resource instances in your current state have moved to new addresses in the latest configuration. Terraform must include those resource instances while planning in order to ensure a correct result, but your -target=... options to not fully cover all of those resource instances.

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
				`Resource instances in your current state have moved to new addresses in the latest configuration. Terraform must include those resource instances while planning in order to ensure a correct result, but your -target=... options to not fully cover all of those resource instances.

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

		// We're semi-abusing "ForRPC" here just to get diagnostics that are
		// more easily comparable than the various different diagnostics types
		// tfdiags uses internally. The RPC-friendly diagnostics are also
		// comparison-friendly, by discarding all of the dynamic type information.
		gotDiags := diags.ForRPC()
		wantDiags := tfdiags.Diagnostics{
			// Still get the warning about the -target option...
			tfdiags.Sourceless(
				tfdiags.Warning,
				"Resource targeting is in effect",
				`You are creating a plan with the -target option, which means that the result of this plan may not represent all of the changes requested by the current configuration.

The -target option is not for routine use, and is provided only for exceptional situations such as recovering from errors or mistakes, or when Terraform specifically suggests to use it as part of an error message.`,
			),
			// ...but now we have no error about test_object.a
		}.ForRPC()

		if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
			t.Errorf("wrong diagnostics\n%s", diff)
		}
	})
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

	p := new(MockProvider)
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp.PlannedState = cty.ObjectVal(map[string]cty.Value{
			"sensitive": cty.UnknownVal(cty.String),
		})
		return resp
	}
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
			AttrSensitivePaths: []cty.PathValueMarks{
				{
					Path:  cty.GetAttrPath("foo"),
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
			},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_instance.bar").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"sensitive":"old"}`),
			AttrSensitivePaths: []cty.PathValueMarks{
				{
					Path:  cty.GetAttrPath("sensitive"),
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
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
			t.Errorf("unexpected %s plan for %s", res.Action, res.Addr)
		}
	}
}
