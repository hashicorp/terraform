// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate/statekeys"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/internal/terraform"
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
		PlanningMode: plans.DestroyMode,
		ProviderFactories: ProviderFactories{
			addrs.NewBuiltInProvider("test"): func() (providers.Interface, error) {
				return &terraform.MockProvider{
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
