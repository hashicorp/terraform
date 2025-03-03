// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestEvalContext_Evaluate(t *testing.T) {
	tests := map[string]struct {
		configs      map[string]string
		state        *states.State
		plan         *plans.Plan
		variables    terraform.InputValues
		testOnlyVars terraform.InputValues
		provider     *testing_provider.MockProvider
		priorOutputs map[string]cty.Value

		expectedDiags   []tfdiags.Description
		expectedStatus  moduletest.Status
		expectedOutputs cty.Value
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
				Changes: plans.NewChangesSrc(),
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
			provider: &testing_provider.MockProvider{
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
			expectedStatus:  moduletest.Pass,
			expectedOutputs: cty.EmptyObjectVal,
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
				Changes: plans.NewChangesSrc(),
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
			variables: terraform.InputValues{
				"value": {
					Value: cty.StringVal("Hello, world!"),
				},
			},
			provider: &testing_provider.MockProvider{
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
			expectedStatus:  moduletest.Pass,
			expectedOutputs: cty.EmptyObjectVal,
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
				Changes: plans.NewChangesSrc(),
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
			provider: &testing_provider.MockProvider{
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
			expectedStatus:  moduletest.Fail,
			expectedOutputs: cty.EmptyObjectVal,
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
				Changes: plans.NewChangesSrc(),
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
			provider: &testing_provider.MockProvider{
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
			expectedStatus:  moduletest.Fail,
			expectedOutputs: cty.EmptyObjectVal,
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
				Changes: plans.NewChangesSrc(),
			},
			state: states.NewState(),
			variables: terraform.InputValues{
				"input": &terraform.InputValue{
					Value:      cty.StringVal("Hello, world!"),
					SourceType: terraform.ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "main.tftest.hcl",
						Start:    tfdiags.SourcePos{Line: 3, Column: 13, Byte: 12},
						End:      tfdiags.SourcePos{Line: 3, Column: 28, Byte: 27},
					},
				},
			},
			provider:        &testing_provider.MockProvider{},
			expectedStatus:  moduletest.Pass,
			expectedOutputs: cty.EmptyObjectVal,
			expectedDiags:   []tfdiags.Description{},
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
				Changes: plans.NewChangesSrc(),
			},
			state: states.NewState(),
			variables: terraform.InputValues{
				"input": &terraform.InputValue{
					Value:      cty.StringVal("Hello, world!"),
					SourceType: terraform.ValueFromConfig,
					SourceRange: tfdiags.SourceRange{
						Filename: "main.tftest.hcl",
						Start:    tfdiags.SourcePos{Line: 3, Column: 13, Byte: 12},
						End:      tfdiags.SourcePos{Line: 3, Column: 28, Byte: 27},
					},
				},
			},
			provider:        &testing_provider.MockProvider{},
			expectedStatus:  moduletest.Fail,
			expectedOutputs: cty.EmptyObjectVal,
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
				Changes: &plans.ChangesSrc{
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
			provider: &testing_provider.MockProvider{
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
			expectedStatus:  moduletest.Pass,
			expectedOutputs: cty.EmptyObjectVal,
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
				Changes: &plans.ChangesSrc{
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
			provider: &testing_provider.MockProvider{
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
			expectedStatus:  moduletest.Fail,
			expectedOutputs: cty.EmptyObjectVal,
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
				Changes: plans.NewChangesSrc(),
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
			priorOutputs: map[string]cty.Value{
				"setup": cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("Hello, world!"),
				}),
			},
			provider: &testing_provider.MockProvider{
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
			expectedStatus:  moduletest.Pass,
			expectedOutputs: cty.EmptyObjectVal,
		},
		"output_values": {
			configs: map[string]string{
				"main.tf": `
					output "foo" {
						value = "foo value"
					}
					output "bar" {
						value = "bar value"
					}
				`,
				"main.tftest.hcl": `
					run "test_case" {}
				`,
			},
			plan: &plans.Plan{
				Changes: plans.NewChangesSrc(),
			},
			state:          states.NewState(),
			provider:       &testing_provider.MockProvider{},
			expectedStatus: moduletest.Pass,
			expectedOutputs: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("foo value"),
				"bar": cty.StringVal("bar value"),
			}),
		},
		"provider_functions": {
			configs: map[string]string{
				"main.tf": `
				    terraform {
                      required_providers {
						test = {
						  source = "hashicorp/test"
                        }
                      }
                    }
					output "true" {
						value = true
					}
				`,
				"main.tftest.hcl": `
					run "test_case" {
						assert {
							condition = provider::test::true() == output.true
							error_message = "invalid value"
						}
					}
					`,
			},
			plan: &plans.Plan{
				Changes: plans.NewChangesSrc(),
			},
			state: states.NewState(),
			provider: &testing_provider.MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					Functions: map[string]providers.FunctionDecl{
						"true": {
							ReturnType: cty.Bool,
						},
					},
				},
				CallFunctionFn: func(request providers.CallFunctionRequest) providers.CallFunctionResponse {
					if request.FunctionName != "true" {
						return providers.CallFunctionResponse{
							Err: errors.New("unexpected function call"),
						}
					}
					return providers.CallFunctionResponse{
						Result: cty.True,
					}
				},
			},
			expectedStatus: moduletest.Pass,
			expectedOutputs: cty.ObjectVal(map[string]cty.Value{
				"true": cty.True,
			}),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			config := testModuleInline(t, test.configs)

			tfCtx, diags := terraform.NewContext(&terraform.ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): providers.FactoryFixed(test.provider),
				},
			})
			if diags.HasErrors() {
				t.Fatalf("unexpected errors from NewContext\n%s", diags.Err().Error())
			}

			// We just need a vaguely-realistic scope here, so we'll make
			// a plan against the given config and state and use its
			// resulting scope.
			_, planScope, diags := tfCtx.PlanAndEval(config, test.state, &terraform.PlanOpts{
				Mode:         plans.NormalMode,
				SetVariables: test.variables,
			})
			if diags.HasErrors() {
				t.Fatalf("unexpected errors\n%s", diags.Err().Error())
			}

			file := config.Module.Tests["main.tftest.hcl"]
			run := &moduletest.Run{
				Config:       file.Runs[len(file.Runs)-1], // We always simulate the last run block.
				Name:         "test_case",                 // and it should be named test_case
				ModuleConfig: config,
			}

			priorOutputs := make(map[addrs.Run]cty.Value, len(test.priorOutputs))
			for name, val := range test.priorOutputs {
				priorOutputs[addrs.Run{Name: name}] = val
			}

			testCtx := NewEvalContext(&EvalContextOpts{
				CancelCtx: context.Background(),
				StopCtx:   context.Background(),
			})
			testCtx.runOutputs = priorOutputs
			gotStatus, gotOutputs, diags := testCtx.EvaluateRun(run, planScope, test.testOnlyVars)

			if got, want := gotStatus, test.expectedStatus; got != want {
				t.Errorf("wrong status %q; want %q", got, want)
			}
			if diff := cmp.Diff(gotOutputs, test.expectedOutputs, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong output values\n%s", diff)
			}

			compareDiagnosticsFromTestResult(t, test.expectedDiags, diags)
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

// testModuleInline takes a map of path -> config strings and yields a config
// structure with those files loaded from disk
func testModuleInline(t *testing.T, sources map[string]string) *configs.Config {
	t.Helper()

	cfgPath := t.TempDir()

	for path, configStr := range sources {
		dir := filepath.Dir(path)
		if dir != "." {
			err := os.MkdirAll(filepath.Join(cfgPath, dir), os.FileMode(0777))
			if err != nil {
				t.Fatalf("Error creating subdir: %s", err)
			}
		}
		// Write the configuration
		cfgF, err := os.Create(filepath.Join(cfgPath, path))
		if err != nil {
			t.Fatalf("Error creating temporary file for config: %s", err)
		}

		_, err = io.Copy(cfgF, strings.NewReader(configStr))
		cfgF.Close()
		if err != nil {
			t.Fatalf("Error creating temporary file for config: %s", err)
		}
	}

	loader, cleanup := configload.NewLoaderForTests(t)
	defer cleanup()

	// Test modules usually do not refer to remote sources, and for local
	// sources only this ultimately just records all of the module paths
	// in a JSON file so that we can load them below.
	inst := initwd.NewModuleInstaller(loader.ModulesDir(), loader, registry.NewClient(nil, nil))
	_, instDiags := inst.InstallModules(context.Background(), cfgPath, "tests", true, false, initwd.ModuleInstallHooksImpl{})
	if instDiags.HasErrors() {
		t.Fatal(instDiags.Err())
	}

	// Since module installer has modified the module manifest on disk, we need
	// to refresh the cache of it in the loader.
	if err := loader.RefreshModules(); err != nil {
		t.Fatalf("failed to refresh modules after installation: %s", err)
	}

	config, diags := loader.LoadConfigWithTests(cfgPath, "tests")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	return config
}
