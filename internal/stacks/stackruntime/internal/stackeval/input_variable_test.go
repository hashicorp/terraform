// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/zclconf/go-cty/cty"
)

func TestInputVariableValue(t *testing.T) {
	ctx := context.Background()
	cfg := testStackConfig(t, "input_variable", "basics")

	// NOTE: This also indirectly tests the propagation of input values
	// from a parent stack into one of itschildren, even though that's
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
