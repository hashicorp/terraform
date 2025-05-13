// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func TestBuiltinEvalContextProviderInput(t *testing.T) {
	var lock sync.Mutex
	cache := make(map[string]map[string]cty.Value)

	ctx1 := testBuiltinEvalContext(t)
	ctx1 = ctx1.withScope(evalContextModuleInstance{Addr: addrs.RootModuleInstance}).(*BuiltinEvalContext)
	ctx1.ProviderInputConfig = cache
	ctx1.ProviderLock = &lock

	ctx2 := testBuiltinEvalContext(t)
	ctx2 = ctx2.withScope(evalContextModuleInstance{Addr: addrs.RootModuleInstance.Child("child", addrs.NoKey)}).(*BuiltinEvalContext)
	ctx2.ProviderInputConfig = cache
	ctx2.ProviderLock = &lock

	providerAddr1 := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}
	providerAddr2 := addrs.AbsProviderConfig{
		Module:   addrs.RootModule.Child("child"),
		Provider: addrs.NewDefaultProvider("foo"),
	}

	expected1 := map[string]cty.Value{"value": cty.StringVal("foo")}
	ctx1.SetProviderInput(providerAddr1, expected1)

	try2 := map[string]cty.Value{"value": cty.StringVal("bar")}
	ctx2.SetProviderInput(providerAddr2, try2) // ignored because not a root module

	actual1 := ctx1.ProviderInput(providerAddr1)
	actual2 := ctx2.ProviderInput(providerAddr2)

	if !reflect.DeepEqual(actual1, expected1) {
		t.Errorf("wrong result 1\ngot:  %#v\nwant: %#v", actual1, expected1)
	}
	if actual2 != nil {
		t.Errorf("wrong result 2\ngot:  %#v\nwant: %#v", actual2, nil)
	}
}

func TestBuildingEvalContextInitProvider(t *testing.T) {
	var lock sync.Mutex

	testP := &testing_provider.MockProvider{}

	ctx := testBuiltinEvalContext(t)
	ctx = ctx.withScope(evalContextModuleInstance{Addr: addrs.RootModuleInstance}).(*BuiltinEvalContext)
	ctx.ProviderLock = &lock
	ctx.ProviderCache = make(map[string]providers.Interface)
	ctx.Plugins = newContextPlugins(map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): providers.FactoryFixed(testP),
	}, nil, nil)

	providerAddrDefault := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("test"),
	}
	providerAddrAlias := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("test"),
		Alias:    "foo",
	}
	providerAddrMock := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("test"),
		Alias:    "mock",
	}

	_, err := ctx.InitProvider(providerAddrDefault, nil)
	if err != nil {
		t.Fatalf("error initializing provider test: %s", err)
	}
	_, err = ctx.InitProvider(providerAddrAlias, nil)
	if err != nil {
		t.Fatalf("error initializing provider test.foo: %s", err)
	}

	_, err = ctx.InitProvider(providerAddrMock, &configs.Provider{
		Mock: true,
	})
	if err != nil {
		t.Fatalf("error initializing provider test.mock: %s", err)
	}
}

func TestBuiltinEvalContext_List_EvaluateBlock(t *testing.T) {
	testCases := map[string]struct {
		block             string
		configs           map[string]string
		schema            *configschema.Block
		keyData           instances.RepetitionData
		self              addrs.Referenceable
		setValState       func(*namedvals.State)
		expectedVal       cty.Value
		expectedDiagCount int
	}{
		"list filter block with var reference": {
			block: "list.test_resource.test",
			configs: map[string]string{
				"main.tfquery.hcl": `
					provider "test" {}

					variable "input" {
						type = string
						default = "test"
					}

					list "test_resource" "test" {
						provider = test

						filter = {
							attr = var.input
						}
					}
				`,
			},
			schema:  getListResourceTestSchema(),
			keyData: EvalDataForNoInstanceKey,
			expectedVal: cty.ObjectVal(map[string]cty.Value{
				"filter": cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("list_resource_input"),
				}),
			}),
			setValState: func(s *namedvals.State) {
				// Set the input variable value
				s.SetInputVariableValue(addrs.AbsInputVariableInstance{
					Module:   addrs.RootModuleInstance,
					Variable: addrs.InputVariable{Name: "input"},
				}, cty.StringVal("list_resource_input"))

				addr := mustAbsResourceAddr("list.test_resource.example")
				resp := cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"instance_type": cty.StringVal("list_resource_value"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"instance_type": cty.StringVal("list_resource_value2"),
					}),
				})
				s.SetResourceListInstance(addr, addrs.NoKey, resp)
			},
		},
		"list filter block with list reference": {
			block: "list.test_resource.example",
			configs: map[string]string{
				"main.tfquery.hcl": `
					provider "test" {}

					variable "input" {
						type = string
						default = "test"
					}

					list "test_resource" "test" {
						provider = test

						filter = {
							attr = var.input
						}
					}

					list "test_resource" "example" {
						provider = test

						filter = {
							attr = list.test_resource.test.data[0].instance_type
						}
					}
				`,
			},
			schema:  getListResourceTestSchema(),
			keyData: EvalDataForNoInstanceKey,
			expectedVal: cty.ObjectVal(map[string]cty.Value{
				"filter": cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("list_resource_value1"),
				}),
			}),
			setValState: func(s *namedvals.State) {
				// Set the input variable value
				s.SetInputVariableValue(addrs.AbsInputVariableInstance{
					Module:   addrs.RootModuleInstance,
					Variable: addrs.InputVariable{Name: "input"},
				}, cty.StringVal("list_resource_input"))

				addr := mustAbsResourceAddr("list.test_resource.test")
				resp := cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"instance_type": cty.StringVal("list_resource_value1"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"instance_type": cty.StringVal("list_resource_value2"),
					}),
				})
				s.SetResourceListInstance(addr, addrs.NoKey, resp)
			},
		},
		"list filter block with complex filter attributes": {
			block: "list.test_resource.complex",
			configs: map[string]string{
				"main.tfquery.hcl": `
					provider "test" {}

					variable "input" {
						type = string
						default = "complex_value"
					}

					list "test_resource" "complex" {
						provider = test

						filter = {
							attr = var.input
							id = "abc-123"
							tags = {
								"Name" = "test-resource"
								"Environment" = "dev"
							}
						}
					}
				`,
			},
			schema:  getComplexListResourceTestSchema(),
			keyData: EvalDataForNoInstanceKey,
			expectedVal: cty.ObjectVal(map[string]cty.Value{
				"filter": cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("complex_value"),
					"id":   cty.StringVal("abc-123"),
					"tags": cty.MapVal(map[string]cty.Value{
						"Name":        cty.StringVal("test-resource"),
						"Environment": cty.StringVal("dev"),
					}),
				}),
			}),
			setValState: func(s *namedvals.State) {
				s.SetInputVariableValue(addrs.AbsInputVariableInstance{
					Module:   addrs.RootModuleInstance,
					Variable: addrs.InputVariable{Name: "input"},
				}, cty.StringVal("complex_value"))
			},
		},
		"list filter block with count index": {
			block: "list.test_resource.counted",
			configs: map[string]string{
				"main.tfquery.hcl": `
					provider "test" {}

					variable "input" {
						type = string
						default = "base_value"
					}

					list "test_resource" "counted" {
						count = 2
						provider = test

						filter = {
							attr = "${var.input}-${count.index}"
						}
					}
				`,
			},
			schema:  getListResourceTestSchema(),
			keyData: EvalDataForInstanceKey(addrs.IntKey(0), nil),
			expectedVal: cty.ObjectVal(map[string]cty.Value{
				"filter": cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("base_value-0"),
				}),
			}),
			setValState: func(s *namedvals.State) {
				s.SetInputVariableValue(addrs.AbsInputVariableInstance{
					Module:   addrs.RootModuleInstance,
					Variable: addrs.InputVariable{Name: "input"},
				}, cty.StringVal("base_value"))
			},
		},
		"list filter block with for_each key": {
			block: "list.test_resource.foreach",
			configs: map[string]string{
				"main.tfquery.hcl": `
					provider "test" {}

					variable "input" {
						type = string
						default = "base_value"
					}

					list "test_resource" "foreach" {
						for_each = toset(["key1", "key2"])
						provider = test

						filter = {
							attr = "${var.input}-${each.key}"
						}
					}
				`,
			},
			schema: getListResourceTestSchema(),
			keyData: EvalDataForInstanceKey(
				addrs.StringKey("key1"),
				map[string]cty.Value{
					"key": cty.StringVal("key1"),
				},
			),
			expectedVal: cty.ObjectVal(map[string]cty.Value{
				"filter": cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("base_value-key1"),
				}),
			}),
			setValState: func(s *namedvals.State) {
				s.SetInputVariableValue(addrs.AbsInputVariableInstance{
					Module:   addrs.RootModuleInstance,
					Variable: addrs.InputVariable{Name: "input"},
				}, cty.StringVal("base_value"))
			},
		},
		"invalid reference causing diagnostic": {
			block: "list.test_resource.invalid",
			configs: map[string]string{
				"main.tfquery.hcl": `
					provider "test" {}

					list "test_resource" "invalid" {
						provider = test

						filter = {
							attr = non_existent_var
						}
					}
				`,
			},
			schema:            getListResourceTestSchema(),
			keyData:           EvalDataForNoInstanceKey,
			expectedDiagCount: 1,
			setValState: func(s *namedvals.State) {
				// No state setup needed for error test case
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			tc.configs["main.tf"] = fmt.Sprintf(`
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}
			`)
			m := testModuleInline(t, tc.configs)
			namedvals := namedvals.NewState()
			expander := instances.NewExpander(nil)
			state := states.BuildState(func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_instance",
						Name: "foo",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"dynamic":{"type":"string","value":"hello"}}`),
					},
					addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				)
			})
			tc.setValState(namedvals)
			testP := getTestProvider()
			providerAddrDefault := addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
			}
			plugins := newContextPlugins(map[addrs.Provider]providers.Factory{
				providerAddrDefault.Provider: providers.FactoryFixed(testP),
			}, nil, map[addrs.Provider]providers.ProviderSchema{
				providerAddrDefault.Provider: testP.GetProviderSchema(),
			})
			// Create a minimal evaluator for testing
			evaluator := &Evaluator{
				Operation:   walkQuery,
				Config:      m,
				Instances:   expander,
				NamedValues: namedvals,
				State:       state.SyncWrapper(),
				Plugins:     plugins,
			}

			// Set up a mock evaluation scope
			var lock sync.Mutex

			ctx := testBuiltinEvalContext(t)
			ctx = ctx.withScope(evalContextModuleInstance{Addr: addrs.RootModuleInstance}).(*BuiltinEvalContext)
			ctx.Evaluator = evaluator
			ctx.NamedValuesValue = namedvals
			ctx.ProviderLock = &lock
			ctx.ProviderCache = make(map[string]providers.Interface)
			ctx.Plugins = plugins

			// Call the method under test
			body := m.Module.ListResources[tc.block].Config
			rsc := mustAbsResourceAddr(tc.block)
			result, _, resultDiags := ctx.EvaluateBlock2(body, tc.schema, tc.self, rsc.Resource, tc.keyData)

			// Check for expected diagnostics
			if resultDiags.HasErrors() != (tc.expectedDiagCount > 0) {
				t.Errorf("unexpected diagnostics status: %s", resultDiags.Err())
			}

			if len(resultDiags) != tc.expectedDiagCount && tc.expectedDiagCount > 0 {
				t.Errorf("expected %d diagnostics, got %d: %s", tc.expectedDiagCount, len(resultDiags), resultDiags.Err())
			}

			// If we expected errors, don't continue with value testing
			if tc.expectedDiagCount > 0 {
				return
			}

			// Verify that the result value matches what we expect
			if !reflect.DeepEqual(result, tc.expectedVal) {
				t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", result, tc.expectedVal)
			}
		})
	}
}

// getListResourceTestSchema returns a schema for list resource tests with a filter attribute
func getListResourceTestSchema() *configschema.Block {
	return &configschema.Block{
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
	}
}

// getComplexListResourceTestSchema returns a schema with more complex filter attributes
func getComplexListResourceTestSchema() *configschema.Block {
	return &configschema.Block{
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
						"id": {
							Type:     cty.String,
							Optional: true,
						},
						"tags": {
							Type:     cty.Map(cty.String),
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func testBuiltinEvalContext(t *testing.T) *BuiltinEvalContext {
	return &BuiltinEvalContext{}
}

func getTestProvider() *testing_provider.MockProvider {
	p := simpleMockProvider()
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"instance_type": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
			"list": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
		ListResourceTypes: map[string]*configschema.Block{
			"test_resource":       getQueryTestSchema(),
			"test_child_resource": getQueryTestSchema(),
		},
	})

	return p
}

// getQueryTestSchema returns a schema for query tests with a filter attribute
func getQueryTestSchema() *configschema.Block {
	return &configschema.Block{
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
	}
}
