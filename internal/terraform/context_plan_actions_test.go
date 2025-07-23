// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestContextPlan_actions(t *testing.T) {

	for name, tc := range map[string]struct {
		toBeImplemented    bool
		module             map[string]string
		buildState         func(*states.SyncState)
		planActionResponse *providers.PlanActionResponse
		planOpts           *PlanOpts

		expectPlanActionCalled    bool
		expectValidateDiagnostics func(m *configs.Config) tfdiags.Diagnostics
		expectPlanDiagnostics     func(m *configs.Config) tfdiags.Diagnostics
		assertPlan                func(*testing.T, *plans.Plan)
	}{
		"unreferenced": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
			`,
			},
			expectPlanActionCalled: false,

			assertPlan: func(t *testing.T, p *plans.Plan) {
				if len(p.Changes.ActionInvocations) != 0 {
					t.Fatalf("expected no actions in plan, got %d", len(p.Changes.ActionInvocations))
				}
			},
		},

		"invalid config": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {
  config {
    unknown_attr = "value"
  }
}
		`,
			},
			expectPlanActionCalled: false,
			expectValidateDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported argument",
					Detail:   `An argument named "unknown_attr" is not expected here.`,
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 4, Column: 5, Byte: 49},
						End:      hcl.Pos{Line: 4, Column: 17, Byte: 61},
					},
				})
			},
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
			expectPlanActionCalled: true,

			assertPlan: func(t *testing.T, p *plans.Plan) {
				if len(p.Changes.ActionInvocations) != 1 {
					t.Fatalf("expected 1 action in plan, got %d", len(p.Changes.ActionInvocations))
				}

				action := p.Changes.ActionInvocations[0]
				if action.Addr.String() != "action.test_unlinked.hello" {
					t.Fatalf("expected action address to be 'action.test_unlinked.hello', got '%s'", action.Addr)
				}

				if !action.TriggeringResourceAddr.Equal(mustResourceInstanceAddr("test_object.a")) {
					t.Fatal("expected action to have a triggering resource address, but it is nil")
				}

				if action.ActionTriggerBlockIndex != 0 {
					t.Fatalf("expected action to have a triggering block index of 0, got %d", action.ActionTriggerBlockIndex)
				}
				if action.TriggerEvent != configs.BeforeCreate {
					t.Fatalf("expected action to have a triggering event of 'before_create', got '%s'", action.TriggerEvent)
				}
				if action.ActionsListIndex != 0 {
					t.Fatalf("expected action to have a actions list index of 0, got %d", action.ActionsListIndex)
				}

				if action.ProviderAddr.Provider != addrs.NewDefaultProvider("test") {
					t.Fatalf("expected action to have a provider address of 'provider[\"registry.terraform.io/hashicorp/test\"]', got '%s'", action.ProviderAddr)
				}
			},
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
			expectPlanActionCalled: true,

			assertPlan: func(t *testing.T, p *plans.Plan) {
				if len(p.Changes.ActionInvocations) != 1 {
					t.Fatalf("expected 1 action in plan, got %d", len(p.Changes.ActionInvocations))
				}

				action := p.Changes.ActionInvocations[0]
				if action.Addr.String() != "action.test_unlinked.hello" {
					t.Fatalf("expected action address to be 'action.test_unlinked.hello', got '%s'", action.Addr)
				}

				// TODO: Test that action the triggering resource address is set correctly
			},
		},

		"before_update triggered - on create": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_update]
      actions = [action.test_unlinked.hello]
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
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_update]
      actions = [action.test_unlinked.hello]
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
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_update]
      actions = [action.test_unlinked.hello]
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
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_update]
      actions = [action.test_unlinked.hello]
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
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_update]
      actions = [action.test_unlinked.hello]
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
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_update]
      actions = [action.test_unlinked.hello]
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

		"action for_each": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {
  for_each = toset(["a", "b"])
  
  config {
    attr = "value-${each.key}"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello["a"], action.test_unlinked.hello["b"]]
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
					"action.test_unlinked.hello[\"a\"]",
					"action.test_unlinked.hello[\"b\"]",
				}) {
					t.Fatalf("expected action addresses to be 'action.test_unlinked.hello[\"a\"]' and 'action.test_unlinked.hello[\"b\"]', got %v", actionAddrs)
				}

				// TODO: Test that action the triggering resource address is set correctly
			},
		},

		"action for_each with auto-expansion": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {
  for_each = toset(["a", "b"])
  
  config {
    attr = "value-${each.key}"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello] # This will auto-expand to action.test_unlinked.hello["a"] and action.test_unlinked.hello["b"]
    }
  }
}
`,
			},
			expectPlanActionCalled: true,
		},

		"action count": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {
  count = 2

  config {
    attr = "value-${count.index}"
  }
}

resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello[0], action.test_unlinked.hello[1]]
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
					"action.test_unlinked.hello[0]",
					"action.test_unlinked.hello[1]",
				}) {
					t.Fatalf("expected action addresses to be 'action.test_unlinked.hello[0]' and 'action.test_unlinked.hello[1]', got %v", actionAddrs)
				}

				// TODO: Test that action the triggering resource address is set correctly
			},
		},

		"action count with auto-expansion": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {
  count = 2

  config {
    attr = "value-${count.index}"
  }
}

resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello] # This will auto-expand to action.test_unlinked.hello[0] and action.test_unlinked.hello[1]
    }
  }
}
`,
			},
			expectPlanActionCalled: true,
		},

		"action for_each invalid access": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {
  for_each = toset(["a", "b"])

  config {
    attr = "value-${each.key}"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello["c"]]
    }
  }
}
`,
			},
			expectPlanActionCalled: false,
			expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to non-existant action instance",
					Detail:   "Action instance was not found in the current context.",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 13, Column: 18, Byte: 226},
						End:      hcl.Pos{Line: 13, Column: 49, Byte: 257},
					},
				})
			},
		},

		"action count invalid access": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {
  count = 2

  config {
    attr = "value-${count.index}"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello[2]]
    }
  }
}
`,
			},
			expectPlanActionCalled: false,
			expectPlanDiagnostics: func(m *configs.Config) (diags tfdiags.Diagnostics) {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to non-existant action instance",
					Detail:   "Action instance was not found in the current context.",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 13, Column: 18, Byte: 210},
						End:      hcl.Pos{Line: 13, Column: 47, Byte: 239},
					},
				})
			},
		},

		"expanded resource - unexpanded action": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  count = 2
  name = "test-${count.index}"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello]
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
					"action.test_unlinked.hello",
					"action.test_unlinked.hello",
				}) {
					t.Fatalf("expected action addresses to be 'action.test_unlinked.hello' and 'action.test_unlinked.hello', got %v", actionAddrs)
				}

				// TODO: Test that action the triggering resource address is set correctly
			},
		},
		"expanded resource - expanded action": {
			toBeImplemented: true, // TODO: Not sure why this panics
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {
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
      actions = [action.test_unlinked.hello[count.index]]
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
					"action.test_unlinked.hello[0]",
					"action.test_unlinked.hello[1]",
				}) {
					t.Fatalf("expected action addresses to be 'action.test_unlinked.hello[0]' and 'action.test_unlinked.hello[1]', got %v", actionAddrs)
				}

				// TODO: Test that action the triggering resource address is set correctly
			},
		},

		"transitive dependencies": {
			module: map[string]string{
				"main.tf": `
resource "test_object" "a" {
  name = "a"
}
action "test_unlinked" "hello" {
  config {
    attr = test_object.a.name
  }
}
resource "test_object" "b" {
  name = "b"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello]
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
action "test_unlinked" "hello_a" {
  config {
    attr = test_object.a.name
  }
}
action "test_unlinked" "hello_b" {
  config {
    attr = test_object.a.name
  }
}
resource "test_object" "c" {
  name = "c"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello_a]
    }
  }
}
resource "test_object" "d" {
  name = "d"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello_b]
    }
  }
}
resource "test_object" "e" {
  name = "e"
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello_a, action.test_unlinked.hello_b]
    }
  }
}
`,
			},
			expectPlanActionCalled: true,
		},

		"failing actions cancel next ones": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "failure" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.failure, action.test_unlinked.failure]
    }
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.failure]
    }
  }
}
`,
			},

			planActionResponse: &providers.PlanActionResponse{
				Diagnostics: tfdiags.Diagnostics{
					tfdiags.Sourceless(tfdiags.Error, "Planning failed", "Test case simulates an error while planning"),
				},
			},

			expectPlanActionCalled: true,
			// We only expect a single diagnostic here, the other should not have been called because the first one failed.
			expectPlanDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{
					tfdiags.Sourceless(tfdiags.Error, "Planning failed", "Test case simulates an error while planning"),
				}
			},
		},

		"actions cant be accessed in resources": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "my_action" {
  config {
    attr = "value"
  }
}
resource "test_object" "a" {
  name = action.test_unlinked.my_action.attr
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.my_action]
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
						Detail:   "Actions can't be referenced in this context, they can only be referenced from within a resources lifecycle events list.",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 8, Column: 10, Byte: 112},
							End:      hcl.Pos{Line: 8, Column: 40, Byte: 142},
						},
					})
			},
		},

		"actions cant be accessed in outputs": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "my_action" {
  config {
    attr = "value"
  }
}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.my_action]
    }
  }
}

output "my_output" {
    value = action.test_unlinked.my_action.attr
}

output "my_output2" {
    value = action.test_unlinked.my_action
}
`,
			},
			expectValidateDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid reference",
						Detail:   "Actions can't be referenced in this context, they can only be referenced from within a resources lifecycle events list.",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 21, Column: 13, Byte: 337},
							End:      hcl.Pos{Line: 21, Column: 43, Byte: 367},
						},
					}).Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid reference",
						Detail:   "Actions can't be referenced in this context, they can only be referenced from within a resources lifecycle events list.",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 17, Column: 13, Byte: 264},
							End:      hcl.Pos{Line: 17, Column: 43, Byte: 294},
						},
					},
				)
			},
		},

		"destroy run": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.test_unlinked.hello]
    }
  }
}
`,
			},
			expectPlanActionCalled: false,
			planOpts:               SimplePlanOpts(plans.DestroyMode, InputValues{}),
		},

		// Since if we just destroy a node there is no reference to an action in config, we try
		// to provoke an error by just removing a resource instance.
		"destroying expanded node": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  count = 2
  lifecycle {
    action_trigger {
      events = [before_create, after_update]
      actions = [action.test_unlinked.hello]
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
	} {
		t.Run(name, func(t *testing.T) {
			if tc.toBeImplemented {
				t.Skip("Test not implemented yet")
			}

			m := testModuleInline(t, tc.module)

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
			}

			if tc.planActionResponse != nil {
				p.PlanActionResponse = *tc.planActionResponse
			}

			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					// The providers never actually going to get called here, we should
					// catch the error long before anything happens.
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
				},
			})

			diags := ctx.Validate(m, &ValidateOpts{})
			if tc.expectValidateDiagnostics != nil {
				tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectValidateDiagnostics(m))
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

			opts := SimplePlanOpts(plans.NormalMode, InputValues{})
			if tc.planOpts != nil {
				opts = tc.planOpts
			}

			plan, diags := ctx.Plan(m, prevRunState, opts)

			if tc.expectPlanDiagnostics != nil {
				tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectPlanDiagnostics(m))
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
}
