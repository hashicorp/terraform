// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
)

type deferredActionsTest struct {
	configs map[string]string

	// The starting state for the first stage. This can be nil, and the test
	// will create a new empty state if so.
	state *states.State

	// This test will execute a plan-apply cycle for every entry in this
	// slice. At each stage the plan and apply outputs will be validated
	// against the expected values.
	stages []deferredActionsTestStage
}

type deferredActionsTestStage struct {
	// The inputs at each plan-apply cycle.
	inputs map[string]cty.Value

	// The values we want to be planned within each cycle.
	wantPlanned map[string]cty.Value

	// The expected actions from the plan step.
	wantActions map[string]plans.Action

	// The values we want to be applied during each cycle. If this is
	// nil, then the apply step will be skipped.
	wantApplied map[string]cty.Value

	// The values we want to be returned by the outputs. If applied is
	// nil, then this should also be nil as the apply step will be
	// skipped.
	wantOutputs map[string]cty.Value

	// Whether the plan should be completed during this stage.
	complete bool

	// buildOpts is an optional field, that lets the test specify additional
	// options to be used when building the plan.
	buildOpts func(opts *PlanOpts)
}

var (
	// We build some fairly complex configurations here, so we'll use separate
	// variables for each one outside of the test function itself for clarity.

	// resourceForEachTest is a test that exercises the deferred actions
	// mechanism with a configuration that has a resource with an unknown
	// for_each attribute.
	//
	// We execute three plan-apply cycles. The first one with an unknown input
	// into the for_each. The second with a known for_each value. The final
	// with the same known for_each value to ensure that the plan is empty.
	resourceForEachTest = deferredActionsTest{
		configs: map[string]string{
			"main.tf": `
// TEMP: unknown for_each currently requires an experiment opt-in.
// We should remove this block if the experiment gets stabilized.
terraform {
	experiments = [unknown_instances]
}

variable "each" {
	type = set(string)
}

resource "test" "a" {
	name = "a"
}

resource "test" "b" {
	for_each = var.each

	name           = "b:${each.key}"
	upstream_names = [test.a.name]
}

resource "test" "c" {
	name = "c"
	upstream_names = setunion(
		[for v in test.b : v.name],
		[test.a.name],
	)
}

output "a" {
	value = test.a
}
output "b" {
	value = test.b
}
output "c" {
	value = test.c
}
		`,
		},
		stages: []deferredActionsTestStage{
			{
				inputs: map[string]cty.Value{
					"each": cty.DynamicVal,
				},
				wantPlanned: map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"name":           cty.StringVal("a"),
						"upstream_names": cty.NullVal(cty.Set(cty.String)),
					}),
					"<unknown>": cty.ObjectVal(map[string]cty.Value{
						"name": cty.UnknownVal(cty.String).Refine().
							StringPrefixFull("b:").
							NotNull().
							NewValue(),
						"upstream_names": cty.SetVal([]cty.Value{
							cty.StringVal("a"),
						}),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"name":           cty.StringVal("c"),
						"upstream_names": cty.UnknownVal(cty.Set(cty.String)).RefineNotNull(),
					}),
				},
				wantActions: map[string]plans.Action{
					"test.a": plans.Create,
					// The other resources will be deferred, so shouldn't
					// have any action at this stage.
				},
				wantApplied: map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"name":           cty.StringVal("a"),
						"upstream_names": cty.NullVal(cty.Set(cty.String)),
					}),
				},
				wantOutputs: map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"name":           cty.StringVal("a"),
						"upstream_names": cty.NullVal(cty.Set(cty.String)),
					}),

					// FIXME: The system is currently producing incorrect
					//   results for output values that are derived from
					//   resources that had deferred actions, because we're
					//   not quite reconstructing all of the deferral state
					//   correctly during the apply phase. The commented-out
					//   lines below show how this _ought_ to look, but
					//   we're accepting the incorrect answer for now so we
					//   can start to gather feedback on the experiment
					//   sooner, since the output value state at the interim
					//   steps isn't really that important for demonstrating
					//   the overall effect. We should fix this before
					//   stabilizing the experiment, though.

					// Currently we produce an incorrect result for output
					// value "b" because the expression evaluator doesn't
					// realize it's supposed to be treating this as deferred
					// during the apply phase, and so it incorrectly decides
					// that there are no instances due to the lack of
					// instances in the state.
					"b": cty.EmptyObjectVal,
					// We can't say anything about test.b until we know what
					// its instance keys are.
					// "b": cty.DynamicVal,

					// Currently we produce an incorrect result for output
					// value "c" because the expression evaluator doesn't
					// realize it's supposed to be treating this as deferred
					// during the apply phase, and so it incorrectly decides
					// that there is instance due to the lack of instances
					// in the state.
					"c": cty.NullVal(cty.DynamicPseudoType),
					// test.c evaluates to the placeholder value that shows
					// what we're expecting this object to look like in the
					// next round.
					// "c": cty.ObjectVal(map[string]cty.Value{
					// 	"name":           cty.StringVal("c"),
					// 	"upstream_names": cty.UnknownVal(cty.Set(cty.String)).RefineNotNull(),
					// }),
				},
			},
			{
				inputs: map[string]cty.Value{
					"each": cty.SetVal([]cty.Value{
						cty.StringVal("1"),
						cty.StringVal("2"),
					}),
				},
				wantPlanned: map[string]cty.Value{
					// test.a gets re-planned (to confirm that nothing has changed)
					"a": cty.ObjectVal(map[string]cty.Value{
						"name":           cty.StringVal("a"),
						"upstream_names": cty.NullVal(cty.Set(cty.String)),
					}),
					// test.b is now planned for real, once for each instance
					"b:1": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("b:1"),
						"upstream_names": cty.SetVal([]cty.Value{
							cty.StringVal("a"),
						}),
					}),
					"b:2": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("b:2"),
						"upstream_names": cty.SetVal([]cty.Value{
							cty.StringVal("a"),
						}),
					}),
					// test.c gets re-planned, so we can finalize its values
					// based on the new results from test.b.
					"c": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("c"),
						"upstream_names": cty.SetVal([]cty.Value{
							cty.StringVal("a"),
							cty.StringVal("b:1"),
							cty.StringVal("b:2"),
						}),
					}),
				},
				wantActions: map[string]plans.Action{
					// Since this plan is "complete", we expect to have a planned
					// action for every resource instance, although test.a is
					// no-op because nothing has changed for it since last round.
					`test.a`:      plans.NoOp,
					`test.b["1"]`: plans.Create,
					`test.b["2"]`: plans.Create,
					`test.c`:      plans.Create,
				},
				wantApplied: map[string]cty.Value{
					// Since test.a is no-op, it isn't visited during apply. The
					// other instances should all be applied, though.
					"b:1": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("b:1"),
						"upstream_names": cty.SetVal([]cty.Value{
							cty.StringVal("a"),
						}),
					}),
					"b:2": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("b:2"),
						"upstream_names": cty.SetVal([]cty.Value{
							cty.StringVal("a"),
						}),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("c"),
						"upstream_names": cty.SetVal([]cty.Value{
							cty.StringVal("a"),
							cty.StringVal("b:1"),
							cty.StringVal("b:2"),
						}),
					}),
				},
				wantOutputs: map[string]cty.Value{
					// Now everything should be fully resolved and known.
					// A is fully resolved and known.
					"a": cty.ObjectVal(map[string]cty.Value{
						"name":           cty.StringVal("a"),
						"upstream_names": cty.NullVal(cty.Set(cty.String)),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"1": cty.ObjectVal(map[string]cty.Value{
							"name": cty.StringVal("b:1"),
							"upstream_names": cty.SetVal([]cty.Value{
								cty.StringVal("a"),
							}),
						}),
						"2": cty.ObjectVal(map[string]cty.Value{
							"name": cty.StringVal("b:2"),
							"upstream_names": cty.SetVal([]cty.Value{
								cty.StringVal("a"),
							}),
						}),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("c"),
						"upstream_names": cty.SetVal([]cty.Value{
							cty.StringVal("a"),
							cty.StringVal("b:1"),
							cty.StringVal("b:2"),
						}),
					}),
				},
				complete: true,
			},
			{
				inputs: map[string]cty.Value{
					"each": cty.SetVal([]cty.Value{
						cty.StringVal("1"),
						cty.StringVal("2"),
					}),
				},
				wantPlanned: map[string]cty.Value{
					// Everything gets re-planned to confirm that nothing has changed.
					"a": cty.ObjectVal(map[string]cty.Value{
						"name":           cty.StringVal("a"),
						"upstream_names": cty.NullVal(cty.Set(cty.String)),
					}),
					"b:1": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("b:1"),
						"upstream_names": cty.SetVal([]cty.Value{
							cty.StringVal("a"),
						}),
					}),
					"b:2": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("b:2"),
						"upstream_names": cty.SetVal([]cty.Value{
							cty.StringVal("a"),
						}),
					}),
					"c": cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("c"),
						"upstream_names": cty.SetVal([]cty.Value{
							cty.StringVal("a"),
							cty.StringVal("b:1"),
							cty.StringVal("b:2"),
						}),
					}),
				},
				wantActions: map[string]plans.Action{
					// No changes needed
					`test.a`:      plans.NoOp,
					`test.b["1"]`: plans.NoOp,
					`test.b["2"]`: plans.NoOp,
					`test.c`:      plans.NoOp,
				},
				complete: true,
				// We won't execute an apply step in this stage, because the
				// plan should be empty.
			},
		},
	}

	// resourceReadTest is a test that covers the behavior of reading resources
	// in a refresh when the refresh is responding with a deferral.
	resourceReadTest = deferredActionsTest{
		configs: map[string]string{
			"main.tf": `
// TEMP: unknown for_each currently requires an experiment opt-in.
// We should remove this block if the experiment gets stabilized.
terraform {
	experiments = [unknown_instances]
}

resource "test" "a" {
	name = "deferred_read"
}

output "a" {
	value = test.a
}
		`,
		},
		stages: []deferredActionsTestStage{
			{
				buildOpts: func(opts *PlanOpts) {
					opts.Mode = plans.RefreshOnlyMode
				},
				inputs:      map[string]cty.Value{},
				wantPlanned: map[string]cty.Value{
					// The all resources will be deferred, so shouldn't
					// have any action at this stage.
				},

				wantActions: map[string]plans.Action{},
				wantApplied: map[string]cty.Value{
					// The all resources will be deferred, so shouldn't
					// have any action at this stage.
				},
				wantOutputs: map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"name":           cty.StringVal("a"),
						"upstream_names": cty.NullVal(cty.Set(cty.String)),
					}),
				},
				complete: true,
			},
		},
	}
)

func TestContextApply_deferredActions(t *testing.T) {
	tests := map[string]deferredActionsTest{
		"resource_for_each": resourceForEachTest,
		"resource_read":     resourceReadTest,
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			// Initialise the context.
			cfg := testModuleInline(t, test.configs)

			// Initialise the state.
			state := test.state
			if state == nil {
				state = states.NewState()
			}

			// Run through our cycle of planning and applying changes, checking
			// the results at each step.
			for ix, stage := range test.stages {
				t.Run(fmt.Sprintf("round-%d", ix), func(t *testing.T) {

					provider := &deferredActionsProvider{
						plannedChanges: &deferredActionsChanges{
							changes: make(map[string]cty.Value),
						},
						appliedChanges: &deferredActionsChanges{
							changes: make(map[string]cty.Value),
						},
					}

					ctx := testContext2(t, &ContextOpts{
						Providers: map[addrs.Provider]providers.Factory{
							addrs.NewDefaultProvider("test"): testProviderFuncFixed(provider.Provider()),
						},
					})

					opts := &PlanOpts{
						Mode: plans.NormalMode,
						SetVariables: func() InputValues {
							values := InputValues{}
							for name, value := range stage.inputs {
								values[name] = &InputValue{
									Value:      value,
									SourceType: ValueFromCaller,
								}
							}
							return values
						}(),
					}

					if stage.buildOpts != nil {
						stage.buildOpts(opts)
					}

					plan, diags := ctx.Plan(cfg, state, opts)
					if plan.Complete != stage.complete {
						t.Errorf("wrong completion status in plan: got %v, want %v", plan.Complete, stage.complete)
					}

					// TODO: Once we are including information about the
					//   individual deferred actions in the plan, this would be
					//   a good place to assert that they are correct!

					// We expect the correct planned changes and no diagnostics.
					assertNoDiagnostics(t, diags)
					provider.plannedChanges.Test(t, stage.wantPlanned)

					// We expect the correct actions.
					gotActions := make(map[string]plans.Action)
					for _, cs := range plan.Changes.Resources {
						gotActions[cs.Addr.String()] = cs.Action
					}
					if diff := cmp.Diff(stage.wantActions, gotActions); diff != "" {
						t.Errorf("wrong actions in plan\n%s", diff)
					}

					if stage.wantApplied == nil {
						// Don't execute the apply stage if wantApplied is nil.
						return
					}

					if opts.Mode == plans.RefreshOnlyMode {
						// Don't execute the apply stage if wantApplied is nil.
						return
					}

					updatedState, diags := ctx.Apply(plan, cfg, nil)

					// We expect the correct applied changes and no diagnostics.
					assertNoDiagnostics(t, diags)
					provider.appliedChanges.Test(t, stage.wantApplied)

					// We also want the correct output values.
					gotOutputs := make(map[string]cty.Value)
					for name, output := range updatedState.RootOutputValues {
						gotOutputs[name] = output.Value
					}
					if diff := cmp.Diff(stage.wantOutputs, gotOutputs, ctydebug.CmpOptions); diff != "" {
						t.Errorf("wrong output values\n%s", diff)
					}

					// Update the state for the next stage.
					state = updatedState
				})
			}
		})
	}
}

// deferredActionsChanges is a concurrent-safe map of changes from a
// deferredActionsProvider.
type deferredActionsChanges struct {
	sync.RWMutex
	changes map[string]cty.Value
}

func (d *deferredActionsChanges) Set(key string, value cty.Value) {
	d.Lock()
	defer d.Unlock()
	if d.changes == nil {
		d.changes = make(map[string]cty.Value)
	}
	d.changes[key] = value
}

func (d *deferredActionsChanges) Get(key string) cty.Value {
	d.RLock()
	defer d.RUnlock()
	return d.changes[key]
}

func (d *deferredActionsChanges) Test(t *testing.T, expected map[string]cty.Value) {
	t.Helper()
	d.RLock()
	defer d.RUnlock()
	if diff := cmp.Diff(expected, d.changes, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

// deferredActionsProvider is a wrapper around the mock provider that keeps
// track of its own planned changes.
type deferredActionsProvider struct {
	plannedChanges *deferredActionsChanges
	appliedChanges *deferredActionsChanges
}

func (provider *deferredActionsProvider) Provider() providers.Interface {
	return &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				"test": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"name": {
								Type:     cty.String,
								Required: true,
							},
							"upstream_names": {
								Type:     cty.Set(cty.String),
								Optional: true,
							},
						},
					},
				},
			},
		},
		ReadResourceFn: func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
			if key := req.PriorState.GetAttr("name"); key.IsKnown() && key.AsString() == "deferred_read" {
				return providers.ReadResourceResponse{
					NewState: req.PriorState,
					Deferred: &providers.Deferred{
						Reason: providers.DeferredReasonProviderConfigUnknown,
					},
				}
			}

			return providers.ReadResourceResponse{
				NewState: req.PriorState,
			}
		},
		PlanResourceChangeFn: func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			key := "<unknown>"
			if v := req.Config.GetAttr("name"); v.IsKnown() {
				key = v.AsString()
			}

			provider.plannedChanges.Set(key, req.ProposedNewState)
			return providers.PlanResourceChangeResponse{
				PlannedState: req.ProposedNewState,
			}
		},
		ApplyResourceChangeFn: func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
			key := req.Config.GetAttr("name").AsString()
			provider.appliedChanges.Set(key, req.PlannedState)
			return providers.ApplyResourceChangeResponse{
				NewState: req.PlannedState,
			}
		},
	}
}
