// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty-debug/ctydebug"
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

func TestContextPlan_actions(t *testing.T) {
	testActionSchema := providers.ActionSchema{
		ConfigSchema: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"attr": {
					Type:     cty.String,
					Optional: true,
				},
			},
		},
	}
	writeOnlyActionSchema := providers.ActionSchema{
		ConfigSchema: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"attr": {
					Type:      cty.String,
					Optional:  true,
					WriteOnly: true,
				},
			},
		},
	}

	// Action schema with nested blocks used for tests exercising block handling.
	nestedActionSchema := providers.ActionSchema{
		ConfigSchema: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"top_attr": {
					Type:     cty.String,
					Optional: true,
				},
			},
			BlockTypes: map[string]*configschema.NestedBlock{
				"settings": {
					Nesting: configschema.NestingSingle,
					Block: configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"name": {
								Type:     cty.String,
								Required: true,
							},
						},
						BlockTypes: map[string]*configschema.NestedBlock{
							"rule": {
								Nesting: configschema.NestingList,
								Block: configschema.Block{
									Attributes: map[string]*configschema.Attribute{
										"value": {
											Type:     cty.String,
											Required: true,
										},
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
							"id": {
								Type:     cty.String,
								Required: true,
							},
						},
					},
				},
			},
		},
	}

	for topic, tcs := range map[string]map[string]struct {
		toBeImplemented bool
		module          map[string]string
		buildState      func(*states.SyncState)
		planActionFn    func(*testing.T, providers.PlanActionRequest) providers.PlanActionResponse
		planResourceFn  func(*testing.T, providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse
		readResourceFn  func(*testing.T, providers.ReadResourceRequest) providers.ReadResourceResponse
		planOpts        *PlanOpts

		expectPlanActionCalled bool

		// Some tests can produce race-conditions in the error messages, so we
		// have two ways of checking the diagnostics. Use expectValidateDiagnostics
		// by default, if there is a race condition and you want to allow multiple
		// versions, please use assertValidateDiagnostics.
		expectValidateDiagnostics func(m *configs.Config) tfdiags.Diagnostics
		assertValidateDiagnostics func(*testing.T, tfdiags.Diagnostics)

		expectPlanDiagnostics func(m *configs.Config) tfdiags.Diagnostics
		assertPlanDiagnostics func(*testing.T, tfdiags.Diagnostics)

		assertPlan func(*testing.T, *plans.Plan)
	}{

		// ======== BASIC ========
		// Fundamental behavior of actions
		// ======== BASIC ========

		"basics": {
			"unreferenced": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
`,
				},
				expectPlanActionCalled: false,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 0 {
						t.Fatalf("expected no actions in plan, got %d", len(p.Changes.ActionInvocations))
					}
					if p.Applyable {
						t.Fatalf("should not be able to apply this plan")
					}
				},
			},
			"query run": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.test_action.hello]
    }
  }
}
`,
					"main.tfquery.hcl": `
list "test_resource" "test1" {
  provider = "test"
	config {
		filter = {
			attr = "foo"
		}
	}
}
`,
				},
				expectPlanActionCalled: false,
				planOpts: &PlanOpts{
					Mode:  plans.NormalMode,
					Query: true,
				},
			},
			"query run, action references resource": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {
  config {
   attr = resource.test_object.a.name
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.test_action.hello]
    }
  }
}
`,
					"main.tfquery.hcl": `
list "test_resource" "test1" {
  provider = "test"
	config {
		filter = {
			attr = "foo"
		}
	}
}
`,
				},
				expectPlanActionCalled: false,
				planOpts: &PlanOpts{
					Mode:  plans.NormalMode,
					Query: true,
				},
			},
			"invalid config": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {
  config {
    unknown_attr = "value"
  }
}`,
				},
				expectPlanActionCalled: false,
				expectValidateDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
					return diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Unsupported argument",
						Detail:   `An argument named "unknown_attr" is not expected here.`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 4, Column: 5, Byte: 47},
							End:      hcl.Pos{Line: 4, Column: 17, Byte: 59},
						},
					})
				},
			},

			"actions can't be accessed in resources": {
				module: map[string]string{
					"main.tf": `
action "test_action" "my_action" {
  config {
    attr = "value"
  }
}
resource "test_object" "a" {
  name = action.test_action.my_action.attr
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.my_action]
    }
  }
}
`,
				},
				expectValidateDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{}.Append(
						&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid reference",
							Detail:   "Actions can not be referenced in this context. They can only be referenced from within a resource's lifecycle actions list.",
							Subject: &hcl.Range{
								Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
								Start:    hcl.Pos{Line: 8, Column: 10, Byte: 110},
								End:      hcl.Pos{Line: 8, Column: 40, Byte: 138},
							},
						})
				},
			},

			"actions can't be accessed in outputs": {
				module: map[string]string{
					"main.tf": `
action "test_action" "my_action" {
  config {
    attr = "value"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.my_action]
    }
  }
}

output "my_output" {
  value = action.test_action.my_action.attr
}

output "my_output2" {
  value = action.test_action.my_action
}
`,
				},
				expectValidateDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{}.Append(
						&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid reference",
							Detail:   "Actions can not be referenced in this context. They can only be referenced from within a resource's lifecycle actions list.",
							Subject: &hcl.Range{
								Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
								Start:    hcl.Pos{Line: 21, Column: 13, Byte: 327},
								End:      hcl.Pos{Line: 21, Column: 43, Byte: 355},
							},
						}).Append(
						&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid reference",
							Detail:   "Actions can not be referenced in this context. They can only be referenced from within a resource's lifecycle actions list.",
							Subject: &hcl.Range{
								Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
								Start:    hcl.Pos{Line: 17, Column: 13, Byte: 258},
								End:      hcl.Pos{Line: 17, Column: 43, Byte: 286},
							},
						},
					)
				},
			},

			"destroy run": {
				module: map[string]string{
					"main.tf": ` 
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
				planOpts:               SimplePlanOpts(plans.DestroyMode, InputValues{}),
			},

			"non-default provider namespace": {
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
resource "other_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.ecosystem.hello]
    }
  }
}
`,
				},

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected 1 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					action := p.Changes.ActionInvocations[0]
					if action.Addr.String() != "action.ecosystem.hello" {
						t.Fatalf("expected action address to be 'action.ecosystem.hello', got '%s'", action.Addr)
					}
					at, ok := action.ActionTrigger.(*plans.LifecycleActionTrigger)
					if !ok {
						t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", action.ActionTrigger)
					}

					if !at.TriggeringResourceAddr.Equal(mustResourceInstanceAddr("other_object.a")) {
						t.Fatalf("expected action to have triggering resource address 'other_object.a', but it is %s", at.TriggeringResourceAddr)
					}

					if action.ProviderAddr.Provider.Namespace != "danielmschmidt" {
						t.Fatalf("expected action to have the namespace 'danielmschmidt', got '%s'", action.ProviderAddr.Provider.Namespace)
					}
				},
			},
		},

		// ======== TRIGGERING ========
		// action_trigger behavior
		// ======== TRIGGERING ========

		"triggering": {
			"before_create triggered": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected 1 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					action := p.Changes.ActionInvocations[0]
					if action.Addr.String() != "action.test_action.hello" {
						t.Fatalf("expected action address to be 'action.test_action.hello', got '%s'", action.Addr)
					}

					at, ok := action.ActionTrigger.(*plans.LifecycleActionTrigger)
					if !ok {
						t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", action.ActionTrigger)
					}

					if !at.TriggeringResourceAddr.Equal(mustResourceInstanceAddr("test_object.a")) {
						t.Fatalf("expected action to have a triggering resource address 'test_object.a', got '%s'", at.TriggeringResourceAddr)
					}

					if at.ActionTriggerBlockIndex != 0 {
						t.Fatalf("expected action to have a triggering block index of 0, got %d", at.ActionTriggerBlockIndex)
					}
					if at.TriggerEvent() != configs.BeforeCreate {
						t.Fatalf("expected action to have a triggering event of 'before_create', got '%s'", at.TriggerEvent())
					}
					if at.ActionsListIndex != 0 {
						t.Fatalf("expected action to have a actions list index of 0, got %d", at.ActionsListIndex)
					}

					if action.ProviderAddr.Provider != addrs.NewDefaultProvider("test") {
						t.Fatalf("expected action to have a provider address of 'provider[\"registry.terraform.io/hashicorp/test\"]', got '%s'", action.ProviderAddr)
					}
				},
			},

			"after_create triggered": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected 1 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					action := p.Changes.ActionInvocations[0]
					if action.Addr.String() != "action.test_action.hello" {
						t.Fatalf("expected action address to be 'action.test_action.hello', got '%s'", action.Addr)
					}

					// TODO: Test that action the triggering resource address is set correctly
				},
			},

			"before_update triggered - on create": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_update]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
			},

			"after_update triggered - on create": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_update]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
			},

			"before_update triggered - on update": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_update]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},

				buildState: func(s *states.SyncState) {
					addr := mustResourceInstanceAddr("test_object.a")
					s.SetResourceInstanceCurrent(addr, &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{"name":"previous_run"}`),
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
				},
				expectPlanActionCalled: true,
			},

			"after_update triggered - on update": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_update]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},

				buildState: func(s *states.SyncState) {
					addr := mustResourceInstanceAddr("test_object.a")
					s.SetResourceInstanceCurrent(addr, &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{"name":"previous_run"}`),
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
				},
				expectPlanActionCalled: true,
			},

			"before_update triggered - on replace": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_update]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},

				buildState: func(s *states.SyncState) {
					addr := mustResourceInstanceAddr("test_object.a")
					s.SetResourceInstanceCurrent(addr, &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{"name":"previous_run"}`),
						Status:    states.ObjectTainted,
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
				},
				expectPlanActionCalled: false,
			},

			"after_update triggered - on replace": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_update]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},

				buildState: func(s *states.SyncState) {
					addr := mustResourceInstanceAddr("test_object.a")
					s.SetResourceInstanceCurrent(addr, &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{"name":"previous_run"}`),
						Status:    states.ObjectTainted,
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
				},
				expectPlanActionCalled: false,
			},

			"failing actions cancel next ones": {
				module: map[string]string{
					"main.tf": `
action "test_action" "failure" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.failure, action.test_action.failure]
    }
    action_trigger {
      events = [before_create]
      actions = [action.test_action.failure]
    }
  }
}
`,
				},

				planActionFn: func(_ *testing.T, _ providers.PlanActionRequest) providers.PlanActionResponse {
					t.Helper()
					return providers.PlanActionResponse{
						Diagnostics: tfdiags.Diagnostics{
							tfdiags.Sourceless(tfdiags.Error, "Planning failed", "Test case simulates an error while planning"),
						},
					}
				},

				expectPlanActionCalled: true,
				// We only expect a single diagnostic here, the other should not have been called because the first one failed.
				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{}.Append(
						&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Failed to plan action",
							Detail:   "Planning failed: Test case simulates an error while planning",
							Subject: &hcl.Range{
								Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
								Start:    hcl.Pos{Line: 7, Column: 8, Byte: 147},
								End:      hcl.Pos{Line: 7, Column: 46, Byte: 173},
							},
						},
					)
				},
			},

			"actions with warnings don't cancel": {
				module: map[string]string{
					"main.tf": `
action "test_action" "failure" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.failure, action.test_action.failure]
    }
    action_trigger {
      events = [before_create]
      actions = [action.test_action.failure]
    }
  }
}
`,
				},

				planActionFn: func(t *testing.T, par providers.PlanActionRequest) providers.PlanActionResponse {
					return providers.PlanActionResponse{
						Diagnostics: tfdiags.Diagnostics{
							tfdiags.Sourceless(tfdiags.Warning, "Warning during planning", "Test case simulates a warning while planning"),
						},
					}
				},

				expectPlanActionCalled: true,
				// We only expect a single diagnostic here, the other should not have been called because the first one failed.
				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{}.Append(
						&hcl.Diagnostic{
							Severity: hcl.DiagWarning,
							Summary:  "Warnings when planning action",
							Detail:   "Warning during planning: Test case simulates a warning while planning",
							Subject: &hcl.Range{
								Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
								Start:    hcl.Pos{Line: 7, Column: 8, Byte: 147},
								End:      hcl.Pos{Line: 7, Column: 46, Byte: 173},
							},
						},
						&hcl.Diagnostic{
							Severity: hcl.DiagWarning,
							Summary:  "Warnings when planning action",
							Detail:   "Warning during planning: Test case simulates a warning while planning",
							Subject: &hcl.Range{
								Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
								Start:    hcl.Pos{Line: 7, Column: 48, Byte: 175},
								End:      hcl.Pos{Line: 7, Column: 76, Byte: 201},
							},
						},
						&hcl.Diagnostic{
							Severity: hcl.DiagWarning,
							Summary:  "Warnings when planning action",
							Detail:   "Warning during planning: Test case simulates a warning while planning",
							Subject: &hcl.Range{
								Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
								Start:    hcl.Pos{Line: 11, Column: 8, Byte: 278},
								End:      hcl.Pos{Line: 11, Column: 46, Byte: 304},
							},
						},
					)
				},
			},
			"splat is not supported": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {
  count = 42
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello[*]]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid action expression",
						Detail:   "Unexpected expression found in action_triggers.actions.",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 9, Column: 18, Byte: 159},
							End:      hcl.Pos{Line: 9, Column: 47, Byte: 186},
						},
					})
				},
			},
			"multiple events triggering in same action trigger": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [
        before_create, // should trigger
        after_create, // should trigger
        before_update // should be ignored
      ]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected 2 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					triggeredEvents := []configs.ActionTriggerEvent{}
					for _, action := range p.Changes.ActionInvocations {
						at, ok := action.ActionTrigger.(*plans.LifecycleActionTrigger)
						if !ok {
							t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", action.ActionTrigger)
						}
						triggeredEvents = append(triggeredEvents, at.ActionTriggerEvent)
					}
					slices.Sort(triggeredEvents)
					if diff := cmp.Diff([]configs.ActionTriggerEvent{configs.BeforeCreate, configs.AfterCreate}, triggeredEvents); diff != "" {
						t.Errorf("wrong result\n%s", diff)
					}
				},
			},

			"multiple events triggered together": {
				module: map[string]string{
					"main.tf": `
action "test_action" "one" {}
action "test_action" "two" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events  = [before_create, after_create, before_update, after_update]
      actions = [action.test_action.one, action.test_action.two]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
			},

			"multiple events triggering in multiple action trigger": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    // should trigger
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
    // should trigger
    action_trigger {
      events = [after_create]
      actions = [action.test_action.hello]
    }
    // should be ignored
    action_trigger {
      events = [before_update]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected 2 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					triggeredEvents := []configs.ActionTriggerEvent{}
					for _, action := range p.Changes.ActionInvocations {
						at, ok := action.ActionTrigger.(*plans.LifecycleActionTrigger)
						if !ok {
							t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", action.ActionTrigger)
						}
						triggeredEvents = append(triggeredEvents, at.ActionTriggerEvent)
					}
					slices.Sort(triggeredEvents)
					if diff := cmp.Diff([]configs.ActionTriggerEvent{configs.BeforeCreate, configs.AfterCreate}, triggeredEvents); diff != "" {
						t.Errorf("wrong result\n%s", diff)
					}
				},
			},
		},

		// ======== EXPANSION ========
		// action expansion behavior (count & for_each)
		// ======== EXPANSION ========

		"expansion": {
			"action for_each": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {
  for_each = toset(["a", "b"])
  
  config {
    attr = "value-${each.key}"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello["a"], action.test_action.hello["b"]]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected 2 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					actionAddrs := []string{}
					for _, action := range p.Changes.ActionInvocations {
						actionAddrs = append(actionAddrs, action.Addr.String())
					}
					slices.Sort(actionAddrs)

					if !slices.Equal(actionAddrs, []string{
						"action.test_action.hello[\"a\"]",
						"action.test_action.hello[\"b\"]",
					}) {
						t.Fatalf("expected action addresses to be 'action.test_action.hello[\"a\"]' and 'action.test_action.hello[\"b\"]', got %v", actionAddrs)
					}

					for _, ai := range p.Changes.ActionInvocations {
						at, ok := ai.ActionTrigger.(*plans.LifecycleActionTrigger)
						if !ok {
							t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", ai.ActionTrigger)
						}

						if !at.TriggeringResourceAddr.Equal(mustResourceInstanceAddr("test_object.a")) {
							t.Fatalf("expected action to have triggering resource address 'test_object.a', but it is %s", at.TriggeringResourceAddr)
						}
					}
				},
			},

			"action count": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {
  count = 2

  config {
    attr = "value-${count.index}"
  }
}

resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello[0], action.test_action.hello[1]]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected 2 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					actionAddrs := []string{}
					for _, action := range p.Changes.ActionInvocations {
						actionAddrs = append(actionAddrs, action.Addr.String())
					}
					slices.Sort(actionAddrs)

					if !slices.Equal(actionAddrs, []string{
						"action.test_action.hello[0]",
						"action.test_action.hello[1]",
					}) {
						t.Fatalf("expected action addresses to be 'action.test_action.hello[0]' and 'action.test_action.hello[1]', got %v", actionAddrs)
					}

					for _, ai := range p.Changes.ActionInvocations {
						at, ok := ai.ActionTrigger.(*plans.LifecycleActionTrigger)
						if !ok {
							t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", ai.ActionTrigger)
						}

						if !at.TriggeringResourceAddr.Equal(mustResourceInstanceAddr("test_object.a")) {
							t.Fatalf("expected action to have triggering resource address 'test_object.a', but it is %s", at.TriggeringResourceAddr)
						}
					}
				},
			},

			"action for_each invalid access": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {
  for_each = toset(["a", "b"])

  config {
    attr = "value-${each.key}"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello["c"]]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
				expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
					return diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to non-existent action instance",
						Detail:   "Action instance was not found in the current context.",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 13, Column: 18, Byte: 224},
							End:      hcl.Pos{Line: 13, Column: 49, Byte: 253},
						},
					})
				},
			},

			"action count invalid access": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {
  count = 2

  config {
    attr = "value-${count.index}"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello[2]]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
				expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
					return diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to non-existent action instance",
						Detail:   "Action instance was not found in the current context.",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 13, Column: 18, Byte: 208},
							End:      hcl.Pos{Line: 13, Column: 47, Byte: 235},
						},
					})
				},
			},

			"expanded resource - unexpanded action": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  count = 2
  name = "test-${count.index}"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected 2 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					actionAddrs := []string{}
					for _, action := range p.Changes.ActionInvocations {
						actionAddrs = append(actionAddrs, action.Addr.String())
					}
					slices.Sort(actionAddrs)

					if !slices.Equal(actionAddrs, []string{
						"action.test_action.hello",
						"action.test_action.hello",
					}) {
						t.Fatalf("expected action addresses to be 'action.test_action.hello' and 'action.test_action.hello', got %v", actionAddrs)
					}

					actionTriggers := []plans.LifecycleActionTrigger{}
					for _, ai := range p.Changes.ActionInvocations {
						at, ok := ai.ActionTrigger.(*plans.LifecycleActionTrigger)
						if !ok {
							t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", ai.ActionTrigger)
						}

						actionTriggers = append(actionTriggers, *at)
					}

					if !actionTriggers[0].TriggeringResourceAddr.Resource.Resource.Equal(actionTriggers[1].TriggeringResourceAddr.Resource.Resource) {
						t.Fatalf("expected both actions to have the same triggering resource address, but got %s and %s", actionTriggers[0].TriggeringResourceAddr, actionTriggers[1].TriggeringResourceAddr)
					}

					if actionTriggers[0].TriggeringResourceAddr.Resource.Key == actionTriggers[1].TriggeringResourceAddr.Resource.Key {
						t.Fatalf("expected both actions to have different triggering resource instance keys, but got the same %s", actionTriggers[0].TriggeringResourceAddr.Resource.Key)
					}
				},
			},
			"expanded resource - expanded action": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {
  count = 2

  config {
    attr = "value-${count.index}"
  }
}
resource "test_object" "a" {
  count = 2
  name = "test-${count.index}"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello[count.index]]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected 2 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					actionAddrs := []string{}
					for _, action := range p.Changes.ActionInvocations {
						actionAddrs = append(actionAddrs, action.Addr.String())
					}
					slices.Sort(actionAddrs)

					if !slices.Equal(actionAddrs, []string{
						"action.test_action.hello[0]",
						"action.test_action.hello[1]",
					}) {
						t.Fatalf("expected action addresses to be 'action.test_action.hello[0]' and 'action.test_action.hello[1]', got %v", actionAddrs)
					}

					actionTriggers := []plans.LifecycleActionTrigger{}
					for _, ai := range p.Changes.ActionInvocations {
						at, ok := ai.ActionTrigger.(*plans.LifecycleActionTrigger)
						if !ok {
							t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", ai.ActionTrigger)
						}

						actionTriggers = append(actionTriggers, *at)
					}

					if !actionTriggers[0].TriggeringResourceAddr.Resource.Resource.Equal(actionTriggers[1].TriggeringResourceAddr.Resource.Resource) {
						t.Fatalf("expected both actions to have the same triggering resource address, but got %s and %s", actionTriggers[0].TriggeringResourceAddr, actionTriggers[1].TriggeringResourceAddr)
					}

					if actionTriggers[0].TriggeringResourceAddr.Resource.Key == actionTriggers[1].TriggeringResourceAddr.Resource.Key {
						t.Fatalf("expected both actions to have different triggering resource instance keys, but got the same %s", actionTriggers[0].TriggeringResourceAddr.Resource.Key)
					}
				},
			},

			// Since if we just destroy a node there is no reference to an action in config, we try
			// to provoke an error by just removing a resource instance.
			"destroying expanded node": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  count = 2
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,

				buildState: func(s *states.SyncState) {
					s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a[0]"), &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{}`),
						Status:    states.ObjectReady,
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))

					s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a[1]"), &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{}`),
						Status:    states.ObjectReady,
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))

					s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a[2]"), &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{}`),
						Status:    states.ObjectReady,
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
				},
			},
		},

		// ======== CONFIG ========
		// action config behavior (secrets, write_only, dependencies)
		// ======== CONFIG ========

		"config": {
			"transitive dependencies": {
				module: map[string]string{
					"main.tf": `
resource "test_object" "a" {
  name = "a"
}
action "test_action" "hello" {
  config {
    attr = test_object.a.name
  }
}
resource "test_object" "b" {
  name = "b"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
			},

			"expanded transitive dependencies": {
				module: map[string]string{
					"main.tf": `
resource "test_object" "a" {
  name = "a"
}
resource "test_object" "b" {
  name = "b"
}
action "test_action" "hello_a" {
  config {
    attr = test_object.a.name
  }
}
action "test_action" "hello_b" {
  config {
    attr = test_object.a.name
  }
}
resource "test_object" "c" {
  name = "c"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello_a]
    }
  }
}
resource "test_object" "d" {
  name = "d"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello_b]
    }
  }
}
resource "test_object" "e" {
  name = "e"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello_a, action.test_action.hello_b]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
			},

			"action config with after_create dependency to triggering resource": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {
  config {
    attr = test_object.a.name
  }
}
resource "test_object" "a" {
  name = "test_name"
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected one action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					if p.Changes.ActionInvocations[0].ActionTrigger.TriggerEvent() != configs.AfterCreate {
						t.Fatalf("expected trigger event to be of type AfterCreate, got: %v", p.Changes.ActionInvocations[0].ActionTrigger)
					}

					if p.Changes.ActionInvocations[0].Addr.Action.String() != "action.test_action.hello" {
						t.Fatalf("expected action to equal 'action.test_action.hello', got '%s'", p.Changes.ActionInvocations[0].Addr)
					}

					decode, err := p.Changes.ActionInvocations[0].ConfigValue.Decode(cty.Object(map[string]cty.Type{"attr": cty.String}))
					if err != nil {
						t.Fatal(err)
					}

					if decode.GetAttr("attr").AsString() != "test_name" {
						t.Fatalf("expected action config field 'attr' to have value 'test_name', got '%s'", decode.GetAttr("attr").AsString())
					}
				},
			},

			"action config refers to before triggering resource leads to validation error": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {
  config {
    attr = test_object.a.name
  }
}
resource "test_object" "a" {
  name = "test_name"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true, // The cycle only appears in the apply graph
				assertPlanDiagnostics: func(t *testing.T, diags tfdiags.Diagnostics) {
					if !diags.HasErrors() {
						t.Fatalf("expected diagnostics to have errors, but it does not")
					}
					if len(diags) != 1 {
						t.Fatalf("expected diagnostics to have 1 error, but it has %d", len(diags))
					}
					// We expect the diagnostic to be about a cycle
					if !strings.Contains(diags[0].Description().Summary, "Cycle") {
						t.Fatalf("expected diagnostic summary to contain 'Cycle', got '%s'", diags[0].Description().Summary)
					}
					// We expect the action node to be part of the cycle
					if !strings.Contains(diags[0].Description().Summary, "action.test_action.hello") {
						t.Fatalf("expected diagnostic summary to contain 'action.test_action.hello', got '%s'", diags[0].Description().Summary)
					}
					// We expect the resource node to be part of the cycle
					if !strings.Contains(diags[0].Description().Summary, "test_object.a") {
						t.Fatalf("expected diagnostic summary to contain 'test_object.a', got '%s'", diags[0].Description().Summary)
					}
				},
			},

			"secret values": {
				module: map[string]string{
					"main.tf": `
variable "secret" {
  type           = string
  sensitive      = true
}
action "test_action" "hello" {
  config {
    attr = var.secret
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				planOpts: &PlanOpts{
					Mode: plans.NormalMode,
					SetVariables: InputValues{
						"secret": &InputValue{
							Value:      cty.StringVal("secret"),
							SourceType: ValueFromCLIArg,
						}},
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected 1 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					action := p.Changes.ActionInvocations[0]
					ac, err := action.Decode(&testActionSchema)
					if err != nil {
						t.Fatalf("expected action to decode successfully, but got error: %v", err)
					}

					if !marks.Has(ac.ConfigValue.GetAttr("attr"), marks.Sensitive) {
						t.Fatalf("expected attribute 'attr' to be marked as sensitive")
					}
				},
			},

			"ephemeral values": {
				module: map[string]string{
					"main.tf": `
variable "secret" {
  type           = string
  ephemeral      = true
}
action "test_action" "hello" {
  config {
    attr = var.secret
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				planOpts: &PlanOpts{
					Mode: plans.NormalMode,
					SetVariables: InputValues{
						"secret": &InputValue{
							Value:      cty.StringVal("secret"),
							SourceType: ValueFromCLIArg,
						}},
				},
				expectPlanActionCalled: false,
				assertValidateDiagnostics: func(t *testing.T, diags tfdiags.Diagnostics) {
					if len(diags) != 1 {
						t.Fatalf("expected exactly 1 diagnostic but had %d", len(diags))
					}

					if diags[0].Severity() != tfdiags.Error {
						t.Error("expected error diagnostic")
					}

					if diags[0].Description().Summary != "Invalid use of ephemeral value" {
						t.Errorf("expected diagnostics to be because of ephemeral values but was %s", diags[0].Description().Summary)
					}
				},
			},

			"write-only attributes": {
				module: map[string]string{
					"main.tf": `
variable "attr" {
  type = string
  ephemeral = true
}

resource "test_object" "resource" {
  name = "hello"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action_wo.hello]
    }
  }
}

action "test_action_wo" "hello" {
  config {
    attr = var.attr
  }
}
`,
				},
				planOpts: SimplePlanOpts(plans.NormalMode, InputValues{
					"attr": {
						Value: cty.StringVal("wo-plan"),
					},
				}),
				expectPlanActionCalled: true,
				assertPlan: func(t *testing.T, plan *plans.Plan) {
					if len(plan.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected exactly one invocation, and found %d", len(plan.Changes.ActionInvocations))
					}

					ais := plan.Changes.ActionInvocations[0]
					ai, err := ais.Decode(&writeOnlyActionSchema)
					if err != nil {
						t.Fatal(err)
					}

					if !ai.ConfigValue.GetAttr("attr").IsNull() {
						t.Fatal("should have converted ephemeral value to null in the plan")
					}
				},
			},

			"action config nested single + list blocks": {
				module: map[string]string{
					"main.tf": `
action "test_nested" "with_blocks" {
  config {
    top_attr = "top"
    settings {
      name = "primary"
      rule {
        value = "r1"
      }
      rule {
        value = "r2"
      }
    }
  }
}
resource "test_object" "a" {
  name = "object"
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.test_nested.with_blocks]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected 1 action invocation, got %d", len(p.Changes.ActionInvocations))
					}
					ais := p.Changes.ActionInvocations[0]
					decoded, err := ais.Decode(&nestedActionSchema)
					if err != nil {
						t.Fatalf("error decoding nested action: %s", err)
					}
					cv := decoded.ConfigValue
					if cv.GetAttr("top_attr").AsString() != "top" {
						t.Fatalf("expected top_attr = top, got %s", cv.GetAttr("top_attr").GoString())
					}
					settings := cv.GetAttr("settings")
					if !settings.Type().IsObjectType() {
						t.Fatalf("expected settings object, got %s", settings.Type().FriendlyName())
					}
					if settings.GetAttr("name").AsString() != "primary" {
						t.Fatalf("expected settings.name = primary, got %s", settings.GetAttr("name").GoString())
					}
					rules := settings.GetAttr("rule")
					if !rules.Type().IsListType() || rules.LengthInt() != 2 {
						t.Fatalf("expected 2 rule blocks, got type %s length %d", rules.Type().FriendlyName(), rules.LengthInt())
					}
					first := rules.Index(cty.NumberIntVal(0)).GetAttr("value").AsString()
					second := rules.Index(cty.NumberIntVal(1)).GetAttr("value").AsString()
					if first != "r1" || second != "r2" {
						t.Fatalf("expected rule values r1,r2 got %s,%s", first, second)
					}
				},
			},

			"action config top-level list block": {
				module: map[string]string{
					"main.tf": `
action "test_nested" "with_list" {
  config {
    settings_list {
      id = "one"
    }
    settings_list {
      id = "two"
    }
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.test_nested.with_list]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected 1 action invocation, got %d", len(p.Changes.ActionInvocations))
					}
					ais := p.Changes.ActionInvocations[0]
					decoded, err := ais.Decode(&nestedActionSchema)
					if err != nil {
						t.Fatalf("error decoding nested action: %s", err)
					}
					cv := decoded.ConfigValue
					if !cv.GetAttr("top_attr").IsNull() {
						t.Fatalf("expected top_attr to be null, got %s", cv.GetAttr("top_attr").GoString())
					}
					sl := cv.GetAttr("settings_list")
					if !sl.Type().IsListType() || sl.LengthInt() != 2 {
						t.Fatalf("expected 2 settings_list blocks, got type %s length %d", sl.Type().FriendlyName(), sl.LengthInt())
					}
					first := sl.Index(cty.NumberIntVal(0)).GetAttr("id").AsString()
					second := sl.Index(cty.NumberIntVal(1)).GetAttr("id").AsString()
					if first != "one" || second != "two" {
						t.Fatalf("expected ids one,two got %s,%s", first, second)
					}
				},
			},
		},

		// ======== MODULES ========
		// actions within modules
		// ======== MODULES ========

		"modules": {
			"triggered within module": {
				module: map[string]string{
					"main.tf": `
module "mod" {
    source = "./mod"
}
`,
					"mod/mod.tf": `
action "test_action" "hello" {}
resource "other_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected 1 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					action := p.Changes.ActionInvocations[0]
					if action.Addr.String() != "module.mod.action.test_action.hello" {
						t.Fatalf("expected action address to be 'module.mod.action.test_action.hello', got '%s'", action.Addr)
					}

					at, ok := action.ActionTrigger.(*plans.LifecycleActionTrigger)
					if !ok {
						t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", action.ActionTrigger)
					}

					if !at.TriggeringResourceAddr.Equal(mustResourceInstanceAddr("module.mod.other_object.a")) {
						t.Fatalf("expected action to have triggering resource address 'module.mod.other_object.a', but it is %s", at.TriggeringResourceAddr)
					}

					if at.ActionTriggerBlockIndex != 0 {
						t.Fatalf("expected action to have a triggering block index of 0, got %d", at.ActionTriggerBlockIndex)
					}
					if at.TriggerEvent() != configs.BeforeCreate {
						t.Fatalf("expected action to have a triggering event of 'before_create', got '%s'", at.TriggerEvent())
					}
					if at.ActionsListIndex != 0 {
						t.Fatalf("expected action to have a actions list index of 0, got %d", at.ActionsListIndex)
					}

					if action.ProviderAddr.Provider != addrs.NewDefaultProvider("test") {
						t.Fatalf("expected action to have a provider address of 'provider[\"registry.terraform.io/hashicorp/test\"]', got '%s'", action.ProviderAddr)
					}
				},
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
action "test_action" "hello" {}
resource "other_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected 1 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					// We know we are run within two child modules, so we can just sort by the triggering resource address
					slices.SortFunc(p.Changes.ActionInvocations, func(a, b *plans.ActionInvocationInstanceSrc) int {
						at, ok := a.ActionTrigger.(*plans.LifecycleActionTrigger)
						if !ok {
							t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", a.ActionTrigger)
						}
						bt, ok := b.ActionTrigger.(*plans.LifecycleActionTrigger)
						if !ok {
							t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", b.ActionTrigger)
						}
						if at.TriggeringResourceAddr.String() < bt.TriggeringResourceAddr.String() {
							return -1
						} else {
							return 1
						}
					})

					action := p.Changes.ActionInvocations[0]
					if action.Addr.String() != "module.mod[0].action.test_action.hello" {
						t.Fatalf("expected action address to be 'module.mod[0].action.test_action.hello', got '%s'", action.Addr)
					}

					at := action.ActionTrigger.(*plans.LifecycleActionTrigger)

					if !at.TriggeringResourceAddr.Equal(mustResourceInstanceAddr("module.mod[0].other_object.a")) {
						t.Fatalf("expected action to have triggering resource address 'module.mod[0].other_object.a', but it is %s", at.TriggeringResourceAddr)
					}

					if at.ActionTriggerBlockIndex != 0 {
						t.Fatalf("expected action to have a triggering block index of 0, got %d", at.ActionTriggerBlockIndex)
					}
					if at.TriggerEvent() != configs.BeforeCreate {
						t.Fatalf("expected action to have a triggering event of 'before_create', got '%s'", at.TriggerEvent())
					}
					if at.ActionsListIndex != 0 {
						t.Fatalf("expected action to have a actions list index of 0, got %d", at.ActionsListIndex)
					}

					if action.ProviderAddr.Provider != addrs.NewDefaultProvider("test") {
						t.Fatalf("expected action to have a provider address of 'provider[\"registry.terraform.io/hashicorp/test\"]', got '%s'", action.ProviderAddr)
					}

					action2 := p.Changes.ActionInvocations[1]
					if action2.Addr.String() != "module.mod[1].action.test_action.hello" {
						t.Fatalf("expected action address to be 'module.mod[1].action.test_action.hello', got '%s'", action2.Addr)
					}

					a2t := action2.ActionTrigger.(*plans.LifecycleActionTrigger)

					if !a2t.TriggeringResourceAddr.Equal(mustResourceInstanceAddr("module.mod[1].other_object.a")) {
						t.Fatalf("expected action to have triggering resource address 'module.mod[1].other_object.a', but it is %s", a2t.TriggeringResourceAddr)
					}
				},
			},

			"not triggered if module is count=0": {
				module: map[string]string{
					"main.tf": `
module "mod" {
    count = 0
    source = "./mod"
}
`,
					"mod/mod.tf": `
action "test_action" "hello" {}
resource "other_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
			},

			"not triggered if for_each is empty": {
				module: map[string]string{
					"main.tf": `
module "mod" {
    for_each = toset([])
    source = "./mod"
}
`,
					"mod/mod.tf": `
action "test_action" "hello" {}
resource "other_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
			},

			"action declaration in module if module is count=0": {
				module: map[string]string{
					"main.tf": `
module "mod" {
    count = 0
    source = "./mod"
}
`,
					"mod/mod.tf": `
action "test_action" "hello" {}
`,
				},
				expectPlanActionCalled: false,
			},

			"action declaration in module if for_each is empty": {
				module: map[string]string{
					"main.tf": `
module "mod" {
    for_each = toset([])
    source = "./mod"
}
`,
					"mod/mod.tf": `
action "test_action" "hello" {}
`,
				},
				expectPlanActionCalled: false,
			},

			"provider is within module": {
				module: map[string]string{
					"main.tf": `
module "mod" {
    source = "./mod"
}
`,
					"mod/mod.tf": `
provider "test" {
    alias = "inthemodule"
}
action "test_action" "hello" {
  provider = test.inthemodule
}
resource "other_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected 1 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					action := p.Changes.ActionInvocations[0]
					if action.Addr.String() != "module.mod.action.test_action.hello" {
						t.Fatalf("expected action address to be 'module.mod.action.test_action.hello', got '%s'", action.Addr)
					}

					at, ok := action.ActionTrigger.(*plans.LifecycleActionTrigger)
					if !ok {
						t.Fatalf("expected action trigger to be a lifecycle action trigger, got %T", action.ActionTrigger)
					}

					if !at.TriggeringResourceAddr.Equal(mustResourceInstanceAddr("module.mod.other_object.a")) {
						t.Fatalf("expected action to have triggering resource address 'module.mod.other_object.a', but it is %s", at.TriggeringResourceAddr)
					}

					if action.ProviderAddr.Module.String() != "module.mod" {
						t.Fatalf("expected action to have a provider module address of 'module.mod' got '%s'", action.ProviderAddr.Module.String())
					}
					if action.ProviderAddr.Alias != "inthemodule" {
						t.Fatalf("expected action to have a provider alias of 'inthemodule', got '%s'", action.ProviderAddr.Alias)
					}
				},
			},
		},

		// ======== PROVIDER ========
		// provider meta-argument
		// ======== PROVIDER ========

		"provider": {
			"aliased provider": {
				module: map[string]string{
					"main.tf": `
provider "test" {
  alias = "aliased"
}
action "test_action" "hello" {
  provider = test.aliased
}
resource "other_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected 1 action in plan, got %d", len(p.Changes.ActionInvocations))
					}

					action := p.Changes.ActionInvocations[0]
					if action.Addr.String() != "action.test_action.hello" {
						t.Fatalf("expected action address to be 'action.test_action.hello', got '%s'", action.Addr)
					}

					at, ok := action.ActionTrigger.(*plans.LifecycleActionTrigger)
					if !ok {
						t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", action.ActionTrigger)
					}

					if !at.TriggeringResourceAddr.Equal(mustResourceInstanceAddr("other_object.a")) {
						t.Fatalf("expected action to have triggering resource address 'other_object.a', but it is %s", at.TriggeringResourceAddr)
					}

					if action.ProviderAddr.Alias != "aliased" {
						t.Fatalf("expected action to have a provider alias of 'aliased', got '%s'", action.ProviderAddr.Alias)
					}
				},
			},
		},

		// ======== DEFERRING ========
		// Deferred actions (partial expansion / provider deferring)
		// ======== DEFERRING ========
		"deferring": {
			"provider deferring action while not allowed": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode:            plans.NormalMode,
					DeferralAllowed: false,
				},
				planActionFn: func(*testing.T, providers.PlanActionRequest) providers.PlanActionResponse {
					return providers.PlanActionResponse{
						Deferred: &providers.Deferred{
							Reason: providers.DeferredReasonAbsentPrereq,
						},
					}
				},
				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{
						tfdiags.Sourceless(
							tfdiags.Error,
							"Provider deferred changes when Terraform did not allow deferrals",
							`The provider signaled a deferred action for "action.test_action.hello", but in this context deferrals are disabled. This is a bug in the provider, please file an issue with the provider developers.`,
						),
					}
				},
			},

			"provider deferring action": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode:            plans.NormalMode,
					DeferralAllowed: true, // actions should ignore this setting
				},
				planActionFn: func(*testing.T, providers.PlanActionRequest) providers.PlanActionResponse {
					return providers.PlanActionResponse{
						Deferred: &providers.Deferred{
							Reason: providers.DeferredReasonAbsentPrereq,
						},
					}
				},
				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{
						tfdiags.Sourceless(
							tfdiags.Error,
							"Provider deferred changes when Terraform did not allow deferrals",
							`The provider signaled a deferred action for "action.test_action.hello", but in this context deferrals are disabled. This is a bug in the provider, please file an issue with the provider developers.`,
						),
					}
				},
			},

			"deferred after actions defer following actions": {
				module: map[string]string{
					"main.tf": `
// Using this provider to have another provider type for an easier assertion
terraform {
  required_providers {
    ecosystem = {
      source = "danielmschmidt/ecosystem"
    }
  }
}
action "test_action" "hello" {}
action "ecosystem" "world" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.test_action.hello, action.ecosystem.world]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode:            plans.NormalMode,
					DeferralAllowed: true,
				},
				planActionFn: func(t *testing.T, r providers.PlanActionRequest) providers.PlanActionResponse {
					if r.ActionType == "ecosystem" {
						t.Fatalf("expected second action to not be planned, but it was planned")
					}
					return providers.PlanActionResponse{
						Deferred: &providers.Deferred{
							Reason: providers.DeferredReasonAbsentPrereq,
						},
					}
				},
				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					// for now, it's just an error for any deferrals but when
					// this gets implemented we should check that all the
					// actions are deferred even though only one of them
					// was actually marked as deferred.
					return tfdiags.Diagnostics{
						tfdiags.Sourceless(
							tfdiags.Error,
							"Provider deferred changes when Terraform did not allow deferrals",
							`The provider signaled a deferred action for "action.test_action.hello", but in this context deferrals are disabled. This is a bug in the provider, please file an issue with the provider developers.`,
						),
					}
				},
			},

			"deferred before actions defer following actions and resource": {
				module: map[string]string{
					"main.tf": `
// Using this provider to have another provider type for an easier assertion
terraform {
  required_providers {
    ecosystem = {
      source = "danielmschmidt/ecosystem"
    }
  }
}
action "test_action" "hello" {}
action "ecosystem" "world" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
    action_trigger {
      events = [after_create]
      actions = [action.ecosystem.world]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode:            plans.NormalMode,
					DeferralAllowed: true,
				},
				planActionFn: func(t *testing.T, r providers.PlanActionRequest) providers.PlanActionResponse {
					if r.ActionType == "ecosystem" {
						t.Fatalf("expected second action to not be planned, but it was planned")
					}
					return providers.PlanActionResponse{
						Deferred: &providers.Deferred{
							Reason: providers.DeferredReasonAbsentPrereq,
						},
					}
				},
				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					// for now, it's just an error for any deferrals but when
					// this gets implemented we should check that all the
					// actions are deferred even though only one of them
					// was actually marked as deferred.
					return tfdiags.Diagnostics{
						tfdiags.Sourceless(
							tfdiags.Error,
							"Provider deferred changes when Terraform did not allow deferrals",
							`The provider signaled a deferred action for "action.test_action.hello", but in this context deferrals are disabled. This is a bug in the provider, please file an issue with the provider developers.`,
						),
					}
				},
			},

			"deferred resources also defer the actions they trigger": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
    action_trigger {
      events = [after_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
				planOpts: &PlanOpts{
					Mode:            plans.NormalMode,
					DeferralAllowed: true,
				},

				planResourceFn: func(_ *testing.T, req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
					return providers.PlanResourceChangeResponse{
						PlannedState:   req.ProposedNewState,
						PlannedPrivate: req.PriorPrivate,
						Diagnostics:    nil,
						Deferred: &providers.Deferred{
							Reason: providers.DeferredReasonAbsentPrereq,
						},
					}
				},

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 0 {
						t.Fatalf("expected 0 actions in plan, got %d", len(p.Changes.ActionInvocations))
					}

					if len(p.DeferredActionInvocations) != 2 {
						t.Fatalf("expected 2 deferred actions in plan, got %d", len(p.DeferredActionInvocations))
					}

					sort.Slice(p.DeferredActionInvocations, func(i, j int) bool {
						return p.DeferredActionInvocations[i].ActionInvocationInstanceSrc.Addr.String() < p.DeferredActionInvocations[j].ActionInvocationInstanceSrc.Addr.String()
					})

					firstDeferredActionInvocation := p.DeferredActionInvocations[0]
					if firstDeferredActionInvocation.DeferredReason != providers.DeferredReasonDeferredPrereq {
						t.Fatalf("expected deferred action to be deferred due to deferred prereq, but got %s", firstDeferredActionInvocation.DeferredReason)
					}

					if firstDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(*plans.LifecycleActionTrigger).TriggeringResourceAddr.String() != "test_object.a" {
						t.Fatalf("expected deferred action to be triggered by test_object.a, but got %s", firstDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(*plans.LifecycleActionTrigger).TriggeringResourceAddr.String())
					}

					if firstDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String() != "action.test_action.hello" {
						t.Fatalf("expected deferred action to be triggered by action.test_action.hello, but got %s", firstDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String())
					}

					secondDeferredActionInvocation := p.DeferredActionInvocations[1]
					if secondDeferredActionInvocation.DeferredReason != providers.DeferredReasonDeferredPrereq {
						t.Fatalf("expected second deferred action to be deferred due to deferred prereq, but got %s", secondDeferredActionInvocation.DeferredReason)
					}
					if secondDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(*plans.LifecycleActionTrigger).TriggeringResourceAddr.String() != "test_object.a" {
						t.Fatalf("expected second deferred action to be triggered by test_object.a, but got %s", secondDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(*plans.LifecycleActionTrigger).TriggeringResourceAddr.String())
					}

					if secondDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String() != "action.test_action.hello" {
						t.Fatalf("expected second deferred action to be triggered by action.test_action.hello, but got %s", secondDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String())
					}

					if len(p.DeferredResources) != 1 {
						t.Fatalf("expected 1 resource to be deferred, got %d", len(p.DeferredResources))
					}
					deferredResource := p.DeferredResources[0]

					if deferredResource.ChangeSrc.Addr.String() != "test_object.a" {
						t.Fatalf("Expected resource %s to be deferred, but it was not", deferredResource.ChangeSrc.Addr)
					}

					if deferredResource.DeferredReason != providers.DeferredReasonAbsentPrereq {
						t.Fatalf("Expected deferred reason to be absent prereq, got %s", deferredResource.DeferredReason)
					}
				},
			},
			"action expansion with unknown instances": {
				module: map[string]string{
					"main.tf": `
variable "each" {
  type = set(string)
}
action "test_action" "hello" {
  for_each = var.each
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello["a"]]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
				planOpts: &PlanOpts{
					Mode:            plans.NormalMode,
					DeferralAllowed: true,
					SetVariables: InputValues{
						"each": &InputValue{
							Value:      cty.UnknownVal(cty.Set(cty.String)),
							SourceType: ValueFromCLIArg,
						},
					},
				},
				assertPlanDiagnostics: func(t *testing.T, diagnostics tfdiags.Diagnostics) {
					if len(diagnostics) != 1 {
						t.Fatal("wrong number of diagnostics")
					}

					if diagnostics[0].Severity() != tfdiags.Error {
						t.Error("expected error severity")
					}

					if diagnostics[0].Description().Summary != "Invalid for_each argument" {
						t.Errorf("expected for_each argument to be source of error but was %s", diagnostics[0].Description().Summary)
					}
				},
			},
			"action with unknown module expansion": {
				// We have an unknown module expansion (for_each over an unknown value). The
				// action and its triggering resource both live inside the (currently
				// un-expanded) module instances. Since we cannot expand the module yet, the
				// action invocation must be deferred.
				module: map[string]string{
					"main.tf": `
variable "mods" {
  type = set(string)
}
module "mod" {
  source   = "./mod"
  for_each = var.mods
}
`,
					"mod/mod.tf": `
action "test_action" "hello" {
  config {
    attr = "static"
  }
}
resource "other_object" "a" {
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode:            plans.NormalMode,
					DeferralAllowed: true,
					SetVariables: InputValues{
						"mods": &InputValue{
							Value:      cty.UnknownVal(cty.Set(cty.String)),
							SourceType: ValueFromCLIArg,
						},
					},
				},
				assertPlan: func(t *testing.T, p *plans.Plan) {
					// No concrete action invocations can be produced yet.
					if got := len(p.Changes.ActionInvocations); got != 0 {
						t.Fatalf("expected 0 planned action invocations, got %d", got)
					}

					if got := len(p.DeferredActionInvocations); got != 1 {
						t.Fatalf("expected 1 deferred action invocations, got %d", got)
					}
					ac, err := p.DeferredActionInvocations[0].Decode(&testActionSchema)
					if err != nil {
						t.Fatalf("error decoding action invocation: %s", err)
					}
					if ac.DeferredReason != providers.DeferredReasonInstanceCountUnknown {
						t.Fatalf("expected DeferredReasonInstanceCountUnknown, got %s", ac.DeferredReason)
					}
					if ac.ActionInvocationInstance.ConfigValue.GetAttr("attr").AsString() != "static" {
						t.Fatalf("expected attr to be static, got %s", ac.ActionInvocationInstance.ConfigValue.GetAttr("attr").AsString())
					}

				},
			},
			"action with unknown module expansion and unknown instances": {
				// Here both the module expansion and the action for_each expansion are unknown.
				// The action is referenced (with a specific key) inside the module so we should
				// get a single deferred action invocation for that specific (yet still
				// unresolved) instance address.
				module: map[string]string{
					"main.tf": `
variable "mods" {
  type = set(string)
}
variable "actions" {
  type = set(string)
}
module "mod" {
  source   = "./mod"
  for_each = var.mods
  
  actions = var.actions
}
`,
					"mod/mod.tf": `
variable "actions" {
  type = set(string)
}
action "test_action" "hello" {
  // Unknown for_each inside the module instance.
  for_each = var.actions
}
resource "other_object" "a" {
  lifecycle {
    action_trigger {
      events  = [before_create]
      // We reference a specific (yet unknown) action instance key.
      actions = [action.test_action.hello["a"]]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode:            plans.NormalMode,
					DeferralAllowed: true,
					SetVariables: InputValues{
						"mods": &InputValue{
							Value:      cty.UnknownVal(cty.Set(cty.String)),
							SourceType: ValueFromCLIArg,
						},
						"actions": &InputValue{
							Value:      cty.UnknownVal(cty.Set(cty.String)),
							SourceType: ValueFromCLIArg,
						},
					},
				},
				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 0 {
						t.Fatalf("expected 0 planned action invocations, got %d", len(p.Changes.ActionInvocations))
					}

					if len(p.DeferredActionInvocations) != 1 {
						t.Fatalf("expected 1 deferred partial action invocations, got %d", len(p.DeferredActionInvocations))
					}

					ac, err := p.DeferredActionInvocations[0].Decode(&testActionSchema)
					if err != nil {
						t.Fatalf("error decoding action invocation: %s", err)
					}
					if ac.DeferredReason != providers.DeferredReasonInstanceCountUnknown {
						t.Fatalf("expected deferred reason to be DeferredReasonInstanceCountUnknown, got %s", ac.DeferredReason)
					}
					if !ac.ActionInvocationInstance.ConfigValue.IsNull() {
						t.Fatalf("expected config value to be null")
					}
				},
			},

			"deferring resource dependencies should defer action": {
				module: map[string]string{
					"main.tf": `
resource "test_object" "origin" {
  name = "origin"
}
action "test_action" "hello" {
  config {
    attr = test_object.origin.name
  }
}
resource "test_object" "a" {
  name = "a"
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
				planOpts: &PlanOpts{
					Mode:            plans.NormalMode,
					DeferralAllowed: true,
				},
				planResourceFn: func(t *testing.T, req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
					if req.Config.GetAttr("name").AsString() == "origin" {
						return providers.PlanResourceChangeResponse{
							Deferred: &providers.Deferred{
								Reason: providers.DeferredReasonAbsentPrereq,
							},
						}
					}
					return providers.PlanResourceChangeResponse{
						PlannedState:    req.ProposedNewState,
						PlannedPrivate:  req.PriorPrivate,
						PlannedIdentity: req.PriorIdentity,
					}
				},

				assertPlanDiagnostics: func(t *testing.T, diagnostics tfdiags.Diagnostics) {
					if len(diagnostics) != 1 {
						t.Fatal("wrong number of diagnostics")
					}

					if diagnostics[0].Severity() != tfdiags.Error {
						t.Error("expected error diagnostics")
					}

					if diagnostics[0].Description().Summary != "Invalid action deferral" {
						t.Errorf("expected deferral to be source of error was %s", diagnostics[0].Description().Summary)
					}
				},
			},
		},

		// ======== INVOKE ========
		// -invoke flag
		// ======== INVOKE ========
		"invoke": {
			"simple action invoke": {
				module: map[string]string{
					"main.tf": `
action "test_action" "one" {
  config {
    attr = "one"
  }
}
action "test_action" "two" {
  config {
    attr = "two"
  }
}
`,
				},
				planOpts: &PlanOpts{
					Mode: plans.RefreshOnlyMode,
					ActionTargets: []addrs.Targetable{
						addrs.AbsActionInstance{
							Action: addrs.ActionInstance{
								Action: addrs.Action{
									Type: "test_action",
									Name: "one",
								},
								Key: addrs.NoKey,
							},
						},
					},
				},
				expectPlanActionCalled: true,
				assertPlan: func(t *testing.T, plan *plans.Plan) {
					if len(plan.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected exactly one invocation, and found %d", len(plan.Changes.ActionInvocations))
					}

					ais := plan.Changes.ActionInvocations[0]
					ai, err := ais.Decode(&testActionSchema)
					if err != nil {
						t.Fatal(err)
					}

					if _, ok := ai.ActionTrigger.(*plans.InvokeActionTrigger); !ok {
						t.Fatalf("expected invoke action trigger type but was %T", ai.ActionTrigger)
					}

					expected := cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("one"),
					})
					if diff := cmp.Diff(ai.ConfigValue, expected, ctydebug.CmpOptions); len(diff) > 0 {
						t.Fatalf("wrong value in plan: %s", diff)
					}

					if !ai.Addr.Equal(mustActionInstanceAddr("action.test_action.one")) {
						t.Fatalf("wrong address in plan: %s", ai.Addr)
					}
				},
			},

			"action invoke in module": {
				module: map[string]string{
					"mod/main.tf": `
action "test_action" "one" {
  config {
    attr = "one"
  }
}
action "test_action" "two" {
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
				planOpts: &PlanOpts{
					Mode: plans.RefreshOnlyMode,
					ActionTargets: []addrs.Targetable{
						addrs.AbsActionInstance{
							Module: addrs.RootModuleInstance.Child("mod", addrs.NoKey),
							Action: addrs.ActionInstance{
								Action: addrs.Action{
									Type: "test_action",
									Name: "one",
								},
								Key: addrs.NoKey,
							},
						},
					},
				},
				expectPlanActionCalled: true,
				assertPlan: func(t *testing.T, plan *plans.Plan) {
					if len(plan.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected exactly one invocation, and found %d", len(plan.Changes.ActionInvocations))
					}

					ais := plan.Changes.ActionInvocations[0]
					ai, err := ais.Decode(&testActionSchema)
					if err != nil {
						t.Fatal(err)
					}

					if _, ok := ai.ActionTrigger.(*plans.InvokeActionTrigger); !ok {
						t.Fatalf("expected invoke action trigger type but was %T", ai.ActionTrigger)
					}

					expected := cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("one"),
					})
					if diff := cmp.Diff(ai.ConfigValue, expected, ctydebug.CmpOptions); len(diff) > 0 {
						t.Fatalf("wrong value in plan: %s", diff)
					}

					if !ai.Addr.Equal(mustActionInstanceAddr("module.mod.action.test_action.one")) {
						t.Fatalf("wrong address in plan: %s", ai.Addr)
					}
				},
			},

			"action invoke in expanded module": {
				module: map[string]string{
					"mod/main.tf": `
action "test_action" "one" {
  config {
    attr = "one"
  }
}
action "test_action" "two" {
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
				planOpts: &PlanOpts{
					Mode: plans.RefreshOnlyMode,
					ActionTargets: []addrs.Targetable{
						addrs.AbsActionInstance{
							Module: addrs.RootModuleInstance.Child("mod", addrs.IntKey(1)),
							Action: addrs.ActionInstance{
								Action: addrs.Action{
									Type: "test_action",
									Name: "one",
								},
								Key: addrs.NoKey,
							},
						},
					},
				},
				expectPlanActionCalled: true,
				assertPlan: func(t *testing.T, plan *plans.Plan) {
					if len(plan.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected exactly one invocation, and found %d", len(plan.Changes.ActionInvocations))
					}

					ais := plan.Changes.ActionInvocations[0]
					ai, err := ais.Decode(&testActionSchema)
					if err != nil {
						t.Fatal(err)
					}

					if _, ok := ai.ActionTrigger.(*plans.InvokeActionTrigger); !ok {
						t.Fatalf("expected invoke action trigger type but was %T", ai.ActionTrigger)
					}

					expected := cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("one"),
					})
					if diff := cmp.Diff(ai.ConfigValue, expected, ctydebug.CmpOptions); len(diff) > 0 {
						t.Fatalf("wrong value in plan: %s", diff)
					}

					if !ai.Addr.Equal(mustActionInstanceAddr("module.mod[1].action.test_action.one")) {
						t.Fatalf("wrong address in plan: %s", ai.Addr)
					}
				},
			},

			"action invoke with count (all)": {
				module: map[string]string{
					"main.tf": `
action "test_action" "one" {
  count = 2

  config {
    attr = "${count.index}"
  }
}
action "test_action" "two" {
  count = 2

  config {
    attr = "two"
  }
}
`,
				},
				planOpts: &PlanOpts{
					Mode: plans.RefreshOnlyMode,
					ActionTargets: []addrs.Targetable{
						addrs.AbsAction{
							Action: addrs.Action{
								Type: "test_action",
								Name: "one",
							},
						},
					},
				},
				expectPlanActionCalled: true,
				assertPlan: func(t *testing.T, plan *plans.Plan) {
					if len(plan.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected exactly two invocations, and found %d", len(plan.Changes.ActionInvocations))
					}

					sort.Slice(plan.Changes.ActionInvocations, func(i, j int) bool {
						return plan.Changes.ActionInvocations[i].Addr.Less(plan.Changes.ActionInvocations[j].Addr)
					})

					ais := plan.Changes.ActionInvocations[0]
					ai, err := ais.Decode(&testActionSchema)
					if err != nil {
						t.Fatal(err)
					}

					if _, ok := ai.ActionTrigger.(*plans.InvokeActionTrigger); !ok {
						t.Fatalf("expected invoke action trigger type but was %T", ai.ActionTrigger)
					}

					expected := cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("0"),
					})
					if diff := cmp.Diff(ai.ConfigValue, expected, ctydebug.CmpOptions); len(diff) > 0 {
						t.Fatalf("wrong value in plan: %s", diff)
					}

					if !ai.Addr.Equal(mustActionInstanceAddr("action.test_action.one[0]")) {
						t.Fatalf("wrong address in plan: %s", ai.Addr)
					}

					ais = plan.Changes.ActionInvocations[1]
					ai, err = ais.Decode(&testActionSchema)
					if err != nil {
						t.Fatal(err)
					}

					if _, ok := ai.ActionTrigger.(*plans.InvokeActionTrigger); !ok {
						t.Fatalf("expected invoke action trigger type but was %T", ai.ActionTrigger)
					}

					expected = cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("1"),
					})
					if diff := cmp.Diff(ai.ConfigValue, expected, ctydebug.CmpOptions); len(diff) > 0 {
						t.Fatalf("wrong value in plan: %s", diff)
					}

					if !ai.Addr.Equal(mustActionInstanceAddr("action.test_action.one[1]")) {
						t.Fatalf("wrong address in plan: %s", ai.Addr)
					}
				},
			},

			"action invoke with count (instance)": {
				module: map[string]string{
					"main.tf": `
action "test_action" "one" {
  count = 2

  config {
    attr = "${count.index}"
  }
}
action "test_action" "two" {
  count = 2

  config {
    attr = "two"
  }
}
`,
				},
				planOpts: &PlanOpts{
					Mode: plans.RefreshOnlyMode,
					ActionTargets: []addrs.Targetable{
						addrs.AbsActionInstance{
							Action: addrs.ActionInstance{
								Action: addrs.Action{
									Type: "test_action",
									Name: "one",
								},
								Key: addrs.IntKey(0),
							},
						},
					},
				},
				expectPlanActionCalled: true,
				assertPlan: func(t *testing.T, plan *plans.Plan) {
					if len(plan.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected exactly one invocation, and found %d", len(plan.Changes.ActionInvocations))
					}

					ais := plan.Changes.ActionInvocations[0]
					ai, err := ais.Decode(&testActionSchema)
					if err != nil {
						t.Fatal(err)
					}

					if _, ok := ai.ActionTrigger.(*plans.InvokeActionTrigger); !ok {
						t.Fatalf("expected invoke action trigger type but was %T", ai.ActionTrigger)
					}

					expected := cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("0"),
					})
					if diff := cmp.Diff(ai.ConfigValue, expected, ctydebug.CmpOptions); len(diff) > 0 {
						t.Fatalf("wrong value in plan: %s", diff)
					}

					if !ai.Addr.Equal(mustActionInstanceAddr("action.test_action.one[0]")) {
						t.Fatalf("wrong address in plan: %s", ai.Addr)
					}
				},
			},

			"invoke action with reference": {
				module: map[string]string{
					"main.tf": `
resource "test_object" "a" {
  name = "hello"
}

action "test_action" "one" {
  config {
    attr = test_object.a.name
  }
}
`,
				},
				planOpts: &PlanOpts{
					Mode: plans.RefreshOnlyMode,
					ActionTargets: []addrs.Targetable{
						addrs.AbsAction{
							Action: addrs.Action{
								Type: "test_action",
								Name: "one",
							},
						},
					},
				},
				expectPlanActionCalled: true,
				buildState: func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"), &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{"name":"hello"}`),
						Status:    states.ObjectReady,
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
				},
				assertPlan: func(t *testing.T, plan *plans.Plan) {
					if len(plan.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected exactly one invocation, and found %d", len(plan.Changes.ActionInvocations))
					}

					ais := plan.Changes.ActionInvocations[0]
					ai, err := ais.Decode(&testActionSchema)
					if err != nil {
						t.Fatal(err)
					}

					if _, ok := ai.ActionTrigger.(*plans.InvokeActionTrigger); !ok {
						t.Fatalf("expected invoke action trigger type but was %T", ai.ActionTrigger)
					}

					expected := cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("hello"),
					})
					if diff := cmp.Diff(ai.ConfigValue, expected, ctydebug.CmpOptions); len(diff) > 0 {
						t.Fatalf("wrong value in plan: %s", diff)
					}

					if !ai.Addr.Equal(mustActionInstanceAddr("action.test_action.one")) {
						t.Fatalf("wrong address in plan: %s", ai.Addr)
					}
				},
			},

			"invoke action with reference (drift)": {
				module: map[string]string{
					"main.tf": `
resource "test_object" "a" {
  name = "hello"
}

action "test_action" "one" {
  config {
    attr = test_object.a.name
  }
}
`,
				},
				planOpts: &PlanOpts{
					Mode: plans.RefreshOnlyMode,
					ActionTargets: []addrs.Targetable{
						addrs.AbsAction{
							Action: addrs.Action{
								Type: "test_action",
								Name: "one",
							},
						},
					},
				},
				expectPlanActionCalled: true,
				buildState: func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"), &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{"name":"hello"}`),
						Status:    states.ObjectReady,
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
				},
				assertPlan: func(t *testing.T, plan *plans.Plan) {
					if len(plan.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected exactly one invocation, and found %d", len(plan.Changes.ActionInvocations))
					}

					ais := plan.Changes.ActionInvocations[0]
					ai, err := ais.Decode(&testActionSchema)
					if err != nil {
						t.Fatal(err)
					}

					if _, ok := ai.ActionTrigger.(*plans.InvokeActionTrigger); !ok {
						t.Fatalf("expected invoke action trigger type but was %T", ai.ActionTrigger)
					}

					expected := cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("drifted value"),
					})
					if diff := cmp.Diff(ai.ConfigValue, expected, ctydebug.CmpOptions); len(diff) > 0 {
						t.Fatalf("wrong value in plan: %s", diff)
					}

					if !ai.Addr.Equal(mustActionInstanceAddr("action.test_action.one")) {
						t.Fatalf("wrong address in plan: %s", ai.Addr)
					}
				},
				readResourceFn: func(t *testing.T, request providers.ReadResourceRequest) providers.ReadResourceResponse {
					return providers.ReadResourceResponse{
						NewState: cty.ObjectVal(map[string]cty.Value{
							"name": cty.StringVal("drifted value"),
						}),
					}
				},
			},

			"invoke action with reference (drift, no refresh)": {
				module: map[string]string{
					"main.tf": `
resource "test_object" "a" {
  name = "hello"
}

action "test_action" "one" {
  config {
    attr = test_object.a.name
  }
}
`,
				},
				planOpts: &PlanOpts{
					Mode:        plans.RefreshOnlyMode,
					SkipRefresh: true,
					ActionTargets: []addrs.Targetable{
						addrs.AbsAction{
							Action: addrs.Action{
								Type: "test_action",
								Name: "one",
							},
						},
					},
				},
				expectPlanActionCalled: true,
				buildState: func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"), &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{"name":"hello"}`),
						Status:    states.ObjectReady,
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
				},
				assertPlan: func(t *testing.T, plan *plans.Plan) {
					if len(plan.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected exactly one invocation, and found %d", len(plan.Changes.ActionInvocations))
					}

					ais := plan.Changes.ActionInvocations[0]
					ai, err := ais.Decode(&testActionSchema)
					if err != nil {
						t.Fatal(err)
					}

					if _, ok := ai.ActionTrigger.(*plans.InvokeActionTrigger); !ok {
						t.Fatalf("expected invoke action trigger type but was %T", ai.ActionTrigger)
					}

					expected := cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("hello"),
					})
					if diff := cmp.Diff(ai.ConfigValue, expected, ctydebug.CmpOptions); len(diff) > 0 {
						t.Fatalf("wrong value in plan: %s", diff)
					}

					if !ai.Addr.Equal(mustActionInstanceAddr("action.test_action.one")) {
						t.Fatalf("wrong address in plan: %s", ai.Addr)
					}
				},
				readResourceFn: func(t *testing.T, request providers.ReadResourceRequest) providers.ReadResourceResponse {
					return providers.ReadResourceResponse{
						NewState: cty.ObjectVal(map[string]cty.Value{
							"name": cty.StringVal("drifted value"),
						}),
					}
				},
			},

			"invoke action with partially applied configuration": {
				module: map[string]string{
					"main.tf": `
resource "test_object" "a" {
  name = "hello"
}

action "test_action" "one" {
  config {
    attr = test_object.a.name
  }
}
`,
				},
				planOpts: &PlanOpts{
					Mode: plans.RefreshOnlyMode,
					ActionTargets: []addrs.Targetable{
						addrs.AbsAction{
							Action: addrs.Action{
								Type: "test_action",
								Name: "one",
							},
						},
					},
				},
				expectPlanActionCalled: false,
				assertPlanDiagnostics: func(t *testing.T, diagnostics tfdiags.Diagnostics) {
					if len(diagnostics) != 1 {
						t.Errorf("expected exactly one diagnostic but got %d", len(diagnostics))
					}

					if diagnostics[0].Description().Summary != "Partially applied configuration" {
						t.Errorf("wrong diagnostic: %s", diagnostics[0].Description().Summary)
					}
				},
			},

			"non-referenced resource isn't refreshed during invoke": {
				module: map[string]string{
					"main.tf": `
resource "test_object" "a" {
  name = "hello"
}

action "test_action" "one" {
  config {
    attr = "world"
  }
}
`,
				},
				planOpts: &PlanOpts{
					Mode: plans.RefreshOnlyMode,
					ActionTargets: []addrs.Targetable{
						addrs.AbsAction{
							Action: addrs.Action{
								Type: "test_action",
								Name: "one",
							},
						},
					},
				},
				expectPlanActionCalled: true,
				buildState: func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_object.a"), &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{"name":"hello"}`),
						Status:    states.ObjectReady,
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
				},
				assertPlan: func(t *testing.T, plan *plans.Plan) {
					if len(plan.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected exactly one invocation, and found %d", len(plan.Changes.ActionInvocations))
					}

					ais := plan.Changes.ActionInvocations[0]
					ai, err := ais.Decode(&testActionSchema)
					if err != nil {
						t.Fatal(err)
					}

					if _, ok := ai.ActionTrigger.(*plans.InvokeActionTrigger); !ok {
						t.Fatalf("expected invoke action trigger type but was %T", ai.ActionTrigger)
					}

					expected := cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("world"),
					})
					if diff := cmp.Diff(ai.ConfigValue, expected, ctydebug.CmpOptions); len(diff) > 0 {
						t.Fatalf("wrong value in plan: %s", diff)
					}

					if !ai.Addr.Equal(mustActionInstanceAddr("action.test_action.one")) {
						t.Fatalf("wrong address in plan: %s", ai.Addr)
					}

					if len(plan.DriftedResources) > 0 {
						t.Fatalf("shouldn't have refreshed any resources")
					}
				},
				readResourceFn: func(t *testing.T, request providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
					t.Fatalf("should not have tried to refresh any resources")
					return
				},
			},
		},

		// ======== CONDITIONS ========
		// condition action_trigger attribute
		// ======== CONDITIONS ========

		"conditions": {
			"boolean condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
action "test_action" "world" {}
action "test_action" "bye" {}
resource "test_object" "foo" {
name = "foo"
}
resource "test_object" "a" {
lifecycle {
  action_trigger {
    events = [before_create]
    condition = test_object.foo.name == "foo"
    actions = [action.test_action.hello, action.test_action.world]
  }
  action_trigger {
    events = [after_create]
    condition = test_object.foo.name == "bye"
    actions = [action.test_action.bye]
  }
}
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected 2 actions in plan, got %d", len(p.Changes.ActionInvocations))
					}

					invokedActionAddrs := []string{}
					for _, action := range p.Changes.ActionInvocations {
						invokedActionAddrs = append(invokedActionAddrs, action.Addr.String())
					}
					slices.Sort(invokedActionAddrs)
					expectedActions := []string{
						"action.test_action.hello",
						"action.test_action.world",
					}
					if !cmp.Equal(expectedActions, invokedActionAddrs) {
						t.Fatalf("expected actions: %v, got %v", expectedActions, invokedActionAddrs)
					}
				},
			},

			"unknown condition": {
				module: map[string]string{
					"main.tf": `
variable "cond" {
    type = string
}
action "test_action" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      condition = var.cond == "foo"
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,
				planOpts: &PlanOpts{
					Mode: plans.NormalMode,
					SetVariables: InputValues{
						"cond": &InputValue{
							Value:      cty.UnknownVal(cty.String),
							SourceType: ValueFromCaller,
						},
					},
				},
				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Condition must be known",
						Detail:   "The condition expression resulted in an unknown value, but it must be a known boolean value.",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 10, Column: 19, Byte: 184},
							End:      hcl.Pos{Line: 10, Column: 36, Byte: 201},
						},
					})
				},
			},

			"non-boolean condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "foo" {
  name = "foo"
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      condition = test_object.foo.name
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,

				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Incorrect value type",
						Detail:   "Invalid expression value: a bool is required.",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 10, Column: 19, Byte: 194},
							End:      hcl.Pos{Line: 10, Column: 39, Byte: 214},
						},
					})
				},
			},

			"using self in before_* condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
action "test_action" "world" {}
resource "test_object" "a" {
  name = "foo"
  lifecycle {
    action_trigger {
      events = [before_create]
      condition = self.name == "foo"
      actions = [action.test_action.hello]
    }
    action_trigger {
      events = [after_update]
      condition = self.name == "bar"
      actions = [action.test_action.world]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,

				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Self reference not allowed",
						Detail:   `The condition expression cannot reference "self".`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 9, Column: 19, Byte: 193},
							End:      hcl.Pos{Line: 9, Column: 37, Byte: 211},
						},
					})
				},
			},

			"using self in after_* condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
action "test_action" "world" {}
resource "test_object" "a" {
  name = "foo"
  lifecycle {
    action_trigger {
      events = [after_create]
      condition = self.name == "foo"
      actions = [action.test_action.hello]
    }
    action_trigger {
      events = [after_update]
      condition = self.name == "bar"
      actions = [action.test_action.world]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,

				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					// We only expect one diagnostic, as the other condition is valid
					return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Self reference not allowed",
						Detail:   `The condition expression cannot reference "self".`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 9, Column: 19, Byte: 192},
							End:      hcl.Pos{Line: 9, Column: 37, Byte: 210},
						},
					})
				},
			},

			"referencing triggering resource in before_* condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
action "test_action" "world" {}
resource "test_object" "a" {
  name = "foo"
  lifecycle {
    action_trigger {
      events = [before_create]
      condition = test_object.a.name == "foo"
      actions = [action.test_action.hello]
    }
    action_trigger {
      events = [before_update]
      condition = test_object.a.name == "bar"
      actions = [action.test_action.world]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlanDiagnostics: func(t *testing.T, diags tfdiags.Diagnostics) {
					if !diags.HasErrors() {
						t.Errorf("expected errors, got none")
					}

					err := diags.Err().Error()
					if !strings.Contains(err, "Cycle:") || !strings.Contains(err, "action.test_action.hello") || !strings.Contains(err, "test_object.a") {
						t.Fatalf("Expected '[Error] Cycle: action.test_action.hello (instance), test_object.a', got '%s'", err)
					}
				},
			},

			"referencing triggering resource in after_* condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
action "test_action" "world" {}
resource "test_object" "a" {
  name = "foo"
  lifecycle {
    action_trigger {
      events = [after_create]
      condition = test_object.a.name == "foo"
      actions = [action.test_action.hello]
    }
    action_trigger {
      events = [after_update]
      condition = test_object.a.name == "bar"
      actions = [action.test_action.world]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Errorf("expected 1 action invocation, got %d", len(p.Changes.ActionInvocations))
					}
				},
			},

			"using each in before_* condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
action "test_action" "world" {}
resource "test_object" "a" {
  for_each = toset(["foo", "bar"])
  name = each.key
  lifecycle {
    action_trigger {
      events = [before_create]
      condition = each.key == "foo"
      actions = [action.test_action.hello]
    }
  }
}
`,
				},
				expectPlanActionCalled: false,

				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Each reference not allowed",
						Detail:   `The condition expression cannot reference "each" if the action is run before the resource is applied.`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 10, Column: 19, Byte: 231},
							End:      hcl.Pos{Line: 10, Column: 36, Byte: 248},
						},
					}).Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Each reference not allowed",
						Detail:   `The condition expression cannot reference "each" if the action is run before the resource is applied.`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 10, Column: 19, Byte: 231},
							End:      hcl.Pos{Line: 10, Column: 36, Byte: 248},
						},
					})
				},
			},

			"using each in after_* condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
action "test_action" "world" {}
resource "test_object" "a" {
  for_each = toset(["foo", "bar"])
  name = each.key
  lifecycle {
    action_trigger {
      events = [after_create]
      condition = each.key == "foo"
      actions = [action.test_action.hello]
    }
    action_trigger {
      events = [after_update]
      condition = each.key == "bar"
      actions = [action.test_action.world]
    }
  }
}`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Errorf("Expected 1 action invocations, got %d", len(p.Changes.ActionInvocations))
					}
					if p.Changes.ActionInvocations[0].Addr.String() != "action.test_action.hello" {
						t.Errorf("Expected action 'action.test_action.hello', got %s", p.Changes.ActionInvocations[0].Addr.String())
					}
				},
			},

			"using count.index in before_* condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
action "test_action" "world" {}
resource "test_object" "a" {
	count = 3
	name = "item-${count.index}"
	lifecycle {
		action_trigger {
			events = [before_create]
			condition = count.index == 1
			actions = [action.test_action.hello]
		}
		action_trigger {
			events = [before_update]
			condition = count.index == 2
			actions = [action.test_action.world]
		}
	}
}`,
				},
				expectPlanActionCalled: false,

				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Count reference not allowed",
						Detail:   `The condition expression cannot reference "count" if the action is run before the resource is applied.`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 10, Column: 21, Byte: 210},
							End:      hcl.Pos{Line: 10, Column: 37, Byte: 226},
						},
					}).Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Count reference not allowed",
						Detail:   `The condition expression cannot reference "count" if the action is run before the resource is applied.`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 10, Column: 21, Byte: 210},
							End:      hcl.Pos{Line: 10, Column: 37, Byte: 226},
						},
					}).Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Count reference not allowed",
						Detail:   `The condition expression cannot reference "count" if the action is run before the resource is applied.`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 10, Column: 21, Byte: 210},
							End:      hcl.Pos{Line: 10, Column: 37, Byte: 226},
						},
					})
				},
			},

			"using count.index in after_* condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
action "test_action" "world" {}
resource "test_object" "a" {
	count = 3
	name = "item-${count.index}"
	lifecycle {
		action_trigger {
			events = [after_create]
			condition = count.index == 1
			actions = [action.test_action.hello]
		}
		action_trigger {
			events = [after_update]
			condition = count.index == 2
			actions = [action.test_action.world]
		}
	}
}
				`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Errorf("Expected 1 action invocation, got %d", len(p.Changes.ActionInvocations))
					}
					if p.Changes.ActionInvocations[0].Addr.String() != "action.test_action.hello" {
						t.Errorf("Expected action invocation %q, got %q", "action.test_action.hello", p.Changes.ActionInvocations[0].Addr.String())
					}
				},
			},

			"using each.value in before_* condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
action "test_action" "world" {}
resource "test_object" "a" {
				for_each = {"foo" = "value1", "bar" = "value2"}
				name = each.value
				lifecycle {
						action_trigger {
								events = [before_create]
								condition = each.value == "value1"
								actions = [action.test_action.hello]
						}
						action_trigger {
								events = [before_update]
								condition = each.value == "value2"
								actions = [action.test_action.world]
						}
				}
}
				`,
				},
				expectPlanActionCalled: false,

				expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
					return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Each reference not allowed",
						Detail:   `The condition expression cannot reference "each" if the action is run before the resource is applied.`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 10, Column: 21, Byte: 260},
							End:      hcl.Pos{Line: 10, Column: 43, Byte: 282},
						},
					}).Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Each reference not allowed",
						Detail:   `The condition expression cannot reference "each" if the action is run before the resource is applied.`,
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 10, Column: 21, Byte: 260},
							End:      hcl.Pos{Line: 10, Column: 43, Byte: 282},
						},
					})
				},
			},

			"using each.value in after_* condition": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
action "test_action" "world" {}
resource "test_object" "a" {
				for_each = {"foo" = "value1", "bar" = "value2"}
				name = each.value
				lifecycle {
						action_trigger {
								events = [after_create]
								condition = each.value == "value1"
								actions = [action.test_action.hello]
						}
						action_trigger {
								events = [after_update]
								condition = each.value == "value2"
								actions = [action.test_action.world]
						}
				}
}
				`,
				},
				expectPlanActionCalled: true,

				assertPlan: func(t *testing.T, p *plans.Plan) {
					if len(p.Changes.ActionInvocations) != 1 {
						t.Errorf("Expected 1 action invocations, got %d", len(p.Changes.ActionInvocations))
					}
					if p.Changes.ActionInvocations[0].Addr.String() != "action.test_action.hello" {
						t.Errorf("Expected action 'action.test_action.hello', got %s", p.Changes.ActionInvocations[0].Addr.String())
					}
				},
			},
		},
		// ======== TARGETING ========
		// -target flag behavior
		// ======== TARGETING ========
		"targeting": {
			"targeted run": {
				module: map[string]string{
					"main.tf": ` 
action "test_action" "hello" {}
action "test_action" "there" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
    action_trigger {
      events = [after_create]
      actions = [action.test_action.there]
    }
  }
}
action "test_action" "general" {}
action "test_action" "kenobi" {}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.test_action.general]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode: plans.NormalMode,
					// We only target resource a
					Targets: []addrs.Targetable{
						addrs.RootModuleInstance.Resource(addrs.ManagedResourceMode, "test_object", "a"),
					},
				},
				// There is a warning related to targeting that we will just ignore
				assertPlanDiagnostics: func(t *testing.T, d tfdiags.Diagnostics) {
					if d.HasErrors() {
						t.Fatalf("expected no errors, got %s", d.Err().Error())
					}
				},
				assertPlan: func(t *testing.T, p *plans.Plan) {
					// Validate we are targeting resource a out of paranoia
					if len(p.Changes.Resources) != 1 {
						t.Fatalf("expected plan to have 1 resource change, got %d", len(p.Changes.Resources))
					}
					if p.Changes.Resources[0].Addr.String() != "test_object.a" {
						t.Fatalf("expected plan to target resource 'test_object.a', got %s", p.Changes.Resources[0].Addr.String())
					}

					// Ensure the actions for test_object.a are planned
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected plan to have 2 action invocations, got %d", len(p.Changes.ActionInvocations))
					}

					actionAddrs := []string{
						p.Changes.ActionInvocations[0].Addr.String(),
						p.Changes.ActionInvocations[1].Addr.String(),
					}

					slices.Sort(actionAddrs)
					if actionAddrs[0] != "action.test_action.hello" || actionAddrs[1] != "action.test_action.there" {
						t.Fatalf("expected action addresses to be ['action.test_action.hello', 'action.test_action.there'], got %v", actionAddrs)
					}

				},
			},
			"targeted run with ancestor that has actions": {
				module: map[string]string{
					"main.tf": ` 
action "test_action" "hello" {}
action "test_action" "there" {}
resource "test_object" "origin" {
  name = "origin"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
  }
}
resource "test_object" "a" {
  name = test_object.origin.name
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.test_action.there]
    }
  }
}
action "test_action" "general" {}
action "test_action" "kenobi" {}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.test_action.general]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode: plans.NormalMode,
					// We only target resource a
					Targets: []addrs.Targetable{
						mustResourceInstanceAddr("test_object.a"),
					},
				},
				// There is a warning related to targeting that we will just ignore
				assertPlanDiagnostics: func(t *testing.T, d tfdiags.Diagnostics) {
					if d.HasErrors() {
						t.Fatalf("expected no errors, got %s", d.Err().Error())
					}
				},
				assertPlan: func(t *testing.T, p *plans.Plan) {
					// Validate we are targeting resource a out of paranoia
					if len(p.Changes.Resources) != 2 {
						t.Fatalf("expected plan to have 2 resource changes, got %d", len(p.Changes.Resources))
					}
					resourceAddrs := []string{
						p.Changes.Resources[0].Addr.String(),
						p.Changes.Resources[1].Addr.String(),
					}

					slices.Sort(resourceAddrs)
					if resourceAddrs[0] != "test_object.a" || resourceAddrs[1] != "test_object.origin" {
						t.Fatalf("expected resource addresses to be ['test_object.a', 'test_object.origin'], got %v", resourceAddrs)
					}

					// Ensure the actions for test_object.a are planned
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected plan to have 2 action invocations, got %d", len(p.Changes.ActionInvocations))
					}

					actionAddrs := []string{
						p.Changes.ActionInvocations[0].Addr.String(),
						p.Changes.ActionInvocations[1].Addr.String(),
					}

					slices.Sort(actionAddrs)
					if actionAddrs[0] != "action.test_action.hello" || actionAddrs[1] != "action.test_action.there" {
						t.Fatalf("expected action addresses to be ['action.test_action.hello', 'action.test_action.there'], got %v", actionAddrs)
					}

				},
			},
			"targeted run with expansion": {
				module: map[string]string{
					"main.tf": ` 
action "test_action" "hello" {
  count = 3
}
action "test_action" "there" {
  count = 3
}
resource "test_object" "a" {
  count = 3
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello[count.index]]
    }
    action_trigger {
      events = [after_create]
      actions = [action.test_action.there[count.index]]
    }
  }
}
action "test_action" "general" {}
action "test_action" "kenobi" {}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.test_action.general]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode: plans.NormalMode,
					// We only target resource a
					Targets: []addrs.Targetable{
						addrs.RootModuleInstance.Resource(addrs.ManagedResourceMode, "test_object", "a").Instance(addrs.IntKey(2)),
					},
				},
				// There is a warning related to targeting that we will just ignore
				assertPlanDiagnostics: func(t *testing.T, d tfdiags.Diagnostics) {
					if d.HasErrors() {
						t.Fatalf("expected no errors, got %s", d.Err().Error())
					}
				},
				assertPlan: func(t *testing.T, p *plans.Plan) {
					// Validate we are targeting resource a out of paranoia
					if len(p.Changes.Resources) != 1 {
						t.Fatalf("expected plan to have 1 resource change, got %d", len(p.Changes.Resources))
					}
					if p.Changes.Resources[0].Addr.String() != "test_object.a[2]" {
						t.Fatalf("expected plan to target resource 'test_object.a[2]', got %s", p.Changes.Resources[0].Addr.String())
					}

					// Ensure the actions for test_object.a are planned
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected plan to have 2 action invocations, got %d", len(p.Changes.ActionInvocations))
					}

					actionAddrs := []string{
						p.Changes.ActionInvocations[0].Addr.String(),
						p.Changes.ActionInvocations[1].Addr.String(),
					}

					slices.Sort(actionAddrs)
					if actionAddrs[0] != "action.test_action.hello[2]" || actionAddrs[1] != "action.test_action.there[2]" {
						t.Fatalf("expected action addresses to be ['action.test_action.hello[2]', 'action.test_action.there[2]'], got %v", actionAddrs)
					}
				},
			},
			"targeted run with resource reference": {
				module: map[string]string{
					"main.tf": ` 
resource "test_object" "source" {}
action "test_action" "hello" {
  config {
    attr = test_object.source.name
  }
}
action "test_action" "there" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_action.hello]
    }
    action_trigger {
      events = [after_create]
      actions = [action.test_action.there]
    }
  }
}
action "test_action" "general" {}
action "test_action" "kenobi" {}
resource "test_object" "b" {
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.test_action.general]
    }
  }
}
`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode: plans.NormalMode,
					// We only target resource a
					Targets: []addrs.Targetable{
						addrs.RootModuleInstance.Resource(addrs.ManagedResourceMode, "test_object", "a"),
					},
				},
				// There is a warning related to targeting that we will just ignore
				assertPlanDiagnostics: func(t *testing.T, d tfdiags.Diagnostics) {
					if d.HasErrors() {
						t.Fatalf("expected no errors, got %s", d.Err().Error())
					}
				},
				assertPlan: func(t *testing.T, p *plans.Plan) {
					// Validate we are targeting resource a out of paranoia
					if len(p.Changes.Resources) != 2 {
						t.Fatalf("expected plan to have 2 resource changes, got %d", len(p.Changes.Resources))
					}
					resourceAddrs := []string{
						p.Changes.Resources[0].Addr.String(),
						p.Changes.Resources[1].Addr.String(),
					}
					slices.Sort(resourceAddrs)
					if resourceAddrs[0] != "test_object.a" || resourceAddrs[1] != "test_object.source" {
						t.Fatalf("expected resource addresses to be ['test_object.a', 'test_object.source'], got %v", resourceAddrs)
					}

					// Ensure the actions for test_object.a are planned
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected plan to have 2 action invocations, got %d", len(p.Changes.ActionInvocations))
					}

					actionAddrs := []string{
						p.Changes.ActionInvocations[0].Addr.String(),
						p.Changes.ActionInvocations[1].Addr.String(),
					}

					slices.Sort(actionAddrs)
					if actionAddrs[0] != "action.test_action.hello" || actionAddrs[1] != "action.test_action.there" {
						t.Fatalf("expected action addresses to be ['action.test_action.hello', 'action.test_action.there'], got %v", actionAddrs)
					}
				},
			},

			"targeted run with condition referencing another resource": {
				module: map[string]string{
					"main.tf": `
resource "test_object" "source" {
		name = "source"
}
action "test_action" "hello" {
		config {
			attr = test_object.source.name
		}
}
resource "test_object" "a" {
		lifecycle {
			action_trigger {
				events = [before_create]
				condition = test_object.source.name == "source"
				actions = [action.test_action.hello]
			}
		}
}
				`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode: plans.NormalMode,
					// Only target resource a
					Targets: []addrs.Targetable{
						addrs.RootModuleInstance.Resource(addrs.ManagedResourceMode, "test_object", "a"),
					},
				},
				assertPlanDiagnostics: func(t *testing.T, d tfdiags.Diagnostics) {
					if d.HasErrors() {
						t.Fatalf("expected no errors, got %s", d.Err().Error())
					}
				},
				assertPlan: func(t *testing.T, p *plans.Plan) {
					// Only resource a should be planned
					if len(p.Changes.Resources) != 2 {
						t.Fatalf("expected plan to have 2 resource changes, got %d", len(p.Changes.Resources))
					}
					resourceAddrs := []string{p.Changes.Resources[0].Addr.String(), p.Changes.Resources[1].Addr.String()}
					slices.Sort(resourceAddrs)

					if resourceAddrs[0] != "test_object.a" || resourceAddrs[1] != "test_object.source" {
						t.Fatalf("expected resource addresses to be ['test_object.a', 'test_object.source'], got %v", resourceAddrs)
					}

					// Only one action invocation for resource a
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected plan to have 1 action invocation, got %d", len(p.Changes.ActionInvocations))
					}
					if p.Changes.ActionInvocations[0].Addr.String() != "action.test_action.hello" {
						t.Fatalf("expected action address to be 'action.test_action.hello', got '%s'", p.Changes.ActionInvocations[0].Addr)
					}
				},
			},

			"targeted run with action referencing another resource that also triggers actions": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "source" {
		name = "source"
		
		lifecycle {
			action_trigger {
				events = [before_create]
				actions = [action.test_action.hello]
			}
		}
}
action "test_action" "there" {
		config {
			attr = test_object.source.name
		}
}
resource "test_object" "a" {
		lifecycle {
			action_trigger {
				events = [after_create]
				actions = [action.test_action.there]
			}
		}
}
resource "test_object" "b" {
		lifecycle {
			action_trigger {
				events = [before_create]
				actions = [action.test_action.hello]
			}
		}
}
				`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode: plans.NormalMode,
					// Only target resource a
					Targets: []addrs.Targetable{
						addrs.RootModuleInstance.Resource(addrs.ManagedResourceMode, "test_object", "a"),
					},
				},
				assertPlanDiagnostics: func(t *testing.T, d tfdiags.Diagnostics) {
					if d.HasErrors() {
						t.Fatalf("expected no errors, got %s", d.Err().Error())
					}
				},
				assertPlan: func(t *testing.T, p *plans.Plan) {
					// Should plan for resource a and its dependency source, but not b
					if len(p.Changes.Resources) != 2 {
						t.Fatalf("expected plan to have 2 resource changes, got %d", len(p.Changes.Resources))
					}
					resourceAddrs := []string{
						p.Changes.Resources[0].Addr.String(),
						p.Changes.Resources[1].Addr.String(),
					}
					slices.Sort(resourceAddrs)
					if resourceAddrs[0] != "test_object.a" || resourceAddrs[1] != "test_object.source" {
						t.Fatalf("expected resource addresses to be ['test_object.a', 'test_object.source'], got %v", resourceAddrs)
					}

					// Should plan both actions for resource a
					if len(p.Changes.ActionInvocations) != 2 {
						t.Fatalf("expected plan to have 2 action invocations, got %d", len(p.Changes.ActionInvocations))
					}
					actionAddrs := []string{
						p.Changes.ActionInvocations[0].Addr.String(),
						p.Changes.ActionInvocations[1].Addr.String(),
					}
					slices.Sort(actionAddrs)
					if actionAddrs[0] != "action.test_action.hello" || actionAddrs[1] != "action.test_action.there" {
						t.Fatalf("expected action addresses to be ['action.test_action.hello', 'action.test_action.there'], got %v", actionAddrs)
					}
				},
			},
			"targeted run with not-triggered action referencing another resource that also triggers actions": {
				module: map[string]string{
					"main.tf": `
action "test_action" "hello" {}
resource "test_object" "source" {
		name = "source"
		
		lifecycle {
			action_trigger {
				events = [before_create]
				actions = [action.test_action.hello]
			}
		}
}
action "test_action" "there" {
		config {
			attr = test_object.source.name
		}
}
resource "test_object" "a" {
		lifecycle {
			action_trigger {
				events = [after_update]
				actions = [action.test_action.there]
			}
		}
}
resource "test_object" "b" {
		lifecycle {
			action_trigger {
				events = [before_update]
				actions = [action.test_action.hello]
			}
		}
}
				`,
				},
				expectPlanActionCalled: true,
				planOpts: &PlanOpts{
					Mode: plans.NormalMode,
					// Only target resource a
					Targets: []addrs.Targetable{
						addrs.RootModuleInstance.Resource(addrs.ManagedResourceMode, "test_object", "a"),
					},
				},
				assertPlanDiagnostics: func(t *testing.T, d tfdiags.Diagnostics) {
					if d.HasErrors() {
						t.Fatalf("expected no errors, got %s", d.Err().Error())
					}
				},
				assertPlan: func(t *testing.T, p *plans.Plan) {
					// Should plan for resource a and its dependency source, but not b
					if len(p.Changes.Resources) != 2 {
						t.Fatalf("expected plan to have 2 resource changes, got %d", len(p.Changes.Resources))
					}
					resourceAddrs := []string{
						p.Changes.Resources[0].Addr.String(),
						p.Changes.Resources[1].Addr.String(),
					}
					slices.Sort(resourceAddrs)
					if resourceAddrs[0] != "test_object.a" || resourceAddrs[1] != "test_object.source" {
						t.Fatalf("expected resource addresses to be ['test_object.a', 'test_object.source'], got %v", resourceAddrs)
					}

					// Should plan only the before_create action of the dependant resource
					if len(p.Changes.ActionInvocations) != 1 {
						t.Fatalf("expected plan to have 1 action invocation, got %d", len(p.Changes.ActionInvocations))
					}
					if p.Changes.ActionInvocations[0].Addr.String() != "action.test_action.hello" {
						t.Fatalf("expected action addresses to be 'action.test_action.hello', got %q", p.Changes.ActionInvocations[0].Addr.String())
					}
				},
			},
		},
	} {
		t.Run(topic, func(t *testing.T) {
			for name, tc := range tcs {
				t.Run(name, func(t *testing.T) {
					if tc.toBeImplemented {
						t.Skip("Test not implemented yet")
					}

					opts := SimplePlanOpts(plans.NormalMode, InputValues{})
					if tc.planOpts != nil {
						opts = tc.planOpts
					}

					configOpts := []configs.Option{}
					if opts.Query {
						configOpts = append(configOpts, configs.MatchQueryFiles())
					}

					m := testModuleInline(t, tc.module, configOpts...)

					p := &testing_provider.MockProvider{
						GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
							Actions: map[string]providers.ActionSchema{
								"test_action":    testActionSchema,
								"test_action_wo": writeOnlyActionSchema,
								"test_nested":    nestedActionSchema,
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
							ListResourceTypes: map[string]providers.Schema{
								"test_resource": {
									Body: &configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"data": {
												Type:     cty.DynamicPseudoType,
												Computed: true,
											},
										},
										BlockTypes: map[string]*configschema.NestedBlock{
											"config": {
												Block: configschema.Block{
													Attributes: map[string]*configschema.Attribute{
														"filter": {
															Required: true,
															NestedType: &configschema.Object{
																Nesting: configschema.NestingSingle,
																Attributes: map[string]*configschema.Attribute{
																	"attr": {
																		Type:     cty.String,
																		Required: true,
																	},
																},
															},
														},
													},
												},
												Nesting: configschema.NestingSingle,
											},
										},
									},
								},
							},
						},
						ListResourceFn: func(req providers.ListResourceRequest) providers.ListResourceResponse {
							resp := []cty.Value{}
							ret := req.Config.AsValueMap()
							maps.Copy(ret, map[string]cty.Value{
								"data": cty.TupleVal(resp),
							})
							return providers.ListResourceResponse{Result: cty.ObjectVal(ret)}
						},
					}

					other := &testing_provider.MockProvider{
						GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
							ResourceTypes: map[string]providers.Schema{
								"other_object": {
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

					ecosystem := &testing_provider.MockProvider{
						GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
							Actions: map[string]providers.ActionSchema{
								"ecosystem": {
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
						},
					}

					if tc.planActionFn != nil {
						p.PlanActionFn = func(r providers.PlanActionRequest) providers.PlanActionResponse {
							return tc.planActionFn(t, r)
						}
					}

					if tc.planResourceFn != nil {
						p.PlanResourceChangeFn = func(r providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
							return tc.planResourceFn(t, r)
						}
					}

					if tc.readResourceFn != nil {
						p.ReadResourceFn = func(r providers.ReadResourceRequest) providers.ReadResourceResponse {
							return tc.readResourceFn(t, r)
						}
					}

					ctx := testContext2(t, &ContextOpts{
						Providers: map[addrs.Provider]providers.Factory{
							// The providers never actually going to get called here, we should
							// catch the error long before anything happens.
							addrs.NewDefaultProvider("test"):  testProviderFuncFixed(p),
							addrs.NewDefaultProvider("other"): testProviderFuncFixed(other),
							{
								Type:      "ecosystem",
								Namespace: "danielmschmidt",
								Hostname:  addrs.DefaultProviderRegistryHost,
							}: testProviderFuncFixed(ecosystem),
						},
					})

					diags := ctx.Validate(m, &ValidateOpts{
						Query: opts.Query,
					})
					if tc.expectValidateDiagnostics != nil {
						tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectValidateDiagnostics(m))
					} else if tc.assertValidateDiagnostics != nil {
						tc.assertValidateDiagnostics(t, diags)
					} else {
						tfdiags.AssertNoDiagnostics(t, diags)
					}

					if diags.HasErrors() {
						return
					}

					var prevRunState *states.State
					if tc.buildState != nil {
						prevRunState = states.BuildState(tc.buildState)
					}

					plan, diags := ctx.Plan(m, prevRunState, opts)

					if tc.expectPlanDiagnostics != nil {
						tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectPlanDiagnostics(m))
					} else if tc.assertPlanDiagnostics != nil {
						tc.assertPlanDiagnostics(t, diags)
					} else {
						tfdiags.AssertNoDiagnostics(t, diags)
					}

					if tc.expectPlanActionCalled && !p.PlanActionCalled {
						t.Errorf("expected plan action to be called, but it was not")
					} else if !tc.expectPlanActionCalled && p.PlanActionCalled {
						t.Errorf("expected plan action to not be called, but it was")
					}

					if tc.assertPlan != nil {
						tc.assertPlan(t, plan)
					}
				})
			}
		})
	}
}

func TestContextPlan_validateActionInTriggerExists(t *testing.T) {
	// this validation occurs during TransformConfig
	module := `
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.test_action.hello]
    }
  }
}
`
	m := testModuleInline(t, map[string]string{"main.tf": module})
	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, nil, DefaultPlanOpts)
	if !diags.HasErrors() {
		t.Fatal("expected errors, got success!")
	}
	expectedErr := "action_trigger actions references non-existent action: The lifecycle action_trigger actions list contains a reference to the action \"action.test_action.hello\" that does not exist in the configuration of this module."
	if diags.Err().Error() != expectedErr {
		t.Fatalf("wrong error!, got %q, expected %q", diags.Err().Error(), expectedErr)
	}
}

func TestContextPlan_validateActionInTriggerExistsWithSimilarAction(t *testing.T) {
	// this validation occurs during TransformConfig
	module := `
action "test_action" "hello_word" {}
	
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.test_action.hello_world]
    }
  }
}
`
	m := testModuleInline(t, map[string]string{"main.tf": module})
	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, nil, DefaultPlanOpts)
	if !diags.HasErrors() {
		t.Fatal("expected errors, got success!")
	}
	expectedErr := "action_trigger actions references non-existent action: The lifecycle action_trigger actions list contains a reference to the action \"action.test_action.hello_world\" that does not exist in the configuration of this module. Did you mean \"action.test_action.hello_word\"?"
	if diags.Err().Error() != expectedErr {
		t.Fatalf("wrong error!, got %q, expected %q", diags.Err().Error(), expectedErr)
	}
}
