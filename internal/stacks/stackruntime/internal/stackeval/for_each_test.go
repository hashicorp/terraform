// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestEvaluateForEachExpr(t *testing.T) {
	tests := map[string]struct {
		Expr    hcl.Expression
		Want    cty.Value
		WantErr string
	}{
		// Objects
		"empty object": {
			Expr: hcltest.MockExprLiteral(cty.EmptyObjectVal),
			Want: cty.EmptyObjectVal,
		},
		"non-empty object": {
			Expr: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("beep"),
				"b": cty.StringVal("beep"),
			})),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("beep"),
				"b": cty.StringVal("beep"),
			}),
		},
		"unknown object": {
			Expr: hcltest.MockExprLiteral(cty.UnknownVal(cty.EmptyObject)),
			Want: cty.UnknownVal(cty.EmptyObject),
		},

		// Maps
		"map of string": {
			Expr: hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
				"a": cty.StringVal("beep"),
				"b": cty.StringVal("boop"),
			})),
			Want: cty.MapVal(map[string]cty.Value{
				"a": cty.StringVal("beep"),
				"b": cty.StringVal("boop"),
			}),
		},
		"empty map of string": {
			Expr: hcltest.MockExprLiteral(cty.MapValEmpty(cty.String)),
			Want: cty.MapValEmpty(cty.String),
		},
		"unknown map of string": {
			Expr: hcltest.MockExprLiteral(cty.UnknownVal(cty.Map(cty.String))),
			Want: cty.UnknownVal(cty.Map(cty.String)),
		},
		"sensitive map of string": {
			Expr:    hcltest.MockExprLiteral(cty.MapValEmpty(cty.String).Mark(marks.Sensitive)),
			WantErr: `Invalid for_each value`,
		},
		"map of object": {
			Expr: hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
				"a": cty.EmptyObjectVal,
				"b": cty.EmptyObjectVal,
			})),
			Want: cty.MapVal(map[string]cty.Value{
				"a": cty.EmptyObjectVal,
				"b": cty.EmptyObjectVal,
			}),
		},
		"empty map of object": {
			Expr: hcltest.MockExprLiteral(cty.MapValEmpty(cty.EmptyObject)),
			Want: cty.MapValEmpty(cty.EmptyObject),
		},

		// Sets
		"set of string": {
			Expr: hcltest.MockExprLiteral(cty.SetVal([]cty.Value{
				cty.StringVal("beep"),
				cty.StringVal("boop"),
			})),
			Want: cty.SetVal([]cty.Value{
				cty.StringVal("beep"),
				cty.StringVal("boop"),
			}),
		},
		"empty set of string": {
			Expr: hcltest.MockExprLiteral(cty.SetValEmpty(cty.String)),
			Want: cty.SetValEmpty(cty.String),
		},
		"unknown set of string": {
			Expr: hcltest.MockExprLiteral(cty.UnknownVal(cty.Set(cty.String))),
			Want: cty.UnknownVal(cty.Set(cty.String)),
		},
		"empty set": {
			Expr: hcltest.MockExprLiteral(cty.SetValEmpty(cty.EmptyTuple)),
			Want: cty.SetValEmpty(cty.EmptyTuple),
		},
		"sensitive set of string": {
			Expr:    hcltest.MockExprLiteral(cty.SetValEmpty(cty.String).Mark(marks.Sensitive)),
			WantErr: `Invalid for_each value`,
		},
		"empty set of object": {
			Expr: hcltest.MockExprLiteral(cty.SetValEmpty(cty.EmptyObject)),
			Want: cty.SetValEmpty(cty.EmptyObject),
		},
		"set with null": {
			Expr:    hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.StringVal("valid"), cty.NullVal(cty.String)})),
			WantErr: `Invalid for_each value`,
		},

		// Nulls of any type are not allowed
		"null object": {
			Expr:    hcltest.MockExprLiteral(cty.NullVal(cty.EmptyObject)),
			WantErr: `Invalid for_each value`,
		},
		"null map": {
			Expr:    hcltest.MockExprLiteral(cty.NullVal(cty.Map(cty.String))),
			WantErr: `Invalid for_each value`,
		},
		"null set": {
			Expr:    hcltest.MockExprLiteral(cty.NullVal(cty.Set(cty.String))),
			WantErr: `Invalid for_each value`,
		},
		"null string": {
			Expr:    hcltest.MockExprLiteral(cty.NullVal(cty.String)),
			WantErr: `Invalid for_each value`,
		},
	}

	ctx := context.Background()
	scope := newStaticExpressionScope()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotResult, diags := evaluateForEachExpr(ctx, test.Expr, PlanPhase, scope, "test")
			got := gotResult.Value

			if test.WantErr != "" {
				if !diags.HasErrors() {
					t.Fatalf("unexpected success; want error\ngot: %#v", got)
				}
				foundErr := false
				for _, diag := range diags {
					if diag.Severity() != tfdiags.Error {
						continue
					}
					if diag.Description().Summary == test.WantErr {
						foundErr = true
						break
					}
				}
				if !foundErr {
					t.Errorf("missing expected error\nwant summary: %s\ngot: %s", test.WantErr, spew.Sdump(diags.ForRPC()))
				}
				return
			}

			if diags.HasErrors() {
				t.Errorf("unexpected errors\n%s", spew.Sdump(diags.ForRPC()))
			}
			if !test.Want.RawEquals(got) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestInstancesMap(t *testing.T) {
	type InstanceObj struct {
		Key addrs.InstanceKey
		Rep instances.RepetitionData
	}
	makeObj := func(k addrs.InstanceKey, r instances.RepetitionData) InstanceObj {
		return InstanceObj{
			Key: k,
			Rep: r,
		}
	}

	tests := []struct {
		Input cty.Value
		Want  map[addrs.InstanceKey]InstanceObj

		// This function always either succeeds or panics, because it
		// expects to be given already-validated input from another function.
		// We're only testing the success cases here.
	}{
		// No for_each at all
		{
			cty.NilVal,
			map[addrs.InstanceKey]InstanceObj{
				addrs.NoKey: {
					Key: addrs.NoKey,
					Rep: instances.RepetitionData{
						// No data available for the non-repeating case
					},
				},
			},
		},

		// Unknowns
		{
			cty.UnknownVal(cty.EmptyObject),
			nil, // a nil map means "unknown" for this function
		},
		{
			cty.UnknownVal(cty.Map(cty.Bool)),
			nil, // a nil map means "unknown" for this function
		},
		{
			cty.UnknownVal(cty.Set(cty.String)),
			nil, // a nil map means "unknown" for this function
		},

		// Empties
		{
			cty.EmptyObjectVal,
			map[addrs.InstanceKey]InstanceObj{
				// intentionally a non-nil empty map to assert that we know
				// that there are zero instances, rather than that we don't
				// know how many there are.
			},
		},
		{
			cty.MapValEmpty(cty.String),
			map[addrs.InstanceKey]InstanceObj{
				// intentionally a non-nil empty map to assert that we know
				// that there are zero instances, rather than that we don't
				// know how many there are.
			},
		},
		{
			cty.SetValEmpty(cty.String),
			map[addrs.InstanceKey]InstanceObj{
				// intentionally a non-nil empty map to assert that we know
				// that there are zero instances, rather than that we don't
				// know how many there are.
			},
		},

		// Known and not empty
		{
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("beep"),
				"b": cty.StringVal("boop"),
			}),
			map[addrs.InstanceKey]InstanceObj{
				addrs.StringKey("a"): {
					Key: addrs.StringKey("a"),
					Rep: instances.RepetitionData{
						EachKey:   cty.StringVal("a"),
						EachValue: cty.StringVal("beep"),
					},
				},
				addrs.StringKey("b"): {
					Key: addrs.StringKey("b"),
					Rep: instances.RepetitionData{
						EachKey:   cty.StringVal("b"),
						EachValue: cty.StringVal("boop"),
					},
				},
			},
		},
		{
			cty.MapVal(map[string]cty.Value{
				"a": cty.StringVal("beep"),
				"b": cty.StringVal("boop"),
			}),
			map[addrs.InstanceKey]InstanceObj{
				addrs.StringKey("a"): {
					Key: addrs.StringKey("a"),
					Rep: instances.RepetitionData{
						EachKey:   cty.StringVal("a"),
						EachValue: cty.StringVal("beep"),
					},
				},
				addrs.StringKey("b"): {
					Key: addrs.StringKey("b"),
					Rep: instances.RepetitionData{
						EachKey:   cty.StringVal("b"),
						EachValue: cty.StringVal("boop"),
					},
				},
			},
		},
		{
			cty.SetVal([]cty.Value{
				cty.StringVal("beep"),
				cty.StringVal("boop"),
			}),
			map[addrs.InstanceKey]InstanceObj{
				addrs.StringKey("beep"): {
					Key: addrs.StringKey("beep"),
					Rep: instances.RepetitionData{
						EachKey:   cty.StringVal("beep"),
						EachValue: cty.StringVal("beep"),
					},
				},
				addrs.StringKey("boop"): {
					Key: addrs.StringKey("boop"),
					Rep: instances.RepetitionData{
						EachKey:   cty.StringVal("boop"),
						EachValue: cty.StringVal("boop"),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s", test.Input), func(t *testing.T) {
			got := instancesMap(test.Input, makeObj)

			if diff := cmp.Diff(test.Want, got, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong result\ninput: %#v\n%s", test.Input, diff)
			}
		})
	}
}
