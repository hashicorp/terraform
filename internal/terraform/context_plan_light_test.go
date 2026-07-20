// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContext2Plan_light(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "noop" {
  arg = "same"
}

resource "test_object" "changed" {
  arg = "new"
}
`,
	})

	p := lightPlanTestProvider(0)

	var mu sync.Mutex
	refreshedIDs := map[string]bool{}
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		mu.Lock()
		defer mu.Unlock()
		if id := req.PriorState.GetAttr("id"); !id.IsNull() {
			refreshedIDs[id.AsString()] = true
		}
		return providers.ReadResourceResponse{NewState: req.PriorState}
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.noop"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"noop","arg":"same","computed":"boop"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.changed"),
			&states.ResourceInstanceObjectSrc{
				// Will prompt a refresh since the config value (arg) has changed
				AttrsJSON: []byte(`{"id":"changed","arg":"old","computed":"boop"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	hook := &testHook{}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		Hooks: []Hook{hook},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:      plans.NormalMode,
		PlanLight: true,
	})
	tfdiags.AssertNoErrors(t, diags)

	if refreshedIDs["noop"] {
		t.Error("test_object.noop should not have been refreshed")
	}
	if !refreshedIDs["changed"] {
		t.Error("test_object.changed should have been refreshed")
	}

	noop := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.noop"))
	if got, want := noop.Action, plans.NoOp; got != want {
		t.Errorf("test_object.noop: wrong plan action - got: %s, want: %s", got, want)
	}
	changed := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.changed"))
	if got, want := changed.Action, plans.Update; got != want {
		t.Errorf("test_object.changed: wrong plan action - got: %s, want: %s", got, want)
	}

	// Assert that the correct hooks are called + no duplicates
	wantHookCalls := []*testHookCall{
		{"PreRefresh", "test_object.changed"},
		{"PostRefresh", "test_object.changed"},
		{"PreDiff", "test_object.changed"},
		{"PostDiff", "test_object.changed"},
		{"PreDiff", "test_object.noop"},
		{"PostDiff", "test_object.noop"},
	}

	sortHookCalls := cmpopts.SortSlices(func(a, b *testHookCall) bool {
		if a.InstanceID == b.InstanceID {
			return a.Action > b.Action
		}
		return a.InstanceID > b.InstanceID
	})

	if diff := cmp.Diff(wantHookCalls, hook.Calls, sortHookCalls); diff != "" {
		t.Errorf("wrong hook events\n%s", diff)
	}
}

func TestContext2Plan_light_for_each(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  for_each = {
    noop    = "same"
    changed = "new"
  }
  arg = each.value
}
`,
	})

	p := lightPlanTestProvider(0)

	var mu sync.Mutex
	refreshedIDs := map[string]bool{}
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		mu.Lock()
		defer mu.Unlock()
		if id := req.PriorState.GetAttr("id"); !id.IsNull() {
			refreshedIDs[id.AsString()] = true
		}
		return providers.ReadResourceResponse{NewState: req.PriorState}
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr(`test_object.a["noop"]`),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"noop","arg":"same","computed":"boop"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr(`test_object.a["changed"]`),
			&states.ResourceInstanceObjectSrc{
				// Will prompt a refresh since the config value (arg) has changed
				AttrsJSON: []byte(`{"id":"changed","arg":"old","computed":"boop"}`),
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
		Mode:      plans.NormalMode,
		PlanLight: true,
	})
	tfdiags.AssertNoErrors(t, diags)

	if refreshedIDs["noop"] {
		t.Error(`test_object.a["noop"] should not have been refreshed`)
	}
	if !refreshedIDs["changed"] {
		t.Error(`test_object.a["changed"] should have been refreshed`)
	}

	noop := plan.Changes.ResourceInstance(mustResourceInstanceAddr(`test_object.a["noop"]`))
	if got, want := noop.Action, plans.NoOp; got != want {
		t.Errorf(`test_object.a["noop"]: wrong plan action - got: %s, want: %s`, got, want)
	}
	changed := plan.Changes.ResourceInstance(mustResourceInstanceAddr(`test_object.a["changed"]`))
	if got, want := changed.Action, plans.Update; got != want {
		t.Errorf(`test_object.a["changed"]: wrong plan action - got: %s, want: %s`, got, want)
	}
}

func TestContext2Plan_light_provider_update_will_refresh(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "foo"
}
`,
	})

	p := lightPlanTestProvider(0)
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		// The provider always plans "computed" as "forced", which differs from the
		// value in state, producing a change even though "arg" is unchanged in the config.
		planned := req.ProposedNewState.AsValueMap()
		planned["computed"] = cty.StringVal("forced")
		return providers.PlanResourceChangeResponse{PlannedState: cty.ObjectVal(planned)}
	}
	state := lightPlanTestState(t, `{"id":"a","arg":"foo","computed":"old"}`, 0, false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:      plans.NormalMode,
		PlanLight: true,
	})
	tfdiags.AssertNoErrors(t, diags)

	if !p.ReadResourceCalled {
		t.Fatal(`Expected a call to ReadResource but received none. The resource in this test should be refreshed with ` +
			`the -light flag as the provider produced a change.`)
	}

	change := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.a"))
	if got, want := change.Action, plans.Update; got != want {
		t.Fatalf("wrong plan action - got: %s, want: %s", got, want)
	}
}

func TestContext2Plan_light_schema_upgrade_will_refresh(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "foo"
}
`,
	})

	// Provider schema is at version 1, but the stored state is at version 0, so
	// reading the state performs a schema version upgrade, which will prompt a refresh
	p := lightPlanTestProvider(1)
	state := lightPlanTestState(t, `{"id":"a","arg":"foo","computed":"boop"}`, 0, false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:      plans.NormalMode,
		PlanLight: true,
	})
	tfdiags.AssertNoErrors(t, diags)

	if !p.UpgradeResourceStateCalled {
		t.Fatal(`Expected a call to UpgradeResourceState but received none.`)
	}
	if !p.ReadResourceCalled {
		t.Fatal(`Expected a call to ReadResource but received none. The resource in this test should be refreshed with ` +
			`the -light flag as the provider schema version was upgraded.`)
	}

	change := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.a"))
	if got, want := change.Action, plans.NoOp; got != want {
		t.Fatalf("wrong plan action - got: %s, want: %s", got, want)
	}
}

func TestContext2Plan_light_ignore_changes_noop(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "new"
  lifecycle {
    ignore_changes = [arg]
  }
}
`,
	})

	p := lightPlanTestProvider(0)
	state := lightPlanTestState(t, `{"id":"a","arg":"old","computed":"boop"}`, 0, false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:      plans.NormalMode,
		PlanLight: true,
	})
	tfdiags.AssertNoErrors(t, diags)

	if p.ReadResourceCalled {
		t.Fatal(`Unexpected call to ReadResource. The resource in this test should not be refreshed with ` +
			`the -light flag as ignore_changes should make the plan a no-op.`)
	}

	change := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.a"))
	if got, want := change.Action, plans.NoOp; got != want {
		t.Fatalf("wrong plan action - got: %s, want: %s", got, want)
	}
}

func TestContext2Plan_light_lifecycle_conditions_noop(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": lightPlanConditionsConfig("foo"),
	})

	p := lightPlanTestProvider(0)
	state := lightPlanTestState(t, `{"id":"a","arg":"foo","computed":"boop"}`, 0, false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	opts := &PlanOpts{
		Mode:         plans.NormalMode,
		PlanLight:    true,
		SetVariables: testInputValuesUnset(m.Module.Variables),
	}
	plan, diags := ctx.Plan(m, state, opts)
	tfdiags.AssertNoErrors(t, diags)

	if p.ReadResourceCalled {
		t.Fatal(`Unexpected call to ReadResource. The resource in this test should not be refreshed with ` +
			`the -light flag as the configuration did not change from prior state.`)
	}

	change := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.a"))
	if got, want := change.Action, plans.NoOp; got != want {
		t.Fatalf("wrong plan action - got: %s, want: %s", got, want)
	}
}

func TestContext2Plan_light_lifecycle_conditions_update(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": lightPlanConditionsConfig("new"),
	})

	p := lightPlanTestProvider(0)
	// Will prompt a refresh since the config value (arg) has changed
	state := lightPlanTestState(t, `{"id":"a","arg":"old","computed":"boop"}`, 0, false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	opts := &PlanOpts{
		Mode:         plans.NormalMode,
		PlanLight:    true,
		SetVariables: testInputValuesUnset(m.Module.Variables),
	}

	plan, diags := ctx.Plan(m, state, opts)
	tfdiags.AssertNoErrors(t, diags)

	if !p.ReadResourceCalled {
		t.Fatal(`Expected a call to ReadResource but received none. The resource in this test should be refreshed with ` +
			`the -light flag as the configuration changed from prior state.`)
	}
	change := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.a"))
	if got, want := change.Action, plans.Update; got != want {
		t.Fatalf("wrong plan action - got: %s, want: %s", got, want)
	}
}

func TestContext2Plan_light_precondition_error(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": lightPlanConditionsConfig("foo"),
	})

	p := lightPlanTestProvider(0)
	state := lightPlanTestState(t, `{"id":"a","arg":"foo","computed":"boop"}`, 0, false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:      plans.NormalMode,
		PlanLight: true,
		SetVariables: InputValues{
			"precond": &InputValue{Value: cty.False, SourceType: ValueFromCaller},
		},
	})
	if !diags.HasErrors() {
		t.Fatal("expected precondition failure, got none")
	}
	if got := diags.Err().Error(); !strings.Contains(got, "precondition failed") {
		t.Fatalf("wrong error, want precondition failure, got: %s", got)
	}
	if p.ReadResourceCalled {
		t.Fatal(`Unexpected call to ReadResource. The resource in this test should not be refreshed with ` +
			`the -light flag as the configuration did not change from prior state.`)
	}
}

func TestContext2Plan_light_postcondition_error(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "foo"
  lifecycle {
    postcondition {
      condition     = self.arg == "wrong"
      error_message = "postcondition failed"
    }
  }
}
`,
	})

	p := lightPlanTestProvider(0)
	state := lightPlanTestState(t, `{"id":"a","arg":"foo","computed":"boop"}`, 0, false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:      plans.NormalMode,
		PlanLight: true,
	})
	if !diags.HasErrors() {
		t.Fatal("expected postcondition failure, got none")
	}
	if got := diags.Err().Error(); !strings.Contains(got, "postcondition failed") {
		t.Fatalf("wrong error, want postcondition failure, got:\n%s", got)
	}
	if p.ReadResourceCalled {
		t.Fatal(`Unexpected call to ReadResource. The resource in this test should not be refreshed with ` +
			`the -light flag as the configuration did not change from prior state.`)
	}
}

func TestContext2Plan_light_create_before_destroy_no_refresh(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "foo"
  lifecycle {
    create_before_destroy = true
  }
}
`,
	})

	p := lightPlanTestProvider(0)
	// The state matches the config (no-op) but is not yet marked create_before_destroy.
	state := lightPlanTestState(t, `{"id":"a","arg":"foo","computed":"boop"}`, 0, false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:      plans.NormalMode,
		PlanLight: true,
	})
	tfdiags.AssertNoErrors(t, diags)

	if p.ReadResourceCalled {
		t.Fatal(`Unexpected call to ReadResource. The resource in this test should not be refreshed with ` +
			`the -light flag as the configuration did not change from prior state.`)
	}

	newState, diags := ctx.Apply(plan, m, nil)
	tfdiags.AssertNoErrors(t, diags)

	instance := newState.ResourceInstance(mustResourceInstanceAddr("test_object.a"))
	if instance == nil || instance.Current == nil {
		t.Fatal("missing state for test_object.a")
	}
	if !instance.Current.CreateBeforeDestroy {
		t.Fatal("create_before_destroy should have been recorded even though the refresh was skipped")
	}
}

func TestContext2Plan_light_force_replace(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "foo"
}
`,
	})

	p := lightPlanTestProvider(0)
	state := lightPlanTestState(t, `{"id":"a","arg":"foo","computed":"boop"}`, 0, false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	opts := &PlanOpts{
		Mode:         plans.NormalMode,
		PlanLight:    true,
		ForceReplace: []addrs.AbsResourceInstance{mustResourceInstanceAddr("test_object.a")},
	}
	plan, diags := ctx.Plan(m, state, opts)
	tfdiags.AssertNoErrors(t, diags)

	if !p.ReadResourceCalled {
		t.Fatal(`Expected a call to ReadResource but received none. The resource in this test should be refreshed due to -replace forcing a replacement`)
	}

	change := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.a"))
	if !change.Action.IsReplace() {
		t.Fatalf("wrong plan action - got: %s, wanted a replace action", change.Action)
	}
}
func TestContext2Plan_light_replace_triggered_by(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "new"
}

resource "test_object" "b" {
  arg = "same"
  lifecycle {
    replace_triggered_by = [test_object.a.arg]
  }
}
`,
	})

	p := lightPlanTestProvider(0)

	var mu sync.Mutex
	refreshedIDs := map[string]bool{}
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		mu.Lock()
		defer mu.Unlock()
		if id := req.PriorState.GetAttr("id"); !id.IsNull() {
			refreshedIDs[id.AsString()] = true
		}
		return providers.ReadResourceResponse{NewState: req.PriorState}
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.a"),
			&states.ResourceInstanceObjectSrc{
				// Will prompt a refresh to both resources
				AttrsJSON: []byte(`{"id":"a","arg":"old","computed":"boop"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.b"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"b","arg":"same","computed":"boop"}`),
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
		Mode:      plans.NormalMode,
		PlanLight: true,
	})
	tfdiags.AssertNoErrors(t, diags)

	if !refreshedIDs["a"] {
		t.Error("test_object.a should have been refreshed")
	}
	if !refreshedIDs["b"] {
		t.Error("test_object.b should have been refreshed due to replace_triggered_by")
	}

	b := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.b"))
	if !b.Action.IsReplace() {
		t.Fatalf("test_object.b: wrong plan action - got: %s, wanted a replace action", b.Action)
	}
}

func TestContext2Plan_light_tainted(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "foo"
}
`,
	})

	p := lightPlanTestProvider(0)
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.a"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"a","arg":"foo","computed":"boop"}`),
				Status:    states.ObjectTainted,
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
		Mode:      plans.NormalMode,
		PlanLight: true,
	})
	tfdiags.AssertNoErrors(t, diags)

	if !p.ReadResourceCalled {
		t.Fatal(`Expected a call to ReadResource but received none. The resource in this test should be refreshed with ` +
			`the -light flag as the resource was tainted.`)
	}

	change := plan.Changes.ResourceInstance(mustResourceInstanceAddr("test_object.a"))
	if !change.Action.IsReplace() {
		t.Fatalf("wrong plan action - got: %s, wanted a replace action", change.Action)
	}
}

func TestContext2Plan_light_no_duplicate_warnings(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "new"
}
`,
	})

	p := lightPlanTestProvider(0)
	planCallCount := 0
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		planCallCount++

		var d tfdiags.Diagnostics
		// Despite plan being called twice, the warning diags from the first call should be discarded
		d = d.Append(tfdiags.SimpleWarning("provider warning during plan"))
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
			Diagnostics:  d,
		}
	}

	state := lightPlanTestState(t, `{"id":"a","arg":"old","computed":"boop"}`, 0, false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:      plans.NormalMode,
		PlanLight: true,
	})

	if len(diags.Warnings()) != 1 {
		t.Fatalf("expected exactly 1 warning diagnostic, got %d", len(diags.Warnings()))
	}
	if planCallCount != 2 {
		t.Fatalf("expected exactly 2 PlanResourceChange calls, got %d", planCallCount)
	}
}

func TestContext2Plan_light_initial_plan_error(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "new"
}
`,
	})

	p := lightPlanTestProvider(0)
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		var d tfdiags.Diagnostics
		d = d.Append(fmt.Errorf("plan error!"))
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
			Diagnostics:  d,
		}
	}

	state := lightPlanTestState(t, `{"id":"a","arg":"old","computed":"boop"}`, 0, false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, state, &PlanOpts{
		Mode:      plans.NormalMode,
		PlanLight: true,
	})
	if !diags.HasErrors() {
		t.Fatal("expected an error from the plan")
	}
	if p.ReadResourceCalled {
		t.Fatal(`Unexpected call to ReadResource. The resource in this test should not be refreshed with ` +
			`the -light flag as the initial plan should error.`)
	}
}

func TestContext2Plan_light_validation_errors(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  arg = "foo"
}
`,
	})

	testCases := map[string]struct {
		opts    *PlanOpts
		wantErr string
	}{
		"destroy mode": {
			opts:    &PlanOpts{Mode: plans.DestroyMode, PlanLight: true},
			wantErr: "The -light planning option is only allowed in normal planning mode, got DestroyMode. This is a bug in Terraform.",
		},
		"refresh-only mode": {
			opts:    &PlanOpts{Mode: plans.RefreshOnlyMode, PlanLight: true},
			wantErr: "The -light planning option is only allowed in normal planning mode, got RefreshOnlyMode. This is a bug in Terraform.",
		},
		"skip refresh": {
			opts:    &PlanOpts{Mode: plans.NormalMode, PlanLight: true, SkipRefresh: true},
			wantErr: "The -light planning option cannot be combined with skipping refresh, because it only affects whether Terraform refreshes. This is a bug in Terraform.",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			p := lightPlanTestProvider(0)
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

func lightPlanConditionsConfig(arg string) string {
	return fmt.Sprintf(`
variable "precond" {
  type    = bool
  default = true
}

resource "test_object" "a" {
  arg = %q
  lifecycle {
    precondition {
      condition     = var.precond
      error_message = "precondition failed"
    }
    postcondition {
      condition     = self.arg != ""
      error_message = "postcondition failed"
    }
  }
}
`, arg)
}

func lightPlanTestProvider(version uint64) *testing_provider.MockProvider {
	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"test_object": {
					Version: int64(version),
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
		PlanResourceChangeFn: func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
		},
		ApplyResourceChangeFn: func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
			return providers.ApplyResourceChangeResponse{NewState: req.PlannedState}
		},
	}
	return p
}

func lightPlanTestState(t *testing.T, attrsJSON string, schemaVersion uint64, createBeforeDestroy bool) *states.State {
	t.Helper()
	return states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("test_object.a"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:           []byte(attrsJSON),
				Status:              states.ObjectReady,
				SchemaVersion:       schemaVersion,
				CreateBeforeDestroy: createBeforeDestroy,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})
}
