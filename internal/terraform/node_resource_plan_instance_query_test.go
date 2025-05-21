// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

func TestNodeResourcePlanInstanceQuery_Execute(t *testing.T) {
	schemaResp := getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"instance_type": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
			"test_child_resource": {
				Attributes: map[string]*configschema.Attribute{
					"instance_type": {
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

	cases := []struct {
		name           string
		mainConfig     string
		queryConfig    string
		listResourceFn func(request providers.ListResourceRequest) providers.ListResourceResponse
		inputVar       string
		expectedConfig map[string]cty.Value
		resourceErrMap map[string]bool // map of resource address to expected error status
	}{
		{
			name: "valid list reference",
			mainConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}
			`,
			queryConfig: `
				variable "input" {
					type = string
					default = "foo"
				}

				list "test_resource" "test" {
					count = 1
					provider = test

					filter = {
						attr = var.input
					}
				}

				list "test_child_resource" "test2" {
					provider = test

					filter = {
						attr = list.test_resource.test[0].data[0].instance_type
					}
				}
			`,
			inputVar: "foo",
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				madeUp := []cty.Value{
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-123456")}),
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-654321")}),
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-789012")}),
				}

				return func(yield func(providers.ListResourceEvent, error) bool) {
					for i, v := range madeUp {
						evt := providers.ListResourceEvent{
							ResourceObject: v,
							Identity: cty.ObjectVal(map[string]cty.Value{
								"id": cty.StringVal(fmt.Sprintf("i-v%d", i+1)),
							}),
						}
						if !yield(evt, nil) {
							return
						}
					}
				}
			},
			expectedConfig: map[string]cty.Value{
				"test_resource": cty.ObjectVal(map[string]cty.Value{
					"filter": cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("foo"),
					}),
				}),
				"test_child_resource": cty.ObjectVal(map[string]cty.Value{
					"filter": cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("ami-123456"),
					}),
				}),
			},
			resourceErrMap: map[string]bool{
				"list.test_resource.test[0]":     false,
				"list.test_child_resource.test2": false,
			},
		},
		{
			name: "empty list result",
			mainConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}
			`,
			queryConfig: `
				variable "input" {
					type = string
				}

				list "test_resource" "test" {
					count = 1
					provider = test

					filter = {
						attr = var.input
					}
				}

				list "test_child_resource" "test2" {
					provider = test

					filter = {
						attr = list.test_resource.test[0].data[0].instance_type
					}
				}
			`,
			inputVar: "empty",
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				return func(yield func(providers.ListResourceEvent, error) bool) {}
			},
			expectedConfig: map[string]cty.Value{
				"test_resource": cty.ObjectVal(map[string]cty.Value{
					"filter": cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("empty"),
					}),
				}),
			},
			resourceErrMap: map[string]bool{
				"list.test_resource.test[0]":     false,
				"list.test_child_resource.test2": true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := states.NewState()
			p := testProvider("test")
			p.ConfigureProvider(providers.ConfigureProviderRequest{})
			p.GetProviderSchemaResponse = schemaResp

			var requestConfigs = make(map[string]cty.Value)
			p.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
				requestConfigs[request.TypeName] = request.Config
				return tc.listResourceFn(request)
			}

			mod := testModuleInline(t, map[string]string{
				"main.tf":           tc.mainConfig,
				"query.tfquery.hcl": tc.queryConfig,
			})
			valState := namedvals.NewState()
			valState.SetInputVariableValue(addrs.AbsInputVariableInstance{
				Variable: addrs.InputVariable{Name: "input"},
			}, cty.StringVal(tc.inputVar))
			ctx := testBuiltinEvalContext(t, walkQuery, mod, state, valState)
			ctx = ctx.withScope(evalContextModuleInstance{Addr: addrs.RootModuleInstance}).(*BuiltinEvalContext)
			providerAddr := `provider["registry.terraform.io/hashicorp/test"]`
			ctx.ProviderCache[providerAddr] = p

			ctx.Plugins = newContextPlugins(map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): providers.FactoryFixed(p),
			}, nil, nil)
			ctx.Evaluator.Plugins = ctx.Plugins

			// Helper function to execute a resource node and validate results
			executeResourceNode := func(addr addrs.AbsResourceInstance, shouldError bool) {
				t.Helper()
				nodeSchema := p.GetProviderSchemaResponse.SchemaForResourceAddr(addr.Resource.Resource)
				node := NodePlannableResourceInstance{
					NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
						Addr: addr,
						NodeAbstractResource: NodeAbstractResource{
							ResolvedProvider: mustProviderConfig(providerAddr),
							Config:           mod.Module.ListResources[addr.ConfigResource().String()],
							Schema:           &nodeSchema,
						},
					},
				}

				err := node.Execute(ctx, walkQuery)
				if shouldError {
					if err == nil {
						t.Fatalf("expected error for %s, got nil", addr)
					}
					return
				}
				if err != nil {
					t.Fatalf("unexpected error for %s: %s", addr, err)
				}

				if !p.ValidateListResourceConfigCalled {
					t.Errorf("ValidateListResourceConfigCalled wasn't called for %s", addr)
				}
				if !p.ListResourceCalled {
					t.Errorf("ListResourceCalled wasn't called for %s", addr)
				}

				// Reset provider call flags to test the next resource
				p.ValidateListResourceConfigCalled = false
				p.ListResourceCalled = false
			}

			for resourceStr, expectedError := range tc.resourceErrMap {
				addr := mustResourceInstanceAddr(resourceStr)
				executeResourceNode(addr, expectedError)
				allLists := state.AllListResourceInstances()
				resultMap := allLists[addr.Resource.Resource.String()]
				result := resultMap.Get(addr)
				if result != nil && !expectedError {
					if !result.Value.Length().Equals(result.Identity.Length()).True() {
						t.Fatalf("expected value and result for %s to be equal, got %s and %s", addr, result.Value, result.Identity)
					}
				}
			}

			if diff := cmp.Diff(requestConfigs, tc.expectedConfig, ctydebug.CmpOptions); diff != "" {
				t.Errorf("unexpected request configs (-want +got):\n%s", diff)
			}
		})
	}
}
