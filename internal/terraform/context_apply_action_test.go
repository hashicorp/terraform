// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Apply_actions(t *testing.T) {
	for name, tc := range map[string]struct {
		module                          map[string]string
		mode                            plans.Mode
		prevRunState                    *states.State
		events                          []providers.InvokeActionEvent
		callingInvokeReturnsDiagnostics tfdiags.Diagnostics
		planOpts                        *PlanOpts

		expectInvokeActionCalled bool
		expectInvokeActionCalls  []providers.InvokeActionRequest

		expectDiagnostics tfdiags.Diagnostics
	}{
		"unreferenced": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
		`,
			},
			expectInvokeActionCalled: false,
		},

		"before_create triggered": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
		},

		"after_create triggered": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.test_unlinked.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
		},

		"before_update triggered": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  name = "new name"
  lifecycle {
    action_trigger {
      events = [before_update]
      actions = [action.test_unlinked.hello]
    }
  }
}
`,
			},
			prevRunState: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_object",
						Name: "a",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"name":"old name"}`),
					},
					addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				)
			}),
			expectInvokeActionCalled: true,
		},

		"after_update triggered": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  name = "new name"
  lifecycle {
    action_trigger {
      events = [after_update]
      actions = [action.test_unlinked.hello]
    }
  }
}
`,
			},
			prevRunState: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_object",
						Name: "a",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"name":"old"}`),
					},
					addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				)
			}),
			expectInvokeActionCalled: true,
		},

		"before_create failing": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			events: []providers.InvokeActionEvent{
				providers.InvokeActionEvent_Completed{
					Diagnostics: tfdiags.Diagnostics{
						tfdiags.Sourceless(
							tfdiags.Error,
							"test case for failing",
							"this simulates a provider failing",
						),
					},
				},
			},

			expectDiagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"test case for failing",
					"this simulates a provider failing",
				),
			},
		},

		"before_create failing to call invoke": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			callingInvokeReturnsDiagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"test case for failing",
					"this simulates a provider failing before the action is invoked",
				),
			},
			expectDiagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"test case for failing",
					"this simulates a provider failing before the action is invoked",
				),
			},
		},

		"action with configuration": {
			module: map[string]string{
				"main.tf": `
resource "test_object" "a" {
  name = "foo"
}
action "test_unlinked" "hello" {
  config {
    attr = resource.test_object.a.name
  }
}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "test_unlinked",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("foo"),
				}),
			}},
		},

		// Providers can handle unknown values in the configuration
		"action with unknown configuration": {
			module: map[string]string{
				"main.tf": `
variable "unknown_value" {
  type = string
}
action "test_unlinked" "hello" {
  config {
    attr = var.unknown_value
  }
}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello]
    }
  }
}
`,
			},
			planOpts: SimplePlanOpts(plans.NormalMode, InputValues{
				"unknown_value": &InputValue{
					Value: cty.UnknownVal(cty.String),
				},
			}),

			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "test_unlinked",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.UnknownVal(cty.String),
				}),
			}},
		},

		"action with secrets in configuration": {
			module: map[string]string{
				"main.tf": `
variable "secret_value" {
  type = string
  sensitive = true
}
action "test_unlinked" "hello" {
  config {
    attr = var.secret_value
  }
}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello]
    }
  }
}
`,
			},
			planOpts: SimplePlanOpts(plans.NormalMode, InputValues{
				"secret_value": &InputValue{
					Value: cty.StringVal("psst, I'm secret").Mark(marks.Sensitive), // Not sure if we need the mark here, but it doesn't hurt
				},
			}),

			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "test_unlinked",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("psst, I'm secret"),
				}),
			}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := testModuleInline(t, tc.module)

			invokeActionCalls := []providers.InvokeActionRequest{}

			p := &testing_provider.MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					Actions: map[string]providers.ActionSchema{
						"test_unlinked": {
							ConfigSchema: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"attr": {
										Type:     cty.String,
										Optional: true,
									},
								},
							},

							Unlinked: &providers.UnlinkedAction{},
						},
					},
					ResourceTypes: map[string]providers.Schema{
						"test_object": {
							Body: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"name": {
										Type:     cty.String,
										Optional: true,
									},
								},
							},
						},
					},
				},
				InvokeActionFn: func(req providers.InvokeActionRequest) providers.InvokeActionResponse {
					invokeActionCalls = append(invokeActionCalls, req)

					if len(tc.callingInvokeReturnsDiagnostics) > 0 {
						return providers.InvokeActionResponse{
							Diagnostics: tc.callingInvokeReturnsDiagnostics,
						}
					}

					defaultEvents := []providers.InvokeActionEvent{}
					defaultEvents = append(defaultEvents, providers.InvokeActionEvent_Progress{
						Message: "Hello world!",
					})
					defaultEvents = append(defaultEvents, providers.InvokeActionEvent_Completed{})

					events := defaultEvents
					if len(tc.events) > 0 {
						events = tc.events
					}

					return providers.InvokeActionResponse{
						Events: func(yield func(providers.InvokeActionEvent) bool) {
							for _, event := range events {
								if !yield(event) {
									return
								}
							}
						},
					}
				},
			}

			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					// The providers never actually going to get called here, we should
					// catch the error long before anything happens.
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
				},
			})

			// Just a sanity check that the module is valid
			diags := ctx.Validate(m, &ValidateOpts{})
			tfdiags.AssertNoDiagnostics(t, diags)

			planOpts := SimplePlanOpts(plans.NormalMode, InputValues{})
			if tc.planOpts != nil {
				planOpts = tc.planOpts
			}

			plan, diags := ctx.Plan(m, tc.prevRunState, planOpts)
			tfdiags.AssertNoDiagnostics(t, diags)

			_, diags = ctx.Apply(plan, m, nil)
			if tc.expectDiagnostics.HasErrors() {
				tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectDiagnostics)
				return
			}
			tfdiags.AssertNoDiagnostics(t, diags)

			if tc.expectInvokeActionCalled && len(invokeActionCalls) == 0 {
				t.Fatalf("expected invoke action to be called, but it was not")
			}

			if len(tc.expectInvokeActionCalls) > 0 && len(invokeActionCalls) != len(tc.expectInvokeActionCalls) {
				t.Fatalf("expected %d invoke action calls, got %d", len(tc.expectInvokeActionCalls), len(invokeActionCalls))
			}
			for i, expectedCall := range tc.expectInvokeActionCalls {
				actualCall := invokeActionCalls[i]
				if actualCall.ActionType != expectedCall.ActionType {
					t.Fatalf("expected invoke action call %d ActionType to be %s, got %s", i, expectedCall.ActionType, actualCall.ActionType)
				}
				if !actualCall.PlannedActionData.RawEquals(expectedCall.PlannedActionData) {
					t.Fatalf("expected invoke action call %d PlannedActionData to be %s, got %s", i, expectedCall.PlannedActionData, actualCall.PlannedActionData)
				}
			}
		})
	}
}
