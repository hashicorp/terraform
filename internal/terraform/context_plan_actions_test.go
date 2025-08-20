// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContextPlan_actions(t *testing.T) {
	for name, tc := range map[string]struct {
		toBeImplemented bool
		module          map[string]string
		buildState      func(*states.SyncState)
		planActionFn    func(*testing.T, providers.PlanActionRequest) providers.PlanActionResponse
		planResourceFn  func(*testing.T, providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse
		planOpts        *PlanOpts

		expectPlanActionCalled bool

		// Some tests can produce race-conditions in the error messages, so we
		// have two ways of checking the diagnostics. Use expectValidateDiagnostics
		// by default, if there is a race condition and you want to allow multiple
		// versions, please use assertValidateDiagnostics.
		expectValidateDiagnostics func(m *configs.Config) tfdiags.Diagnostics
		assertValidateDiagnostics func(*testing.T, tfdiags.Diagnostics)

		expectPlanDiagnostics func(m *configs.Config) tfdiags.Diagnostics
		assertPlan            func(*testing.T, *plans.Plan)
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

				at, ok := action.ActionTrigger.(plans.LifecycleActionTrigger)
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
			toBeImplemented: true, // TODO: Look into this
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
			toBeImplemented: true, // TODO: Look into this
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
							Start:    hcl.Pos{Line: 7, Column: 8, Byte: 149},
							End:      hcl.Pos{Line: 7, Column: 46, Byte: 177},
						},
					},
				)
			},
		},

		"actions with warnings don't cancel": {
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
							Start:    hcl.Pos{Line: 7, Column: 8, Byte: 149},
							End:      hcl.Pos{Line: 7, Column: 46, Byte: 177},
						},
					},
					&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "Warnings when planning action",
						Detail:   "Warning during planning: Test case simulates a warning while planning",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 7, Column: 48, Byte: 179},
							End:      hcl.Pos{Line: 7, Column: 76, Byte: 207},
						},
					},
					&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "Warnings when planning action",
						Detail:   "Warning during planning: Test case simulates a warning while planning",
						Subject: &hcl.Range{
							Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
							Start:    hcl.Pos{Line: 11, Column: 8, Byte: 284},
							End:      hcl.Pos{Line: 11, Column: 46, Byte: 312},
						},
					},
				)
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
		// We don't yet support these action types yet
		"fails with lifecycle actions": {
			module: map[string]string{
				"main.tf": `
action "test_lifecycle" "hello" {}
`,
			},
			expectValidateDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Lifecycle actions are not supported",
					Detail:   "This version of Terraform does not support lifecycle actions",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
						End:      hcl.Pos{Line: 2, Column: 32, Byte: 32},
					},
				})
			},
		},
		"fails with linked actions": {
			module: map[string]string{
				"main.tf": `
action "test_linked" "hello" {}
`,
			},
			expectValidateDiagnostics: func(m *configs.Config) tfdiags.Diagnostics {
				return tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Linked actions are not supported",
					Detail:   "This version of Terraform does not support linked actions",
					Subject: &hcl.Range{
						Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
						Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
						End:      hcl.Pos{Line: 2, Column: 29, Byte: 29},
					},
				})
			},
		},

		"triggered within module": {
			module: map[string]string{
				"main.tf": `
module "mod" {
    source = "./mod"
}
`,
				"mod/mod.tf": `
action "test_unlinked" "hello" {}
resource "other_object" "a" {
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
				if action.Addr.String() != "module.mod.action.test_unlinked.hello" {
					t.Fatalf("expected action address to be 'module.mod.action.test_unlinked.hello', got '%s'", action.Addr)
				}

				at, ok := action.ActionTrigger.(plans.LifecycleActionTrigger)
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
action "test_unlinked" "hello" {}
resource "other_object" "a" {
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
					t.Fatalf("expected 1 action in plan, got %d", len(p.Changes.ActionInvocations))
				}

				// We know we are run within two child modules, so we can just sort by the triggering resource address
				slices.SortFunc(p.Changes.ActionInvocations, func(a, b *plans.ActionInvocationInstanceSrc) int {
					at, ok := a.ActionTrigger.(plans.LifecycleActionTrigger)
					if !ok {
						t.Fatalf("expected action trigger to be a LifecycleActionTrigger, got %T", a.ActionTrigger)
					}
					bt, ok := b.ActionTrigger.(plans.LifecycleActionTrigger)
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
				if action.Addr.String() != "module.mod[0].action.test_unlinked.hello" {
					t.Fatalf("expected action address to be 'module.mod[0].action.test_unlinked.hello', got '%s'", action.Addr)
				}

				at := action.ActionTrigger.(plans.LifecycleActionTrigger)

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
				if action2.Addr.String() != "module.mod[1].action.test_unlinked.hello" {
					t.Fatalf("expected action address to be 'module.mod[1].action.test_unlinked.hello', got '%s'", action2.Addr)
				}

				a2t := action2.ActionTrigger.(plans.LifecycleActionTrigger)

				if !a2t.TriggeringResourceAddr.Equal(mustResourceInstanceAddr("module.mod[1].other_object.a")) {
					t.Fatalf("expected action to have triggering resource address 'module.mod[1].other_object.a', but it is %s", a2t.TriggeringResourceAddr)
				}
			},
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
action "test_unlinked" "hello" {
  provider = test.inthemodule
}
resource "other_object" "a" {
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
				if action.Addr.String() != "module.mod.action.test_unlinked.hello" {
					t.Fatalf("expected action address to be 'module.mod.action.test_unlinked.hello', got '%s'", action.Addr)
				}

				at, ok := action.ActionTrigger.(plans.LifecycleActionTrigger)
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
action "ecosystem_unlinked" "hello" {}
resource "other_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.ecosystem_unlinked.hello]
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
				if action.Addr.String() != "action.ecosystem_unlinked.hello" {
					t.Fatalf("expected action address to be 'action.ecosystem_unlinked.hello', got '%s'", action.Addr)
				}
				at, ok := action.ActionTrigger.(plans.LifecycleActionTrigger)
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

		"aliased provider": {
			module: map[string]string{
				"main.tf": `
provider "test" {
  alias = "aliased"
}
action "test_unlinked" "hello" {
  provider = test.aliased
}
resource "other_object" "a" {
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

				at, ok := action.ActionTrigger.(plans.LifecycleActionTrigger)
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

		"action config with after_create dependency to triggering resource": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {
  config {
    attr = test_object.a.name
  }
}
resource "test_object" "a" {
  name = "test_name"
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
					t.Fatalf("expected one action in plan, got %d", len(p.Changes.ActionInvocations))
				}

				if p.Changes.ActionInvocations[0].ActionTrigger.TriggerEvent() != configs.AfterCreate {
					t.Fatalf("expected trigger event to be of type AfterCreate, got: %v", p.Changes.ActionInvocations[0].ActionTrigger)
				}

				if p.Changes.ActionInvocations[0].Addr.Action.String() != "action.test_unlinked.hello" {
					t.Fatalf("expected action to equal 'action.test_unlinked.hello', got '%s'", p.Changes.ActionInvocations[0].Addr)
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
			toBeImplemented: true,
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {
  config {
    attr = test_object.a.name
  }
}
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
			expectPlanActionCalled: false,
			assertValidateDiagnostics: func(t *testing.T, diags tfdiags.Diagnostics) {
				if !diags.HasErrors() {
					t.Fatalf("expected diagnostics to have errors, but it does not")
				}
				if len(diags) != 1 {
					t.Fatalf("expected diagnostics to have 1 error, but it has %d", len(diags))
				}
				if diags[0].Description().Summary != "Cycle: test_object.a, action.test_unlinked.hello (expand)" && diags[0].Description().Summary != "Cycle: action.test_unlinked.hello (expand), test_object.a" {
					t.Fatalf("expected diagnostic to have summary 'Cycle: test_object.a, action.test_unlinked.hello (expand)' or 'Cycle: action.test_unlinked.hello (expand), test_object.a', but got '%s'", diags[0].Description().Summary)
				}
			},
		},
		"provider deferring action while not allowed": {
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
						`The provider signaled a deferred action for "action.test_unlinked.hello", but in this context deferrals are disabled. This is a bug in the provider, please file an issue with the provider developers.`,
					),
				}
			},
		},

		"provider deferring action": {
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
			planOpts: &PlanOpts{
				Mode:            plans.NormalMode,
				DeferralAllowed: true,
			},
			planActionFn: func(*testing.T, providers.PlanActionRequest) providers.PlanActionResponse {
				return providers.PlanActionResponse{
					Deferred: &providers.Deferred{
						Reason: providers.DeferredReasonAbsentPrereq,
					},
				}
			},

			assertPlan: func(t *testing.T, p *plans.Plan) {
				if len(p.Changes.ActionInvocations) != 0 {
					t.Fatalf("expected 0 actions in plan, got %d", len(p.Changes.ActionInvocations))
				}

				if len(p.DeferredActionInvocations) != 1 {
					t.Fatalf("expected 1 deferred action in plan, got %d", len(p.DeferredActionInvocations))
				}
				deferredActionInvocation := p.DeferredActionInvocations[0]
				if deferredActionInvocation.DeferredReason != providers.DeferredReasonAbsentPrereq {
					t.Fatalf("expected deferred action to be deferred due to absent prereq, but got %s", deferredActionInvocation.DeferredReason)
				}
				if deferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String() != "test_object.a" {
					t.Fatalf("expected deferred action to be triggered by test_object.a, but got %s", deferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String())
				}

				if deferredActionInvocation.ActionInvocationInstanceSrc.Addr.String() != "action.test_unlinked.hello" {
					t.Fatalf("expected deferred action to be triggered by action.test_unlinked.hello, but got %s", deferredActionInvocation.ActionInvocationInstanceSrc.Addr.String())
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
action "test_unlinked" "hello" {}
action "ecosystem_unlinked" "world" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.test_unlinked.hello, action.ecosystem_unlinked.world]
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
				if r.ActionType == "ecosystem_unlinked" {
					t.Fatalf("expected second action to not be planned, but it was planned")
				}
				return providers.PlanActionResponse{
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
				firstDeferredActionInvocation := p.DeferredActionInvocations[0]
				if firstDeferredActionInvocation.DeferredReason != providers.DeferredReasonAbsentPrereq {
					t.Fatalf("expected deferred action to be deferred due to absent prereq, but got %s", firstDeferredActionInvocation.DeferredReason)
				}
				if firstDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String() != "test_object.a" {
					t.Fatalf("expected deferred action to be triggered by test_object.a, but got %s", firstDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String())
				}

				if firstDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String() != "action.test_unlinked.hello" {
					t.Fatalf("expected deferred action to be triggered by action.test_unlinked.hello, but got %s", firstDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String())
				}

				secondDeferredActionInvocation := p.DeferredActionInvocations[1]
				if secondDeferredActionInvocation.DeferredReason != providers.DeferredReasonDeferredPrereq {
					t.Fatalf("expected second deferred action to be deferred due to deferred prereq, but got %s", secondDeferredActionInvocation.DeferredReason)
				}
				if secondDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String() != "test_object.a" {
					t.Fatalf("expected second deferred action to be triggered by test_object.a, but got %s", secondDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String())
				}

				if secondDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String() != "action.ecosystem_unlinked.world" {
					t.Fatalf("expected second deferred action to be triggered by action.ecosystem_unlinked.world, but got %s", secondDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String())
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
action "test_unlinked" "hello" {}
action "ecosystem_unlinked" "world" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello]
    }
    action_trigger {
      events = [after_create]
      actions = [action.ecosystem_unlinked.world]
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
				if r.ActionType == "ecosystem_unlinked" {
					t.Fatalf("expected second action to not be planned, but it was planned")
				}
				return providers.PlanActionResponse{
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
				firstDeferredActionInvocation := p.DeferredActionInvocations[0]
				if firstDeferredActionInvocation.DeferredReason != providers.DeferredReasonAbsentPrereq {
					t.Fatalf("expected deferred action to be deferred due to absent prereq, but got %s", firstDeferredActionInvocation.DeferredReason)
				}
				if firstDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String() != "test_object.a" {
					t.Fatalf("expected deferred action to be triggered by test_object.a, but got %s", firstDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String())
				}

				if firstDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String() != "action.test_unlinked.hello" {
					t.Fatalf("expected deferred action to be triggered by action.test_unlinked.hello, but got %s", firstDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String())
				}

				secondDeferredActionInvocation := p.DeferredActionInvocations[1]
				if secondDeferredActionInvocation.DeferredReason != providers.DeferredReasonDeferredPrereq {
					t.Fatalf("expected second deferred action to be deferred due to deferred prereq, but got %s", secondDeferredActionInvocation.DeferredReason)
				}
				if secondDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String() != "test_object.a" {
					t.Fatalf("expected second deferred action to be triggered by test_object.a, but got %s", secondDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String())
				}

				if secondDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String() != "action.ecosystem_unlinked.world" {
					t.Fatalf("expected second deferred action to be triggered by action.ecosystem_unlinked.world, but got %s", secondDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String())
				}

				if len(p.DeferredResources) != 1 {
					t.Fatalf("expected 1 resource to be deferred, got %d", len(p.DeferredResources))
				}
				deferredResource := p.DeferredResources[0]

				if deferredResource.ChangeSrc.Addr.String() != "test_object.a" {
					t.Fatalf("Expected resource %s to be deferred, but it was not", deferredResource.ChangeSrc.Addr)
				}

				if deferredResource.DeferredReason != providers.DeferredReasonDeferredPrereq {
					t.Fatalf("Expected deferred reason to be deferred prereq, got %s", deferredResource.DeferredReason)
				}
			},
		},

		"deferred resources also defer the actions they trigger": {
			module: map[string]string{
				"main.tf": `
action "test_unlinked" "hello" {}
resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events = [before_create]
      actions = [action.test_unlinked.hello]
    }
    action_trigger {
      events = [after_create]
      actions = [action.test_unlinked.hello]
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
				firstDeferredActionInvocation := p.DeferredActionInvocations[0]
				if firstDeferredActionInvocation.DeferredReason != providers.DeferredReasonDeferredPrereq {
					t.Fatalf("expected deferred action to be deferred due to deferred prereq, but got %s", firstDeferredActionInvocation.DeferredReason)
				}
				if firstDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String() != "test_object.a" {
					t.Fatalf("expected deferred action to be triggered by test_object.a, but got %s", firstDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String())
				}

				if firstDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String() != "action.test_unlinked.hello" {
					t.Fatalf("expected deferred action to be triggered by action.test_unlinked.hello, but got %s", firstDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String())
				}

				secondDeferredActionInvocation := p.DeferredActionInvocations[1]
				if secondDeferredActionInvocation.DeferredReason != providers.DeferredReasonDeferredPrereq {
					t.Fatalf("expected second deferred action to be deferred due to deferred prereq, but got %s", secondDeferredActionInvocation.DeferredReason)
				}
				if secondDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String() != "test_object.a" {
					t.Fatalf("expected second deferred action to be triggered by test_object.a, but got %s", secondDeferredActionInvocation.ActionInvocationInstanceSrc.ActionTrigger.(plans.LifecycleActionTrigger).TriggeringResourceAddr.String())
				}

				if secondDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String() != "action.test_unlinked.hello" {
					t.Fatalf("expected second deferred action to be triggered by action.test_unlinked.hello, but got %s", secondDeferredActionInvocation.ActionInvocationInstanceSrc.Addr.String())
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

						"test_lifecycle": {
							ConfigSchema: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"attr": {
										Type:     cty.String,
										Optional: true,
									},
								},
							},

							Lifecycle: &providers.LifecycleAction{
								LinkedResource: providers.LinkedResourceSchema{
									TypeName: "test_object",
								},
							},
						},

						"test_linked": {
							ConfigSchema: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"attr": {
										Type:     cty.String,
										Optional: true,
									},
								},
							},

							Linked: &providers.LinkedAction{
								LinkedResources: []providers.LinkedResourceSchema{
									{
										TypeName: "test_object",
									},
								},
							},
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
						"ecosystem_unlinked": {
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

			diags := ctx.Validate(m, &ValidateOpts{})
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
