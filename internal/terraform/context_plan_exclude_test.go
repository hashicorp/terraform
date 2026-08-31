// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"
	"sync"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContext2Plan_excludes(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "excluded" {
  arg = "excluded"
}

resource "test_object" "foo" {
  arg = "bar"
}
`,
	})

	p := excludePlanTestProvider()

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.excluded"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"exc-id","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.foo"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo-id","arg":"changeme","computed":"c"}`),
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

	excludeAddr := mustResourceInstanceAddr("test_object.excluded")
	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:            plans.NormalMode,
		DeferralAllowed: true,
		Excludes:        []addrs.Targetable{excludeAddr},
	})
	tfdiags.AssertNoErrors(t, diags)

	assertDeferredResource(t, plan, "test_object.excluded", providers.DeferredReasonExcluded)
	assertPlannedChange(t, plan, "test_object.foo", plans.Update)
}

func TestContext2Plan_excludes_resource_expansion(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "excluded" {
  for_each = { a = "val-a", b = "val-b" }
  arg      = each.value
}

resource "test_object" "foo" {
  arg = "foo"
}
`,
	})

	p := excludePlanTestProvider()

	var mu sync.Mutex
	plannedAddrs := map[string]bool{}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		if !req.ProposedNewState.IsNull() {
			if arg := req.ProposedNewState.GetAttr("arg"); arg.IsKnown() && !arg.IsNull() {
				mu.Lock()
				plannedAddrs[arg.AsString()] = true
				mu.Unlock()
			}
		}
		return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr(`test_object.excluded["a"]`),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"a-id","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr(`test_object.excluded["b"]`),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"b-id","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.foo"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo-id","arg":"changeme","computed":"c"}`),
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

	// Exclude the entire resource (no instance key)
	excludeAddr := mustAbsResourceAddr("test_object.excluded")
	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:            plans.NormalMode,
		DeferralAllowed: true,
		Excludes:        []addrs.Targetable{excludeAddr},
	})
	tfdiags.AssertNoErrors(t, diags)

	assertDeferredResource(t, plan, `test_object.excluded["a"]`, providers.DeferredReasonExcluded)
	assertDeferredResource(t, plan, `test_object.excluded["b"]`, providers.DeferredReasonExcluded)
	assertPlannedChange(t, plan, "test_object.foo", plans.Update)

	if plannedAddrs["val-a"] || plannedAddrs["val-b"] {
		t.Error("excluded resource instances should not have been planned")
	}
	if !plannedAddrs["foo"] {
		t.Error("test_object.foo should have been planned")
	}
}

func TestContext2Plan_excludes_resource_instance(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "excluded" {
  for_each = { a = "excluded", b = "included" }
  arg      = each.value
}

resource "test_object" "foo" {
  arg = "foo"
}
`,
	})

	p := excludePlanTestProvider()

	var mu sync.Mutex
	plannedAddrs := map[string]bool{}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		if !req.ProposedNewState.IsNull() {
			if arg := req.ProposedNewState.GetAttr("arg"); arg.IsKnown() && !arg.IsNull() {
				mu.Lock()
				plannedAddrs[arg.AsString()] = true
				mu.Unlock()
			}
		}
		return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr(`test_object.excluded["a"]`),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"a-id","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr(`test_object.excluded["b"]`),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"b-id","arg":"changeme","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.foo"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo-id","arg":"changeme","computed":"c"}`),
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

	excludeAddr := mustResourceInstanceAddr(`test_object.excluded["a"]`)
	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:            plans.NormalMode,
		DeferralAllowed: true,
		Excludes:        []addrs.Targetable{excludeAddr},
	})
	tfdiags.AssertNoErrors(t, diags)

	assertDeferredResource(t, plan, `test_object.excluded["a"]`, providers.DeferredReasonExcluded)
	assertPlannedChange(t, plan, `test_object.excluded["b"]`, plans.Update)
	assertPlannedChange(t, plan, "test_object.foo", plans.Update)

	if plannedAddrs["excluded"] {
		t.Error("excluded resource instances should not have been planned")
	}
	if !plannedAddrs["included"] {
		t.Error(`test_object.excluded["b"] should have been planned`)
	}
	if !plannedAddrs["foo"] {
		t.Error("test_object.foo should have been planned")
	}
}

func TestContext2Plan_excludes_module(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "excluded" {
  source = "./child"
  input = "excluded"
}

module "foo" {
  source = "./child"
  input = "foo"
}
`,
		"child/main.tf": `
variable "input" {
  type = string
}
resource "test_object" "r" {
  arg = var.input
}
`,
	})

	p := excludePlanTestProvider()

	var mu sync.Mutex
	plannedAddrs := map[string]bool{}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		if !req.ProposedNewState.IsNull() {
			if arg := req.ProposedNewState.GetAttr("arg"); arg.IsKnown() && !arg.IsNull() {
				mu.Lock()
				plannedAddrs[arg.AsString()] = true
				mu.Unlock()
			}
		}
		return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("module.excluded.test_object.r"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"excl-r-id","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("module.foo.test_object.r"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo-r-id","arg":"changeme","computed":"c"}`),
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

	excludeAddr := mustModuleInstance("module.excluded")
	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:            plans.NormalMode,
		DeferralAllowed: true,
		Excludes:        []addrs.Targetable{excludeAddr},
	})
	tfdiags.AssertNoErrors(t, diags)

	assertDeferredResource(t, plan, "module.excluded.test_object.r", providers.DeferredReasonExcluded)
	assertPlannedChange(t, plan, "module.foo.test_object.r", plans.Update)

	if plannedAddrs["excluded"] {
		t.Error("excluded resource instances should not have been planned")
	}
	if !plannedAddrs["foo"] {
		t.Error("test_object.foo should have been planned")
	}
}

func TestContext2Plan_excludes_module_expansion(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "excluded" {
  count  = 2
  input = "excluded-${count.index}"
  source = "./child"
}

resource "test_object" "foo" {
  arg = "foo"
}
`,
		"child/main.tf": `
variable "input" {
  type = string
}
resource "test_object" "child_obj" {
  arg = var.input
}
`,
	})

	p := excludePlanTestProvider()

	var mu sync.Mutex
	plannedAddrs := map[string]bool{}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		if !req.ProposedNewState.IsNull() {
			if arg := req.ProposedNewState.GetAttr("arg"); arg.IsKnown() && !arg.IsNull() {
				mu.Lock()
				plannedAddrs[arg.AsString()] = true
				mu.Unlock()
			}
		}
		return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("module.excluded[0].test_object.child_obj"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"excl-0-id","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("module.excluded[1].test_object.child_obj"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"excl-1-id","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.foo"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo-id","arg":"changeme","computed":"c"}`),
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

	excludeAddr := mustModuleInstance("module.excluded")
	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:            plans.NormalMode,
		DeferralAllowed: true,
		Excludes:        []addrs.Targetable{excludeAddr},
	})
	tfdiags.AssertNoErrors(t, diags)

	assertDeferredResource(t, plan, "module.excluded[0].test_object.child_obj", providers.DeferredReasonExcluded)
	assertDeferredResource(t, plan, "module.excluded[1].test_object.child_obj", providers.DeferredReasonExcluded)
	assertPlannedChange(t, plan, "test_object.foo", plans.Update)

	if plannedAddrs["excluded-0"] || plannedAddrs["excluded-1"] {
		t.Error("excluded resource instances should not have been planned")
	}
	if !plannedAddrs["foo"] {
		t.Error("test_object.foo should have been planned")
	}
}

func TestContext2Plan_excludes_module_instance(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "partially_excluded" {
  count  = 3
  input = count.index == 1 ? "excluded-1" : "included-${count.index}"
  source = "./child"
}

resource "test_object" "foo" {
  arg = "foo"
}
`,
		"child/main.tf": `
variable "input" {
  type = string
}
resource "test_object" "child_obj" {
  arg = var.input
}
`,
	})

	p := excludePlanTestProvider()

	var mu sync.Mutex
	plannedAddrs := map[string]bool{}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		if !req.ProposedNewState.IsNull() {
			if arg := req.ProposedNewState.GetAttr("arg"); arg.IsKnown() && !arg.IsNull() {
				mu.Lock()
				plannedAddrs[arg.AsString()] = true
				mu.Unlock()
			}
		}
		return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("module.partially_excluded[0].test_object.child_obj"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"excl-0-id","arg":"changeme","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("module.partially_excluded[1].test_object.child_obj"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"excl-1-id","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("module.partially_excluded[2].test_object.child_obj"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"excl-1-id","arg":"changeme","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.foo"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo-id","arg":"changeme","computed":"c"}`),
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

	excludeAddr := mustModuleInstance("module.partially_excluded[1]")
	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:            plans.NormalMode,
		DeferralAllowed: true,
		Excludes:        []addrs.Targetable{excludeAddr},
	})
	tfdiags.AssertNoErrors(t, diags)

	assertDeferredResource(t, plan, "module.partially_excluded[1].test_object.child_obj", providers.DeferredReasonExcluded)
	assertPlannedChange(t, plan, "module.partially_excluded[0].test_object.child_obj", plans.Update)
	assertPlannedChange(t, plan, "module.partially_excluded[2].test_object.child_obj", plans.Update)
	assertPlannedChange(t, plan, "test_object.foo", plans.Update)

	if plannedAddrs["excluded-1"] {
		t.Error("excluded resource instances should not have been planned")
	}
	if !plannedAddrs["included-0"] {
		t.Error("module.excluded[0].test_object.child_obj should have been planned")
	}
	if !plannedAddrs["included-2"] {
		t.Error("module.excluded[2].test_object.child_obj should have been planned")
	}
	if !plannedAddrs["foo"] {
		t.Error("test_object.foo should have been planned")
	}
}

func TestContext2Plan_excludes_dependencies(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "excluded" {
  arg = "excluded"
}

resource "test_object" "dependent_1" {
  arg        = "excluded-dependent-1"
  depends_on = [test_object.excluded]
}

resource "test_object" "dependent_2" {
  arg = test_object.dependent_1.id
}

resource "test_object" "foo" {
  arg = "foo"
}
`,
	})

	p := excludePlanTestProvider()

	var mu sync.Mutex
	plannedAddrs := map[string]bool{}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		if !req.ProposedNewState.IsNull() {
			if arg := req.ProposedNewState.GetAttr("arg"); arg.IsKnown() && !arg.IsNull() {
				mu.Lock()
				plannedAddrs[arg.AsString()] = true
				mu.Unlock()
			}
		}
		return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.excluded"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"exc-id","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.dependent_1"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"excluded-dependent-2","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.dependent_2"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"dep-2-id","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.foo"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"ind-id","arg":"changeme","computed":"c"}`),
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

	excludeAddr := mustResourceInstanceAddr("test_object.excluded")
	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:            plans.NormalMode,
		DeferralAllowed: true,
		Excludes:        []addrs.Targetable{excludeAddr},
	})
	tfdiags.AssertNoErrors(t, diags)

	assertDeferredResource(t, plan, "test_object.excluded", providers.DeferredReasonExcluded)
	assertDeferredResource(t, plan, "test_object.dependent_1", providers.DeferredReasonExcludedPrereq)
	assertDeferredResource(t, plan, "test_object.dependent_2", providers.DeferredReasonExcludedPrereq)
	assertPlannedChange(t, plan, "test_object.foo", plans.Update)

	if plannedAddrs["excluded"] || plannedAddrs["excluded-dependent-1"] || plannedAddrs["excluded-dependent-2"] {
		t.Error("excluded resource instances should not have been planned")
	}
	if !plannedAddrs["foo"] {
		t.Error("test_object.foo should have been planned")
	}
}

func TestContext2Plan_excludes_validation_errors(t *testing.T) {

	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "foo"
}
`,
	})

	testAddr := mustResourceInstanceAddr("test_object.a")
	testCases := map[string]struct {
		opts    *PlanOpts
		wantErr string
	}{
		"excludes-with-target": {
			opts: &PlanOpts{
				Mode:            plans.NormalMode,
				DeferralAllowed: true,
				Excludes:        []addrs.Targetable{testAddr},
				Targets:         []addrs.Targetable{testAddr},
			},
			wantErr: "The resource targeting options -target and -exclude cannot be combined as they are mutually exclusive. This is a bug in Terraform.",
		},
		"skip refresh": {
			opts: &PlanOpts{
				Mode:            plans.NormalMode,
				DeferralAllowed: false,
				Excludes:        []addrs.Targetable{testAddr},
			},
			wantErr: "The resource targeting option -exclude must be combined with -allow-deferral as excluding resources will create a partial plan. This is a bug in Terraform.",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			p := excludePlanTestProvider()
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
				},
			})

			_, diags := ctx.Plan(m, states.NewState(), tc.opts)
			if !diags.HasErrors() {
				t.Fatal("expected an error but got none")
			}
			if got := diags.Err().Error(); !strings.Contains(got, tc.wantErr) {
				t.Fatalf("wrong error\n got: %s\nwant: %s", got, tc.wantErr)
			}
		})
	}
}

func TestContext2Plan_excludes_output_evaluation(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "excluded" {
  arg = "excluded"
}

resource "test_object" "foo" {
  arg = "foo"
}

output "from_excluded" {
  value = test_object.excluded.arg
}

output "from_foo" {
  value = test_object.foo.arg
}
`,
	})

	p := excludePlanTestProvider()

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.excluded"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"exc-id","arg":"dontchange","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.foo"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo-id","arg":"changeme","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetOutputValue(mustAbsOutputValue(`output.from_excluded`), cty.StringVal("dontchange"), false)
		s.SetOutputValue(mustAbsOutputValue(`output.from_foo`), cty.StringVal("changeme"), false)
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	excludeAddr := mustResourceInstanceAddr("test_object.excluded")
	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:            plans.NormalMode,
		DeferralAllowed: true,
		Excludes:        []addrs.Targetable{excludeAddr},
	})
	tfdiags.AssertNoErrors(t, diags)
	assertDeferredResource(t, plan, "test_object.excluded", providers.DeferredReasonExcluded)
	assertPlannedChange(t, plan, "test_object.foo", plans.Update)

	fooOutput := plan.Changes.OutputValue(mustAbsOutputValue(`output.from_foo`))
	excludedOutput := plan.Changes.OutputValue(mustAbsOutputValue(`output.from_excluded`))
	if fooOutput.Action != plans.Update {
		t.Errorf("output.from_foo should have a planned change (update), got, %s", fooOutput.Action)
	}
	if excludedOutput.Action != plans.NoOp {
		t.Errorf("output.from_foo should have a planned change (no-op), got, %s", excludedOutput.Action)
	}
}

func TestContext2Plan_excludes_orphan(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "foo" {
  arg = "bar"
}
`,
	})

	p := excludePlanTestProvider()

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.orphan_excluded"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"exc-id","arg":"orphan-excluded","computed":"c"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.foo"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo-id","arg":"changeme","computed":"c"}`),
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

	excludeAddr := mustResourceInstanceAddr("test_object.orphan_excluded")
	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:            plans.NormalMode,
		DeferralAllowed: true,
		Excludes:        []addrs.Targetable{excludeAddr},
	})
	tfdiags.AssertNoErrors(t, diags)

	assertDeferredResource(t, plan, "test_object.orphan_excluded", providers.DeferredReasonExcluded)
	assertPlannedChange(t, plan, "test_object.foo", plans.Update)
}

func excludePlanTestProvider() *testing_provider.MockProvider {
	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"test_object": {
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"id":       {Type: cty.String, Computed: true},
							"arg":      {Type: cty.String, Optional: true},
							"computed": {Type: cty.String, Computed: true},
						},
					},
				},
			},
		},

		// These provider methods assert that any excluded resources (identified via the arg attribute)
		// are not refreshed or planned, which differs from the behavior of other deferral reasons.
		ReadResourceFn: func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
			if arg := req.PriorState.GetAttr("arg"); strings.Contains(arg.AsString(), "excluded") {
				return providers.ReadResourceResponse{
					Diagnostics: tfdiags.Diagnostics{
						tfdiags.Sourceless(tfdiags.Error,
							"Excluded resource refreshed",
							"A resource that was expected to be excluded was refreshed",
						),
					},
				}
			}
			return providers.ReadResourceResponse{NewState: req.PriorState}
		},
		PlanResourceChangeFn: func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			if !req.ProposedNewState.IsNull() {
				if arg := req.ProposedNewState.GetAttr("arg"); strings.Contains(arg.AsString(), "excluded") {
					return providers.PlanResourceChangeResponse{
						Diagnostics: tfdiags.Diagnostics{
							tfdiags.Sourceless(tfdiags.Error,
								"Excluded resource planned",
								"A resource that was expected to be excluded was planned",
							),
						},
					}
				}
			}
			return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
		},
	}
	return p
}

func assertDeferredResource(t *testing.T, plan *plans.Plan, addrStr string, wantReason providers.DeferredReason) {
	t.Helper()
	addr := mustResourceInstanceAddr(addrStr)
	for _, d := range plan.DeferredResources {
		if d.ChangeSrc.Addr.Equal(addr) {
			if d.DeferredReason != wantReason {
				t.Errorf("deferred resource %s: got reason %q, want %q", addrStr, d.DeferredReason, wantReason)
			}
			return
		}
	}
	t.Errorf("expected %s to be deferred, but it was not found in plan.DeferredResources", addrStr)
}

func assertPlannedChange(t *testing.T, plan *plans.Plan, addrStr string, wantAction plans.Action) {
	t.Helper()

	gotChange := plan.Changes.ResourceInstance(mustResourceInstanceAddr(addrStr))
	if gotChange == nil {
		t.Errorf("expected %s to have a planned change (%s) but it was not found in plan.Changes", addrStr, wantAction)
		return
	}
	if gotChange.Action != wantAction {
		t.Errorf("%s: got action %s, want %s", addrStr, gotChange.Action, wantAction)
	}
}
