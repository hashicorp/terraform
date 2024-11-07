// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestOutputValueResultValue(t *testing.T) {
	ctx := context.Background()
	cfg := testStackConfig(t, "output_value", "basics")

	// NOTE: This also indirectly tests the propagation of output values
	// from a child stack into its parent, even though that's technically
	// the responsibility of [StackCall] rather than [OutputValue],
	// because propagating upward from child stacks is a major purpose
	// of output values that must keep working.
	childStackAddr := stackaddrs.RootStackInstance.Child("child", addrs.NoKey)

	tests := map[string]struct {
		RootVal      cty.Value
		ChildVal     cty.Value
		WantRootVal  cty.Value
		WantChildVal cty.Value
		WantRootErr  string
		WantChildErr string
	}{
		"valid with no type conversions": {
			RootVal:  cty.StringVal("root value"),
			ChildVal: cty.StringVal("child value"),

			WantRootVal:  cty.StringVal("root value"),
			WantChildVal: cty.StringVal("child value"),
		},
		"valid after type conversions": {
			RootVal:  cty.True,
			ChildVal: cty.NumberIntVal(4),

			WantRootVal:  cty.StringVal("true"),
			WantChildVal: cty.StringVal("4"),
		},
		"type mismatch root": {
			RootVal:  cty.EmptyObjectVal,
			ChildVal: cty.StringVal("irrelevant"),

			WantRootVal:  cty.UnknownVal(cty.String),
			WantChildVal: cty.StringVal("irrelevant"),

			WantRootErr: `Unsuitable value for output "root": string required.`,
		},
		"type mismatch child": {
			RootVal:  cty.StringVal("irrelevant"),
			ChildVal: cty.EmptyTupleVal,

			WantRootVal:  cty.StringVal("irrelevant"),
			WantChildVal: cty.UnknownVal(cty.String),

			WantChildErr: `Unsuitable value for output "foo": string required.`,
		},
		"dynamic value placeholders": {
			RootVal:  cty.DynamicVal,
			ChildVal: cty.DynamicVal,

			WantRootVal:  cty.UnknownVal(cty.String),
			WantChildVal: cty.UnknownVal(cty.String),
		},
		"ephemeral value when not allowed": {
			RootVal:  cty.StringVal("root value").Mark(marks.Ephemeral),
			ChildVal: cty.StringVal("child value").Mark(marks.Ephemeral),

			WantRootVal:  cty.UnknownVal(cty.String),
			WantChildVal: cty.UnknownVal(cty.String),

			WantRootErr:  `The output value "root" does not accept ephemeral values.`,
			WantChildErr: `The output value "foo" does not accept ephemeral values.`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"root_output":  test.RootVal,
					"child_output": test.ChildVal,
				},
			})

			t.Run("root", func(t *testing.T) {
				promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
					mainStack := main.MainStack(ctx)
					rootOutput := mainStack.OutputValues(ctx)[stackaddrs.OutputValue{Name: "root"}]
					if rootOutput == nil {
						t.Fatal("root output value doesn't exist at all")
					}
					got, diags := rootOutput.CheckResultValue(ctx, InspectPhase)

					if wantErr := test.WantRootErr; wantErr != "" {
						if !diags.HasErrors() {
							t.Errorf("unexpected success\ngot: %#v\nwant error: %s", got, wantErr)
						}
						if len(diags) != 1 {
							t.Fatalf("extraneous diagnostics\n%s", diags.Err())
						}
						if gotErr := diags[0].Description().Detail; gotErr != wantErr {
							t.Errorf("wrong error message detail\ngot:  %s\nwant: %s", gotErr, wantErr)
						}
						return struct{}{}, nil
					}

					if diags.HasErrors() {
						t.Errorf("unexpected errors\n%s", diags.Err())
					}
					want := test.WantRootVal
					if !want.RawEquals(got) {
						t.Errorf("wrong value\ngot:  %#v\nwant: %#v", got, want)
					}
					return struct{}{}, nil
				})
			})
			t.Run("child", func(t *testing.T) {
				t.Run("from the child perspective", func(t *testing.T) {
					promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
						childStack := main.Stack(ctx, childStackAddr, InspectPhase)
						if childStack == nil {
							t.Fatal("child stack doesn't exist at all")
						}
						childOutput := childStack.OutputValues(ctx)[stackaddrs.OutputValue{Name: "foo"}]
						if childOutput == nil {
							t.Fatal("child output value doesn't exist at all")
						}
						got, diags := childOutput.CheckResultValue(ctx, InspectPhase)

						if wantErr := test.WantChildErr; wantErr != "" {
							if !diags.HasErrors() {
								t.Errorf("unexpected success\ngot: %#v\nwant error: %s", got, wantErr)
							}
							if len(diags) != 1 {
								t.Fatalf("extraneous diagnostics\n%s", diags.Err())
							}
							if gotErr := diags[0].Description().Detail; gotErr != wantErr {
								t.Errorf("wrong error message detail\ngot:  %s\nwant: %s", gotErr, wantErr)
							}
							return struct{}{}, nil
						}

						if diags.HasErrors() {
							t.Errorf("unexpected errors\n%s", diags.Err())
						}
						want := test.WantChildVal
						if !want.RawEquals(got) {
							t.Errorf("wrong value\ngot:  %#v\nwant: %#v", got, want)
						}
						return struct{}{}, nil
					})
				})
				t.Run("from the root perspective", func(t *testing.T) {
					promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
						mainStack := main.MainStack(ctx)
						childOutput := mainStack.OutputValues(ctx)[stackaddrs.OutputValue{Name: "child"}]
						if childOutput == nil {
							t.Fatal("child output value doesn't exist at all")
						}
						got, diags := childOutput.CheckResultValue(ctx, InspectPhase)

						// We should never see any errors when viewed from the
						// root perspective, because the root output value
						// only reports its _own_ errors, not the indirect
						// errors caused by things it refers to.
						if diags.HasErrors() {
							t.Errorf("unexpected errors\n%s", diags.Err())
						}
						want := test.WantChildVal
						if !want.RawEquals(got) {
							t.Errorf("wrong value\ngot:  %#v\nwant: %#v", got, want)
						}
						return struct{}{}, nil
					})
				})
			})
		})
	}
}

func TestOutputValueEphemeral(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		fixtureName                 string
		givenVal                    cty.Value
		allowed                     bool
		expectedDiagnosticSummaries []string
		wantVal                     cty.Value
	}{
		"ephemeral and declared as ephemeral": {
			fixtureName:                 "ephemeral_yes",
			givenVal:                    cty.StringVal("beep").Mark(marks.Ephemeral),
			allowed:                     false,
			expectedDiagnosticSummaries: []string{"Ephemeral output value not allowed on root stack"},
			wantVal:                     cty.StringVal("beep").Mark(marks.Ephemeral),
		},
		"ephemeral and not declared as ephemeral": {
			fixtureName:                 "ephemeral_no",
			givenVal:                    cty.StringVal("beep").Mark(marks.Ephemeral),
			allowed:                     false,
			expectedDiagnosticSummaries: []string{"Ephemeral value not allowed"},
			wantVal:                     cty.UnknownVal(cty.String),
		},
		"non-ephemeral and declared as ephemeral": {
			fixtureName:                 "ephemeral_yes",
			givenVal:                    cty.StringVal("beep"),
			allowed:                     false,
			expectedDiagnosticSummaries: []string{"Ephemeral output value not allowed on root stack", "Expected ephemeral value"},
			wantVal:                     cty.StringVal("beep").Mark(marks.Ephemeral),
		},
		"non-ephemeral and not declared as ephemeral": {
			fixtureName: "ephemeral_no",
			givenVal:    cty.StringVal("beep"),
			allowed:     true,
			wantVal:     cty.StringVal("beep"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := testStackConfig(t, "output_value", test.fixtureName)
			outputAddr := stackaddrs.OutputValue{Name: "result"}

			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"result": test.givenVal,
				},
			})

			promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
				stack := main.MainStack(ctx)
				output := stack.OutputValues(ctx)[outputAddr]
				if output == nil {
					t.Fatalf("missing %s", outputAddr)
				}
				want := test.wantVal
				got, diags := output.CheckResultValue(ctx, InspectPhase)
				if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
					t.Errorf("wrong value for %s\n%s", outputAddr, diff)
				}

				if test.allowed {
					if diags.HasErrors() {
						t.Errorf("unexpected errors\n%s", diags.Err().Error())
					}
				} else {
					if !diags.HasErrors() {
						t.Fatalf("no errors; should have failed")
					}

					foundDiagSummaries := make(map[string]bool)
					for _, diag := range diags {
						summary := diag.Description().Summary
						foundDiagSummaries[summary] = true
					}

					if len(foundDiagSummaries) != len(test.expectedDiagnosticSummaries) {
						t.Fatalf("wrong number of diagnostics, expected %v, got \n%s", test.expectedDiagnosticSummaries, diags.Err().Error())
					}

					for _, expectedSummary := range test.expectedDiagnosticSummaries {
						if !foundDiagSummaries[expectedSummary] {
							t.Fatalf("missing diagnostic with summary %s", expectedSummary)
						}
					}
				}
				return struct{}{}, nil
			})
		})
	}
}

func TestOutputValueEphemeralInChildStack(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		fixtureName                 string
		givenVal                    cty.Value
		allowed                     bool
		expectedDiagnosticSummaries []string
		wantVal                     cty.Value
	}{
		"ephemeral and declared as ephemeral": {
			fixtureName: "ephemeral_child",
			givenVal:    cty.StringVal("beep").Mark(marks.Ephemeral),
			allowed:     true,
			wantVal:     cty.StringVal("beep").Mark(marks.Ephemeral),
		},
		"non-ephemeral and declared as ephemeral": {
			fixtureName:                 "ephemeral_child",
			givenVal:                    cty.StringVal("beep"),
			allowed:                     false,
			expectedDiagnosticSummaries: []string{"Expected ephemeral value"},
			wantVal:                     cty.StringVal("beep").Mark(marks.Ephemeral),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := testStackConfig(t, "output_value", test.fixtureName)
			outputAddr := stackaddrs.OutputValue{Name: "result"}

			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"result": test.givenVal,
				},
			})

			promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
				rootStack := main.MainStack(ctx)
				childStackStep := stackaddrs.StackInstanceStep{
					Name: "child",
					Key:  addrs.NoKey,
				}
				stack := rootStack.ChildStackChecked(ctx, childStackStep, ValidatePhase)
				output := stack.OutputValues(ctx)[outputAddr]
				if output == nil {
					t.Fatalf("missing %s", outputAddr)
				}
				want := test.wantVal
				got, diags := output.CheckResultValue(ctx, InspectPhase)
				if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
					t.Errorf("wrong value for %s\n%s", outputAddr, diff)
				}

				if test.allowed {
					if diags.HasErrors() {
						t.Errorf("unexpected errors\n%s", diags.Err().Error())
					}
				} else {
					if !diags.HasErrors() {
						t.Fatalf("no errors; should have failed")
					}

					foundDiagSummaries := make(map[string]bool)
					for _, diag := range diags {
						summary := diag.Description().Summary
						foundDiagSummaries[summary] = true
					}

					if len(foundDiagSummaries) != len(test.expectedDiagnosticSummaries) {
						t.Fatalf("wrong number of diagnostics, expected %v, got \n%s", test.expectedDiagnosticSummaries, diags.Err().Error())
					}

					for _, expectedSummary := range test.expectedDiagnosticSummaries {
						if !foundDiagSummaries[expectedSummary] {
							t.Fatalf("missing diagnostic with summary %s", expectedSummary)
						}
					}
				}
				return struct{}{}, nil
			})
		})
	}
}
