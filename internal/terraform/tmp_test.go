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

func TestContext2Apply_forget_createBeforeDestroy(t *testing.T) {
	tests := map[string]string{
		"explicit": `
resource "test_object" "forget" {
	lifecycle {
		destroy               = false
		create_before_destroy = true
	}
}
`,
		"propagated": `
resource "test_object" "forget" {
	lifecycle {
		destroy = false
	}
}

resource "test_object" "dependent" {
	test_string = test_object.forget.test_string

	lifecycle {
		create_before_destroy = true
	}
}
`,
	}

	for name, config := range tests {
		t.Run(name, func(t *testing.T) {
			m := testModuleInline(t, map[string]string{
				"main.tf": config,
			})

			p := simpleMockProvider()
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
				},
			})

			forget := mustResourceInstanceAddr("test_object.forget")
			state := states.NewState()
			root := state.EnsureModule(addrs.RootModuleInstance)
			root.SetResourceInstanceCurrent(
				forget.Resource,
				&states.ResourceInstanceObjectSrc{
					Status:    states.ObjectTainted,
					AttrsJSON: []byte(`{"test_string":"old"}`),
				},
				mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
			)

			plan, diags := ctx.Plan(m, state, nil)
			tfdiags.AssertNoErrors(t, diags)
			change := plan.Changes.ResourceInstance(forget)
			if change == nil {
				t.Fatal("expected a change")
			}
			if got, want := change.Action, plans.CreateThenForget; got != want {
				t.Fatalf("wrong change type for %s: got %s, want %s", forget, got, want)
			}

			state, diags = ctx.Apply(plan, m, nil)
			tfdiags.AssertNoErrors(t, diags)

			instance := state.ResourceInstance(forget)
			if instance == nil || instance.Current == nil {
				t.Fatalf("%s has no current object after replacement", forget)
			}
			if got := len(instance.Deposed); got != 0 {
				t.Fatalf("%s has %d deposed objects after replacement; want none", forget, got)
			}
		})
	}
}

func TestContext2Apply_forget_createBeforeDestroyInvalidResult(t *testing.T) {
	initialConfig := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "forget" {
	test_string = "old"

	lifecycle {
		destroy               = false
		create_before_destroy = true
	}
}
`,
	})

	replacementConfig := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "forget" {
	test_string = "new"

	lifecycle {
		destroy               = false
		create_before_destroy = true
	}
}
`,
	})

	p := simpleMockProvider()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp.PlannedState = req.ProposedNewState
		if req.PriorState.IsNull() || req.ProposedNewState.IsNull() {
			return resp
		}
		if !req.PriorState.GetAttr("test_string").RawEquals(req.ProposedNewState.GetAttr("test_string")) {
			resp.RequiresReplace = []cty.Path{cty.GetAttrPath("test_string")}
		}
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(initialConfig, states.NewState(), nil)
	tfdiags.AssertNoErrors(t, diags)
	state, diags := ctx.Apply(plan, initialConfig, nil)
	tfdiags.AssertNoErrors(t, diags)

	plan, diags = ctx.Plan(replacementConfig, state, nil)
	tfdiags.AssertNoErrors(t, diags)
	_, diags = ctx.Apply(plan, replacementConfig, nil)
	tfdiags.AssertNoErrors(t, diags)
}

func TestContext2Apply_forget_replace(t *testing.T) {
	// https://github.com/hashicorp/terraform/issues/39088
	// we have a resource in state that needs replacing, and forget statement, but we're doing a replace
	// don't be weird about it
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "forget" {
	test_string = "hello"
	lifecycle {
		destroy = false
	}
}
`})

	p := simpleMockProvider()

	hook := new(MockHook)
	ctx := testContext2(t, &ContextOpts{
		Hooks: []Hook{hook},
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	forget := mustResourceInstanceAddr("test_object.forget")

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		forget.Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"test_string":"hi"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	// need the provider to return a requires_replace (or otherwise force replace)
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		obj := req.ProposedNewState.AsValueMap()

		if req.Config.IsNull() {
			t.Fatal("should not plan a destroy")
		}

		if !req.PriorState.IsNull() {
			if req.PriorState.GetAttr("test_string").AsString() != "hello" {
				resp.RequiresReplace = append(resp.RequiresReplace, cty.GetAttrPath("test_string"))
			}
		}

		resp.PlannedState = cty.ObjectVal(obj)
		return resp
	}

	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		if req.PlannedState.IsNull() {
			t.Fatal("should not be applying a destroy")
		}

		resp.NewState = req.PlannedState
		return resp
	}

	plan, diags := ctx.Plan(m, state, nil)
	if !diags.HasWarnings() { // forgetting emits a warning, but there should be no errors.
		t.Errorf("missing expected forget warning")
	}
	assertNoDiagnostics(t, diags.ErrorsOnly())
	change := plan.Changes.ResourceInstance(forget)
	if change == nil {
		t.Fatal("expected a change")
	}
	if got, want := change.Action, plans.ForgetThenCreate; got != want {
		t.Fatalf("wrong change type for %s: got %s, want %s", forget, got, want)
	}

	state, applyDiags := ctx.Apply(plan, m, nil)
	assertNoDiagnostics(t, applyDiags)

	replaced := state.Resources(forget.ConfigResource())
	if len(replaced) != 1 {
		t.Fatal("expected one resource in state")
	}

	if len(replaced[0].Instances[addrs.NoKey].Deposed) != 0 {
		t.Fatal("should be no deposed instances")
	}
}
