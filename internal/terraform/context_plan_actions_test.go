// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"path/filepath"
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

		expectPlanActionCalled    bool
		expectValidateDiagnostics func(m *configs.Config) tfdiags.Diagnostics
		expectPlanDiagnostics     func(m *configs.Config) tfdiags.Diagnostics
	}{
		"unreferenced": {
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
action "test_unlinked" "hello" {}
			`,
			},
			expectPlanActionCalled: false,
		},

		"invalid config": {
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
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
						Start:    hcl.Pos{Line: 5, Column: 5, Byte: 87},
						End:      hcl.Pos{Line: 5, Column: 17, Byte: 99},
					},
				})
			},
		},

		"before_create triggered": {
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
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
		},

		"after_create triggered": {
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
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
		},

		"before_update triggered - on create": {
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
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
terraform { experiments = [actions] }
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
terraform { experiments = [actions] }
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
terraform { experiments = [actions] }
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
terraform { experiments = [actions] }
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
terraform { experiments = [actions] }
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
terraform { experiments = [actions] }
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
		},

		"action for_each with auto-expansion": {
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
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
terraform { experiments = [actions] }
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
		},

		"action count with auto-expansion": {
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
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
terraform { experiments = [actions] }
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
				return diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					`action trigger #0 refers to a non-existent action instance action.test_unlinked.hello["c"]`,
					"Action instance not found in the current context.",
				))
			},
		},

		"action count invalid access": {
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
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
				return diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					`action trigger #0 refers to a non-existent action instance action.test_unlinked.hello[2]`,
					"Action instance not found in the current context.",
				))
			},
		},

		"expanded resource - unexpanded action": {
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
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
		},
		"expanded resource - expanded action": {
			toBeImplemented: true, // TODO: Not sure why this panics
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
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
		},

		"transitive dependencies": {
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
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
terraform { experiments = [actions] }
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
terraform { experiments = [actions] }
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
terraform { experiments = [actions] }
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
				return tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "Can not access actions", "Actions can not be accessed, they have no state and can only be referenced with a resources lifecycle action_trigger events list. Tried to access action.test_unlinked.my_action.")}
			},
		},

		"actions cant be accessed in outputs": {
			module: map[string]string{
				"main.tf": `
terraform { experiments = [actions] }
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
				return tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "Can not access actions", "Actions can not be accessed, they have no state and can only be referenced with a resources lifecycle action_trigger events list. Tried to access action.test_unlinked.my_action."), tfdiags.Sourceless(tfdiags.Error, "Can not access actions", "Actions can not be accessed, they have no state and can only be referenced with a resources lifecycle action_trigger events list. Tried to access action.test_unlinked.my_action.")}
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

			_, diags = ctx.Plan(m, prevRunState, SimplePlanOpts(plans.NormalMode, InputValues{}))

			if tc.expectPlanDiagnostics != nil {
				tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectPlanDiagnostics(m))
			} else {
				tfdiags.AssertNoDiagnostics(t, diags)
			}

			// TODO: add tc.assertPlan once we have a plan implementation for actions

			if tc.expectPlanActionCalled && !p.PlanActionCalled {
				t.Errorf("expected plan action to be called, but it was not")
			} else if !tc.expectPlanActionCalled && p.PlanActionCalled {
				t.Errorf("expected plan action to not be called, but it was")
			}
		})
	}
}
