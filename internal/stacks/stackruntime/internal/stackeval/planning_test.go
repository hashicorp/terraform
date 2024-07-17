// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	providerTesting "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/stacks/stackstate/statekeys"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestPlanning_DestroyMode(t *testing.T) {
	// This integration test aims to verify the overall problem of planning in
	// destroy mode, which has some special exceptions to deal with the fact
	// that downstream components need to plan against the _current_ outputs of
	// other component instances they depend on, rather than the _planned_
	// outputs which would necessarily be missing in a full-destroy situation.
	//
	// This behavior differs from other planning modes because when applying
	// destroys we work in reverse dependency order, destroying the dependent
	// before we destroy the dependency, and therefore the downstream destroy
	// action happens _before_ the upstream has been destroyed, when its prior
	// state outputs are still available.)

	cfg := testStackConfig(t, "planning", "plan_destroy")
	aComponentInstAddr := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{
				Name: "a",
			},
		},
	}
	aResourceInstAddr := stackaddrs.AbsResourceInstance{
		Component: aComponentInstAddr,
		Item: addrs.AbsResourceInstance{
			Module: addrs.RootModuleInstance,
			Resource: addrs.ResourceInstance{
				Resource: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test",
					Name: "foo",
				},
			},
		},
	}
	bComponentInstAddr := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{
				Name: "b",
			},
		},
	}
	bResourceInstAddr := stackaddrs.AbsResourceInstance{
		Component: bComponentInstAddr,
		Item: addrs.AbsResourceInstance{
			Module: addrs.RootModuleInstance,
			Resource: addrs.ResourceInstance{
				Resource: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test",
					Name: "foo",
				},
			},
		},
	}
	providerAddr := addrs.NewBuiltInProvider("test")
	providerInstAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: providerAddr,
	}
	priorState := testPriorState(t, map[string]protoreflect.ProtoMessage{
		statekeys.String(statekeys.ComponentInstance{
			ComponentInstanceAddr: aComponentInstAddr,
		}): &tfstackdata1.StateComponentInstanceV1{
			// Intentionally unpopulated because this operation doesn't
			// actually depend on anything other than knowing that the
			// component instance used to exist.
		},

		statekeys.String(statekeys.ComponentInstance{
			ComponentInstanceAddr: bComponentInstAddr,
		}): &tfstackdata1.StateComponentInstanceV1{
			// Intentionally unpopulated because this operation doesn't
			// actually depend on anything other than knowing that the
			// component instance used to exist.
		},

		statekeys.String(statekeys.ResourceInstanceObject{
			ResourceInstance: aResourceInstAddr,
		}): &tfstackdata1.StateResourceInstanceObjectV1{
			Status:             tfstackdata1.StateResourceInstanceObjectV1_READY,
			ProviderConfigAddr: providerInstAddr.String(),
			ValueJson: []byte(`
				{
					"for_module": "a",
					"arg": null,
					"result": "result for \"a\" from prior state"
				}
			`),
		},

		statekeys.String(statekeys.ResourceInstanceObject{
			ResourceInstance: bResourceInstAddr,
		}): &tfstackdata1.StateResourceInstanceObjectV1{
			Status:             tfstackdata1.StateResourceInstanceObjectV1_READY,
			ProviderConfigAddr: providerInstAddr.String(),
			ValueJson: []byte(`
				{
					"for_module": "b",
					"arg": "result for \"a\" from prior state",
					"result": "result for \"b\" from prior state"
				}
			`),
		},
	})

	resourceTypeSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"for_module": {
				Type:     cty.String,
				Required: true,
			},
			"arg": {
				Type:     cty.String,
				Optional: true,
			},
			"result": {
				Type:     cty.String,
				Computed: true,
			},
		},
	}
	main := NewForPlanning(cfg, priorState, PlanOpts{
		PlanningMode:  plans.DestroyMode,
		PlanTimestamp: time.Now().UTC(),
		ProviderFactories: ProviderFactories{
			addrs.NewBuiltInProvider("test"): func() (providers.Interface, error) {
				return &providerTesting.MockProvider{
					GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
						Provider: providers.Schema{
							Block: &configschema.Block{},
						},
						ResourceTypes: map[string]providers.Schema{
							"test": {
								Block: resourceTypeSchema,
							},
						},
						ServerCapabilities: providers.ServerCapabilities{
							PlanDestroy: true,
						},
					},
					ConfigureProviderFn: func(cpr providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
						t.Logf("configuring the provider: %#v", cpr.Config)
						return providers.ConfigureProviderResponse{}
					},
					ReadResourceFn: func(rrr providers.ReadResourceRequest) providers.ReadResourceResponse {
						forModule := rrr.PriorState.GetAttr("for_module").AsString()
						t.Logf("refreshing for_module = %q", forModule)
						arg := rrr.PriorState.GetAttr("arg")
						var result string
						if !arg.IsNull() {
							argStr := arg.AsString()
							result = fmt.Sprintf("result for %q refreshed with %q", forModule, argStr)
						} else {
							result = fmt.Sprintf("result for %q refreshed without arg", forModule)
						}

						return providers.ReadResourceResponse{
							NewState: cty.ObjectVal(map[string]cty.Value{
								"for_module": cty.StringVal(forModule),
								"arg":        arg,
								"result":     cty.StringVal(result),
							}),
						}
					},
					PlanResourceChangeFn: func(prcr providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
						if prcr.ProposedNewState.IsNull() {
							// We're destroying then, which is what we expect.
							forModule := prcr.PriorState.GetAttr("for_module").AsString()
							t.Logf("planning destroy for_module = %q", forModule)
							return providers.PlanResourceChangeResponse{
								PlannedState: prcr.ProposedNewState,
							}
						}

						// Although we're planning for destroy, as an
						// implementation detail the modules runtime uses a
						// normal planning pass to get the prior state updated
						// before doing the real destroy plan, and then
						// discards the result.

						forModule := prcr.ProposedNewState.GetAttr("for_module").AsString()
						t.Logf("planning non-destroy for_module = %q (should be ignored by the modules runtime)", forModule)

						arg := prcr.ProposedNewState.GetAttr("arg")
						var result string
						if !arg.IsNull() {
							argStr := arg.AsString()
							result = fmt.Sprintf("result for %q planned with %q", forModule, argStr)
						} else {
							result = fmt.Sprintf("result for %q planned without arg", forModule)
						}

						return providers.PlanResourceChangeResponse{
							PlannedState: cty.ObjectVal(map[string]cty.Value{
								"for_module": cty.StringVal(forModule),
								"arg":        arg,
								"result":     cty.StringVal(result),
							}),
						}
					},
				}, nil
			},
		},
	})

	plan, diags := testPlan(t, main)
	assertNoDiagnostics(t, diags)

	aCmpPlan := plan.Components.Get(aComponentInstAddr)
	bCmpPlan := plan.Components.Get(bComponentInstAddr)
	if aCmpPlan == nil || bCmpPlan == nil {
		t.Fatalf(
			"incomplete plan\n%s: %#v\n%s: %#v",
			aComponentInstAddr, aCmpPlan,
			bComponentInstAddr, bCmpPlan,
		)
	}

	aPlan, err := aCmpPlan.ForModulesRuntime()
	if err != nil {
		t.Fatalf("inconsistent plan for %s: %s", aComponentInstAddr, err)
	}
	bPlan, err := bCmpPlan.ForModulesRuntime()
	if err != nil {
		t.Fatalf("inconsistent plan for %s: %s", bComponentInstAddr, err)
	}

	if planSrc := aPlan.Changes.ResourceInstance(aResourceInstAddr.Item); planSrc != nil {
		rAddr := aResourceInstAddr
		plan, err := planSrc.Decode(resourceTypeSchema.ImpliedType())
		if err != nil {
			t.Fatalf("can't decode change for %s: %s", rAddr, err)
		}
		if got, want := plan.Action, plans.Delete; got != want {
			t.Errorf("wrong action for %s\ngot:  %s\nwant: %s", rAddr, got, want)
		}
		if !plan.After.IsNull() {
			t.Errorf("unexpected non-nil 'after' value for %s: %#v", rAddr, plan.After)
		}
		wantBefore := cty.ObjectVal(map[string]cty.Value{
			"arg":        cty.NullVal(cty.String),
			"for_module": cty.StringVal("a"),
			"result":     cty.StringVal(`result for "a" refreshed without arg`),
		})
		if !wantBefore.RawEquals(plan.Before) {
			t.Errorf("wrong before value for %s\ngot:  %#v\nwant: %#v", rAddr, plan.Before, wantBefore)
		}
	} else {
		t.Errorf("no plan for %s", aResourceInstAddr)
	}

	if planSrc := bPlan.Changes.ResourceInstance(bResourceInstAddr.Item); planSrc != nil {
		rAddr := bResourceInstAddr
		plan, err := planSrc.Decode(resourceTypeSchema.ImpliedType())
		if err != nil {
			t.Fatalf("can't decode change for %s: %s", rAddr, err)
		}
		if got, want := plan.Action, plans.Delete; got != want {
			t.Errorf("wrong action for %s\ngot:  %s\nwant: %s", rAddr, got, want)
		}
		if !plan.After.IsNull() {
			t.Errorf("unexpected non-nil 'after' value for %s: %#v", rAddr, plan.After)
		}
		wantBefore := cty.ObjectVal(map[string]cty.Value{
			// FIXME: The modules runtime has a long-standing historical quirk
			// that it not-so-secretly does a full normal plan before it runs
			// the destroy plan, as its way to get the prior state updated.
			//
			// Unfortunately, that means that the output values in the prior
			// state end up not reflecting the refreshed state properly,
			// and that's why the below says that "a" came from the prior state.
			// This quirk only really matters if there has been a significant
			// change in the remote system that needs to be considered for
			// destroy to be successful, which thankfully isn't true _often_
			// but does happen somtimes, and so we should find a way to fix
			// the modules runtime to produce its output values based on the
			// refreshed state instead of the prior state. Perhaps using
			// a refresh-only plan instead of a normal plan would do it?
			//
			// Once the quirk in the modules runtime is fixed, "arg" below
			// (and the copy of it embedded in "result") should become:
			//  `result for "a" refreshed without arg`
			"arg":        cty.StringVal(`result for "a" from prior state`),
			"for_module": cty.StringVal("b"),

			// If this appears as `result for "b" refreshed without arg` after
			// future maintenence, then that suggests that the special case
			// for destroy mode in ComponentInstance.ResultValue is no longer
			// working correctly. Propagating the new desired state instead
			// of the prior state will cause the "a" value to be null, and
			// therefore "arg" on this resource instance would also be null.
			"result": cty.StringVal(`result for "b" refreshed with "result for \"a\" from prior state"`),
		})
		if !wantBefore.RawEquals(plan.Before) {
			t.Errorf("wrong before value for %s\ngot:  %#v\nwant: %#v", rAddr, plan.Before, wantBefore)
		}
	} else {
		t.Errorf("no plan for %s", bResourceInstAddr)
	}
}

func TestPlanning_RequiredComponents(t *testing.T) {
	// This test acts both as some unit tests for the component requirement
	// analysis of various different object types and as an integration test
	// for the overall component dependency analysis during the plan phase,
	// ensuring that the dependency graph is reflected correctly in the
	// resulting plan.

	cfg := testStackConfig(t, "planning", "required_components")
	main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
		PlanningMode: plans.NormalMode,
		ProviderFactories: ProviderFactories{
			addrs.NewBuiltInProvider("foo"): func() (providers.Interface, error) {
				return &providerTesting.MockProvider{
					GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
						Provider: providers.Schema{
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"in": {
										Type:     cty.Map(cty.String),
										Optional: true,
									},
								},
							},
						},
					},
					ConfigureProviderFn: func(cpr providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
						t.Logf("configuring the provider: %#v", cpr.Config)
						return providers.ConfigureProviderResponse{}
					},
				}, nil
			},
		},
		PlanTimestamp: time.Now().UTC(),
	})

	cmpA := stackaddrs.AbsComponent{
		Stack: stackaddrs.RootStackInstance,
		Item:  stackaddrs.Component{Name: "a"},
	}
	cmpB := stackaddrs.AbsComponent{
		Stack: stackaddrs.RootStackInstance,
		Item:  stackaddrs.Component{Name: "b"},
	}
	cmpC := stackaddrs.AbsComponent{
		Stack: stackaddrs.RootStackInstance,
		Item:  stackaddrs.Component{Name: "c"},
	}

	cmpOpts := collections.CmpOptions

	t.Run("integrated", func(t *testing.T) {
		// This integration tests runs a full plan of the test configuration
		// and checks that the resulting plan contains the expected component
		// dependency information, without concern for exactly how that
		// information got populated.
		//
		// The other subtests below check that the individual objects
		// participating in this plan are reporting their own component
		// dependencies correctly, and so if this integrated test fails
		// then the simultaneous failure of one of those other tests might be
		// a good clue as to what's broken.

		plan, diags := testPlan(t, main)
		assertNoDiagnostics(t, diags)

		componentPlans := plan.Components

		tests := []struct {
			component        stackaddrs.AbsComponent
			wantDependencies []stackaddrs.AbsComponent
			wantDependents   []stackaddrs.AbsComponent
		}{
			{
				component:        cmpA,
				wantDependencies: []stackaddrs.AbsComponent{},
				wantDependents: []stackaddrs.AbsComponent{
					cmpB,
					cmpC,
				},
			},
			{
				component: cmpB,
				wantDependencies: []stackaddrs.AbsComponent{
					cmpA,
				},
				wantDependents: []stackaddrs.AbsComponent{
					cmpC,
				},
			},
			{
				component: cmpC,
				wantDependencies: []stackaddrs.AbsComponent{
					cmpA,
					cmpB,
				},
				wantDependents: []stackaddrs.AbsComponent{},
			},
		}
		for _, test := range tests {
			t.Run(test.component.String(), func(t *testing.T) {
				instAddr := stackaddrs.AbsComponentInstance{
					Stack: test.component.Stack,
					Item: stackaddrs.ComponentInstance{
						Component: test.component.Item,
					},
				}
				cp := componentPlans.Get(instAddr)
				{
					got := cp.Dependencies
					want := collections.NewSet[stackaddrs.AbsComponent]()
					want.Add(test.wantDependencies...)
					if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
						t.Errorf("wrong dependencies\n%s", diff)
					}
				}
				{
					got := cp.Dependents
					want := collections.NewSet[stackaddrs.AbsComponent]()
					want.Add(test.wantDependents...)
					if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
						t.Errorf("wrong dependents\n%s", diff)
					}
				}
			})
		}
	})

	t.Run("component dependents", func(t *testing.T) {
		ctx := context.Background()
		promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
			tests := []struct {
				componentAddr    stackaddrs.AbsComponent
				wantDependencies []stackaddrs.AbsComponent
			}{
				{
					cmpA,
					[]stackaddrs.AbsComponent{},
				},
				{
					cmpB,
					[]stackaddrs.AbsComponent{
						cmpA,
					},
				},
				{
					cmpC,
					[]stackaddrs.AbsComponent{
						cmpA,
						cmpB,
					},
				},
			}

			for _, test := range tests {
				t.Run(test.componentAddr.String(), func(t *testing.T) {
					stack := main.Stack(ctx, test.componentAddr.Stack, PlanPhase)
					if stack == nil {
						t.Fatalf("no declaration for %s", test.componentAddr.Stack)
					}
					component := stack.Component(ctx, test.componentAddr.Item)
					if component == nil {
						t.Fatalf("no declaration for %s", test.componentAddr)
					}

					got := component.RequiredComponents(ctx)
					want := collections.NewSet[stackaddrs.AbsComponent]()
					want.Add(test.wantDependencies...)

					if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
						t.Errorf("wrong result\n%s", diff)
					}
				})
			}

			return struct{}{}, nil
		})
	})

	subtestInPromisingTask(t, "input variable dependents", func(ctx context.Context, t *testing.T) {
		stack := main.Stack(ctx, stackaddrs.RootStackInstance.Child("child", addrs.NoKey), PlanPhase)
		if stack == nil {
			t.Fatalf("embedded stack isn't declared")
		}
		ivs := stack.InputVariables(ctx)
		iv := ivs[stackaddrs.InputVariable{Name: "in"}]
		if iv == nil {
			t.Fatalf("input variable isn't declared")
		}

		got := iv.RequiredComponents(ctx)
		want := collections.NewSet[stackaddrs.AbsComponent]()
		want.Add(cmpB)

		if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})

	subtestInPromisingTask(t, "output value dependents", func(ctx context.Context, t *testing.T) {
		stack := main.MainStack(ctx)
		ovs := stack.OutputValues(ctx)
		ov := ovs[stackaddrs.OutputValue{Name: "out"}]
		if ov == nil {
			t.Fatalf("output value isn't declared")
		}

		got := ov.RequiredComponents(ctx)
		want := collections.NewSet[stackaddrs.AbsComponent]()
		want.Add(cmpA)

		if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})

	subtestInPromisingTask(t, "embedded stack dependents", func(ctx context.Context, t *testing.T) {
		stack := main.MainStack(ctx)
		sc := stack.EmbeddedStackCall(ctx, stackaddrs.StackCall{Name: "child"})
		if sc == nil {
			t.Fatalf("embedded stack call isn't declared")
		}

		got := sc.RequiredComponents(ctx)
		want := collections.NewSet[stackaddrs.AbsComponent]()
		want.Add(cmpB)

		if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})

	subtestInPromisingTask(t, "provider config dependents", func(ctx context.Context, t *testing.T) {
		stack := main.MainStack(ctx)
		pc := stack.Provider(ctx, stackaddrs.ProviderConfig{
			Provider: addrs.NewBuiltInProvider("foo"),
			Name:     "bar",
		})
		if pc == nil {
			t.Fatalf("provider configuration isn't declared")
		}

		got := pc.RequiredComponents(ctx)
		want := collections.NewSet[stackaddrs.AbsComponent]()
		want.Add(cmpA)
		want.Add(cmpB)

		if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
}

func TestPlanning_DeferredChangesPropagation(t *testing.T) {
	// This test arranges for one component's plan to signal deferred changes,
	// and checks that a downstream component's plan also has everything
	// deferred even though it could potentially have been plannable in
	// isolation, since we need to respect the dependency ordering between
	// components.

	cfg := testStackConfig(t, "planning", "deferred_changes_propagation")
	main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
		PlanningMode:  plans.NormalMode,
		PlanTimestamp: time.Now().UTC(),
		InputVariableValues: map[stackaddrs.InputVariable]ExternalInputValue{
			// This causes the first component to have a module whose
			// instance count isn't known yet.
			{Name: "first_count"}: {
				Value: cty.UnknownVal(cty.Number),
			},
		},
		ProviderFactories: ProviderFactories{
			addrs.NewBuiltInProvider("test"): func() (providers.Interface, error) {
				return &providerTesting.MockProvider{
					GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
						Provider: providers.Schema{
							Block: &configschema.Block{},
						},
						ResourceTypes: map[string]providers.Schema{
							"test": {
								Block: &configschema.Block{},
							},
						},
					},
				}, nil
			},
		},
	})

	componentFirstInstAddr := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{
				Name: "first",
			},
		},
	}
	componentSecondInstAddr := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{
				Name: "second",
			},
		},
	}

	componentPlanResourceActions := func(plan *stackplan.Component) map[string]plans.Action {
		ret := make(map[string]plans.Action)
		for _, elem := range plan.ResourceInstancePlanned.Elems {
			ret[elem.Key.String()] = elem.Value.Action
		}
		return ret
	}

	inPromisingTask(t, func(ctx context.Context, t *testing.T) {
		plan, diags := testPlan(t, main)
		assertNoErrors(t, diags)

		firstPlan := plan.Components.Get(componentFirstInstAddr)
		if firstPlan.PlanComplete {
			t.Error("first component has a complete plan; should be incomplete because it has deferred actions")
		}
		secondPlan := plan.Components.Get(componentSecondInstAddr)
		if secondPlan.PlanComplete {
			t.Error("second component has a complete plan; should be incomplete because everything in it should've been deferred")
		}

		gotFirstActions := componentPlanResourceActions(firstPlan)
		wantFirstActions := map[string]plans.Action{
			// Only test.a is planned, because test.b has unknown count
			// and must therefore be deferred.
			"test.a": plans.Create,
		}
		gotSecondActions := componentPlanResourceActions(secondPlan)
		wantSecondActions := map[string]plans.Action{
			// Nothing at all expected for the second, because all of its
			// planned actions should've been deferred to respect the
			// dependency on the first component.
		}

		if diff := cmp.Diff(wantFirstActions, gotFirstActions); diff != "" {
			t.Errorf("wrong actions for first component\n%s", diff)
		}
		if diff := cmp.Diff(wantSecondActions, gotSecondActions); diff != "" {
			t.Errorf("wrong actions for second component\n%s", diff)
		}
	})
}

func TestPlanning_RemoveDataResource(t *testing.T) {
	// This test is here because there was a historical bug where we'd generate
	// an invalid plan (unparsable) whenever the plan included deletion of
	// a previously-declared data resource, where the provider configuration
	// address would not be populated correctly.
	//
	// Therefore this test is narrowly focused on that specific situation.
	// Anything else it's exercising as a side-effect is not crucial for
	// this test in particular, although of course unrelated regressions might
	// still be important in some other way beyond this test's scope.

	providerFactories := map[addrs.Provider]providers.Factory{
		addrs.NewBuiltInProvider("test"): func() (providers.Interface, error) {
			return &providerTesting.MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					DataSources: map[string]providers.Schema{
						"test": {
							Block: &configschema.Block{},
						},
					},
				},
				ReadDataSourceFn: func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
					return providers.ReadDataSourceResponse{
						State: cty.EmptyObjectVal,
					}
				},
			}, nil
		},
	}
	objAddr := stackaddrs.AbsResourceInstanceObject{
		Component: stackaddrs.AbsComponentInstance{
			Stack: stackaddrs.RootStackInstance,
			Item: stackaddrs.ComponentInstance{
				Component: stackaddrs.Component{Name: "main"},
			},
		},
		Item: addrs.AbsResourceInstanceObject{
			ResourceInstance: addrs.AbsResourceInstance{
				Module: addrs.RootModuleInstance,
				Resource: addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: addrs.DataResourceMode,
						Type: "test",
						Name: "test",
					},
				},
			},
			DeposedKey: addrs.NotDeposed,
		},
	}

	var state *stackstate.State

	// Round 1: data.test.test is present inside component.main
	{
		ctx := context.Background()
		cfg := testStackConfig(t, "planning", "remove_data_resource/step1")

		// Plan
		rawPlan, err := promising.MainTask(ctx, func(ctx context.Context) ([]*anypb.Any, error) {
			main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
				PlanningMode:      plans.NormalMode,
				ProviderFactories: providerFactories,
				PlanTimestamp:     time.Now().UTC(),
			})
			outp, outpTest := testPlanOutput(t)
			main.PlanAll(ctx, outp)
			rawPlan := outpTest.RawChanges(t)
			_, diags := outpTest.Close(t)
			assertNoDiagnostics(t, diags)
			return rawPlan, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		plan, err := stackplan.LoadFromProto(rawPlan)
		if err != nil {
			t.Fatal(err)
		}

		// Apply
		newState, err := promising.MainTask(ctx, func(ctx context.Context) (*stackstate.State, error) {
			outp, outpTest := testApplyOutput(t, nil)
			_, err := ApplyPlan(ctx, cfg, plan, ApplyOpts{
				ProviderFactories: providerFactories,
			}, outp)
			if err != nil {
				t.Fatal(err)
			}
			state, diags := outpTest.Close(t)
			assertNoDiagnostics(t, diags)

			// This test is only valid if the data resource instance is actually
			// tracked in the state.
			obj := state.ResourceInstanceObjectSrc(objAddr)
			if obj == nil {
				t.Fatalf("data.test.test is not in the final state for round 1")
			}

			return state, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		// We'll use the new state as the input for the next round.
		state = newState
	}

	// Round 2: data.test.test has its remnant left in the prior state, but
	// it's no longer present in the configuration.
	{
		ctx := context.Background()
		cfg := testStackConfig(t, "planning", "remove_data_resource/step2")

		// Plan
		type Plans struct {
			Nice *stackplan.Plan
			Raw  []*anypb.Any
		}
		plan, err := promising.MainTask(ctx, func(ctx context.Context) (*stackplan.Plan, error) {
			main := NewForPlanning(cfg, state, PlanOpts{
				PlanningMode:      plans.NormalMode,
				ProviderFactories: providerFactories,
				PlanTimestamp:     time.Now().UTC(),
			})
			outp, outpTest := testPlanOutput(t)
			main.PlanAll(ctx, outp)
			// The original bug would occur at this point, because
			// outpTest.Close attempts to parse the raw plan, which fails if
			// any part of that structure is not syntactically valid.
			plan, diags := outpTest.Close(t)
			assertNoDiagnostics(t, diags)
			return plan, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		// We'll check whether the data resource even appears in the plan,
		// because if not then this test is no longer testing what it thinks
		// it's testing and should probably be revised.
		//
		// (That doesn't necessarily mean that any new behavior is wrong: if
		// plan at all anymore then we can update this test to agree with that.)
		//
		// Specifically we expect to have a prior state and a provider config
		// address for this data resource, but no planned action because
		// dropping a data resource from the state is not an "action" in the
		// usual sense (it doesn't cause any calls to the provider).
		mainPlan := plan.Components.Get(stackaddrs.AbsComponentInstance{
			Stack: stackaddrs.RootStackInstance,
			Item: stackaddrs.ComponentInstance{
				Component: stackaddrs.Component{Name: "main"},
			},
		})
		if mainPlan == nil {
			t.Fatalf("main component not appear in the plan at all")
		}
		riAddr := objAddr.Item
		_, ok := mainPlan.ResourceInstancePriorState.GetOk(riAddr)
		if !ok {
			t.Fatalf("data resource instance does not appear in the prior state at all")
		}
		providerConfig, ok := mainPlan.ResourceInstanceProviderConfig.GetOk(riAddr)
		if !ok {
			t.Fatalf("data resource instance does not have a provider config in the plan")
		}
		if got, want := providerConfig.Provider, addrs.NewBuiltInProvider("test"); got != want {
			t.Errorf("wrong provider configuration address\ngot:  %s\nwant: %s", got, want)
		}

		// For good measure we'll also apply this new plan, to make sure that
		// we're left with no remnant of the data resource in the updated state.
		newState, err := promising.MainTask(ctx, func(ctx context.Context) (*stackstate.State, error) {
			outp, outpTest := testApplyOutput(t, nil)
			_, err := ApplyPlan(ctx, cfg, plan, ApplyOpts{
				ProviderFactories: providerFactories,
			}, outp)
			if err != nil {
				t.Fatal(err)
			}
			state, diags := outpTest.Close(t)
			assertNoDiagnostics(t, diags)

			return state, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		state = newState
	}

	// Our final state should not include the data resource at all.
	objState := state.ResourceInstanceObjectSrc(objAddr)
	if objState != nil {
		t.Errorf("%s is still in the state after it should've been dropped", objAddr)
	}
}

func TestPlanning_PathValues(t *testing.T) {
	cfg := testStackConfig(t, "planning", "path_values")
	main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
		PlanningMode:  plans.NormalMode,
		PlanTimestamp: time.Now().UTC(),
	})

	inPromisingTask(t, func(ctx context.Context, t *testing.T) {
		plan, diags := testPlan(t, main)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %s", diags)
		}

		component, ok := plan.Components.GetOk(stackaddrs.AbsComponentInstance{
			Stack: stackaddrs.RootStackInstance,
			Item: stackaddrs.ComponentInstance{
				Component: stackaddrs.Component{
					Name: "path_values",
				},
				Key: addrs.NoKey,
			},
		})
		if !ok {
			t.Fatalf("component not found in plan")
		}

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current working directory: %s", err)
		}

		normalizePath := func(path string) string {
			rel, err := filepath.Rel(cwd, path)
			if err != nil {
				t.Errorf("rel(%s,%s): %s", cwd, path, err)
				return path
			}
			return rel
		}

		expected := map[string]string{
			"cwd":          ".",
			"root":         "testdata/sourcebundle/planning/path_values/module",       // this is the root module of the component
			"module":       "testdata/sourcebundle/planning/path_values/module",       // this is the root module
			"child_root":   "testdata/sourcebundle/planning/path_values/module",       // should be the same for all modules
			"child_module": "testdata/sourcebundle/planning/path_values/module/child", // this is the child module
		}

		actual := map[string]string{
			"cwd":          normalizePath(component.PlannedOutputValues[addrs.OutputValue{Name: "cwd"}].AsString()),
			"root":         normalizePath(component.PlannedOutputValues[addrs.OutputValue{Name: "root"}].AsString()),
			"module":       normalizePath(component.PlannedOutputValues[addrs.OutputValue{Name: "module"}].AsString()),
			"child_root":   normalizePath(component.PlannedOutputValues[addrs.OutputValue{Name: "child_root"}].AsString()),
			"child_module": normalizePath(component.PlannedOutputValues[addrs.OutputValue{Name: "child_module"}].AsString()),
		}

		if cmp.Diff(expected, actual) != "" {
			t.Fatalf("unexpected path values\n%s", cmp.Diff(expected, actual))
		}
	})
}

func TestPlanning_NoWorkspaceNameRef(t *testing.T) {
	// This test verifies that a reference to terraform.workspace is treated
	// as invalid for modules used in a stacks context, because there's
	// no comparable single string to use in stacks context and we expect
	// modules used in stack components to vary declarations based only
	// on their input variables.
	//
	// (If something needs to vary between stack deployments then that's
	// a good candidate for an input variable on the root stack configuration,
	// set differently for each deployment, and then passed in to the
	// components that need it.)

	cfg := testStackConfig(t, "planning", "no_workspace_name_ref")
	main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
		PlanningMode: plans.NormalMode,
	})

	inPromisingTask(t, func(ctx context.Context, t *testing.T) {
		_, diags := testPlan(t, main)
		if !diags.HasErrors() {
			t.Fatal("success; want error about invalid terraform.workspace reference")
		}

		// At least one of the diagnostics must mention the terraform.workspace
		// attribute in its detail.
		seenRelevantDiag := false
		for _, diag := range diags {
			if diag.Severity() != tfdiags.Error {
				continue
			}
			if strings.Contains(diag.Description().Detail, "terraform.workspace") {
				seenRelevantDiag = true
				break
			}
		}
		if !seenRelevantDiag {
			t.Fatalf("none of the error diagnostics mentions terraform.workspace\n%s", spew.Sdump(diags.ForRPC()))
		}
	})
}

func TestPlanning_Locals(t *testing.T) {
	cfg := testStackConfig(t, "local_value", "basics")
	main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
		PlanningMode: plans.NormalMode,
	})

	inPromisingTask(t, func(ctx context.Context, t *testing.T) {
		_, diags := testPlan(t, main)
		if diags.HasErrors() {
			t.Fatalf("errors encountered\n%s", spew.Sdump(diags.ForRPC()))
		}
	})
}

func TestPlanning_LocalsDataSource(t *testing.T) {
	ctx := context.Background()
	cfg := testStackConfig(t, "local_value", "custom_provider")
	providerFactories := map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
			provider := stacks_testing_provider.NewProvider()
			return provider, nil
		},
	}
	lock := depsfile.NewLocks()
	lock.SetProvider(
		addrs.NewDefaultProvider("testing"),
		providerreqs.MustParseVersion("0.0.0"),
		providerreqs.MustParseVersionConstraints("=0.0.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)

	main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
		PlanningMode:      plans.NormalMode,
		ProviderFactories: providerFactories,
		DependencyLocks:   *lock,
		PlanTimestamp:     time.Now().UTC(),
	})

	comp2Addr := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{Name: "child2"},
		},
	}

	rawPlan, err := promising.MainTask(ctx, func(ctx context.Context) ([]*anypb.Any, error) {
		outp, outpTest := testPlanOutput(t)
		main.PlanAll(ctx, outp)
		rawPlan := outpTest.RawChanges(t)
		_, diags := outpTest.Close(t)
		assertNoDiagnostics(t, diags)
		return rawPlan, nil
	})

	if err != nil {
		t.Fatal(err)
	}

	plan, err := stackplan.LoadFromProto(rawPlan)
	if err != nil {
		t.Fatal(err)
	}

	_, err = promising.MainTask(ctx, func(ctx context.Context) (*stackstate.State, error) {
		outp, outpTest := testApplyOutput(t, nil)
		_, err := ApplyPlan(ctx, cfg, plan, ApplyOpts{
			ProviderFactories: providerFactories,
			DependencyLocks:   *lock,
		}, outp)
		if err != nil {
			t.Fatal(err)
		}
		state, diags := outpTest.Close(t)
		applies := outpTest.AppliedChanges()
		for _, apply := range applies {
			switch v := apply.(type) {
			case *stackstate.AppliedChangeComponentInstance:
				if v.ComponentAddr.Item.Name == comp2Addr.Item.Component.Name {
					stringKey := addrs.OutputValue{
						Name: "bar",
					}
					listKey := addrs.OutputValue{
						Name: "list",
					}
					mapKey := addrs.OutputValue{
						Name: "map",
					}

					stringOutput := v.OutputValues[stringKey]
					listOutput := v.OutputValues[listKey].AsValueSlice()
					mapOutput := v.OutputValues[mapKey].AsValueMap()

					expectedString := cty.StringVal("through-local-aloha-foo-foo")
					expectedList := []cty.Value{
						cty.StringVal("through-local-aloha-foo"),
						cty.StringVal("foo")}

					expectedMap := map[string]cty.Value{
						"key":   cty.StringVal("through-local-aloha-foo"),
						"value": cty.StringVal("foo"),
					}

					if cmp.Diff(stringOutput, expectedString, ctydebug.CmpOptions) != "" {
						t.Fatalf("string output is wrong, expected %q", expectedString.AsString())
					}

					if cmp.Diff(listOutput, expectedList, ctydebug.CmpOptions) != "" {
						t.Fatalf("list output is wrong, expected \n%+v,\ngot\n%+v", expectedList, listOutput)
					}

					if cmp.Diff(mapOutput, expectedMap, ctydebug.CmpOptions) != "" {
						t.Fatalf("map output is wrong, expected \n%+v,\ngot\n%+v", expectedMap, mapOutput)
					}
				}
			default:
				break
			}
		}
		assertNoDiagnostics(t, diags)

		return state, nil
	})

	if err != nil {
		t.Fatal(err)
	}
}
