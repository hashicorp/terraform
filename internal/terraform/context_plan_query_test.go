// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

// queryTestCase is a test case for verifying query behavior
type queryTestCase struct {
	name     string
	configs  map[string]string
	variable cty.Value
	want     map[string]cty.Value
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

// getQueryTestProvider returns a provider configured for query tests
func getQueryTestProvider() *testing_provider.MockProvider {
	p := simpleMockProvider()
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
			"test_child_resource": {
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

	p.ListResourceFn = func(request providers.ListResourceRequest) error {
		filter := request.Config.GetAttr("filter")
		str := filter.GetAttr("attr").AsString()

		if str != "parent" && request.TypeName == "test_resource" {
			return fmt.Errorf("Expected filter attr to be 'inputed' for test_resource, got '%s'", str)
		}

		if request.TypeName == "test_child_resource" {
			request.ResourceEmitter(providers.ListResult{
				ResourceObject: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal(fmt.Sprintf("child_%s", str)),
				}),
			})
		} else {
			for _, attr := range []string{"0", "1"} {
				request.ResourceEmitter(providers.ListResult{
					ResourceObject: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal(str + attr),
					}),
				})
			}
		}
		request.DoneCh <- struct{}{}
		return nil
	}

	return p
}

func TestContext2Plan_QueryExamples(t *testing.T) {
	testCases := []queryTestCase{
		{
			name: "basic query",
			configs: map[string]string{
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
			variable: cty.StringVal("parent"),
			want: map[string]cty.Value{
				"list.test_resource.test": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("parent0"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("parent1"),
					}),
				}),
			},
		},
		{
			name: "query with count",
			configs: map[string]string{
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

					list "test_child_resource" "test_child" {
						count = 2
						provider = test

						filter = {
							attr = join("-",["attr", "${count.index}"])
						}
					}
				`,
			},
			variable: cty.StringVal("parent"),
			want: map[string]cty.Value{
				"list.test_child_resource.test_child[0]": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("child_attr-0"),
					}),
				}),
			},
		},
		{
			name: "query with count and reference",
			configs: map[string]string{
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
				"main.tfquery.hcl": `
					provider "test" {}

					variable "input" {
						type = string
						default = "test"
					}

					list "test_resource" "test" {
						count = 2
						provider = test

						filter = {
							attr = var.input
						}
					}

					list "test_child_resource" "test_child" {
						provider = test

						filter = {
							attr = join("-",["attr", list.test_resource.test[0].data[0].attr])
						}
					}
				`,
			},
			variable: cty.StringVal("parent"),
			want: map[string]cty.Value{
				"list.test_child_resource.test_child": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("child_attr-parent0"),
					}),
				}),
			},
		},
		{
			name: "query with for_each",
			configs: map[string]string{
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

					# looping of the results from a single list resource
					list "test_child_resource" "test_child" {
						for_each = toset([for el in list.test_resource.test.data : el.attr])
						provider = test

						filter = {
							attr = each.key
						}
					}
				`,
			},
			variable: cty.StringVal("parent"),
			want: map[string]cty.Value{
				"list.test_child_resource.test_child[\"parent0\"]": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("child_parent0"),
					}),
				}),
			},
		},
		{
			name: "query with for_each reference",
			configs: map[string]string{
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
				"main.tfquery.hcl": `
					provider "test" {}

					variable "input" {
						type = string
						default = "test"
					}

					list "test_resource" "test" {
						for_each = toset(["attr1", "attr2"])
						provider = test

						filter = {
							attr = var.input
						}
					}

					list "test_child_resource" "test_child" {
						for_each = list.test_resource.test
						provider = test

						filter = {
							attr = each.value.data[0].attr
						}
					}
				`,
			},
			variable: cty.StringVal("parent"),
			want: map[string]cty.Value{
				"list.test_child_resource.test_child[\"attr1\"]": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("child_parent0"),
					}),
				}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := testModuleInline(t, tc.configs)
			p := getQueryTestProvider()

			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
				},
				Parallelism: 1,
			})

			qv := &MockQueryViews{
				ResourceAddrs: addrs.MakeMap[addrs.AbsResourceInstance, []*states.ResourceInstanceObjectSrc](),
			}

			plan, _, diags := ctx.PlanAndEval(m, states.NewState(), &PlanOpts{
				QueryViews: qv,
				SetVariables: InputValues{
					"input": &InputValue{
						Value: tc.variable,
					},
				},
				Mode: plans.QueryMode,
			})

			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			// Check the expected query results
			for addr, expected := range tc.want {
				partial, diags := addrs.ParsePartialResourceInstanceStr(addr)
				if diags.HasErrors() {
					t.Fatalf("Failed to parse address %s: %s", addr, diags.Err())
				}
				resourceAddr, ok := plan.QueryResults.GetOk(partial.Resource.Resource)
				if !ok {
					t.Fatalf("Expected %s to be in the query results", addr)
				}

				instance, ok := resourceAddr.GetOk(mustResourceInstanceAddr(addr))
				if !ok {
					t.Fatalf("Expected %s to be in the query results", addr)
				}

				if diff := cmp.Diff(instance, expected, ctydebug.CmpOptions); diff != "" {
					t.Fatalf("Unexpected query result for %s:\n%s", addr, diff)
				}
			}
		})
	}
}

// MockQueryViews is a mock implementation of the QueryViews interface for testing.
type MockQueryViews struct {
	ListCalled     bool
	ListStatesArg  ListStates
	ResourceCalled bool
	ResourceAddrs  addrs.Map[addrs.AbsResourceInstance, []*states.ResourceInstanceObjectSrc]
}

func (m *MockQueryViews) List(states ListStates) {
	m.ListCalled = true
	m.ListStatesArg = states
}

func (m *MockQueryViews) Resource(addr addrs.AbsResourceInstance, obj *states.ResourceInstanceObjectSrc) {
	m.ResourceCalled = true
	if !m.ResourceAddrs.Has(addr) {
		m.ResourceAddrs.Put(addr, []*states.ResourceInstanceObjectSrc{})
	}
	m.ResourceAddrs.Put(addr, append(m.ResourceAddrs.Get(addr), obj))
}
