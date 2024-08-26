// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/encoding/prototext"
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
					mainStack := main.MainStack(ctx)
					rootVar := mainStack.InputVariable(ctx, stackaddrs.InputVariable{Name: "name"})
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
						rootVar := childStack.InputVariable(ctx, stackaddrs.InputVariable{Name: "name"})
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
				childStackCall := main.MainStack(ctx).EmbeddedStackCall(ctx, childStackCallAddr)
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

				aVar := childStack.InputVariable(ctx, aVarAddr)
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
				mainStack := main.MainStack(ctx)
				rootVar := mainStack.InputVariable(ctx, stackaddrs.InputVariable{Name: "name"})
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
