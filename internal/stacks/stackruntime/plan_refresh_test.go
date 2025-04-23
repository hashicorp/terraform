// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/version"
)

func TestRefreshPlan(t *testing.T) {
	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1991-08-25T20:57:08Z")
	if err != nil {
		t.Fatal(err)
	}

	tcs := map[string]struct {
		path  string
		state *stackstate.State
		store *stacks_testing_provider.ResourceStore
		cycle TestCycle
	}{
		"simple-valid": {
			path: filepath.Join("with-single-input", "valid"),
			state: stackstate.NewStateBuilder().
				AddComponentInstance(stackstate.NewComponentInstanceBuilder(mustAbsComponentInstance("component.self"))).
				AddResourceInstance(stackstate.NewResourceInstanceBuilder().
					SetAddr(mustAbsResourceInstanceObject("component.self.testing_resource.data")).
					SetProviderAddr(mustDefaultRootProvider("testing")).
					SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]any{
							"id":    "old",
							"value": "old",
						}),
					})).
				Build(),
			store: stacks_testing_provider.NewResourceStoreBuilder().
				AddResource("old", cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("old"),
					"value": cty.StringVal("new"),
				})).
				Build(),
			cycle: TestCycle{
				planInputs: map[string]cty.Value{
					"id":    cty.StringVal("old"),
					"input": cty.StringVal("old"),
				},
				wantPlannedChanges: []stackplan.PlannedChange{
					&stackplan.PlannedChangeApplyable{
						Applyable: true,
					},
					&stackplan.PlannedChangeComponentInstance{
						Addr:          mustAbsComponentInstance("component.self"),
						PlanApplyable: true,
						PlanComplete:  true,
						Action:        plans.Read,
						Mode:          plans.RefreshOnlyMode,
						PlannedInputValues: map[string]plans.DynamicValue{
							"id":    mustPlanDynamicValueDynamicType(cty.StringVal("old")),
							"input": mustPlanDynamicValueDynamicType(cty.StringVal("old")),
						},
						PlannedInputValueMarks: map[string][]cty.PathValueMarks{
							"id":    nil,
							"input": nil,
						},
						PlannedOutputValues: make(map[string]cty.Value),
						PlannedCheckResults: &states.CheckResults{},
						PlanTimestamp:       fakePlanTimestamp,
					},
					&stackplan.PlannedChangeResourceInstancePlanned{
						ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.testing_resource.data"),
						PriorStateSrc: &states.ResourceInstanceObjectSrc{
							Status: states.ObjectReady,
							AttrsJSON: mustMarshalJSONAttrs(map[string]any{
								"id":    "old",
								"value": "new",
							}),
							Dependencies: make([]addrs.ConfigResource, 0),
						},
						ProviderConfigAddr: mustDefaultRootProvider("testing"),
						Schema:             stacks_testing_provider.TestingResourceSchema,
					},
					&stackplan.PlannedChangeHeader{
						TerraformVersion: version.SemVer,
					},
					&stackplan.PlannedChangePlannedTimestamp{
						PlannedTimestamp: fakePlanTimestamp,
					},
					&stackplan.PlannedChangeRootInputValue{
						Addr:   mustStackInputVariable("id"),
						Action: plans.Create,
						Before: cty.NullVal(cty.DynamicPseudoType),
						After:  cty.StringVal("old"),
					},
					&stackplan.PlannedChangeRootInputValue{
						Addr:   mustStackInputVariable("input"),
						Action: plans.Create,
						Before: cty.NullVal(cty.DynamicPseudoType),
						After:  cty.StringVal("old"),
					},
				},
			},
		},
		"removed-component": {
			path: filepath.Join("with-single-input", "removed-component"),
			state: stackstate.NewStateBuilder().
				AddComponentInstance(stackstate.NewComponentInstanceBuilder(mustAbsComponentInstance("component.self")).
					AddInputVariable("id", cty.StringVal("old")).
					AddInputVariable("input", cty.StringVal("old"))).
				AddResourceInstance(stackstate.NewResourceInstanceBuilder().
					SetAddr(mustAbsResourceInstanceObject("component.self.testing_resource.data")).
					SetProviderAddr(mustDefaultRootProvider("testing")).
					SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]any{
							"id":    "old",
							"value": "old",
						}),
					})).
				Build(),
			store: stacks_testing_provider.NewResourceStoreBuilder().
				AddResource("old", cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("old"),
					"value": cty.StringVal("new"),
				})).
				Build(),
			cycle: TestCycle{
				wantPlannedChanges: []stackplan.PlannedChange{
					&stackplan.PlannedChangeApplyable{
						Applyable: true,
					},
					&stackplan.PlannedChangeComponentInstance{
						Addr:          mustAbsComponentInstance("component.self"),
						PlanApplyable: true,
						PlanComplete:  true,
						Action:        plans.Read,
						Mode:          plans.RefreshOnlyMode,
						PlannedInputValues: map[string]plans.DynamicValue{
							"id":    mustPlanDynamicValueDynamicType(cty.StringVal("old")),
							"input": mustPlanDynamicValueDynamicType(cty.StringVal("old")),
						},
						PlannedInputValueMarks: map[string][]cty.PathValueMarks{
							"id":    nil,
							"input": nil,
						},
						PlannedOutputValues: make(map[string]cty.Value),
						PlannedCheckResults: &states.CheckResults{},
						PlanTimestamp:       fakePlanTimestamp,
					},
					&stackplan.PlannedChangeResourceInstancePlanned{
						ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.testing_resource.data"),
						PriorStateSrc: &states.ResourceInstanceObjectSrc{
							Status: states.ObjectReady,
							AttrsJSON: mustMarshalJSONAttrs(map[string]any{
								"id":    "old",
								"value": "new",
							}),
							Dependencies: make([]addrs.ConfigResource, 0),
						},
						ProviderConfigAddr: mustDefaultRootProvider("testing"),
						Schema:             stacks_testing_provider.TestingResourceSchema,
					},
					&stackplan.PlannedChangeHeader{
						TerraformVersion: version.SemVer,
					},
					&stackplan.PlannedChangePlannedTimestamp{
						PlannedTimestamp: fakePlanTimestamp,
					},
				},
			},
		},
		"removed-stack": {
			path: filepath.Join("with-single-input", "removed-stack-instance-dynamic"),
			state: stackstate.NewStateBuilder().
				AddComponentInstance(stackstate.NewComponentInstanceBuilder(mustAbsComponentInstance("stack.simple[\"old\"].component.self")).
					AddInputVariable("id", cty.StringVal("old")).
					AddInputVariable("input", cty.StringVal("old"))).
				AddResourceInstance(stackstate.NewResourceInstanceBuilder().
					SetAddr(mustAbsResourceInstanceObject("stack.simple[\"old\"].component.self.testing_resource.data")).
					SetProviderAddr(mustDefaultRootProvider("testing")).
					SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]any{
							"id":    "old",
							"value": "old",
						}),
					})).
				Build(),
			store: stacks_testing_provider.NewResourceStoreBuilder().
				AddResource("old", cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("old"),
					"value": cty.StringVal("new"),
				})).
				Build(),
			cycle: TestCycle{
				planInputs: map[string]cty.Value{
					"removed": cty.MapVal(map[string]cty.Value{
						"old": cty.StringVal("old"),
					}),
				},
				wantPlannedChanges: []stackplan.PlannedChange{
					&stackplan.PlannedChangeApplyable{
						Applyable: true,
					},
					&stackplan.PlannedChangeHeader{
						TerraformVersion: version.SemVer,
					},
					&stackplan.PlannedChangePlannedTimestamp{
						PlannedTimestamp: fakePlanTimestamp,
					},
					&stackplan.PlannedChangeComponentInstance{
						Addr:          mustAbsComponentInstance("stack.simple[\"old\"].component.self"),
						PlanApplyable: true,
						PlanComplete:  true,
						Action:        plans.Read,
						Mode:          plans.RefreshOnlyMode,
						PlannedInputValues: map[string]plans.DynamicValue{
							"id":    mustPlanDynamicValueDynamicType(cty.StringVal("old")),
							"input": mustPlanDynamicValueDynamicType(cty.StringVal("old")),
						},
						PlannedInputValueMarks: map[string][]cty.PathValueMarks{
							"id":    nil,
							"input": nil,
						},
						PlannedOutputValues: make(map[string]cty.Value),
						PlannedCheckResults: &states.CheckResults{},
						PlanTimestamp:       fakePlanTimestamp,
					},
					&stackplan.PlannedChangeResourceInstancePlanned{
						ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("stack.simple[\"old\"].component.self.testing_resource.data"),
						PriorStateSrc: &states.ResourceInstanceObjectSrc{
							Status: states.ObjectReady,
							AttrsJSON: mustMarshalJSONAttrs(map[string]any{
								"id":    "old",
								"value": "new",
							}),
							Dependencies: make([]addrs.ConfigResource, 0),
						},
						ProviderConfigAddr: mustDefaultRootProvider("testing"),
						Schema:             stacks_testing_provider.TestingResourceSchema,
					},
					&stackplan.PlannedChangeRootInputValue{
						Addr:   stackaddrs.InputVariable{Name: "input"},
						Action: plans.Create,
						Before: cty.NullVal(cty.DynamicPseudoType),
						After:  cty.MapValEmpty(cty.String),
					},
					&stackplan.PlannedChangeRootInputValue{
						Addr:   stackaddrs.InputVariable{Name: "removed"},
						Action: plans.Create,
						Before: cty.NullVal(cty.DynamicPseudoType),
						After: cty.MapVal(map[string]cty.Value{
							"old": cty.StringVal("old"),
						}),
					},
					&stackplan.PlannedChangeRootInputValue{
						Addr:   stackaddrs.InputVariable{Name: "removed-direct"},
						Action: plans.Create,
						Before: cty.NullVal(cty.DynamicPseudoType),
						After:  cty.SetValEmpty(cty.String),
					},
				},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			cycle := tc.cycle
			cycle.planMode = plans.RefreshOnlyMode // set this for all the tests here

			ctx := context.Background()

			lock := depsfile.NewLocks()
			lock.SetProvider(
				addrs.NewDefaultProvider("testing"),
				providerreqs.MustParseVersion("0.0.0"),
				providerreqs.MustParseVersionConstraints("=0.0.0"),
				providerreqs.PreferredHashes([]providerreqs.Hash{}),
			)

			store := tc.store
			if store == nil {
				store = stacks_testing_provider.NewResourceStore()
			}

			testContext := TestContext{
				timestamp: &fakePlanTimestamp,
				config:    loadMainBundleConfigForTest(t, tc.path),
				providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
						return stacks_testing_provider.NewProviderWithData(t, store), nil
					},
				},
				dependencyLocks: *lock,
			}

			testContext.Plan(t, ctx, tc.state, cycle)
		})
	}
}
