// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestStackCallCheckInstances(t *testing.T) {
	getStackCall := func(ctx context.Context, main *Main) *StackCall {
		mainStack := main.MainStack(ctx)
		call := mainStack.EmbeddedStackCall(ctx, stackaddrs.StackCall{Name: "child"})
		if call == nil {
			t.Fatal("stack.child does not exist, but it should exist")
		}
		return call
	}

	subtestInPromisingTask(t, "single instance", func(ctx context.Context, t *testing.T) {
		cfg := testStackConfig(t, "stack_call", "single_instance")
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			TestOnlyGlobals: map[string]cty.Value{
				"child_stack_inputs": cty.EmptyObjectVal,
			},
		})

		call := getStackCall(ctx, main)
		forEachVal, diags := call.CheckForEachValue(ctx, InspectPhase)
		assertNoDiags(t, diags)
		if forEachVal != cty.NilVal {
			t.Fatalf("unexpected for_each value\ngot:  %#v\nwant: cty.NilVal", forEachVal)
		}

		insts, diags := call.CheckInstances(ctx, InspectPhase)
		assertNoDiags(t, diags)
		if got, want := len(insts), 1; got != want {
			t.Fatalf("wrong number of instances %d; want %d\n%#v", got, want, insts)
		}
		inst, ok := insts[addrs.NoKey]
		if !ok {
			t.Fatalf("missing expected addrs.NoKey instance\n%s", spew.Sdump(insts))
		}
		if diff := cmp.Diff(instances.RepetitionData{}, inst.RepetitionData(), ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong repetition data\n%s", diff)
		}
	})
	t.Run("for_each", func(t *testing.T) {
		cfg := testStackConfig(t, "stack_call", "for_each")

		subtestInPromisingTask(t, "no instances", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"child_stack_for_each": cty.MapValEmpty(cty.EmptyObject),
				},
			})

			call := getStackCall(ctx, main)
			forEachVal, diags := call.CheckForEachValue(ctx, InspectPhase)
			assertNoDiags(t, diags)
			if got, want := forEachVal, cty.MapValEmpty(cty.EmptyObject); !want.RawEquals(got) {
				t.Fatalf("unexpected for_each value\ngot:  %#v\nwant: %#v", got, want)
			}
			insts, diags := call.CheckInstances(ctx, InspectPhase)
			assertNoDiags(t, diags)
			if got, want := len(insts), 0; got != want {
				t.Fatalf("wrong number of instances %d; want %d\n%#v", got, want, insts)
			}

			// For this particular function we take the unusual approach of
			// distinguishing between a nil map and a non-nil empty map so
			// we can distinguish between "definitely no instances" (this case)
			// and "we don't know how many instances there are" (tested in other
			// subtests of this test, below.)
			if insts == nil {
				t.Error("CheckInstances result is nil; should be non-nil empty map")
			}
		})
		subtestInPromisingTask(t, "two instances", func(ctx context.Context, t *testing.T) {
			wantForEachVal := cty.MapVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("in a"),
				}),
				"b": cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("in b"),
				}),
			})
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"child_stack_for_each": wantForEachVal,
				},
			})

			call := getStackCall(ctx, main)
			gotForEachVal, diags := call.CheckForEachValue(ctx, InspectPhase)
			assertNoDiags(t, diags)
			if !wantForEachVal.RawEquals(gotForEachVal) {
				t.Fatalf("unexpected for_each value\ngot:  %#v\nwant: %#v", gotForEachVal, wantForEachVal)
			}
			insts, diags := call.CheckInstances(ctx, InspectPhase)
			assertNoDiags(t, diags)
			if got, want := len(insts), 2; got != want {
				t.Fatalf("wrong number of instances %d; want %d\n%#v", got, want, insts)
			}
			t.Run("instance a", func(t *testing.T) {
				inst, ok := insts[addrs.StringKey("a")]
				if !ok {
					t.Fatalf("missing expected addrs.StringKey(\"a\") instance\n%s", spew.Sdump(insts))
				}
				wantRepData := instances.RepetitionData{
					EachKey: cty.StringVal("a"),
					EachValue: cty.ObjectVal(map[string]cty.Value{
						"test_string": cty.StringVal("in a"),
					}),
				}
				if diff := cmp.Diff(wantRepData, inst.RepetitionData(), ctydebug.CmpOptions); diff != "" {
					t.Errorf("wrong repetition data\n%s", diff)
				}
			})
			t.Run("instance b", func(t *testing.T) {
				inst, ok := insts[addrs.StringKey("b")]
				if !ok {
					t.Fatalf("missing expected addrs.StringKey(\"b\") instance\n%s", spew.Sdump(insts))
				}
				wantRepData := instances.RepetitionData{
					EachKey: cty.StringVal("b"),
					EachValue: cty.ObjectVal(map[string]cty.Value{
						"test_string": cty.StringVal("in b"),
					}),
				}
				if diff := cmp.Diff(wantRepData, inst.RepetitionData(), ctydebug.CmpOptions); diff != "" {
					t.Errorf("wrong repetition data\n%s", diff)
				}
			})
		})
		subtestInPromisingTask(t, "null", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"child_stack_for_each": cty.NullVal(cty.Map(cty.EmptyObject)),
				},
			})

			call := getStackCall(ctx, main)
			gotVal, diags := call.CheckForEachValue(ctx, InspectPhase)
			assertMatchingDiag(t, diags, func(diag tfdiags.Diagnostic) bool {
				return diag.Severity() == tfdiags.Error && strings.Contains(diag.Description().Detail, "The for_each expression produced a null value")
			})
			wantVal := cty.DynamicVal // placeholder for invalid result
			if !wantVal.RawEquals(gotVal) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", gotVal, wantVal)
			}
		})
		subtestInPromisingTask(t, "string", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"child_stack_for_each": cty.StringVal("nope"),
				},
			})

			call := getStackCall(ctx, main)
			gotVal, diags := call.CheckForEachValue(ctx, InspectPhase)
			assertMatchingDiag(t, diags, func(diag tfdiags.Diagnostic) bool {
				return (diag.Severity() == tfdiags.Error &&
					diag.Description().Detail == "The for_each expression must produce either a map of any type or a set of strings. The keys of the map or the set elements will serve as unique identifiers for multiple instances of this stack.")
			})
			wantVal := cty.DynamicVal // placeholder for invalid result
			if !wantVal.RawEquals(gotVal) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", gotVal, wantVal)
			}

			// When the for_each expression is invalid, CheckInstances should
			// return nil to represent that we don't know enough to predict
			// how many instances there are. This is a different result than
			// when we know there are zero instances, which would be a non-nil
			// empty map.
			gotInsts, diags := call.CheckInstances(ctx, InspectPhase)
			assertNoDiags(t, diags)
			if gotInsts != nil {
				t.Errorf("wrong instances; want nil\n%#v", gotInsts)
			}
		})
		subtestInPromisingTask(t, "unknown", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"child_stack_for_each": cty.UnknownVal(cty.Map(cty.EmptyObject)),
				},
			})

			// For now it's invalid to use an unknown value in for_each.
			// Later we're expecting to make this succeed but announce that
			// planning everything beneath this call must be deferred to a
			// future plan after everything else has been applied first.
			call := getStackCall(ctx, main)
			gotVal, diags := call.CheckForEachValue(ctx, InspectPhase)
			assertMatchingDiag(t, diags, func(diag tfdiags.Diagnostic) bool {
				return (diag.Severity() == tfdiags.Error &&
					diag.Description().Detail == "The for_each value must not be derived from values that will be determined only during the apply phase.")
			})
			wantVal := cty.UnknownVal(cty.Map(cty.EmptyObject))
			if !wantVal.RawEquals(gotVal) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", gotVal, wantVal)
			}

			// When the for_each expression is invalid, CheckInstances should
			// return nil to represent that we don't know enough to predict
			// how many instances there are. This is a different result than
			// when we know there are zero instances, which would be a non-nil
			// empty map.
			gotInsts, diags := call.CheckInstances(ctx, InspectPhase)
			assertNoDiags(t, diags)
			if gotInsts != nil {
				t.Errorf("wrong instances; want nil\n%#v", gotInsts)
			}
		})
	})
}

func TestStackCallResultValue(t *testing.T) {
	getStackCall := func(ctx context.Context, main *Main) *StackCall {
		mainStack := main.MainStack(ctx)
		call := mainStack.EmbeddedStackCall(ctx, stackaddrs.StackCall{Name: "child"})
		if call == nil {
			t.Fatal("stack.child does not exist, but it should exist")
		}
		return call
	}

	subtestInPromisingTask(t, "single instance", func(ctx context.Context, t *testing.T) {
		cfg := testStackConfig(t, "stack_call", "single_instance")
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			TestOnlyGlobals: map[string]cty.Value{
				"child_stack_inputs": cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("hello"),
					"test_map":    cty.MapValEmpty(cty.String),
				}),
			},
		})

		call := getStackCall(ctx, main)
		got := call.ResultValue(ctx, InspectPhase)
		want := cty.ObjectVal(map[string]cty.Value{
			"test_map":    cty.MapValEmpty(cty.String),
			"test_string": cty.StringVal("hello"),
		})
		if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
			t.Fatalf("wrong result\n%s", diff)
		}
	})
	t.Run("for_each", func(t *testing.T) {
		cfg := testStackConfig(t, "stack_call", "for_each")

		subtestInPromisingTask(t, "no instances", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"child_stack_for_each": cty.MapValEmpty(cty.EmptyObject),
				},
			})

			call := getStackCall(ctx, main)
			got := call.ResultValue(ctx, InspectPhase)
			want := cty.MapValEmpty(cty.Map(cty.Object(map[string]cty.Type{
				"test_string": cty.String,
				"test_map":    cty.Map(cty.String),
			})))
			if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
				t.Fatalf("wrong result\n%s", diff)
			}
		})
		subtestInPromisingTask(t, "two instances", func(ctx context.Context, t *testing.T) {
			forEachVal := cty.MapVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("in a"),
				}),
				"b": cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("in b"),
				}),
			})
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"child_stack_for_each": forEachVal,
				},
			})

			call := getStackCall(ctx, main)
			got := call.ResultValue(ctx, InspectPhase)
			want := cty.MapVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("in a"),
					"test_map":    cty.NullVal(cty.Map(cty.String)),
				}),
				"b": cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("in b"),
					"test_map":    cty.NullVal(cty.Map(cty.String)),
				}),
			})
			// FIXME: the cmp transformer ctydebug.CmpOptions seems to find
			// this particular pair of values troubling, causing it to get
			// into an infinite recursion. For now we'll just use RawEquals,
			// at the expense of a less helpful failure message. This seems
			// to be a bug in upstream ctydebug.
			if !want.RawEquals(got) {
				t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, want)
			}
		})
		subtestInPromisingTask(t, "null", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"child_stack_for_each": cty.NullVal(cty.Map(cty.EmptyObject)),
				},
			})

			call := getStackCall(ctx, main)
			got := call.ResultValue(ctx, InspectPhase)
			// When the for_each expression is invalid, the result value
			// is unknown so we can use it as a placeholder for partial
			// downstream checking.
			want := cty.UnknownVal(cty.Map(cty.Object(map[string]cty.Type{
				"test_map":    cty.Map(cty.String),
				"test_string": cty.String,
			})))
			// FIXME: the cmp transformer ctydebug.CmpOptions seems to find
			// this particular pair of values troubling, causing it to get
			// into an infinite recursion. For now we'll just use RawEquals,
			// at the expense of a less helpful failure message. This seems
			// to be a bug in upstream ctydebug.
			if !want.RawEquals(got) {
				t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, want)
			}
		})
		subtestInPromisingTask(t, "string", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"child_stack_for_each": cty.StringVal("nope"),
				},
			})

			call := getStackCall(ctx, main)
			got := call.ResultValue(ctx, InspectPhase)
			// When the for_each expression is invalid, the result value
			// is unknown so we can use it as a placeholder for partial
			// downstream checking.
			want := cty.UnknownVal(cty.Map(cty.Object(map[string]cty.Type{
				"test_map":    cty.Map(cty.String),
				"test_string": cty.String,
			})))
			// FIXME: the cmp transformer ctydebug.CmpOptions seems to find
			// this particular pair of values troubling, causing it to get
			// into an infinite recursion. For now we'll just use RawEquals,
			// at the expense of a less helpful failure message. This seems
			// to be a bug in upstream ctydebug.
			if !want.RawEquals(got) {
				t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, want)
			}
		})
		subtestInPromisingTask(t, "unknown", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"child_stack_for_each": cty.UnknownVal(cty.Map(cty.EmptyObject)),
				},
			})

			call := getStackCall(ctx, main)
			got := call.ResultValue(ctx, InspectPhase)
			// When the for_each expression is unknown, the result value
			// is unknown too so we can use it as a placeholder for partial
			// downstream checking.
			want := cty.UnknownVal(cty.Map(cty.Object(map[string]cty.Type{
				"test_map":    cty.Map(cty.String),
				"test_string": cty.String,
			})))
			// FIXME: the cmp transformer ctydebug.CmpOptions seems to find
			// this particular pair of values troubling, causing it to get
			// into an infinite recursion. For now we'll just use RawEquals,
			// at the expense of a less helpful failure message. This seems
			// to be a bug in upstream ctydebug.
			if !want.RawEquals(got) {
				t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	})
}
