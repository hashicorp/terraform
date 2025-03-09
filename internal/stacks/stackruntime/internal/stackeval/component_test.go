// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestComponentInstances is a test of the [Component.CheckInstances] function.
//
// In particular, note that it's _not_ a test of the [ComponentInstance] type
// as a whole, although [Component.CheckInstances] does return a collection of
// those so there is _some_ coverage of that in here.
func TestComponentCheckInstances(t *testing.T) {
	getComponent := func(ctx context.Context, main *Main) *Component {
		mainStack := main.MainStack(ctx)
		component := mainStack.Component(ctx, stackaddrs.Component{Name: "foo"})
		if component == nil {
			t.Fatal("component.foo does not exist, but it should exist")
		}
		return component
	}

	subtestInPromisingTask(t, "single instance", func(ctx context.Context, t *testing.T) {
		cfg := testStackConfig(t, "component", "single_instance")
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			TestOnlyGlobals: map[string]cty.Value{
				"component_inputs": cty.EmptyObjectVal,
			},
		})

		component := getComponent(ctx, main)
		forEachVal, diags := component.CheckForEachValue(ctx, InspectPhase)
		assertNoDiags(t, diags)
		if forEachVal != cty.NilVal {
			t.Fatalf("unexpected for_each value\ngot:  %#v\nwant: cty.NilVal", forEachVal)
		}

		insts, unknown, diags := component.CheckInstances(ctx, InspectPhase)
		assertNoDiags(t, diags)
		assertFalse(t, unknown)
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
		cfg := testStackConfig(t, "component", "for_each")

		subtestInPromisingTask(t, "no instances", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"component_instances": cty.MapValEmpty(cty.EmptyObject),
				},
			})

			component := getComponent(ctx, main)
			forEachVal, diags := component.CheckForEachValue(ctx, InspectPhase)
			assertNoDiags(t, diags)
			if got, want := forEachVal, cty.MapValEmpty(cty.EmptyObject); !want.RawEquals(got) {
				t.Fatalf("unexpected for_each value\ngot:  %#v\nwant: %#v", got, want)
			}
			insts, unknown, diags := component.CheckInstances(ctx, InspectPhase)
			assertNoDiags(t, diags)
			assertFalse(t, unknown)
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
					"component_instances": wantForEachVal,
				},
			})

			component := getComponent(ctx, main)
			gotForEachVal, diags := component.CheckForEachValue(ctx, InspectPhase)
			assertNoDiags(t, diags)
			if !wantForEachVal.RawEquals(gotForEachVal) {
				t.Fatalf("unexpected for_each value\ngot:  %#v\nwant: %#v", gotForEachVal, wantForEachVal)
			}
			insts, unknown, diags := component.CheckInstances(ctx, InspectPhase)
			assertNoDiags(t, diags)
			assertFalse(t, unknown)
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
					"component_instances": cty.NullVal(cty.Map(cty.EmptyObject)),
				},
			})

			component := getComponent(ctx, main)
			gotVal, diags := component.CheckForEachValue(ctx, InspectPhase)
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
					"component_instances": cty.StringVal("nope"),
				},
			})

			component := getComponent(ctx, main)
			gotVal, diags := component.CheckForEachValue(ctx, InspectPhase)
			assertMatchingDiag(t, diags, func(diag tfdiags.Diagnostic) bool {
				return (diag.Severity() == tfdiags.Error &&
					diag.Description().Detail == "The for_each expression must produce either a map of any type or a set of strings. The keys of the map or the set elements will serve as unique identifiers for multiple instances of this component.")
			})
			wantVal := cty.DynamicVal // placeholder for invalid result
			if !wantVal.RawEquals(gotVal) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", gotVal, wantVal)
			}

			// When the for_each expression is invalid, CheckInstances should
			// return nil and diagnostics.
			gotInsts, unknown, diags := component.CheckInstances(ctx, InspectPhase)
			assertFalse(t, unknown)
			if gotInsts != nil {
				t.Fatalf("unexpected instances\ngot:  %#v\nwant: nil", gotInsts)
			}

			assertMatchingDiag(t, diags, func(diag tfdiags.Diagnostic) bool {
				return (diag.Severity() == tfdiags.Error &&
					diag.Description().Detail == "The for_each expression must produce either a map of any type or a set of strings. The keys of the map or the set elements will serve as unique identifiers for multiple instances of this component.")
			})
		})
		subtestInPromisingTask(t, "unknown", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"component_instances": cty.UnknownVal(cty.Map(cty.EmptyObject)),
				},
			})

			component := getComponent(ctx, main)
			gotVal, diags := component.CheckForEachValue(ctx, InspectPhase)
			assertNoDiags(t, diags)

			wantVal := cty.UnknownVal(cty.Map(cty.EmptyObject))
			if !wantVal.RawEquals(gotVal) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", gotVal, wantVal)
			}

			// When the for_each expression is unknown, CheckInstances should
			// return a single instance with dynamic values in the repetition data.
			gotInsts, unknown, diags := component.CheckInstances(ctx, InspectPhase)
			assertNoDiags(t, diags)
			assertTrue(t, unknown)
			if got, want := len(gotInsts), 0; got != want {
				t.Fatalf("wrong number of instances %d; want %d\n%#v", got, want, gotInsts)
			}
		})
	})

}

func TestComponentResultValue(t *testing.T) {
	getComponent := func(ctx context.Context, t *testing.T, main *Main) *Component {
		mainStack := main.MainStack(ctx)
		component := mainStack.Component(ctx, stackaddrs.Component{Name: "foo"})
		if component == nil {
			t.Fatal("component.foo does not exist, but it should exist")
		}
		return component
	}

	subtestInPromisingTask(t, "single instance", func(ctx context.Context, t *testing.T) {
		cfg := testStackConfig(t, "component", "single_instance")
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			TestOnlyGlobals: map[string]cty.Value{
				"child_stack_inputs": cty.ObjectVal(map[string]cty.Value{
					"test": cty.StringVal("hello"),
				}),
			},
		})

		component := getComponent(ctx, t, main)
		got := component.ResultValue(ctx, InspectPhase)
		want := cty.ObjectVal(map[string]cty.Value{
			// FIXME: This currently returns an unknown value because we
			// aren't tracking component output values in prior state.
			// Once we fix that, we should see an output value called "test"
			// here.
			"test": cty.DynamicVal,
		})
		if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
			t.Fatalf("wrong result\n%s", diff)
		}
	})
	t.Run("for_each", func(t *testing.T) {
		cfg := testStackConfig(t, "component", "for_each")

		subtestInPromisingTask(t, "no instances", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"component_instances": cty.MapValEmpty(cty.EmptyObject),
				},
			})

			component := getComponent(ctx, t, main)
			got := component.ResultValue(ctx, InspectPhase)
			want := cty.EmptyObjectVal
			if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
				t.Fatalf("wrong result\n%s", diff)
			}
		})
		subtestInPromisingTask(t, "two instances", func(ctx context.Context, t *testing.T) {
			forEachVal := cty.MapVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"test": cty.StringVal("in a"),
				}),
				"b": cty.ObjectVal(map[string]cty.Value{
					"test": cty.StringVal("in b"),
				}),
			})
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"component_instances": forEachVal,
				},
			})

			component := getComponent(ctx, t, main)
			got := component.ResultValue(ctx, InspectPhase)
			want := cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					// FIXME: This currently returns an unknown value because we
					// aren't tracking component output values in prior state.
					// Once we fix that, we should see an output value called "test"
					// here.
					"test": cty.DynamicVal,
				}),
				"b": cty.ObjectVal(map[string]cty.Value{
					// FIXME: This currently returns an unknown value because we
					// aren't tracking component output values in prior state.
					// Once we fix that, we should see an output value called "test"
					// here.
					"test": cty.DynamicVal,
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
					"component_instances": cty.NullVal(cty.Map(cty.EmptyObject)),
				},
			})

			component := getComponent(ctx, t, main)
			got := component.ResultValue(ctx, InspectPhase)
			// When the for_each expression is null, the result value should
			// be a cty.NilVal.
			want := cty.NilVal
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
					"component_instances": cty.StringVal("nope"),
				},
			})

			component := getComponent(ctx, t, main)
			got := component.ResultValue(ctx, InspectPhase)
			// When the for_each expression is null, the result value should
			// be a cty.NilVal.
			want := cty.NilVal
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
					"component_instances": cty.UnknownVal(cty.Map(cty.EmptyObject)),
				},
			})

			component := getComponent(ctx, t, main)
			got := component.ResultValue(ctx, InspectPhase)
			// When the for_each expression is unknown, the result value
			// is a dynamic instance.
			want := cty.DynamicVal

			if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
				t.Fatalf("wrong result\n%s", diff)
			}
		})
	})
}
