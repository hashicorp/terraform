// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
)

func TestContextEval(t *testing.T) {
	// This test doesn't check the "Want" value for impure funcs, so the value
	// on those doesn't matter.
	tests := []struct {
		Input      string
		Want       cty.Value
		ImpureFunc bool
	}{
		{ // An impure function: allowed in the console, but the result is nondeterministic
			`bcrypt("example")`,
			cty.NilVal,
			true,
		},
		{
			`keys(var.map)`,
			cty.ListVal([]cty.Value{
				cty.StringVal("foo"),
				cty.StringVal("baz"),
			}),
			true,
		},
		{
			`local.result`,
			cty.NumberIntVal(6),
			false,
		},
		{
			`module.child.result`,
			cty.NumberIntVal(6),
			false,
		},
	}

	// This module has a little bit of everything (and if it is missing somehitng, add to it):
	// resources, variables, locals, modules, output
	m := testModule(t, "eval-context-basic")
	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	scope, diags := ctx.Eval(m, states.NewState(), addrs.RootModuleInstance, &EvalOpts{
		SetVariables: testInputValuesUnset(m.Module.Variables),
	})
	if diags.HasErrors() {
		t.Fatalf("Eval errors: %s", diags.Err())
	}

	// Since we're testing 'eval' (used by terraform console), impure functions
	// should be allowed by the scope.
	if scope.PureOnly == true {
		t.Fatal("wrong result: eval should allow impure funcs")
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			// Parse the test input as an expression
			expr, _ := hclsyntax.ParseExpression([]byte(test.Input), "<test-input>", hcl.Pos{Line: 1, Column: 1})
			got, diags := scope.EvalExpr(expr, cty.DynamicPseudoType)

			if diags.HasErrors() {
				t.Fatalf("unexpected error: %s", diags.Err())
			}

			if !test.ImpureFunc {
				if !got.RawEquals(test.Want) {
					t.Fatalf("wrong result: want %#v, got %#v", test.Want, got)
				}
			}
		})
	}
}

// ensure that we can execute a console when outputs have preconditions
func TestContextEval_outputsWithPreconditions(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod" {
  source = "./mod"
  input  = "ok"
}

output "out" {
  value = module.mod.out
}
`,

		"./mod/main.tf": `
variable "input" {
  type = string
}

output "out" {
  value = var.input

  precondition {
    condition     = var.input != ""
    error_message = "error"
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

	_, diags := ctx.Eval(m, states.NewState(), addrs.RootModuleInstance, &EvalOpts{
		SetVariables: testInputValuesUnset(m.Module.Variables),
	})
	assertNoErrors(t, diags)
}

func TestContextPlanAndEval(t *testing.T) {
	// This test actually performs a plan walk rather than an eval walk, but
	// it's here because PlanAndEval is thematically related to the evaluation
	// walk, with the same effect of producing a lang.Scope that the caller
	// can use to evaluate arbitrary expressions.

	m := testModule(t, "planandeval-basic")
	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"test_thing": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"arg": {
								Type:     cty.String,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, scope, diags := ctx.PlanAndEval(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"a": {
				Value: cty.StringVal("a value"),
			},
		},
	})
	assertNoDiagnostics(t, diags)

	// This test isn't really about whether the plan is correct, but we'll
	// do some basic checks on it anyway because if the plan is incorrect
	// then the evaluation scope will probably behave oddly too.
	if plan.Errored {
		t.Error("plan is marked as errored; want success")
	}
	riAddr := mustResourceInstanceAddr("test_thing.a")
	if plan.Changes != nil {
		if rc := plan.Changes.ResourceInstance(riAddr); rc == nil {
			t.Errorf("plan does not include a change for test_thing.a")
		} else if got, want := rc.Action, plans.Create; got != want {
			t.Errorf("wrong planned action for test_thing.a\ngot:  %s\nwant: %s", got, want)
		}
		if _, ok := plan.VariableValues["a"]; !ok {
			t.Errorf("plan does not track value for var.a")
		}
	} else {
		t.Fatalf("plan has no Changes")
	}
	if plan.PlannedState != nil {
		if rs := plan.PlannedState.ResourceInstance(riAddr); rs == nil {
			t.Errorf("planned satte does not include test_thing.a")
		}
	} else {
		t.Fatalf("plan has no PlannedState")
	}
	if plan.PriorState == nil {
		t.Fatalf("plan has no PriorState")
	}
	if plan.PrevRunState == nil {
		t.Fatalf("plan has no PrevRunState")
	}

	if scope == nil {
		// It's okay for scope to be nil when there are errors, but if we
		// successfully created a plan then it should always be set.
		t.Fatal("PlanAndEval returned nil scope")
	}

	t.Run("var.a", func(t *testing.T) {
		expr := hcltest.MockExprTraversalSrc(`var.a`)
		want := cty.StringVal("a value")
		got, diags := scope.EvalExpr(expr, cty.String)
		assertNoDiagnostics(t, diags)

		if !want.RawEquals(got) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("test_thing.a", func(t *testing.T) {
		expr := hcltest.MockExprTraversalSrc(`test_thing.a`)
		want := cty.ObjectVal(map[string]cty.Value{
			"arg": cty.StringVal("a value"),
		})
		got, diags := scope.EvalExpr(expr, cty.DynamicPseudoType)
		assertNoDiagnostics(t, diags)

		if !want.RawEquals(got) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
}

func TestContextApplyAndEval(t *testing.T) {
	// This test actually performs plan and apply walks rather than an eval
	// walk, but it's here because ApplyAndEval is thematically related to the
	// evaluation walk, with the same effect of producing a lang.Scope that the
	// caller can use to evaluate arbitrary expressions.

	m := testModule(t, "planandeval-basic")
	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"test_thing": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"arg": {
								Type:     cty.String,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
		SetVariables: InputValues{
			"a": {
				Value: cty.StringVal("a value"),
			},
		},
	})
	assertNoDiagnostics(t, diags)

	// This test isn't really about whether the plan is correct, but we'll
	// do some basic checks on it anyway because if the plan is incorrect
	// then the evaluation scope will probably behave oddly too.
	if plan.Errored {
		t.Error("plan is marked as errored; want success")
	}
	riAddr := mustResourceInstanceAddr("test_thing.a")
	if plan.Changes != nil {
		if rc := plan.Changes.ResourceInstance(riAddr); rc == nil {
			t.Errorf("plan does not include a change for test_thing.a")
		} else if got, want := rc.Action, plans.Create; got != want {
			t.Errorf("wrong planned action for test_thing.a\ngot:  %s\nwant: %s", got, want)
		}
		if _, ok := plan.VariableValues["a"]; !ok {
			t.Errorf("plan does not track value for var.a")
		}
	} else {
		t.Fatalf("plan has no Changes")
	}
	if plan.PlannedState != nil {
		if rs := plan.PlannedState.ResourceInstance(riAddr); rs == nil {
			t.Errorf("planned satte does not include test_thing.a")
		}
	} else {
		t.Fatalf("plan has no PlannedState")
	}
	if plan.PriorState == nil {
		t.Fatalf("plan has no PriorState")
	}
	if plan.PrevRunState == nil {
		t.Fatalf("plan has no PrevRunState")
	}

	finalState, scope, diags := ctx.ApplyAndEval(plan, m, nil)
	assertNoDiagnostics(t, diags)
	if finalState == nil {
		t.Fatalf("no final state")
	}

	if scope == nil {
		// It's okay for scope to be nil when there are errors, but if we
		// successfully applied the plan then it should always be set.
		t.Fatal("ApplyAndEval returned nil scope")
	}

	t.Run("var.a", func(t *testing.T) {
		expr := hcltest.MockExprTraversalSrc(`var.a`)
		want := cty.StringVal("a value")
		got, diags := scope.EvalExpr(expr, cty.String)
		assertNoDiagnostics(t, diags)

		if !want.RawEquals(got) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("test_thing.a", func(t *testing.T) {
		expr := hcltest.MockExprTraversalSrc(`test_thing.a`)
		want := cty.ObjectVal(map[string]cty.Value{
			"arg": cty.StringVal("a value"),
		})
		got, diags := scope.EvalExpr(expr, cty.DynamicPseudoType)
		assertNoDiagnostics(t, diags)

		if !want.RawEquals(got) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
}
