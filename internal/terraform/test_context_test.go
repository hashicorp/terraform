// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestTestContext_Evaluate(t *testing.T) {
	tcs := map[string]struct {
		configs     map[string]string
		state       *states.State
		plan        *plans.Plan
		variables   InputValues
		provider    *MockProvider
		priorStates map[string]func(config *configs.Config) *TestContext

		expectedDiags  []tfdiags.Description
		expectedStatus moduletest.Status
	}{
		"basic_passing": {
			configs: map[string]string{
				"main.tf": `
resource "test_resource" "a" {
	value = "Hello, world!"
}
`,
				"main.tftest.hcl": `
run "test_case" {
	assert {
		condition = test_resource.a.value == "Hello, world!"
		error_message = "invalid value"
	}
}
`,
			},
			plan: &plans.Plan{
				Changes: plans.NewChanges(),
			},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "a",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: encodeCtyValue(t, cty.ObjectVal(map[string]cty.Value{
							"value": cty.StringVal("Hello, world!"),
						})),
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			provider: &MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					ResourceTypes: map[string]providers.Schema{
						"test_resource": {
							Block: &configschema.Block{
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
			expectedStatus: moduletest.Pass,
		},
		"with_variables": {
			configs: map[string]string{
				"main.tf": `
variable "value" {
	type = string
}

resource "test_resource" "a" {
	value = var.value
}
`,
				"main.tftest.hcl": `
variables {
	value = "Hello, world!"
}

run "test_case" {
	assert {
		condition = test_resource.a.value == var.value
		error_message = "invalid value"
	}
}
`,
			},
			plan: &plans.Plan{
				Changes: plans.NewChanges(),
			},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "a",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: encodeCtyValue(t, cty.ObjectVal(map[string]cty.Value{
							"value": cty.StringVal("Hello, world!"),
						})),
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			variables: InputValues{
				"value": {
					Value: cty.StringVal("Hello, world!"),
				},
			},
			provider: &MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					ResourceTypes: map[string]providers.Schema{
						"test_resource": {
							Block: &configschema.Block{
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
			expectedStatus: moduletest.Pass,
		},
		"basic_failing": {
			configs: map[string]string{
				"main.tf": `
resource "test_resource" "a" {
	value = "Hello, world!"
}
`,
				"main.tftest.hcl": `
run "test_case" {
	assert {
		condition = test_resource.a.value == "incorrect!"
		error_message = "invalid value"
	}
}
`,
			},
			plan: &plans.Plan{
				Changes: plans.NewChanges(),
			},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "a",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: encodeCtyValue(t, cty.ObjectVal(map[string]cty.Value{
							"value": cty.StringVal("Hello, world!"),
						})),
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			provider: &MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					ResourceTypes: map[string]providers.Schema{
						"test_resource": {
							Block: &configschema.Block{
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
			expectedStatus: moduletest.Fail,
			expectedDiags: []tfdiags.Description{
				{
					Summary: "Test assertion failed",
					Detail:  "invalid value",
				},
			},
		},
		"two_failing_assertions": {
			configs: map[string]string{
				"main.tf": `
resource "test_resource" "a" {
	value = "Hello, world!"
}
`,
				"main.tftest.hcl": `
run "test_case" {
	assert {
		condition = test_resource.a.value == "incorrect!"
		error_message = "invalid value"
	}

    assert {
        condition = test_resource.a.value == "also incorrect!"
        error_message = "still invalid"
    }
}
`,
			},
			plan: &plans.Plan{
				Changes: plans.NewChanges(),
			},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "a",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: encodeCtyValue(t, cty.ObjectVal(map[string]cty.Value{
							"value": cty.StringVal("Hello, world!"),
						})),
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			provider: &MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					ResourceTypes: map[string]providers.Schema{
						"test_resource": {
							Block: &configschema.Block{
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
			expectedStatus: moduletest.Fail,
			expectedDiags: []tfdiags.Description{
				{
					Summary: "Test assertion failed",
					Detail:  "invalid value",
				},
				{
					Summary: "Test assertion failed",
					Detail:  "still invalid",
				},
			},
		},
		"sensitive_variables": {
			configs: map[string]string{
				"main.tf": `
variable "input" {
  type = string
  sensitive = true
}
`,
				"main.tftest.hcl": `
run "test" {
  variables {
    input = "Hello, world!"
  }

  assert {
    condition = var.input == "Hello, world!"
    error_message = "bad"
  }
}
`,
			},
			plan: &plans.Plan{
				Changes: plans.NewChanges(),
			},
			state: states.NewState(),
			variables: InputValues{
				"input": &InputValue{
					Value:      cty.StringVal("Hello, world!"),
					SourceType: ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "main.tftest.hcl",
						Start:    tfdiags.SourcePos{Line: 3, Column: 13, Byte: 12},
						End:      tfdiags.SourcePos{Line: 3, Column: 28, Byte: 27},
					},
				},
			},
			provider:       &MockProvider{},
			expectedStatus: moduletest.Pass,
			expectedDiags:  []tfdiags.Description{},
		},
		"sensitive_variables_fail": {
			configs: map[string]string{
				"main.tf": `
variable "input" {
  type = string
  sensitive = true
}
`,
				"main.tftest.hcl": `
run "test" {
  variables {
    input = "Hello, world!"
  }

  assert {
    condition = var.input == "Hello, universe!"
    error_message = "bad ${var.input}"
  }
}
`,
			},
			plan: &plans.Plan{
				Changes: plans.NewChanges(),
			},
			state: states.NewState(),
			variables: InputValues{
				"input": &InputValue{
					Value:      cty.StringVal("Hello, world!"),
					SourceType: ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "main.tftest.hcl",
						Start:    tfdiags.SourcePos{Line: 3, Column: 13, Byte: 12},
						End:      tfdiags.SourcePos{Line: 3, Column: 28, Byte: 27},
					},
				},
			},
			provider:       &MockProvider{},
			expectedStatus: moduletest.Fail,
			expectedDiags: []tfdiags.Description{
				{
					Summary: "Error message refers to sensitive values",
					Detail:  "The error expression used to explain this condition refers to sensitive values, so Terraform will not display the resulting message.\n\nYou can correct this by removing references to sensitive values, or by carefully using the nonsensitive() function if the expression will not reveal the sensitive data.",
				},
				{
					Summary: "Test assertion failed",
				},
			},
		},
		"basic_passing_with_plan": {
			configs: map[string]string{
				"main.tf": `
resource "test_resource" "a" {
	value = "Hello, world!"
}
`,
				"main.tftest.hcl": `
run "test_case" {
	command = plan

	assert {
		condition = test_resource.a.value == "Hello, world!"
		error_message = "invalid value"
	}
}
`,
			},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "a",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectPlanned,
						AttrsJSON: encodeCtyValue(t, cty.NullVal(cty.Object(map[string]cty.Type{
							"value": cty.String,
						}))),
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			plan: &plans.Plan{
				Changes: &plans.Changes{
					Resources: []*plans.ResourceInstanceChangeSrc{
						{
							Addr: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "test_resource",
								Name: "a",
							}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
							ProviderAddr: addrs.AbsProviderConfig{
								Module:   addrs.RootModule,
								Provider: addrs.NewDefaultProvider("test"),
							},
							ChangeSrc: plans.ChangeSrc{
								Action: plans.Create,
								Before: nil,
								After: encodeDynamicValue(t, cty.ObjectVal(map[string]cty.Value{
									"value": cty.StringVal("Hello, world!"),
								})),
							},
						},
					},
				},
			},
			provider: &MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					ResourceTypes: map[string]providers.Schema{
						"test_resource": {
							Block: &configschema.Block{
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
			expectedStatus: moduletest.Pass,
		},
		"basic_failing_with_plan": {
			configs: map[string]string{
				"main.tf": `
resource "test_resource" "a" {
	value = "Hello, world!"
}
`,
				"main.tftest.hcl": `
run "test_case" {
	command = plan

	assert {
		condition = test_resource.a.value == "incorrect!"
		error_message = "invalid value"
	}
}
`,
			},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "a",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectPlanned,
						AttrsJSON: encodeCtyValue(t, cty.NullVal(cty.Object(map[string]cty.Type{
							"value": cty.String,
						}))),
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			plan: &plans.Plan{
				Changes: &plans.Changes{
					Resources: []*plans.ResourceInstanceChangeSrc{
						{
							Addr: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "test_resource",
								Name: "a",
							}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
							ProviderAddr: addrs.AbsProviderConfig{
								Module:   addrs.RootModule,
								Provider: addrs.NewDefaultProvider("test"),
							},
							ChangeSrc: plans.ChangeSrc{
								Action: plans.Create,
								Before: nil,
								After: encodeDynamicValue(t, cty.ObjectVal(map[string]cty.Value{
									"value": cty.StringVal("Hello, world!"),
								})),
							},
						},
					},
				},
			},
			provider: &MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					ResourceTypes: map[string]providers.Schema{
						"test_resource": {
							Block: &configschema.Block{
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
			expectedStatus: moduletest.Fail,
			expectedDiags: []tfdiags.Description{
				{
					Summary: "Test assertion failed",
					Detail:  "invalid value",
				},
			},
		},
		"with_prior_state": {
			configs: map[string]string{
				"main.tf": `
resource "test_resource" "a" {
	value = "Hello, world!"
}
`,
				"main.tftest.hcl": `
run "setup" {}

run "test_case" {
	assert {
		condition = test_resource.a.value == run.setup.value
		error_message = "invalid value"
	}
}
`,
			},
			plan: &plans.Plan{
				Changes: plans.NewChanges(),
			},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "a",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: encodeCtyValue(t, cty.ObjectVal(map[string]cty.Value{
							"value": cty.StringVal("Hello, world!"),
						})),
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			priorStates: map[string]func(config *configs.Config) *TestContext{
				"setup": func(config *configs.Config) *TestContext {
					return &TestContext{
						Context: &Context{},
						Run: &moduletest.Run{
							Config: config.Module.Tests["main.tftest.hcl"].Runs[0],
							Name:   "setup",
						},
						Config: &configs.Config{
							Module: &configs.Module{
								Outputs: map[string]*configs.Output{
									"value": {
										Name: "value",
									},
								},
							},
						},
						Plan: &plans.Plan{
							Changes: plans.NewChanges(),
						},
						State: states.BuildState(func(state *states.SyncState) {
							state.SetOutputValue(addrs.AbsOutputValue{
								Module: addrs.RootModuleInstance,
								OutputValue: addrs.OutputValue{
									Name: "value",
								},
							}, cty.StringVal("Hello, world!"), false)
						}),
					}
				},
			},
			provider: &MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					ResourceTypes: map[string]providers.Schema{
						"test_resource": {
							Block: &configschema.Block{
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
			expectedStatus: moduletest.Pass,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			config := testModuleInline(t, tc.configs)
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(tc.provider),
				},
			})

			priorStates := make(map[string]*TestContext)
			for run, ps := range tc.priorStates {
				priorStates[run] = ps(config)
			}

			file := config.Module.Tests["main.tftest.hcl"]
			run := moduletest.Run{
				Config: file.Runs[len(file.Runs)-1], // We always simulate the last run block.
				Name:   "test_case",                 // and it should be named test_case
			}

			tctx := ctx.TestContext(&run, config, tc.state, tc.plan, tc.variables)
			tctx.Evaluate(priorStates)

			if expected, actual := tc.expectedStatus, run.Status; expected != actual {
				t.Errorf("expected status \"%s\" but got \"%s\"", expected, actual)
			}

			compareDiagnosticsFromTestResult(t, tc.expectedDiags, run.Diagnostics)
		})
	}
}

func compareDiagnosticsFromTestResult(t *testing.T, expected []tfdiags.Description, actual tfdiags.Diagnostics) {
	if len(expected) != len(actual) {
		t.Errorf("found invalid number of diagnostics, expected %d but found %d", len(expected), len(actual))
	}

	length := len(expected)
	if len(actual) > length {
		length = len(actual)
	}

	for ix := 0; ix < length; ix++ {
		if ix >= len(expected) {
			t.Errorf("found extra diagnostic at %d:\n%v", ix, actual[ix].Description())
		} else if ix >= len(actual) {
			t.Errorf("missing diagnostic at %d:\n%v", ix, expected[ix])
		} else {
			expected := expected[ix]
			actual := actual[ix].Description()
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Errorf("found different diagnostics at %d:\nexpected:\n%s\nactual:\n%s\ndiff:%s", ix, expected, actual, diff)
			}
		}
	}
}

func encodeDynamicValue(t *testing.T, value cty.Value) []byte {
	data, err := ctymsgpack.Marshal(value, value.Type())
	if err != nil {
		t.Fatalf("failed to marshal JSON: %s", err)
	}
	return data
}

func encodeCtyValue(t *testing.T, value cty.Value) []byte {
	data, err := ctyjson.Marshal(value, value.Type())
	if err != nil {
		t.Fatalf("failed to marshal JSON: %s", err)
	}
	return data
}
