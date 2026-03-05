// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/encoding/prototext"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestInputVariableValue(t *testing.T) {
	ctx := context.Background()
	cfg := testStackConfig(t, "input_variable", "basics")

	// NOTE: This also indirectly tests the propagation of input values
	// from a parent stack into one of its children, even though that's
	// technically the responsibility of [StackCall] rather than [InputVariable],
	// because propagating downward into child stacks is a major purpose
	// of input variables that must keep working.
	childStackAddr := stackaddrs.RootStackInstance.Child("child", addrs.NoKey)

	tests := map[string]struct {
		NameVal      cty.Value
		WantRootVal  cty.Value
		WantChildVal cty.Value

		WantRootErr bool
	}{
		"known string": {
			NameVal:      cty.StringVal("jackson"),
			WantRootVal:  cty.StringVal("jackson"),
			WantChildVal: cty.StringVal("child of jackson"),
		},
		"unknown string": {
			NameVal:     cty.UnknownVal(cty.String),
			WantRootVal: cty.UnknownVal(cty.String),
			WantChildVal: cty.UnknownVal(cty.String).Refine().
				NotNull().
				StringPrefix("child of ").
				NewValue(),
		},
		"unknown of unknown type": {
			NameVal:     cty.DynamicVal,
			WantRootVal: cty.UnknownVal(cty.String),
			WantChildVal: cty.UnknownVal(cty.String).Refine().
				NotNull().
				StringPrefix("child of ").
				NewValue(),
		},
		"bool": {
			// This one is testing that the given value gets converted to
			// the declared type constraint, which is string in this case.
			NameVal:      cty.True,
			WantRootVal:  cty.StringVal("true"),
			WantChildVal: cty.StringVal("child of true"),
		},
		"object": {
			// This one is testing that the given value gets converted to
			// the declared type constraint, which is string in this case.
			NameVal:     cty.EmptyObjectVal,
			WantRootErr: true, // Type mismatch error
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				InputVariableValues: map[string]cty.Value{
					"name": test.NameVal,
				},
			})

			t.Run("root", func(t *testing.T) {
				promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
					mainStack := main.MainStack()
					rootVar := mainStack.InputVariable(stackaddrs.InputVariable{Name: "name"})
					got, diags := rootVar.CheckValue(ctx, InspectPhase)

					if test.WantRootErr {
						if !diags.HasErrors() {
							t.Errorf("succeeded; want error\ngot: %#v", got)
						}
						return struct{}{}, nil
					}

					if diags.HasErrors() {
						t.Errorf("unexpected errors\n%s", diags.Err().Error())
					}
					want := test.WantRootVal
					if !want.RawEquals(got) {
						t.Errorf("wrong value\ngot:  %#v\nwant: %#v", got, want)
					}
					return struct{}{}, nil
				})
			})
			if !test.WantRootErr {
				t.Run("child", func(t *testing.T) {
					promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
						childStack := main.Stack(ctx, childStackAddr, InspectPhase)
						rootVar := childStack.InputVariable(stackaddrs.InputVariable{Name: "name"})
						got, diags := rootVar.CheckValue(ctx, InspectPhase)
						if diags.HasErrors() {
							t.Errorf("unexpected errors\n%s", diags.Err().Error())
						}
						want := test.WantChildVal
						if !want.RawEquals(got) {
							t.Errorf("wrong value\ngot:  %#v\nwant: %#v", got, want)
						}
						return struct{}{}, nil
					})
				})
			}
		})
	}
}

func TestInputVariableEphemeral(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		fixtureName string
		givenVal    cty.Value
		allowed     bool
		wantInputs  cty.Value
		wantVal     cty.Value
	}{
		"ephemeral and allowed": {
			fixtureName: "ephemeral_yes",
			givenVal:    cty.StringVal("beep").Mark(marks.Ephemeral),
			allowed:     true,
			wantInputs: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("beep").Mark(marks.Ephemeral),
			}),
			wantVal: cty.StringVal("beep").Mark(marks.Ephemeral),
		},
		"ephemeral and not allowed": {
			fixtureName: "ephemeral_no",
			givenVal:    cty.StringVal("beep").Mark(marks.Ephemeral),
			allowed:     false,
			wantInputs: cty.UnknownVal(cty.Object(map[string]cty.Type{
				"a": cty.String,
			})),
			wantVal: cty.UnknownVal(cty.String),
		},
		"non-ephemeral and allowed": {
			fixtureName: "ephemeral_yes",
			givenVal:    cty.StringVal("beep"),
			allowed:     true,
			wantInputs: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("beep"), // not marked on the input side...
			}),
			wantVal: cty.StringVal("beep").Mark(marks.Ephemeral), // ...but marked on the result side
		},
		"non-ephemeral and not allowed": {
			fixtureName: "ephemeral_no",
			givenVal:    cty.StringVal("beep"),
			allowed:     true,
			wantInputs: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("beep"),
			}),
			wantVal: cty.StringVal("beep"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := testStackConfig(t, "input_variable", test.fixtureName)
			childStackAddr := stackaddrs.RootStackInstance.Child("child", addrs.NoKey)
			childStackCallAddr := stackaddrs.StackCall{Name: "child"}
			aVarAddr := stackaddrs.InputVariable{Name: "a"}

			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"var_val": test.givenVal,
				},
			})

			promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
				childStack := main.Stack(ctx, childStackAddr, InspectPhase)
				if childStack == nil {
					t.Fatalf("missing %s", childStackAddr)
				}
				childStackCall := main.MainStack().EmbeddedStackCall(childStackCallAddr)
				if childStackCall == nil {
					t.Fatalf("missing %s", childStackCallAddr)
				}
				insts, unknown := childStackCall.Instances(ctx, InspectPhase)
				if unknown {
					t.Fatalf("stack call instances are unknown")
				}
				childStackCallInst := insts[addrs.NoKey]
				if childStackCallInst == nil {
					t.Fatalf("missing %s instance", childStackCallAddr)
				}

				// The responsibility for handling ephemeral input variables
				// is split between the stack call which decides whether an
				// ephemeral value is acceptable, and the variable declaration
				// itself which ensures that variables declared as ephemeral
				// always appear as ephemeral inside even if the given value
				// wasn't.

				wantInputs := test.wantInputs
				gotInputs, diags := childStackCallInst.CheckInputVariableValues(ctx, InspectPhase)
				if diff := cmp.Diff(wantInputs, gotInputs, ctydebug.CmpOptions); diff != "" {
					t.Errorf("wrong inputs for %s\n%s", childStackCallAddr, diff)
				}

				aVar := childStack.InputVariable(aVarAddr)
				if aVar == nil {
					t.Fatalf("missing %s", stackaddrs.Absolute(childStackAddr, aVarAddr))
				}
				want := test.wantVal
				got, moreDiags := aVar.CheckValue(ctx, InspectPhase)
				diags = diags.Append(moreDiags)
				if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
					t.Errorf("wrong value for %s\n%s", aVarAddr, diff)
				}

				if test.allowed {
					if diags.HasErrors() {
						t.Errorf("unexpected errors\n%s", diags.Err().Error())
					}
				} else {
					if !diags.HasErrors() {
						t.Fatalf("no errors; should have failed")
					}
					found := 0
					for _, diag := range diags {
						summary := diag.Description().Summary
						if summary == "Ephemeral value not allowed" {
							found++
						}
					}
					if found == 0 {
						t.Errorf("no diagnostics about disallowed ephemeral values\n%s", diags.Err().Error())
					} else if found > 1 {
						t.Errorf("found %d errors about disallowed ephemeral values, but wanted only one\n%s", found, diags.Err().Error())
					}
				}
				return struct{}{}, nil
			})
		})
	}
}

func TestInputVariablePlanApply(t *testing.T) {
	ctx := context.Background()
	cfg := testStackConfig(t, "input_variable", "basics")

	tests := map[string]struct {
		PlanVal  cty.Value
		ApplyVal cty.Value
		WantErr  bool
	}{
		"unmarked": {
			PlanVal:  cty.StringVal("alisdair"),
			ApplyVal: cty.StringVal("alisdair"),
		},
		"sensitive": {
			PlanVal:  cty.StringVal("alisdair").Mark(marks.Sensitive),
			ApplyVal: cty.StringVal("alisdair").Mark(marks.Sensitive),
		},
		"changed": {
			PlanVal:  cty.StringVal("alice"),
			ApplyVal: cty.StringVal("bob"),
			WantErr:  true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			planOutput, err := promising.MainTask(ctx, func(ctx context.Context) (*planOutputTester, error) {
				main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
					PlanningMode:  plans.NormalMode,
					PlanTimestamp: time.Now().UTC(),
					InputVariableValues: map[stackaddrs.InputVariable]ExternalInputValue{
						{Name: "name"}: {
							Value: test.PlanVal,
						},
					},
				})

				outp, outpTester := testPlanOutput(t)
				main.PlanAll(ctx, outp)

				return outpTester, nil
			})
			if err != nil {
				t.Fatalf("planning failed: %s", err)
			}

			rawPlan := planOutput.RawChanges(t)
			plan, diags := planOutput.Close(t)
			assertNoDiagnostics(t, diags)

			if !plan.Applyable {
				m := prototext.MarshalOptions{
					Multiline: true,
					Indent:    "  ",
				}
				for _, raw := range rawPlan {
					t.Log(m.Format(raw))
				}
				t.Fatalf("plan is not applyable")
			}

			_, err = promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
				main := NewForApplying(cfg, plan, nil, ApplyOpts{
					InputVariableValues: map[stackaddrs.InputVariable]ExternalInputValue{
						{Name: "name"}: {
							Value: test.ApplyVal,
						},
					},
				})
				mainStack := main.MainStack()
				rootVar := mainStack.InputVariable(stackaddrs.InputVariable{Name: "name"})
				got, diags := rootVar.CheckValue(ctx, ApplyPhase)

				if test.WantErr {
					if !diags.HasErrors() {
						t.Errorf("succeeded; want error\ngot: %#v", got)
					}
					return struct{}{}, nil
				}

				if diags.HasErrors() {
					t.Errorf("unexpected errors\n%s", diags.Err().Error())
				}
				want := test.ApplyVal
				if !want.RawEquals(got) {
					t.Errorf("wrong value\ngot:  %#v\nwant: %#v", got, want)
				}

				return struct{}{}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInputVariablePlanChanges(t *testing.T) {
	ctx := context.Background()
	cfg := testStackConfig(t, "input_variable", "basics")

	tests := map[string]struct {
		PlanVal            cty.Value
		PreviousPlanVal    cty.Value
		WantPlannedChanges []stackplan.PlannedChange
	}{
		"unmarked": {
			PlanVal:         cty.StringVal("value_1"),
			PreviousPlanVal: cty.NullVal(cty.String),
			WantPlannedChanges: []stackplan.PlannedChange{
				&stackplan.PlannedChangeRootInputValue{
					Addr:            stackaddrs.InputVariable{Name: "name"},
					Action:          plans.Update,
					Before:          cty.NullVal(cty.String),
					After:           cty.StringVal("value_1"),
					RequiredOnApply: false,
					DeleteOnApply:   false,
				},
			},
		},
		"sensitive": {
			PlanVal:         cty.StringVal("value_2").Mark(marks.Sensitive),
			PreviousPlanVal: cty.NullVal(cty.String),
			WantPlannedChanges: []stackplan.PlannedChange{
				&stackplan.PlannedChangeRootInputValue{
					Addr:            stackaddrs.InputVariable{Name: "name"},
					Action:          plans.Update,
					Before:          cty.NullVal(cty.String),
					After:           cty.StringVal("value_2").Mark(marks.Sensitive),
					RequiredOnApply: false,
					DeleteOnApply:   false,
				},
			},
		},
		"ephemeral": {
			PlanVal:         cty.StringVal("value_3").Mark(marks.Ephemeral),
			PreviousPlanVal: cty.NullVal(cty.String),
			WantPlannedChanges: []stackplan.PlannedChange{
				&stackplan.PlannedChangeRootInputValue{
					Addr:            stackaddrs.InputVariable{Name: "name"},
					Action:          plans.Update,
					Before:          cty.NullVal(cty.String),
					After:           cty.StringVal("value_3").Mark(marks.Ephemeral),
					RequiredOnApply: false,
					DeleteOnApply:   false,
				},
			},
		},
		"sensitive_and_ephemeral": {
			PlanVal:         cty.StringVal("value_4").Mark(marks.Ephemeral).Mark(marks.Sensitive),
			PreviousPlanVal: cty.NullVal(cty.String),
			WantPlannedChanges: []stackplan.PlannedChange{
				&stackplan.PlannedChangeRootInputValue{
					Addr:            stackaddrs.InputVariable{Name: "name"},
					Action:          plans.Update,
					Before:          cty.NullVal(cty.String),
					After:           cty.StringVal("value_4").Mark(marks.Ephemeral).Mark(marks.Sensitive),
					RequiredOnApply: false,
					DeleteOnApply:   false,
				},
			},
		},
		"from_non_null_to_sensitive": {
			PlanVal:         cty.StringVal("value_2").Mark(marks.Sensitive),
			PreviousPlanVal: cty.StringVal("value_1"),
			WantPlannedChanges: []stackplan.PlannedChange{
				&stackplan.PlannedChangeRootInputValue{
					Addr:            stackaddrs.InputVariable{Name: "name"},
					Action:          plans.Update,
					Before:          cty.StringVal("value_1"),
					After:           cty.StringVal("value_2").Mark(marks.Sensitive),
					RequiredOnApply: false,
					DeleteOnApply:   false,
				},
			},
		},
		"from_ephemeral_to_unmark": {
			PlanVal:         cty.StringVal("value_2"),
			PreviousPlanVal: cty.StringVal("value_1").Mark(marks.Ephemeral),
			WantPlannedChanges: []stackplan.PlannedChange{
				&stackplan.PlannedChangeRootInputValue{
					Addr:            stackaddrs.InputVariable{Name: "name"},
					Action:          plans.Update,
					Before:          cty.StringVal("value_1").Mark(marks.Ephemeral),
					After:           cty.StringVal("value_2"),
					RequiredOnApply: false,
					DeleteOnApply:   false,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := promising.MainTask(ctx, func(ctx context.Context) (*planOutputTester, error) {
				previousState := stackstate.NewStateBuilder().AddInput("name", test.PreviousPlanVal).Build()

				main := NewForPlanning(cfg, previousState, PlanOpts{
					PlanningMode:  plans.NormalMode,
					PlanTimestamp: time.Now().UTC(),
					InputVariableValues: map[stackaddrs.InputVariable]ExternalInputValue{
						{Name: "name"}: {
							Value: test.PlanVal,
						},
					},
				})

				mainStack := main.MainStack()
				rootVar := mainStack.InputVariable(stackaddrs.InputVariable{Name: "name"})
				got, diags := rootVar.PlanChanges(ctx)
				if diags.HasErrors() {
					t.Errorf("unexpected errors\n%s", diags.Err().Error())
				}

				opts := cmp.Options{ctydebug.CmpOptions}
				if diff := cmp.Diff(test.WantPlannedChanges, got, opts); len(diff) > 0 {
					t.Errorf("wrong planned changes\n%s", diff)
				}

				return nil, nil
			})
			if err != nil {
				t.Fatalf("planning failed: %s", err)
			}
		})
	}
}

// TestEvalVariableValidation tests the evalVariableValidation function directly,
// covering all the "invalid" cases: sensitive/ephemeral values in the error message,
// unknown/null condition results, and unknown error messages.  These tests
// exercise the logic independently of the full stack-evaluator machinery.
func TestEvalVariableValidation(t *testing.T) {
	// parseExpr parses a real HCL expression from a source string.
	parseExpr := func(t *testing.T, src string) hcl.Expression {
		t.Helper()
		expr, diags := hclsyntax.ParseExpression([]byte(src), "test.hcl", hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			t.Fatalf("failed to parse expression %q: %s", src, diags.Error())
		}
		return expr
	}

	// makeFakeRule builds a minimal stackconfig.CheckRule from two expressions.
	makeFakeRule := func(condition, errorMessage hcl.Expression) *stackconfig.CheckRule {
		return &stackconfig.CheckRule{
			Condition:    condition,
			ErrorMessage: errorMessage,
			DeclRange: hcl.Range{
				Filename: "test.hcl",
				Start:    hcl.Pos{Line: 2, Column: 1},
				End:      hcl.Pos{Line: 5, Column: 1},
			},
		}
	}

	// makeVarCtx builds an HCL evaluation context that exposes var.foo = val.
	makeVarCtx := func(val cty.Value) *hcl.EvalContext {
		return &hcl.EvalContext{
			Variables: map[string]cty.Value{
				"var": cty.ObjectVal(map[string]cty.Value{
					"foo": val,
				}),
			},
		}
	}

	valueRange := hcl.Range{
		Filename: "test.hcl",
		Start:    hcl.Pos{Line: 1, Column: 1},
		End:      hcl.Pos{Line: 1, Column: 10},
	}

	// --- Basic pass/fail ---

	t.Run("condition passes, clean message → no diagnostics", func(t *testing.T) {
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.True),
			hcltest.MockExprLiteral(cty.StringVal("Value is invalid.")),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("good")), valueRange)
		assertNoDiags(t, diags)
	})

	t.Run("condition fails, clean message → Invalid value for variable", func(t *testing.T) {
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.False),
			hcltest.MockExprLiteral(cty.StringVal("Value must be 'good'.")),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("bad")), valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid value for variable"
		})
	})

	// --- Sensitive error message ---

	t.Run("condition passes, sensitive error_message → flagged even on success", func(t *testing.T) {
		// The error_message evaluates to a sensitive string even though the
		// condition passes.  This structural problem must always be reported.
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.True),
			hcltest.MockExprLiteral(cty.StringVal("Contains secret").Mark(marks.Sensitive)),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("good")), valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Error message refers to sensitive values"
		})
		// Condition passed, so there must be no "Invalid value for variable".
		for _, d := range diags {
			if d.Description().Summary == "Invalid value for variable" {
				t.Errorf("unexpected 'Invalid value for variable' diagnostic when condition passed")
			}
		}
	})

	t.Run("condition fails, sensitive error_message → both diagnostics", func(t *testing.T) {
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.False),
			hcltest.MockExprLiteral(cty.StringVal("Contains secret").Mark(marks.Sensitive)),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("bad")), valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Error message refers to sensitive values"
		})
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid value for variable"
		})
	})

	// --- Ephemeral error message ---

	t.Run("condition passes, ephemeral error_message → flagged even on success", func(t *testing.T) {
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.True),
			hcltest.MockExprLiteral(cty.StringVal("Contains ephemeral").Mark(marks.Ephemeral)),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("good")), valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Error message refers to ephemeral values"
		})
		// Condition passed, so there must be no "Invalid value for variable".
		for _, d := range diags {
			if d.Description().Summary == "Invalid value for variable" {
				t.Errorf("unexpected 'Invalid value for variable' diagnostic when condition passed")
			}
		}
	})

	t.Run("condition fails, ephemeral error_message → both diagnostics", func(t *testing.T) {
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.False),
			hcltest.MockExprLiteral(cty.StringVal("Contains ephemeral").Mark(marks.Ephemeral)),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("bad")), valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Error message refers to ephemeral values"
		})
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid value for variable"
		})
	})

	// --- Unknown / null condition results ---

	t.Run("condition result unknown → no diagnostics", func(t *testing.T) {
		// Unknown condition means we cannot determine validity yet; skip quietly.
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.UnknownVal(cty.Bool)),
			hcltest.MockExprLiteral(cty.StringVal("Value is invalid.")),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.UnknownVal(cty.String)), valueRange)
		assertNoDiags(t, diags)
	})

	t.Run("condition result null → Invalid variable validation result", func(t *testing.T) {
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.NullVal(cty.Bool)),
			hcltest.MockExprLiteral(cty.StringVal("Value is invalid.")),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("anything")), valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid variable validation result"
		})
	})

	// --- Unknown error message ---

	t.Run("error message unknown, condition fails → Invalid error message only", func(t *testing.T) {
		// An unknown error_message is always a structural problem: the validation
		// block is invalid regardless of whether the condition passes or fails,
		// because Terraform can never safely display the message.
		// We return early on the unknown message, so "Invalid value for variable"
		// must NOT also be emitted.
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.False),
			hcltest.MockExprLiteral(cty.UnknownVal(cty.String)),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("bad")), valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid error message"
		})
		for _, d := range diags {
			if d.Description().Summary == "Invalid value for variable" {
				t.Errorf("unexpected 'Invalid value for variable' when error message is unknown")
			}
		}
	})

	t.Run("error message unknown, condition passes → Invalid error message only", func(t *testing.T) {
		// An unknown error_message is always a structural problem: the validation
		// block is invalid regardless of whether the condition passes or fails,
		// because Terraform can never safely display the message.
		// We return early on the unknown message, so "Invalid value for variable"
		// must NOT be emitted even though the condition passed.
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.True),
			hcltest.MockExprLiteral(cty.UnknownVal(cty.String)),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("good")), valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid error message"
		})
		for _, d := range diags {
			if d.Description().Summary == "Invalid value for variable" {
				t.Errorf("unexpected 'Invalid value for variable' when error message is unknown")
			}
		}
	})

	// --- Sensitive variable value in condition expression ---

	t.Run("sensitive variable value, plain error message, condition fails → only Invalid value for variable", func(t *testing.T) {
		// var.foo carries a sensitive mark.  The condition expression
		// (var.foo == "good") evaluates to a sensitive bool; Unmark() peels
		// off the mark so the check works correctly.  The error_message is a
		// plain literal → no "Error message refers to sensitive values" diag.
		hclCtx := makeVarCtx(cty.StringVal("bad").Mark(marks.Sensitive))
		rule := makeFakeRule(
			parseExpr(t, `var.foo == "good"`),
			hcltest.MockExprLiteral(cty.StringVal("Value is not allowed.")),
		)
		diags := evalVariableValidation(rule, hclCtx, valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid value for variable"
		})
		for _, d := range diags {
			if d.Description().Summary == "Error message refers to sensitive values" {
				t.Errorf("unexpected 'Error message refers to sensitive values' when error message is plain text")
			}
		}
	})

	t.Run("sensitive variable referenced in error message, condition fails → both diagnostics", func(t *testing.T) {
		// When the error_message interpolates a sensitive variable the
		// evaluated message is itself sensitive — both the sensitive-value
		// diagnostic and the generic failure diagnostic must be emitted.
		hclCtx := makeVarCtx(cty.StringVal("secret").Mark(marks.Sensitive))
		rule := makeFakeRule(
			parseExpr(t, `var.foo == "good"`),
			parseExpr(t, `"Value '${var.foo}' is not allowed."`),
		)
		diags := evalVariableValidation(rule, hclCtx, valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Error message refers to sensitive values"
		})
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid value for variable"
		})
	})

	t.Run("ephemeral variable referenced in error message, condition passes → flagged even on success", func(t *testing.T) {
		// The condition passes but the error_message references an ephemeral
		// variable, making the message itself ephemeral.  This structural
		// problem must still be reported.
		hclCtx := makeVarCtx(cty.StringVal("good").Mark(marks.Ephemeral))
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.True),
			parseExpr(t, `"Value '${var.foo}' is not allowed."`),
		)
		diags := evalVariableValidation(rule, hclCtx, valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Error message refers to ephemeral values"
		})
		for _, d := range diags {
			if d.Description().Summary == "Invalid value for variable" {
				t.Errorf("unexpected 'Invalid value for variable' when condition passed")
			}
		}
	})

	// --- Condition evaluation error ---

	t.Run("condition evaluation error → early return with HCL error", func(t *testing.T) {
		// When the condition expression itself fails to evaluate (e.g. it references
		// an undefined variable), evalVariableValidation must return early with the
		// evaluation error and must NOT emit "Invalid value for variable" or
		// "Invalid variable validation result".
		rule := makeFakeRule(
			parseExpr(t, "undefined_var.foo"),
			hcltest.MockExprLiteral(cty.StringVal("Value is invalid.")),
		)
		// hclCtx only has "var", so "undefined_var" is unknown → evaluation error.
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("anything")), valueRange)
		if !diags.HasErrors() {
			t.Fatal("expected at least one error diagnostic, got none")
		}
		for _, d := range diags {
			if d.Description().Summary == "Invalid value for variable" {
				t.Errorf("unexpected 'Invalid value for variable' on condition evaluation error")
			}
			if d.Description().Summary == "Invalid variable validation result" {
				t.Errorf("unexpected 'Invalid variable validation result' on condition evaluation error")
			}
		}
	})

	// --- Non-bool condition result ---

	t.Run("condition result is non-bool (list) → Invalid variable validation result", func(t *testing.T) {
		// A condition that returns a list (or any value that cannot be converted
		// to bool) hits the convert.Convert(result, cty.Bool) failure path.
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.ListValEmpty(cty.String)),
			hcltest.MockExprLiteral(cty.StringVal("Value is invalid.")),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("anything")), valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid variable validation result"
		})
		// Must return early — no "Invalid value for variable" should follow.
		for _, d := range diags {
			if d.Description().Summary == "Invalid value for variable" {
				t.Errorf("unexpected 'Invalid value for variable' when condition type conversion failed")
			}
		}
	})

	// --- Null error message ---

	t.Run("null error message, condition fails → Invalid value for variable with fallback text", func(t *testing.T) {
		// A null error_message is skipped during string conversion; the framework
		// falls back to "Failed to evaluate condition error message." in the
		// detail of the "Invalid value for variable" diagnostic.
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.False),
			hcltest.MockExprLiteral(cty.NullVal(cty.String)),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("bad")), valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid value for variable" &&
				strings.Contains(d.Description().Detail, "Failed to evaluate condition error message.")
		})
	})

	// --- Non-string error message ---

	t.Run("non-string error message (list), condition fails → Invalid error message + fallback in failure diag", func(t *testing.T) {
		// An error_message that evaluates to a list (unconvertible to string)
		// hits the convert.Convert(errorValue, cty.String) failure path. Both
		// "Invalid error message" and "Invalid value for variable" (with the
		// fallback text) must be emitted.
		rule := makeFakeRule(
			hcltest.MockExprLiteral(cty.False),
			hcltest.MockExprLiteral(cty.ListValEmpty(cty.String)),
		)
		diags := evalVariableValidation(rule, makeVarCtx(cty.StringVal("bad")), valueRange)
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid error message"
		})
		assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
			return d.Severity() == tfdiags.Error &&
				d.Description().Summary == "Invalid value for variable" &&
				strings.Contains(d.Description().Detail, "Failed to evaluate condition error message.")
		})
	})
}

// TestInputVariableValidation exercises evalVariableValidations end-to-end
// through CheckValue, using the "validation" fixture that declares variables
// with validation blocks.
func TestInputVariableValidation(t *testing.T) {
	cfg := testStackConfig(t, "input_variable", "validation")

	tests := map[string]struct {
		varName       string
		inputVal      cty.Value
		wantSummaries []string // diagnostics that MUST be present (by Summary)
		wantNoErrors  bool     // if true, no error diagnostics are expected
	}{
		// --- validated (plain error message) ---
		"validated: clean pass": {
			varName:      "validated",
			inputVal:     cty.StringVal("good"),
			wantNoErrors: true,
		},
		"validated: clean fail": {
			varName:       "validated",
			inputVal:      cty.StringVal("bad"),
			wantSummaries: []string{"Invalid value for variable"},
		},

		// --- with_msg_ref (error message interpolates var.with_msg_ref) ---
		"with_msg_ref: clean pass": {
			varName:      "with_msg_ref",
			inputVal:     cty.StringVal("good"),
			wantNoErrors: true,
		},
		"with_msg_ref: clean fail": {
			varName:       "with_msg_ref",
			inputVal:      cty.StringVal("bad"),
			wantSummaries: []string{"Invalid value for variable"},
		},
		"with_msg_ref: sensitive value passes → error message diag only": {
			// Condition passes, but the interpolated error_message is sensitive
			// → we should still flag the structural problem.
			varName:       "with_msg_ref",
			inputVal:      cty.StringVal("good").Mark(marks.Sensitive),
			wantSummaries: []string{"Error message refers to sensitive values"},
		},
		"with_msg_ref: sensitive value fails → both diags": {
			varName:  "with_msg_ref",
			inputVal: cty.StringVal("bad").Mark(marks.Sensitive),
			wantSummaries: []string{
				"Error message refers to sensitive values",
				"Invalid value for variable",
			},
		},
		"with_msg_ref: ephemeral value passes → error message diag only": {
			varName:       "with_msg_ref",
			inputVal:      cty.StringVal("good").Mark(marks.Ephemeral),
			wantSummaries: []string{"Error message refers to ephemeral values"},
		},
		"with_msg_ref: ephemeral value fails → both diags": {
			varName:  "with_msg_ref",
			inputVal: cty.StringVal("bad").Mark(marks.Ephemeral),
			wantSummaries: []string{
				"Error message refers to ephemeral values",
				"Invalid value for variable",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				InputVariableValues: map[string]cty.Value{
					tc.varName: tc.inputVal,
				},
			})
			inPromisingTask(t, func(ctx context.Context, t *testing.T) {
				mainStack := main.MainStack()
				rootVar := mainStack.InputVariable(stackaddrs.InputVariable{Name: tc.varName})
				_, diags := rootVar.CheckValue(ctx, InspectPhase)

				if tc.wantNoErrors {
					if diags.HasErrors() {
						t.Errorf("unexpected errors: %s", diags.Err())
					}
					return
				}

				for _, wantSummary := range tc.wantSummaries {
					wantSummary := wantSummary // capture for closure
					assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
						return d.Severity() == tfdiags.Error &&
							d.Description().Summary == wantSummary
					})
				}
			})
		})
	}
}

// TestInputVariableValidationWithProviderFunction verifies that provider-defined
// functions can be called inside a variable validation condition expression.
// It uses the "validation_provider_function" fixture together with a mock provider
// that exposes a simple "upper" string function.
func TestInputVariableValidationWithProviderFunction(t *testing.T) {
	cfg := testStackConfig(t, "input_variable", "validation_provider_function")
	providerTypeAddr := addrs.MustParseProviderSourceString("terraform.io/builtin/testing")

	newMockProvider := func(t *testing.T) (*testing_provider.MockProvider, providers.Factory) {
		t.Helper()
		mockProvider := &testing_provider.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				Functions: map[string]providers.FunctionDecl{
					"upper": {
						Parameters: []providers.FunctionParam{
							{Name: "input", Type: cty.String},
						},
						ReturnType: cty.String,
						Summary:    "Converts a string to upper-case.",
					},
				},
			},
			CallFunctionFn: func(req providers.CallFunctionRequest) providers.CallFunctionResponse {
				if req.FunctionName != "upper" {
					return providers.CallFunctionResponse{
						Err: fmt.Errorf("unexpected function call: %s", req.FunctionName),
					}
				}
				input, _ := req.Arguments[0].Unmark()
				return providers.CallFunctionResponse{
					Result: cty.StringVal(strings.ToUpper(input.AsString())),
				}
			},
		}
		return mockProvider, providers.FactoryFixed(mockProvider)
	}

	t.Run("passes validation", func(t *testing.T) {
		_, providerFactory := newMockProvider(t)
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			InputVariableValues: map[string]cty.Value{
				"foo": cty.StringVal("hello"), // upper("hello") == "HELLO" → condition passes
			},
			ProviderFactories: ProviderFactories{
				providerTypeAddr: providerFactory,
			},
		})
		inPromisingTask(t, func(ctx context.Context, t *testing.T) {
			mainStack := main.MainStack()
			rootVar := mainStack.InputVariable(stackaddrs.InputVariable{Name: "foo"})
			_, diags := rootVar.CheckValue(ctx, InspectPhase)
			assertNoDiags(t, diags)
		})
	})

	t.Run("fails validation", func(t *testing.T) {
		_, providerFactory := newMockProvider(t)
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			InputVariableValues: map[string]cty.Value{
				"foo": cty.StringVal("world"), // upper("world") == "WORLD" ≠ "HELLO" → condition fails
			},
			ProviderFactories: ProviderFactories{
				providerTypeAddr: providerFactory,
			},
		})
		inPromisingTask(t, func(ctx context.Context, t *testing.T) {
			mainStack := main.MainStack()
			rootVar := mainStack.InputVariable(stackaddrs.InputVariable{Name: "foo"})
			_, diags := rootVar.CheckValue(ctx, InspectPhase)
			assertMatchingDiag(t, diags, func(d tfdiags.Diagnostic) bool {
				return d.Severity() == tfdiags.Error &&
					d.Description().Summary == "Invalid value for variable"
			})
		})
	})
}

// TestInputVariableMultipleValidationRules verifies that when a variable has
// more than one validation block, every failing rule produces its own
// diagnostic — i.e., all rules are evaluated and none are short-circuited.
//
// The "multi_rule" variable in the "validation" fixture has two rules:
//
//	Rule 1: length(var.multi_rule) >= 5
//	Rule 2: var.multi_rule != "bad"
//
// The value "bad" has length 3 (< 5) and equals "bad", so it violates both
// rules simultaneously, giving us exactly two "Invalid value for variable"
// diagnostics.
func TestInputVariableMultipleValidationRules(t *testing.T) {
	cfg := testStackConfig(t, "input_variable", "validation")

	tests := map[string]struct {
		inputVal     cty.Value
		wantErrCount int // expected number of "Invalid value for variable" diagnostics
	}{
		"passes both rules": {
			inputVal:     cty.StringVal("hello"), // length 5 >= 5, != "bad"
			wantErrCount: 0,
		},
		"fails first rule only": {
			inputVal:     cty.StringVal("hi"), // length 2 < 5, != "bad"
			wantErrCount: 1,
		},
		"fails second rule only": {
			// length("hello!") = 6 >= 5 → first passes; "hello!" != "bad" → second passes.
			// To fail only the second rule we need length >= 5 AND value == "bad".
			// "bad" itself has length 3, so the only way to isolate rule 2 failure
			// is with a longer value that equals "bad" — impossible for a plain
			// string.  We therefore omit this sub-case and rely on the unit-level
			// TestEvalVariableValidation coverage instead.
			inputVal:     cty.StringVal("hello"), // deliberately a pass case
			wantErrCount: 0,
		},
		"fails both rules": {
			inputVal:     cty.StringVal("bad"), // length 3 < 5 AND == "bad"
			wantErrCount: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				InputVariableValues: map[string]cty.Value{
					"multi_rule": tc.inputVal,
				},
			})
			inPromisingTask(t, func(ctx context.Context, t *testing.T) {
				mainStack := main.MainStack()
				rootVar := mainStack.InputVariable(stackaddrs.InputVariable{Name: "multi_rule"})
				_, diags := rootVar.CheckValue(ctx, InspectPhase)

				var failCount int
				for _, d := range diags {
					if d.Severity() == tfdiags.Error && d.Description().Summary == "Invalid value for variable" {
						failCount++
					}
				}
				if failCount != tc.wantErrCount {
					t.Errorf("expected %d 'Invalid value for variable' diagnostic(s), got %d; diags:\n%s",
						tc.wantErrCount, failCount, diags.ErrWithWarnings())
				}
			})
		})
	}
}
