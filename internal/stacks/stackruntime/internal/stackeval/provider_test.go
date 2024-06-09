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
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/stackconfigtypes"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestProviderCheckInstances(t *testing.T) {
	getProvider := func(ctx context.Context, t *testing.T, main *Main) *Provider {
		t.Helper()
		mainStack := main.MainStack(ctx)
		provider := mainStack.Provider(ctx, stackaddrs.ProviderConfig{
			Provider: addrs.MustParseProviderSourceString("terraform.io/builtin/foo"),
			Name:     "bar",
		})
		if provider == nil {
			t.Fatal("provider.foo.bar does not exist, but it should exist")
		}
		return provider
	}

	subtestInPromisingTask(t, "single instance", func(ctx context.Context, t *testing.T) {
		cfg := testStackConfig(t, "provider", "single_instance")
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
		})

		provider := getProvider(ctx, t, main)
		forEachVal, diags := provider.CheckForEachValue(ctx, InspectPhase)
		assertNoDiags(t, diags)
		if forEachVal != cty.NilVal {
			t.Fatalf("unexpected for_each value\ngot:  %#v\nwant: cty.NilVal", forEachVal)
		}

		insts, unknown, diags := provider.CheckInstances(ctx, InspectPhase)
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
		cfg := testStackConfig(t, "provider", "for_each")

		subtestInPromisingTask(t, "no instances", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"provider_instances": cty.MapValEmpty(cty.EmptyObject),
				},
			})

			provider := getProvider(ctx, t, main)
			forEachVal, diags := provider.CheckForEachValue(ctx, InspectPhase)
			assertNoDiags(t, diags)
			if got, want := forEachVal, cty.MapValEmpty(cty.EmptyObject); !want.RawEquals(got) {
				t.Fatalf("unexpected for_each value\ngot:  %#v\nwant: %#v", got, want)
			}
			insts, unknown, diags := provider.CheckInstances(ctx, InspectPhase)
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
				"a": cty.StringVal("in a"),
				"b": cty.StringVal("in b"),
			})
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"provider_instances": wantForEachVal,
				},
			})

			provider := getProvider(ctx, t, main)
			gotForEachVal, diags := provider.CheckForEachValue(ctx, InspectPhase)
			assertNoDiags(t, diags)
			if !wantForEachVal.RawEquals(gotForEachVal) {
				t.Fatalf("unexpected for_each value\ngot:  %#v\nwant: %#v", gotForEachVal, wantForEachVal)
			}
			insts, unknown, diags := provider.CheckInstances(ctx, InspectPhase)
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
					EachKey:   cty.StringVal("a"),
					EachValue: cty.StringVal("in a"),
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
					EachKey:   cty.StringVal("b"),
					EachValue: cty.StringVal("in b"),
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
					"provider_instances": cty.NullVal(cty.Map(cty.EmptyObject)),
				},
			})

			provider := getProvider(ctx, t, main)
			gotVal, diags := provider.CheckForEachValue(ctx, InspectPhase)
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
					"provider_instances": cty.StringVal("nope"),
				},
			})

			provider := getProvider(ctx, t, main)
			gotVal, diags := provider.CheckForEachValue(ctx, InspectPhase)
			assertMatchingDiag(t, diags, func(diag tfdiags.Diagnostic) bool {
				return (diag.Severity() == tfdiags.Error &&
					diag.Description().Detail == "The for_each expression must produce either a map of any type or a set of strings. The keys of the map or the set elements will serve as unique identifiers for multiple instances of this provider.")
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
			gotInsts, unknown, diags := provider.CheckInstances(ctx, InspectPhase)
			assertFalse(t, unknown)
			assertMatchingDiag(t, diags, func(diag tfdiags.Diagnostic) bool {
				return (diag.Severity() == tfdiags.Error &&
					diag.Description().Detail == "The for_each expression must produce either a map of any type or a set of strings. The keys of the map or the set elements will serve as unique identifiers for multiple instances of this provider.")
			})
			if gotInsts != nil {
				t.Errorf("wrong instances; want nil\n%#v", gotInsts)
			}
		})
		subtestInPromisingTask(t, "unknown", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"provider_instances": cty.UnknownVal(cty.Map(cty.EmptyObject)),
				},
			})

			// For now it's invalid to use an unknown value in for_each.
			// Later we're expecting to make this succeed but announce that
			// planning everything beneath this provider must be deferred to a
			// future plan after everything else has been applied first.
			provider := getProvider(ctx, t, main)
			gotVal, diags := provider.CheckForEachValue(ctx, InspectPhase)
			assertNoDiags(t, diags)
			wantVal := cty.UnknownVal(cty.Map(cty.EmptyObject))
			if !wantVal.RawEquals(gotVal) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", gotVal, wantVal)
			}

			insts, unknown, diags := provider.CheckInstances(ctx, InspectPhase)
			assertNoDiags(t, diags)
			assertTrue(t, unknown)
			if got, want := len(insts), 0; got != want {
				t.Fatalf("wrong number of instances %d; want %d\n%#v", got, want, insts)
			}
		})
	})

}

func TestProviderExprReferenceValue(t *testing.T) {
	providerTypeAddr := addrs.MustParseProviderSourceString("terraform.io/builtin/foo")
	providerRefType := providerInstanceRefType(providerTypeAddr)
	getProvider := func(ctx context.Context, t *testing.T, main *Main) *Provider {
		t.Helper()
		mainStack := main.MainStack(ctx)
		provider := mainStack.Provider(ctx, stackaddrs.ProviderConfig{
			Provider: providerTypeAddr,
			Name:     "bar",
		})
		if provider == nil {
			t.Fatal("provider.foo.bar does not exist, but it should exist")
		}
		return provider
	}
	getRefFromVal := func(t *testing.T, v cty.Value) stackaddrs.AbsProviderConfigInstance {
		t.Helper()
		if !stackconfigtypes.IsProviderConfigType(v.Type()) {
			t.Fatalf("result is not of a provider configuration reference type\ngot type:  %#v", v.Type())
		}
		return stackconfigtypes.ProviderInstanceForValue(v)
	}

	subtestInPromisingTask(t, "single instance", func(ctx context.Context, t *testing.T) {
		cfg := testStackConfig(t, "provider", "single_instance")
		main := testEvaluator(t, testEvaluatorOpts{
			Config:          cfg,
			TestOnlyGlobals: map[string]cty.Value{},
		})

		provider := getProvider(ctx, t, main)
		got := getRefFromVal(t, provider.ExprReferenceValue(ctx, InspectPhase))
		want := stackaddrs.AbsProviderConfigInstance{
			Stack: stackaddrs.RootStackInstance,
			Item: stackaddrs.ProviderConfigInstance{
				ProviderConfig: stackaddrs.ProviderConfig{
					Provider: providerTypeAddr,
					Name:     "bar",
				},
				Key: addrs.NoKey,
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong result\n%s", diff)
		}
	})
	t.Run("for_each", func(t *testing.T) {
		cfg := testStackConfig(t, "provider", "for_each")

		subtestInPromisingTask(t, "no instances", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"provider_instances": cty.MapValEmpty(cty.EmptyObject),
				},
			})

			provider := getProvider(ctx, t, main)
			got := provider.ExprReferenceValue(ctx, InspectPhase)
			want := cty.MapValEmpty(providerRefType)
			if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
				t.Fatalf("wrong result\n%s", diff)
			}
		})
		subtestInPromisingTask(t, "two instances", func(ctx context.Context, t *testing.T) {
			forEachVal := cty.MapVal(map[string]cty.Value{
				"a": cty.StringVal("in a"),
				"b": cty.StringVal("in b"),
			})
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"provider_instances": forEachVal,
				},
			})

			provider := getProvider(ctx, t, main)
			gotVal := provider.ExprReferenceValue(ctx, InspectPhase)
			if !gotVal.Type().IsMapType() {
				t.Fatalf("wrong result type\ngot type:  %#v\nwant: map of provider references", gotVal.Type())
			}
			if gotVal.IsNull() || !gotVal.IsKnown() {
				t.Fatalf("wrong result\ngot:  %#v\nwant: a known, non-null map of provider references", gotVal)
			}
			gotValMap := gotVal.AsValueMap()
			if got, want := len(gotValMap), 2; got != want {
				t.Errorf("wrong number of instances %d; want %d\n", got, want)
			}
			if gotVal := gotValMap["a"]; gotVal != cty.NilVal {
				got := getRefFromVal(t, gotVal)
				want := stackaddrs.AbsProviderConfigInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ProviderConfigInstance{
						ProviderConfig: stackaddrs.ProviderConfig{
							Provider: providerTypeAddr,
							Name:     "bar",
						},
						Key: addrs.StringKey("a"),
					},
				}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Fatalf("wrong result for instance 'a'\n%s", diff)
				}
			} else {
				t.Errorf("no element for instance 'a'")
			}
			if gotVal := gotValMap["b"]; gotVal != cty.NilVal {
				got := getRefFromVal(t, gotVal)
				want := stackaddrs.AbsProviderConfigInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ProviderConfigInstance{
						ProviderConfig: stackaddrs.ProviderConfig{
							Provider: providerTypeAddr,
							Name:     "bar",
						},
						Key: addrs.StringKey("b"),
					},
				}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Fatalf("wrong result for instance 'b'\n%s", diff)
				}
			} else {
				t.Errorf("no element for instance 'b'")
			}
		})
		subtestInPromisingTask(t, "null", func(ctx context.Context, t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
				TestOnlyGlobals: map[string]cty.Value{
					"provider_instances": cty.NullVal(cty.Map(cty.EmptyObject)),
				},
			})

			provider := getProvider(ctx, t, main)
			got := provider.ExprReferenceValue(ctx, InspectPhase)
			// When the for_each expression is invalid, the result value
			// is unknown so we can use it as a placeholder for partial
			// downstream checking.
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
					"provider_instances": cty.StringVal("nope"),
				},
			})

			provider := getProvider(ctx, t, main)
			got := provider.ExprReferenceValue(ctx, InspectPhase)
			// When the for_each expression is invalid, the result value
			// is unknown so we can use it as a placeholder for partial
			// downstream checking.
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
					"provider_instances": cty.UnknownVal(cty.Map(cty.EmptyObject)),
				},
			})

			provider := getProvider(ctx, t, main)
			got := provider.ExprReferenceValue(ctx, InspectPhase)
			// When the for_each expression is unknown, the result value
			// is unknown too so we can use it as a placeholder for partial
			// downstream checking.
			want := cty.UnknownVal(cty.Map(providerRefType))
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
