// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

func TestNodeResourcePlanInstanceQuery_Execute(t *testing.T) {
	state := states.NewState()
	absResource1 := mustResourceInstanceAddr("list.test_resource.test[0]")
	absResource2 := mustResourceInstanceAddr("list.test_child_resource.test2")

	p := testProvider("test")
	p.ConfigureProvider(providers.ConfigureProviderRequest{})
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

	var requestConfigs = make(map[string]cty.Value)
	p.ListResourceFn = func(request providers.ListResourceRequest) error {
		requestConfigs[request.TypeName] = request.Config
		return nil
	}
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}
				`,
		"query.tfquery.hcl": `
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
	})
	valState := namedvals.NewState()
	valState.SetInputVariableValue(addrs.AbsInputVariableInstance{
		Variable: addrs.InputVariable{Name: "input"},
	}, cty.StringVal("foo"))
	ctx := testBuiltinEvalContext(t, walkQuery, mod, state, valState)
	ctx = ctx.withScope(evalContextModuleInstance{Addr: addrs.RootModuleInstance}).(*BuiltinEvalContext)
	providerAddr := `provider["registry.terraform.io/hashicorp/test"]`
	ctx.ProviderCache[providerAddr] = p

	ctx.Plugins = newContextPlugins(map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): providers.FactoryFixed(p),
	}, nil, nil)
	ctx.Evaluator.Plugins = ctx.Plugins

	// Helper function to execute a resource node and validate results
	executeResourceNode := func(addr addrs.AbsResourceInstance, expectedConfig cty.Value) {
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

		// Check if the expected config was set
		if value, ok := requestConfigs[addr.Resource.Resource.Type]; ok {
			if !value.Equals(expectedConfig).True() {
				t.Errorf("expected %s, got %s", expectedConfig, value)
			}
		} else {
			t.Errorf("expected config for %s not found", addr)
		}
	}

	// Test first resource
	expected1 := cty.ObjectVal(map[string]cty.Value{
		"filter": cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
	})
	executeResourceNode(absResource1, expected1)

	// Test second resource
	expected2 := cty.ObjectVal(map[string]cty.Value{
		"filter": cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("list.test_resource.test[0]"),
		}),
	})
	executeResourceNode(absResource2, expected2)
}
