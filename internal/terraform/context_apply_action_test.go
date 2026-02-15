// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContextApply_actions(t *testing.T) {
	tests := map[string]struct {
		module                          map[string]string
		prevRunState                    *states.State
		events                          func(req providers.InvokeActionRequest) []providers.InvokeActionEvent
		readResourceFn                  func(*testing.T, providers.ReadResourceRequest) providers.ReadResourceResponse
		callingInvokeReturnsDiagnostics func(providers.InvokeActionRequest) tfdiags.Diagnostics

		planOpts  *PlanOpts
		applyOpts *ApplyOpts

		expectInvokeActionCalled            bool
		expectInvokeActionCalls             []providers.InvokeActionRequest
		expectInvokeActionCallsAreUnordered bool
		expectDiagnostics                   func(m *configs.Config) tfdiags.Diagnostics
		ignoreWarnings                      bool

		assertHooks func(*testing.T, actionHookCapture)
	}{
		"before_create triggered": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
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
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
		},

		"before and after created triggered": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create,after_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			assertHooks: func(t *testing.T, capture actionHookCapture) {
				if len(capture.startActionHooks) != 2 {
					t.Error("expected 2 start action hooks")
				}
				if len(capture.completeActionHooks) != 2 {
					t.Error("expected 2 complete action hooks")
				}

				evaluateHook := func(got HookActionIdentity, wantAddr string, wantEvent configs.ActionTriggerEvent) {
					trigger := got.ActionTrigger.(*plans.LifecycleActionTrigger)

					if trigger.ActionTriggerEvent != wantEvent {
						t.Errorf("wrong event, got %s, want %s", trigger.ActionTriggerEvent, wantEvent)
					}
					if diff := cmp.Diff(got.Addr.String(), wantAddr); len(diff) > 0 {
						t.Errorf("wrong address: %s", diff)
					}
				}

				// the before should have happened first, and the order should
				// be correct.
				evaluateHook(capture.startActionHooks[0], "action.action_example.hello", configs.BeforeCreate)
				evaluateHook(capture.completeActionHooks[0], "action.action_example.hello", configs.BeforeCreate)
				evaluateHook(capture.startActionHooks[1], "action.action_example.hello", configs.AfterCreate)
				evaluateHook(capture.completeActionHooks[1], "action.action_example.hello", configs.AfterCreate)
			},
		},

		"before_update triggered": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  test_string = "new name"
  lifecycle {
    action_trigger {
      events = [before_update]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			prevRunState: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"test_string":"old name"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
			expectInvokeActionCalled: true,
		},

		"after_update triggered": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  test_string = "new name"
  lifecycle {
    action_trigger {
      events = [after_update]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			prevRunState: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"test_string":"old"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
			expectInvokeActionCalled: true,
		},

		"before_create failing": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			events: func(req providers.InvokeActionRequest) []providers.InvokeActionEvent {
				return []providers.InvokeActionEvent{
					providers.InvokeActionEvent_Completed{
						Diagnostics: tfdiags.Diagnostics{
							tfdiags.Sourceless(
								tfdiags.Error,
								"test case for failing",
								"this simulates a provider failing",
							),
						},
					},
				}
			},
			expectDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Error when invoking action",
					Detail:   "test case for failing: this simulates a provider failing",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 7, Column: 18, Byte: 148},
						End:      hcl.Pos{Line: 7, Column: 45, Byte: 175},
					},
				})
			},
		},

		"before_create failing with successfully completed actions": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
action "action_example" "world" {}
action "action_example" "failure" {
  config {
    attr = "failure"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello, action.action_example.world, action.action_example.failure]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			events: func(req providers.InvokeActionRequest) []providers.InvokeActionEvent {
				if !req.PlannedActionData.IsNull() && req.PlannedActionData.GetAttr("attr").AsString() == "failure" {
					return []providers.InvokeActionEvent{
						providers.InvokeActionEvent_Completed{
							Diagnostics: tfdiags.Diagnostics{
								tfdiags.Sourceless(
									tfdiags.Error,
									"test case for failing",
									"this simulates a provider failing",
								),
							},
						},
					}
				} else {
					return []providers.InvokeActionEvent{
						providers.InvokeActionEvent_Completed{},
					}
				}
			},
			expectDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Error when invoking action",
						Detail:   `test case for failing: this simulates a provider failing`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 13, Column: 76, Byte: 315},
							End:      hcl.Pos{Line: 13, Column: 105, Byte: 344},
						},
					},
				)

			},
		},

		"before_create failing when calling invoke": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
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
			expectDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Error when invoking action",
						Detail:   "test case for failing: this simulates a provider failing before the action is invoked",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 7, Column: 18, Byte: 148},
							End:      hcl.Pos{Line: 7, Column: 47, Byte: 175},
						},
					},
				)
			},
		},

		"failing an action by action event stops next actions in list": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
action "action_example" "failure" {
  config {
    attr = "failure"
  }
}
action "action_example" "goodbye" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello, action.action_example.failure, action.action_example.goodbye]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			events: func(r providers.InvokeActionRequest) []providers.InvokeActionEvent {
				if !r.PlannedActionData.IsNull() && r.PlannedActionData.GetAttr("attr").AsString() == "failure" {
					return []providers.InvokeActionEvent{
						providers.InvokeActionEvent_Completed{
							Diagnostics: tfdiags.Diagnostics{}.Append(tfdiags.Sourceless(tfdiags.Error, "test case for failing", "this simulates a provider failing")),
						},
					}
				}

				return []providers.InvokeActionEvent{
					providers.InvokeActionEvent_Completed{},
				}

			},
			expectDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Error when invoking action",
						Detail:   "test case for failing: this simulates a provider failing",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 13, Column: 47, Byte: 288},
							End:      hcl.Pos{Line: 13, Column: 76, Byte: 317},
						},
					},
				)
			},
			// We expect two calls but not the third one, because the second action fails
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("failure"),
				}),
			}},
		},

		"failing an action during invocation stops next actions in list": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
action "action_example" "failure" {
  config {
    attr = "failure"
  }
}
action "action_example" "goodbye" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello, action.action_example.failure, action.action_example.goodbye]
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
							"this simulates a provider failing",
						),
					}
				}
				return tfdiags.Diagnostics{}
			},
			expectDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Error when invoking action",
						Detail:   "test case for failing: this simulates a provider failing",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 13, Column: 47, Byte: 288},
							End:      hcl.Pos{Line: 13, Column: 76, Byte: 317},
						},
					},
				)
			},
			// We expect two calls but not the third one, because the second action fails
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("failure"),
				}),
			}},
		},

		"failing an action stops next action triggers": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
action "action_example" "failure" {
  config {
    attr = "failure"
  }
}
action "action_example" "goodbye" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
    action_trigger {
      events = [before_create]
      actions = [action.action_example.failure]
    }
    action_trigger {
      events = [before_create]
      actions = [action.action_example.goodbye]
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
							"this simulates a provider failing",
						),
					}
				}
				return tfdiags.Diagnostics{}
			},
			expectDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Error when invoking action",
						Detail:   "test case for failing: this simulates a provider failing",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 17, Column: 18, Byte: 363},
							End:      hcl.Pos{Line: 17, Column: 47, Byte: 392},
						},
					},
				)
			},
			// We expect two calls but not the third one, because the second action fails
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("failure"),
				}),
			}},
		},

		"action with configuration": {
			module: map[string]string{
				"main.tf": `
resource "test_object" "a" {
  test_string = "foo"
}
action "action_example" "hello" {
  config {
    attr = resource.test_object.a.test_string
  }
}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
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
action "action_example" "hello" {
  config {
    attr = var.unknown_value
  }
}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
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
			expectInvokeActionCalled: false,
			expectDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Action configuration unknown during apply",
					Detail:   "The action action.action_example.hello was not fully known during apply.\n\nThis is a bug in Terraform, please report it.",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 14, Column: 18, Byte: 238},
						End:      hcl.Pos{Line: 14, Column: 45, Byte: 265},
					},
				})
			},
		},

		"action with secrets in configuration": {
			module: map[string]string{
				"main.tf": `
variable "secret_value" {
  type = string
  sensitive = true
}
action "action_example" "hello" {
  config {
    attr = var.secret_value
  }
}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
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
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("psst, I'm secret"),
				}),
			}},
		},

		"aliased provider": {
			module: map[string]string{
				"main.tf": `
provider "action" {
  alias = "aliased"
}
action "action_example" "hello" {
  provider = action.aliased
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
		},

		"non-default namespace provider": {
			module: map[string]string{
				"main.tf": `
terraform {
  required_providers {
    ecosystem = {
      source = "danielmschmidt/ecosystem"
    }
  }
}
action "ecosystem" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.ecosystem.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
		},

		"after_create with config cycle": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {
  config {
    attr = test_object.a.test_string
  }
}
resource "test_object" "a" {
  test_string = "test_object_a"
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("test_object_a"),
				}),
			}},
		},

		"triggered within module": {
			module: map[string]string{
				"main.tf": `
module "mod" {
    source = "./mod"
}
`,
				"mod/mod.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}},
		},

		"triggered within module instance": {
			module: map[string]string{
				"main.tf": `
module "mod" {
    count = 2
    source = "./mod"
}
`,
				"mod/mod.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}},
		},

		"not triggered with no module instances": {
			module: map[string]string{
				"main.tf": `
module "mod" {
    count = 0
    source = "./mod"
}

// an empty plan is not applyable so we have this extra resource here
resource "test_object" "a" {} 
`,
				"mod/mod.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: false,
		},

		"provider is within module": {
			module: map[string]string{
				"main.tf": `
module "mod" {
    source = "./mod"
}
`,
				"mod/mod.tf": `
provider "action" {
    alias = "inthemodule"
}
action "action_example" "hello" {
  provider = action.inthemodule
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}},
		},

		"action for_each": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {
  for_each = toset(["a", "b"])
  
  config {
    attr = "value-${each.key}"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello["a"], action.action_example.hello["b"]]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("value-a"),
				}),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("value-b"),
				}),
			}},
		},

		"action count": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {
  count = 2

  config {
    attr = "value-${count.index}"
  }
}

resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello[0], action.action_example.hello[1]]
    }
  }
}
`,
			},
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("value-0"),
				}),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("value-1"),
				}),
			}},
		},

		"before_update triggered - on create": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_update]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: false,
		},

		"after_update triggered - on create": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_update]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: false,
		},

		"before_update triggered - on replace": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_update]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},

			prevRunState: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectTainted,
						AttrsJSON: []byte(`{"test_string":"previous_run"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
			expectInvokeActionCalled: false,
		},

		"after_update triggered - on replace": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_update]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			prevRunState: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectTainted,
						AttrsJSON: []byte(`{"test_string":"previous_run"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
			expectInvokeActionCalled: false,
		},

		"expanded resource - unexpanded action": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  count = 2
  test_string = "test-${count.index}"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}},
		},

		"transitive dependencies": {
			module: map[string]string{
				"main.tf": `
resource "test_object" "a" {
  test_string = "a"
}
action "action_example" "hello" {
  config {
    attr = test_object.a.test_string
  }
}
resource "test_object" "b" {
  test_string = "b"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("a"),
				}),
			}},
		},

		"expanded transitive dependencies": {
			module: map[string]string{
				"main.tf": `
resource "test_object" "a" {
  test_string = "a"
}
resource "test_object" "b" {
  test_string = "b"
}
action "action_example" "hello_a" {
  config {
    attr = test_object.a.test_string
  }
}
action "action_example" "hello_b" {
  config {
    attr = test_object.a.test_string
  }
}
resource "test_object" "c" {
  test_string = "c"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello_a]
    }
  }
}
resource "test_object" "d" {
  test_string = "d"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello_b]
    }
  }
}
resource "test_object" "e" {
  test_string = "e"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello_a, action.action_example.hello_b]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("a"),
				}),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("a"),
				}),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("a"),
				}),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("a"),
				}),
			}},
		},

		"destroy run": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: false,
			planOpts:                 SimplePlanOpts(plans.DestroyMode, InputValues{}),
			prevRunState: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"test_string":"previous_run"}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
		},

		"destroying expanded node": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  count = 2
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: false,
			prevRunState: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a[0]"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
				s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a[1]"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
				s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a[2]"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				)
			}),
		},

		"action config with after_create dependency to triggering resource": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {
  config {
    attr = test_object.a.test_string
  }
}
resource "test_object" "a" {
  test_string = "test_name"
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("test_name"),
				}),
			}},
		},

		"module action with different resource types": {
			module: map[string]string{
				"main.tf": `
module "action_mod" {
    source = "./action_mod"
}
`,
				"action_mod/main.tf": `
action "action_example" "hello" {}
resource "test_object" "trigger" {
  test_string = "trigger_resource"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
					"attr": cty.String,
				})),
			}},
		},

		"nested module actions": {
			module: map[string]string{
				"main.tf": `
module "parent" {
    source = "./parent"
}
`,
				"parent/main.tf": `
module "child" {
    source = "./child"
}
`,
				"parent/child/main.tf": `
action "action_example" "nested_action" {
  config {
    attr = "nested_value"
  }
}
resource "test_object" "nested_resource" {
  test_string = "nested"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.nested_action]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("nested_value"),
				}),
			}},
		},

		"module with for_each and actions": {
			module: map[string]string{
				"main.tf": `
module "multi_mod" {
    for_each = toset(["a", "b"])
    source = "./multi_mod"
    instance_name = each.key
}
`,
				"multi_mod/main.tf": `
variable "instance_name" {
  type = string
}

action "action_example" "hello" {
  config {
    attr = "instance-${var.instance_name}"
  }
}
resource "test_object" "resource" {
  test_string = "resource-${var.instance_name}"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled:            true,
			expectInvokeActionCallsAreUnordered: true, // The order depends on the order of the modules
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("instance-a"),
				}),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("instance-b"),
				}),
			}},
		},

		"write-only attributes": {
			module: map[string]string{
				"main.tf": `
variable "attr" {
  type = string
  ephemeral = true
}

resource "test_object" "resource" {
  test_string = "hello"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.action_example_wo.hello]
    }
  }
}

action "action_example_wo" "hello" {
  config {
    attr = var.attr
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example_wo",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("wo-apply"),
					}),
				},
			},
			planOpts: SimplePlanOpts(plans.NormalMode, InputValues{
				"attr": {Value: cty.StringVal("wo-plan")},
			}),
			applyOpts: &ApplyOpts{
				SetVariables: InputValues{"attr": {Value: cty.StringVal("wo-apply")}},
			},
		},

		"nested action config single + list blocks applies": {
			module: map[string]string{
				"main.tf": `
action "action_nested" "with_blocks" {
  config {
    top_attr = "top"
    settings {
      name = "primary"
      rule { value = "r1" }
      rule { value = "r2" }
    }
  }
}
resource "test_object" "a" {
  test_string = "object"
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.action_nested.with_blocks]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_nested",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"top_attr": cty.StringVal("top"),
						"settings": cty.ObjectVal(map[string]cty.Value{
							"name": cty.StringVal("primary"),
							"rule": cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{"value": cty.StringVal("r1")}),
								cty.ObjectVal(map[string]cty.Value{"value": cty.StringVal("r2")}),
							}),
						}),
						"settings_list": cty.ListValEmpty(cty.Object(map[string]cty.Type{
							"id": cty.String,
						})),
					}),
				},
			},
		},
		"nested action config top-level list blocks applies": {
			module: map[string]string{
				"main.tf": `
action "action_nested" "with_list" {
  config {
    settings_list { id = "one" }
    settings_list { id = "two" }
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.action_nested.with_list]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_nested",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"top_attr": cty.NullVal(cty.String),
					"settings": cty.NullVal(cty.Object(map[string]cty.Type{
						"name": cty.String,
						"rule": cty.List(cty.Object(map[string]cty.Type{
							"value": cty.String,
						})),
					})),
					"settings_list": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("one")}),
						cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("two")}),
					}),
				}),
			}},
		},

		"conditions": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {
    count = 3
    config {
        attr = "value-${count.index}"
    }
}
resource "test_object" "foo" {
    test_string = "foo"
}
resource "test_object" "resource" {
    test_string = "resource"
    lifecycle {
        action_trigger {
            events = [before_create]
            condition = test_object.foo.test_string == "bar"
            actions = [action.action_example.hello[0]]
        }
  
        action_trigger {
            events = [before_create]
            condition = test_object.foo.test_string == "foo"
            actions = [action.action_example.hello[1], action.action_example.hello[2]]
        }
    }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{{
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("value-1"),
				}),
			}, {
				ActionType: "action_example",
				PlannedActionData: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("value-2"),
				}),
			}},
		},

		"simple condition evaluation - true": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  test_string = "foo"
  lifecycle {
    action_trigger {
      events = [after_create]
      condition = "foo" == "foo"
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
		},

		"simple condition evaluation - false": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  test_string = "foo"
  lifecycle {
    action_trigger {
      events = [after_create]
      condition = "foo" == "bar"
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: false,
		},

		"using count.index in after_create condition": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  count = 3
  test_string = "item-${count.index}"
  lifecycle {
    action_trigger {
      events = [after_create]
      condition = count.index == 1
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
		},

		"using each.key in after_create condition": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  for_each = toset(["foo", "bar"])
  test_string = each.key
  lifecycle {
    action_trigger {
      events = [after_create]
      condition = each.key == "foo"
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
		},

		"using each.value in after_create condition": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  for_each = {"foo" = "value1", "bar" = "value2"}
  test_string = each.value
  lifecycle {
    action_trigger {
      events = [after_create]
      condition = each.value == "value1"
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
		},
		"referencing triggering resource in after_* condition": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {
  config {
    attr = "hello"
  }
}
action "action_example" "world" {
  config {
    attr = "world"
  }
}
resource "test_object" "a" {
  test_string = "foo"
  lifecycle {
    action_trigger {
      events = [after_create]
      condition = test_object.a.test_string == "foo"
      actions = [action.action_example.hello]
    }
    action_trigger {
      events = [after_update]
      condition = test_object.a.test_string == "bar"
      actions = [action.action_example.world]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("hello"),
					}),
				},
			},
		},
		"multiple events triggering in same action trigger": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [
        before_create, // should trigger
        after_create, // should trigger
        before_update // should be ignored
      ]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
						"attr": cty.String,
					})),
				},
				{
					ActionType: "action_example",
					PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
						"attr": cty.String,
					})),
				},
			},
		},

		"multiple events triggering in multiple action trigger": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {}
resource "test_object" "a" {
  lifecycle {
    // should trigger
    action_trigger {
      events = [before_create]
      actions = [action.action_example.hello]
    }
    // should trigger
    action_trigger {
      events = [after_create]
      actions = [action.action_example.hello]
    }
    // should be ignored
    action_trigger {
      events = [before_update]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
						"attr": cty.String,
					})),
				},
				{
					ActionType: "action_example",
					PlannedActionData: cty.NullVal(cty.Object(map[string]cty.Type{
						"attr": cty.String,
					})),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			m := testModuleInline(t, tc.module)

			testProvider := simpleMockProvider()
			if tc.readResourceFn != nil {
				testProvider.ReadResourceFn = func(r providers.ReadResourceRequest) providers.ReadResourceResponse {
					return tc.readResourceFn(t, r)
				}
			}

			invokeActionCalls := []providers.InvokeActionRequest{}
			invokeActionFn := func(req providers.InvokeActionRequest) providers.InvokeActionResponse {
				invokeActionCalls = append(invokeActionCalls, req)
				if tc.callingInvokeReturnsDiagnostics != nil && len(tc.callingInvokeReturnsDiagnostics(req)) > 0 {
					return providers.InvokeActionResponse{
						Diagnostics: tc.callingInvokeReturnsDiagnostics(req),
					}
				}

				events := []providers.InvokeActionEvent{
					providers.InvokeActionEvent_Progress{Message: "Hello world!"},
					providers.InvokeActionEvent_Completed{},
				}

				if tc.events != nil {
					events = tc.events(req)
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
			}

			actionProvider := testContextActionProvider(invokeActionFn)

			ecosystem := &testing_provider.MockProvider{}
			ecosystem.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
				Actions: map[string]providers.ActionSchema{
					"ecosystem": {
						ConfigSchema: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"attr": {Type: cty.String, Optional: true},
							},
						},
					},
				},
			}
			ecosystem.InvokeActionFn = invokeActionFn

			hookCapture := newActionHookCapture()
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"):                                testProviderFuncFixed(testProvider),
					addrs.NewDefaultProvider("action"):                              testProviderFuncFixed(actionProvider),
					addrs.MustParseProviderSourceString("danielmschmidt/ecosystem"): testProviderFuncFixed(ecosystem),
				},
				Hooks: []Hook{&hookCapture},
			})

			// Just a sanity check that the module is valid
			diags := ctx.Validate(m, &ValidateOpts{})
			tfdiags.AssertNoDiagnostics(t, diags)

			planOpts := SimplePlanOpts(plans.NormalMode, InputValues{})
			if tc.planOpts != nil {
				planOpts = tc.planOpts
			}

			plan, diags := ctx.Plan(m, tc.prevRunState, planOpts)
			if tc.ignoreWarnings {
				tfdiags.AssertNoErrors(t, diags)
			} else {
				tfdiags.AssertNoDiagnostics(t, diags)
			}

			if !plan.Applyable {
				t.Fatalf("plan is not applyable but should be")
			}

			_, diags = ctx.Apply(plan, m, tc.applyOpts)
			if tc.expectDiagnostics != nil {
				tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectDiagnostics(m))
			} else {
				if tc.ignoreWarnings {
					tfdiags.AssertNoErrors(t, diags)
				} else {
					tfdiags.AssertNoDiagnostics(t, diags)
				}
			}

			if tc.expectInvokeActionCalled && len(invokeActionCalls) == 0 {
				t.Fatalf("expected invoke action to be called, but it was not")
			}

			if len(tc.expectInvokeActionCalls) > 0 && len(invokeActionCalls) != len(tc.expectInvokeActionCalls) {
				t.Fatalf("expected %d invoke action calls, got %d (%#v)", len(tc.expectInvokeActionCalls), len(invokeActionCalls), invokeActionCalls)
			}

			for i, expectedCall := range tc.expectInvokeActionCalls {
				if tc.expectInvokeActionCallsAreUnordered {
					// We established the length is correct, so we just need to find one call that matches for each
					found := false
					for _, actualCall := range invokeActionCalls {
						if actualCall.ActionType == expectedCall.ActionType && actualCall.PlannedActionData.RawEquals(expectedCall.PlannedActionData) {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("expected invoke action call with ActionType %s and PlannedActionData %s was not found in actual calls", expectedCall.ActionType, expectedCall.PlannedActionData.GoString())
					}
				} else {
					// Expect correct order
					actualCall := invokeActionCalls[i]

					if actualCall.ActionType != expectedCall.ActionType {
						t.Fatalf("expected invoke action call %d ActionType to be %s, got %s", i, expectedCall.ActionType, actualCall.ActionType)
					}
					if !actualCall.PlannedActionData.RawEquals(expectedCall.PlannedActionData) {
						t.Fatalf("expected invoke action call %d PlannedActionData to be %s, got %s", i, expectedCall.PlannedActionData.GoString(), actualCall.PlannedActionData.GoString())
					}
				}
			}

			if tc.assertHooks != nil {
				tc.assertHooks(t, hookCapture)
			}
		})
	}
}

func TestContextApply_targeted_resource_actions(t *testing.T) {
	tests := map[string]struct {
		module                   map[string]string
		planOpts                 *PlanOpts
		expectInvokeActionCalled bool
		expectInvokeActionCalls  []providers.InvokeActionRequest
		ignoreWarnings           bool
		assertHooks              func(*testing.T, actionHookCapture)
	}{
		"targeted run": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {
  config {
    attr = "hello"
  }
}
action "action_example" "there" {
  config {
    attr = "there"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.action_example.hello]
    }
    action_trigger {
      events  = [after_create]
      actions = [action.action_example.there]
    }
  }
}
action "action_example" "general" {
  config {
    attr = "general"
  }
}
action "action_example" "kenobi" {
  config {
    attr = "kenobi"
  }
}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events  = [before_create, after_update]
      actions = [action.action_example.general]
    }
  }
}
`,
			},
			ignoreWarnings:           true,
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("hello"),
					}),
				},
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("there"),
					}),
				},
			},
			planOpts: &PlanOpts{
				Mode: plans.NormalMode,
				Targets: []addrs.Targetable{
					addrs.RootModuleInstance.Resource(addrs.ManagedResourceMode, "test_object", "a"),
				},
			},
		},

		"targeted run with ancestor that has actions": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {
  config {
    attr = "hello"
  }
}
action "action_example" "there" {
  config {
    attr = "there"
  }
}
resource "test_object" "origin" {
  test_string = "origin"
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
resource "test_object" "a" {
  test_string = test_object.origin.test_string
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.action_example.there]
    }
  }
}
action "action_example" "general" {}
action "action_example" "kenobi" {}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events  = [before_create, after_update]
      actions = [action.action_example.general]
    }
  }
}
`,
			},
			ignoreWarnings:           true,
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("hello"),
					}),
				},
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("there"),
					}),
				},
			},
			planOpts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustAbsResourceAddr("test_object.a")},
			},
		},

		"targeted run with expansion": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {
  count = 3
  config {
    attr = "hello-${count.index}"
  }
}
action "action_example" "there" {
  count = 3
  config {
    attr = "there-${count.index}"
  }
}
resource "test_object" "a" {
  count = 3
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.action_example.hello[count.index]]
    }
    action_trigger {
      events  = [after_create]
      actions = [action.action_example.there[count.index]]
    }
  }
}
action "action_example" "general" {}
action "action_example" "kenobi" {}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events  = [before_create, after_update]
      actions = [action.action_example.general]
    }
  }
}
`,
			},
			ignoreWarnings:           true,
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					// action_example.hello[2] before_create
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("hello-2"),
					}),
				},
				{
					// action_example.there[2] after_create
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("there-2"),
					}),
				},
			},
			planOpts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustResourceInstanceAddr("test_object.a[2]")},
			},
		},

		"targeted run with resource reference": {
			module: map[string]string{
				"main.tf": `
resource "test_object" "source" {
  test_string = "src"
}
action "action_example" "hello" {
  config {
    attr = test_object.source.test_string
  }
}
action "action_example" "there" {
  config {
    attr = "there"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.action_example.hello]
    }
    action_trigger {
      events  = [after_create]
      actions = [action.action_example.there]
    }
  }
}
action "action_example" "general" {}
action "action_example" "kenobi" {}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events  = [before_create, after_update]
      actions = [action.action_example.general]
    }
  }
}
`,
			},
			ignoreWarnings:           true,
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					// action_example.hello before_create with config (attr = test_object.source.test_string -> "src")
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("src"),
					}),
				},
				{
					// action_example.there after_create with static config attr = "there"
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("there"),
					}),
				},
			},
			planOpts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustAbsResourceAddr("test_object.a")},
			},
		},

		"targeted run with condition referencing another resource": {
			module: map[string]string{
				"main.tf": `
resource "test_object" "source" {
  test_string = "source"
}
action "action_example" "hello" {
  config {
    attr = test_object.source.test_string
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events    = [before_create]
      condition = test_object.source.test_string == "source"
      actions   = [action.action_example.hello]
    }
  }
}
`,
			},
			ignoreWarnings:           true,
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					// action_example.hello before_create with config (attr = test_object.source.test_string -> "source")
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("source"),
					}),
				},
			},
			planOpts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustAbsResourceAddr("test_object.a")},
			},
		},

		"targeted run with action referencing another resource that also triggers actions": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {
  config {
    attr = "hello"
  }
}
resource "test_object" "source" {
  test_string = "source"
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
action "action_example" "there" {
  config {
    attr = test_object.source.test_string
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.action_example.there]
    }
  }
}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
			},
			ignoreWarnings:           true,
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					// action_example.hello before_create with static config attr = "hello"
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("hello"),
					}),
				},
				{
					// action_example.there after_create with config attr = source.test_string ("source")
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("source"),
					}),
				},
			},
			planOpts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustAbsResourceAddr("test_object.a")},
			},
		},

		"targeted run triggers resources and actions referenced by not-running actions": {
			module: map[string]string{
				"main.tf": `
action "action_example" "hello" {
  config {
    attr = "hello"
  }
}
resource "test_object" "source" {
    test_string = "source"
    lifecycle {
        action_trigger {
            events = [before_create]
            actions = [action.action_example.hello]
        }
    }
}
action "action_example" "there" {
    config {
        attr = test_object.source.test_string
    }
}
resource "test_object" "a" {
    lifecycle {
        action_trigger {
            events = [after_update]
            actions = [action.action_example.there]
        }
    }
}
resource "test_object" "b" {
    lifecycle {
        action_trigger {
            events = [before_update]
            actions = [action.action_example.hello]
        }
    }
}
		`,
			},
			ignoreWarnings:           true,
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("hello"),
					}),
				},
			},
			planOpts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustAbsResourceAddr("test_object.a")},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			m := testModuleInline(t, tc.module)

			invokeActionCalls := []providers.InvokeActionRequest{}
			invokeActionFn := func(req providers.InvokeActionRequest) providers.InvokeActionResponse {
				invokeActionCalls = append(invokeActionCalls, req)
				events := []providers.InvokeActionEvent{
					providers.InvokeActionEvent_Progress{Message: "Hello world!"},
					providers.InvokeActionEvent_Completed{},
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
			}

			actionProvider := testContextActionProvider(invokeActionFn)
			testProvider := simpleMockProvider()
			ecosystem := &testing_provider.MockProvider{}
			ecosystem.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
				Actions: map[string]providers.ActionSchema{
					"ecosystem": {
						ConfigSchema: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"attr": {Type: cty.String, Optional: true},
							},
						},
					},
				},
			}
			ecosystem.InvokeActionFn = invokeActionFn

			hookCapture := newActionHookCapture()
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"):                                testProviderFuncFixed(testProvider),
					addrs.NewDefaultProvider("action"):                              testProviderFuncFixed(actionProvider),
					addrs.MustParseProviderSourceString("danielmschmidt/ecosystem"): testProviderFuncFixed(ecosystem),
				},
				Hooks: []Hook{&hookCapture},
			})

			// Just a sanity check that the module is valid
			diags := ctx.Validate(m, &ValidateOpts{})
			tfdiags.AssertNoDiagnostics(t, diags)

			planOpts := SimplePlanOpts(plans.NormalMode, InputValues{})
			if tc.planOpts != nil {
				planOpts = tc.planOpts
			}

			plan, diags := ctx.Plan(m, nil, planOpts)
			if tc.ignoreWarnings {
				tfdiags.AssertNoErrors(t, diags)
			} else {
				tfdiags.AssertNoDiagnostics(t, diags)
			}

			if !plan.Applyable {
				t.Fatalf("plan is not applyable but should be")
			}

			_, diags = ctx.Apply(plan, m, nil)
			if tc.ignoreWarnings {
				tfdiags.AssertNoErrors(t, diags)
			} else {
				tfdiags.AssertNoDiagnostics(t, diags)
			}

			if tc.expectInvokeActionCalled && len(invokeActionCalls) == 0 {
				t.Fatalf("expected invoke action to be called, but it was not")
			}

			if len(tc.expectInvokeActionCalls) > 0 && len(invokeActionCalls) != len(tc.expectInvokeActionCalls) {
				t.Fatalf("expected %d invoke action calls, got %d (%#v)", len(tc.expectInvokeActionCalls), len(invokeActionCalls), invokeActionCalls)
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

			if tc.assertHooks != nil {
				tc.assertHooks(t, hookCapture)
			}
		})
	}
}
func TestContextApply_invoked_actions(t *testing.T) {
	tests := map[string]struct {
		module                              map[string]string
		prevRunState                        *states.State
		readResourceFn                      func(*testing.T, providers.ReadResourceRequest) providers.ReadResourceResponse
		planOpts                            *PlanOpts
		expectInvokeActionCalled            bool
		expectInvokeActionCalls             []providers.InvokeActionRequest
		expectInvokeActionCallsAreUnordered bool
	}{
		"simple action invoke": {
			module: map[string]string{
				"main.tf": `
	action "action_example" "one" {
	  config {
	    attr = "one"
	  }
	}
	action "action_example" "two" {
	  config {
	    attr = "two"
	  }
	}
	`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("one"),
					}),
				},
			},
			planOpts: &PlanOpts{
				Mode:          plans.RefreshOnlyMode,
				ActionTargets: []addrs.Targetable{mustActionInstanceAddr("action.action_example.one")},
			},
		},

		"action invoke in module": {
			module: map[string]string{
				"mod/main.tf": `
	action "action_example" "one" {
	  config {
	    attr = "one"
	  }
	}
	action "action_example" "two" {
	  config {
	    attr = "two"
	  }
	}
	`,

				"main.tf": `
	module "mod" {
	  source = "./mod"
	}
	`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("one"),
					}),
				},
			},
			planOpts: &PlanOpts{
				Mode:          plans.RefreshOnlyMode,
				ActionTargets: []addrs.Targetable{mustActionInstanceAddr("module.mod.action.action_example.one")},
			},
		},

		"action invoke in expanded module": {
			module: map[string]string{
				"mod/main.tf": `
	action "action_example" "one" {
	  config {
	    attr = "one"
	  }
	}
	action "action_example" "two" {
	  config {
	    attr = "two"
	  }
	}
	`,

				"main.tf": `
	module "mod" {
	  count = 2
	  source = "./mod"
	}
	`,
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("one"),
					}),
				},
			},
			planOpts: &PlanOpts{
				Mode:          plans.RefreshOnlyMode,
				ActionTargets: []addrs.Targetable{mustActionInstanceAddr("module.mod[1].action.action_example.one")},
			},
		},

		"action invoke with count (all)": {
			module: map[string]string{
				"main.tf": `
	action "action_example" "one" {
	  count = 2

	  config {
	    attr = "${count.index}"
	  }
	}
	action "action_example" "two" {
	  count = 2

	  config {
	    attr = "two"
	  }
	}
	`,
			},
			planOpts: &PlanOpts{
				Mode:          plans.RefreshOnlyMode,
				ActionTargets: []addrs.Targetable{mustActionAddr("action.action_example.one")},
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("0"),
					}),
				},
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("1"),
					}),
				},
			},
			expectInvokeActionCallsAreUnordered: true,
		},

		"action invoke with count (instance)": {
			module: map[string]string{
				"main.tf": `
	action "action_example" "one" {
	  count = 2

	  config {
	    attr = "${count.index}"
	  }
	}
	action "action_example" "two" {
	  count = 2

	  config {
	    attr = "two"
	  }
	}
	`,
			},
			planOpts: &PlanOpts{
				Mode:          plans.RefreshOnlyMode,
				ActionTargets: []addrs.Targetable{mustActionInstanceAddr("action.action_example.one[0]")},
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("0"),
					}),
				},
			},
		},

		"invoke action with reference": {
			module: map[string]string{
				"main.tf": `
	resource "test_object" "a" {
	  test_string = "hello"
	}

	action "action_example" "one" {
	  config {
	    attr = test_object.a.test_string
	  }
	}
	`,
			},
			planOpts: &PlanOpts{
				Mode:          plans.RefreshOnlyMode,
				ActionTargets: []addrs.Targetable{mustActionAddr("action.action_example.one")},
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("hello"),
					}),
				},
			},
			prevRunState: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"), &states.ResourceInstanceObjectSrc{
					AttrsJSON: []byte(`{"test_string":"hello"}`),
					Status:    states.ObjectReady,
				}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
			}),
		},

		"invoke action with reference (drift)": {
			module: map[string]string{
				"main.tf": `
	resource "test_object" "a" {
	  test_string = "hello"
	}

	action "action_example" "one" {
	  config {
	    attr = test_object.a.test_string
	  }
	}
	`,
			},
			planOpts: &PlanOpts{
				Mode:          plans.RefreshOnlyMode,
				ActionTargets: []addrs.Targetable{mustActionAddr("action.action_example.one")},
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("drifted value"),
					}),
				},
			},
			readResourceFn: func(t *testing.T, request providers.ReadResourceRequest) providers.ReadResourceResponse {
				return providers.ReadResourceResponse{
					NewState: cty.ObjectVal(map[string]cty.Value{
						"test_string": cty.StringVal("drifted value"),
					}),
				}
			},
			prevRunState: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"), &states.ResourceInstanceObjectSrc{
					AttrsJSON: []byte(`{"test_string":"hello"}`),
					Status:    states.ObjectReady,
				}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
			}),
		},

		"invoke action with reference (drift, skip refresh)": {
			module: map[string]string{
				"main.tf": `
	resource "test_object" "a" {
	  test_string = "hello"
	}

	action "action_example" "one" {
	  config {
	    attr = test_object.a.test_string
	  }
	}
	`,
			},
			planOpts: &PlanOpts{
				Mode:          plans.RefreshOnlyMode,
				SkipRefresh:   true,
				ActionTargets: []addrs.Targetable{mustActionAddr("action.action_example.one")},
			},
			expectInvokeActionCalled: true,
			expectInvokeActionCalls: []providers.InvokeActionRequest{
				{
					ActionType: "action_example",
					PlannedActionData: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("hello"),
					}),
				},
			},
			readResourceFn: func(t *testing.T, request providers.ReadResourceRequest) providers.ReadResourceResponse {
				return providers.ReadResourceResponse{
					NewState: cty.ObjectVal(map[string]cty.Value{
						"test_string": cty.StringVal("drifted value"),
					}),
				}
			},
			prevRunState: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"), &states.ResourceInstanceObjectSrc{
					AttrsJSON: []byte(`{"test_string":"hello"}`),
					Status:    states.ObjectReady,
				}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
			}),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			m := testModuleInline(t, tc.module)

			testProvider := &testing_provider.MockProvider{}
			testProvider.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
				ResourceTypes: map[string]providers.Schema{
					"test_object": {
						Body: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"test_string": {Type: cty.String, Optional: true},
							},
						},
					},
				},
			}

			if tc.readResourceFn != nil {
				testProvider.ReadResourceFn = func(r providers.ReadResourceRequest) providers.ReadResourceResponse {
					return tc.readResourceFn(t, r)
				}
			}

			invokeActionCalls := []providers.InvokeActionRequest{}
			invokeActionFn := func(req providers.InvokeActionRequest) providers.InvokeActionResponse {
				invokeActionCalls = append(invokeActionCalls, req)
				events := []providers.InvokeActionEvent{
					providers.InvokeActionEvent_Progress{Message: "Hello world!"},
					providers.InvokeActionEvent_Completed{},
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
			}

			actionProvider := &testing_provider.MockProvider{}
			actionProvider.InvokeActionFn = invokeActionFn
			actionProvider.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
				Actions: map[string]providers.ActionSchema{
					"action_example": {
						ConfigSchema: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"attr": {Type: cty.String, Optional: true},
							},
						},
					},
					"action_example_wo": {
						ConfigSchema: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"attr": {Type: cty.String, Optional: true, WriteOnly: true},
							},
						},
					},
					// Added nested action schema with nested blocks
					"action_nested": {
						ConfigSchema: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"top_attr": {Type: cty.String, Optional: true},
							},
							BlockTypes: map[string]*configschema.NestedBlock{
								"settings": {
									Nesting: configschema.NestingSingle,
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"name": {Type: cty.String, Required: true},
										},
										BlockTypes: map[string]*configschema.NestedBlock{
											"rule": {
												Nesting: configschema.NestingList,
												Block: configschema.Block{
													Attributes: map[string]*configschema.Attribute{
														"value": {Type: cty.String, Required: true},
													},
												},
											},
										},
									},
								},
								"settings_list": {
									Nesting: configschema.NestingList,
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"id": {Type: cty.String, Required: true},
										},
									},
								},
							},
						},
					},
				},
			}

			ecosystem := &testing_provider.MockProvider{}
			ecosystem.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
				Actions: map[string]providers.ActionSchema{
					"ecosystem": {
						ConfigSchema: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"attr": {Type: cty.String, Optional: true},
							},
						},
					},
				},
			}
			ecosystem.InvokeActionFn = invokeActionFn

			hookCapture := newActionHookCapture()
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"):                                testProviderFuncFixed(testProvider),
					addrs.NewDefaultProvider("action"):                              testProviderFuncFixed(actionProvider),
					addrs.MustParseProviderSourceString("danielmschmidt/ecosystem"): testProviderFuncFixed(ecosystem),
				},
				Hooks: []Hook{&hookCapture},
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

			if !plan.Applyable {
				t.Fatalf("plan is not applyable but should be")
			}

			_, diags = ctx.Apply(plan, m, nil)
			tfdiags.AssertNoDiagnostics(t, diags)

			if tc.expectInvokeActionCalled && len(invokeActionCalls) == 0 {
				t.Fatalf("expected invoke action to be called, but it was not")
			}

			if len(tc.expectInvokeActionCalls) > 0 && len(invokeActionCalls) != len(tc.expectInvokeActionCalls) {
				t.Fatalf("expected %d invoke action calls, got %d (%#v)", len(tc.expectInvokeActionCalls), len(invokeActionCalls), invokeActionCalls)
			}

			for i, expectedCall := range tc.expectInvokeActionCalls {
				if tc.expectInvokeActionCallsAreUnordered {
					// We established the length is correct, so we just need to find one call that matches for each
					found := false
					for _, actualCall := range invokeActionCalls {
						if actualCall.ActionType == expectedCall.ActionType && actualCall.PlannedActionData.RawEquals(expectedCall.PlannedActionData) {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("expected invoke action call with ActionType %s and PlannedActionData %s was not found in actual calls", expectedCall.ActionType, expectedCall.PlannedActionData.GoString())
					}
				} else {
					// Expect correct order
					actualCall := invokeActionCalls[i]

					if actualCall.ActionType != expectedCall.ActionType {
						t.Fatalf("expected invoke action call %d ActionType to be %s, got %s", i, expectedCall.ActionType, actualCall.ActionType)
					}
					if !actualCall.PlannedActionData.RawEquals(expectedCall.PlannedActionData) {
						t.Fatalf("expected invoke action call %d PlannedActionData to be %s, got %s", i, expectedCall.PlannedActionData.GoString(), actualCall.PlannedActionData.GoString())
					}
				}
			}
		})
	}
}

var _ Hook = (*actionHookCapture)(nil)

type actionHookCapture struct {
	NilHook

	mu                  *sync.Mutex
	startActionHooks    []HookActionIdentity
	completeActionHooks []HookActionIdentity
}

func newActionHookCapture() actionHookCapture {
	return actionHookCapture{
		mu: &sync.Mutex{},
	}
}
func (a *actionHookCapture) StartAction(identity HookActionIdentity) (HookAction, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.startActionHooks = append(a.startActionHooks, identity)
	return HookActionContinue, nil
}

func (a *actionHookCapture) ProgressAction(HookActionIdentity, string) (HookAction, error) {
	return HookActionContinue, nil
}

func (a *actionHookCapture) CompleteAction(identity HookActionIdentity, _ error) (HookAction, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.completeActionHooks = append(a.completeActionHooks, identity)
	return HookActionContinue, nil
}

func TestContextApply_actions_after_trigger_runs_after_expanded_resource(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  each = toset(["one"])
}
action "action_example" "hello" {
  config {
    attr = "hello"
  }
}
resource "test_object" "a" {
  for_each = local.each
  name = each.value
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.action_example.hello]
    }
  }
}
`,
	})

	orderedCalls := []string{}

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
		ApplyResourceChangeFn: func(arcr providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
			time.Sleep(100 * time.Millisecond)
			orderedCalls = append(orderedCalls, fmt.Sprintf("ApplyResourceChangeFn %s", arcr.TypeName))
			return providers.ApplyResourceChangeResponse{
				NewState:    arcr.PlannedState,
				NewIdentity: arcr.PlannedIdentity,
			}
		},
	}

	actionProvider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Actions: map[string]providers.ActionSchema{
				"action_example": {
					ConfigSchema: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"attr": {
								Type:     cty.String,
								Optional: true,
							},
						},
					},
				},
			},
			ResourceTypes: map[string]providers.Schema{},
		},
		InvokeActionFn: func(iar providers.InvokeActionRequest) providers.InvokeActionResponse {
			orderedCalls = append(orderedCalls, fmt.Sprintf("InvokeAction %s", iar.ActionType))
			return providers.InvokeActionResponse{
				Events: func(yield func(providers.InvokeActionEvent) bool) {
					yield(providers.InvokeActionEvent_Completed{})
				},
			}
		},
	}

	hookCapture := newActionHookCapture()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"):   testProviderFuncFixed(testProvider),
			addrs.NewDefaultProvider("action"): testProviderFuncFixed(actionProvider),
		},
		Hooks: []Hook{
			&hookCapture,
		},
	})

	// Just a sanity check that the module is valid
	diags := ctx.Validate(m, &ValidateOpts{})
	tfdiags.AssertNoDiagnostics(t, diags)

	planOpts := SimplePlanOpts(plans.NormalMode, InputValues{})

	plan, diags := ctx.Plan(m, nil, planOpts)
	tfdiags.AssertNoDiagnostics(t, diags)

	if !plan.Applyable {
		t.Fatalf("plan is not applyable but should be")
	}

	_, diags = ctx.Apply(plan, m, nil)
	tfdiags.AssertNoDiagnostics(t, diags)

	expectedOrder := []string{
		"ApplyResourceChangeFn test_object",
		"InvokeAction action_example",
	}

	if diff := cmp.Diff(expectedOrder, orderedCalls); diff != "" {
		t.Fatalf("expected calls in order did not match actual calls (-expected +actual):\n%s", diff)
	}
}

func testContextActionProvider(invokeActionFn func(req providers.InvokeActionRequest) providers.InvokeActionResponse) *testing_provider.MockProvider {
	return &testing_provider.MockProvider{
		InvokeActionFn: invokeActionFn,
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Actions: map[string]providers.ActionSchema{
				"action_example": {
					ConfigSchema: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"attr": {Type: cty.String, Optional: true},
						},
					},
				},
				"action_example_wo": {
					ConfigSchema: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"attr": {Type: cty.String, Optional: true, WriteOnly: true},
						},
					},
				},
				// Added nested action schema with nested blocks
				"action_nested": {
					ConfigSchema: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"top_attr": {Type: cty.String, Optional: true},
						},
						BlockTypes: map[string]*configschema.NestedBlock{
							"settings": {
								Nesting: configschema.NestingSingle,
								Block: configschema.Block{
									Attributes: map[string]*configschema.Attribute{
										"name": {Type: cty.String, Required: true},
									},
									BlockTypes: map[string]*configschema.NestedBlock{
										"rule": {
											Nesting: configschema.NestingList,
											Block: configschema.Block{
												Attributes: map[string]*configschema.Attribute{
													"value": {Type: cty.String, Required: true},
												},
											},
										},
									},
								},
							},
							"settings_list": {
								Nesting: configschema.NestingList,
								Block: configschema.Block{
									Attributes: map[string]*configschema.Attribute{
										"id": {Type: cty.String, Required: true},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
