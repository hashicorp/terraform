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
		toBeImplemented                 bool
		module                          map[string]string
		mode                            plans.Mode
		prevRunState                    *states.State
		events                          []providers.InvokeActionEvent
		callingInvokeReturnsDiagnostics func(providers.InvokeActionRequest) tfdiags.Diagnostics
		planOpts                        *PlanOpts

		expectInvokeActionCalled bool
		expectInvokeActionCalls  []providers.InvokeActionRequest

		expectDiagnostics tfdiags.Diagnostics
	}{
		"unreferenced": {
			module: map[string]string{
				"main.tf": `
action "act_unlinked" "hello" {}
		`,
			},
			expectInvokeActionCalled: false,
		},

		"before_create triggered": {
			module: map[string]string{
				"main.tf": `
action "act_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.act_unlinked.hello]
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
action "act_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.act_unlinked.hello]
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
action "act_unlinked" "hello" {}
resource "test_object" "a" {
  name = "new name"
  lifecycle {
    action_trigger {
      events = [before_update]
      actions = [action.act_unlinked.hello]
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
action "act_unlinked" "hello" {}
resource "test_object" "a" {
  name = "new name"
  lifecycle {
    action_trigger {
      events = [after_update]
      actions = [action.act_unlinked.hello]
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
action "act_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.act_unlinked.hello]
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

		"before_create failing when calling invoke": {
			module: map[string]string{
				"main.tf": `
action "act_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.act_unlinked.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			callingInvokeReturnsDiagnostics: func(providers.InvokeActionRequest) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{
					tfdiags.Sourceless(
						tfdiags.Error,
						"test case for failing",
						"this simulates a provider failing before the action is invoked",
					),
				}
			},
			expectDiagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"test case for failing",
					"this simulates a provider failing before the action is invoked",
				),
			},
		},

		"failing an action stops next actions in list": {
			module: map[string]string{
				"main.tf": `
action "act_unlinked" "hello" {}
action "act_unlinked" "failure" {
  config {
    attr = "failure"
  }
}
action "act_unlinked" "goodbye" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.act_unlinked.hello, action.act_unlinked.failure, action.act_unlinked.goodbye]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			callingInvokeReturnsDiagnostics: func(r providers.InvokeActionRequest) tfdiags.Diagnostics {
				if !r.PlannedActionData.IsNull() && r.PlannedActionData.GetAttr("attr").AsString() == "failure" {
					// Simulate a failure for the second action
					return tfdiags.Diagnostics{
						tfdiags.Sourceless(
							tfdiags.Error,
							"test case for failing",
							"this simulates a provider failing before the action is invoked",
						),
					}
				}
				return tfdiags.Diagnostics{}
			},
			expectDiagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"test case for failing",
					"this simulates a provider failing before the action is invoked",
				),
			},
			// We expect two calls but not the third one, because the second action fails
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "act_unlinked",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}, {
				ActionType: "act_unlinked",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("failure"),
				}),
			}},
		},

		"failing an action stops next action triggers": {
			module: map[string]string{
				"main.tf": `
action "act_unlinked" "hello" {}
action "act_unlinked" "failure" {
  config {
    attr = "failure"
  }
}
action "act_unlinked" "goodbye" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.act_unlinked.hello]
    }
    action_trigger {
      events = [before_create]
      actions = [action.act_unlinked.failure]
    }
    action_trigger {
      events = [before_create]
      actions = [action.act_unlinked.goodbye]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			callingInvokeReturnsDiagnostics: func(r providers.InvokeActionRequest) tfdiags.Diagnostics {
				if !r.PlannedActionData.IsNull() && r.PlannedActionData.GetAttr("attr").AsString() == "failure" {
					// Simulate a failure for the second action
					return tfdiags.Diagnostics{
						tfdiags.Sourceless(
							tfdiags.Error,
							"test case for failing",
							"this simulates a provider failing before the action is invoked",
						),
					}
				}
				return tfdiags.Diagnostics{}
			},
			expectDiagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"test case for failing",
					"this simulates a provider failing before the action is invoked",
				),
			},
			// We expect two calls but not the third one, because the second action fails
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "act_unlinked",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}, {
				ActionType: "act_unlinked",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("failure"),
				}),
			}},
		},

		"action with configuration": {
			module: map[string]string{
				"main.tf": `
resource "test_object" "a" {
  name = "foo"
}
action "act_unlinked" "hello" {
  config {
    attr = resource.test_object.a.name
  }
}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.act_unlinked.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "act_unlinked",
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
action "act_unlinked" "hello" {
  config {
    attr = var.unknown_value
  }
}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.act_unlinked.hello]
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
				ActionType: "act_unlinked",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.UnknownVal(cty.String),
				}),
			}},
		},

		"action with secrets in configuration": {
			toBeImplemented: true, // We currently don't suppport sensitive values in the plan
			module: map[string]string{
				"main.tf": `
variable "secret_value" {
  type = string
  sensitive = true
}
action "act_unlinked" "hello" {
  config {
    attr = var.secret_value
  }
}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.act_unlinked.hello]
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
				ActionType: "act_unlinked",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("psst, I'm secret"),
				}),
			}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.toBeImplemented {
				t.Skip("This test is not implemented yet")
			}

			m := testModuleInline(t, tc.module)

			invokeActionCalls := []providers.InvokeActionRequest{}

			testProvider := &testing_provider.MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
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
			}

			actionProvider := &testing_provider.MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					Actions: map[string]providers.ActionSchema{
						"act_unlinked": {
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
					ResourceTypes: map[string]providers.Schema{},
				},
				InvokeActionFn: func(req providers.InvokeActionRequest) providers.InvokeActionResponse {
					invokeActionCalls = append(invokeActionCalls, req)

					if tc.callingInvokeReturnsDiagnostics != nil && len(tc.callingInvokeReturnsDiagnostics(req)) > 0 {
						return providers.InvokeActionResponse{
							Diagnostics: tc.callingInvokeReturnsDiagnostics(req),
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
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(testProvider),
					addrs.NewDefaultProvider("act"):  testProviderFuncFixed(actionProvider),
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
			} else {
				tfdiags.AssertNoDiagnostics(t, diags)
			}

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
					t.Fatalf("expected invoke action call %d PlannedActionData to be %s, got %s", i, expectedCall.PlannedActionData.GoString(), actualCall.PlannedActionData.GoString())
				}
			}
		})
	}
}
