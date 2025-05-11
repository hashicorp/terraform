// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

// queryTestCase is a test case for verifying query behavior
type queryTestCase struct {
	name       string
	configs    map[string]string
	variables  map[string]cty.Value
	wantFilter map[string][]string
	want       map[string]cty.Value
	// If specified, modifies the provider to inject custom behavior
	providerCustomizer func(p *testing_provider.MockProvider)
	// If set, validate diagnostics match expected
	assertErrorDiags func(diags tfdiags.Diagnostics) bool
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
func getQueryTestProvider(attrMap map[string][]string) *testing_provider.MockProvider {
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
		if attrMap[request.TypeName] == nil {
			attrMap[request.TypeName] = []string{}
		}

		attrMap[request.TypeName] = append(attrMap[request.TypeName], str)

		str = strings.TrimPrefix(str, "filter_")
		if request.TypeName == "test_child_resource" {
			request.ResourceEmitter(providers.ListResult{
				ResourceObject: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal(fmt.Sprintf("resp_%s", str)),
				}),
			})
		} else {
			for _, attr := range []string{"foo", "bar"} {
				request.ResourceEmitter(providers.ListResult{
					ResourceObject: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal(fmt.Sprintf("resp_%s_%s", str, attr)),
					}),
				})
			}
		}
		request.DoneCh <- struct{}{}
		return nil
	}

	return p
}

func TestContext2Plan_Query(t *testing.T) {
	testCases := []queryTestCase{
		{
			name: "basic query",
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
			variables: map[string]cty.Value{
				"input": cty.StringVal("filter_parent"),
			},
			wantFilter: map[string][]string{
				"test_resource": {"filter_parent"},
			},
			want: map[string]cty.Value{
				"list.test_resource.test": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("resp_parent_foo"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("resp_parent_bar"),
					}),
				}),
			},
		},
		{
			name: "query with count",
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

					list "test_child_resource" "test_child" {
						count = 2
						provider = test

						filter = {
							attr = join("-",["filter_child", "${count.index}"])
						}
					}
				`,
			},
			variables: map[string]cty.Value{
				"input": cty.StringVal("filter_parent"),
			},
			wantFilter: map[string][]string{
				"test_resource":       {"filter_parent"},
				"test_child_resource": {"filter_child-0", "filter_child-1"},
			},
			want: map[string]cty.Value{
				"list.test_child_resource.test_child[0]": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("resp_child-0"),
					}),
				}),
				"list.test_child_resource.test_child[1]": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("resp_child-1"),
					}),
				}),
			},
		},
		{
			name: "query with count and reference",
			configs: map[string]string{
				"main.tfquery.hcl": `
					provider "test" {}

					variable "input" {
						type = string
						default = "test"
					}

					variable "countvar" {
						type = number
						default = 2
					}

					list "test_resource" "test" {
						count = var.countvar
						provider = test

						filter = {
							attr = join("",[var.input,"${count.index}"])
						}
					}

					list "test_child_resource" "test_child" {
						provider = test

						filter = {
							attr = join("-",["filter_child", list.test_resource.test[0].data[0].attr])
						}
					}
				`,
			},
			variables: map[string]cty.Value{
				"input":    cty.StringVal("filter_parent"),
				"countvar": cty.NumberIntVal(2),
			},
			wantFilter: map[string][]string{
				"test_resource":       {"filter_parent0", "filter_parent1"},
				"test_child_resource": {"filter_child-resp_parent0_foo"},
			},
			want: map[string]cty.Value{
				"list.test_child_resource.test_child": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("resp_child-resp_parent0_foo"),
					}),
				}),
			},
		},
		{
			name: "query with for_each",
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

					# looping of the results from a single list resource
					list "test_child_resource" "test_child" {
						for_each = toset([for el in list.test_resource.test.data : el.attr])
						provider = test

						filter = {
							attr = join("-",["filter_child", each.key])
						}
					}
				`,
			},
			variables: map[string]cty.Value{
				"input": cty.StringVal("filter_parent"),
			},
			wantFilter: map[string][]string{
				"test_resource":       {"filter_parent"},
				"test_child_resource": {"filter_child-resp_parent_foo", "filter_child-resp_parent_bar"},
			},
			want: map[string]cty.Value{
				"list.test_child_resource.test_child[\"resp_parent_foo\"]": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("resp_child-resp_parent_foo"),
					}),
				}),
			},
		},
		{
			name: "query with for_each splat",
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

					# looping of the results from a single list resource
					list "test_child_resource" "test_child" {
						for_each = toset(list.test_resource.test.data[*].attr)
						provider = test

						filter = {
							attr = join("-",["filter_child", each.key])
						}
					}
				`,
			},
			variables: map[string]cty.Value{
				"input": cty.StringVal("filter_parent"),
			},
			wantFilter: map[string][]string{
				"test_resource":       {"filter_parent"},
				"test_child_resource": {"filter_child-resp_parent_foo", "filter_child-resp_parent_bar"},
			},
			want: map[string]cty.Value{
				"list.test_child_resource.test_child[\"resp_parent_foo\"]": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("resp_child-resp_parent_foo"),
					}),
				}),
			},
		},
		{
			name: "query with for_each reference",
			configs: map[string]string{
				"main.tfquery.hcl": `
					provider "test" {}

					variable "input" {
						type = string
						default = "test"
					}

					list "test_resource" "test" {
						for_each = toset(["inst1", "inst2"])
						provider = test

						filter = {
							attr = join("-",[var.input,each.key])
						}
					}

					# looping over multiple list resources
					# and using the first object of each list resource's result
					list "test_child_resource" "test_child" {
						for_each = list.test_resource.test
						provider = test

						filter = {
							attr = join("-",["filter_child", each.value.data[0].attr])
						}
					}
				`,
			},
			variables: map[string]cty.Value{
				"input": cty.StringVal("filter_parent"),
			},
			wantFilter: map[string][]string{
				"test_resource":       {"filter_parent-inst1", "filter_parent-inst2"},
				"test_child_resource": {"filter_child-resp_parent-inst1_foo", "filter_child-resp_parent-inst2_foo"},
			},
			want: map[string]cty.Value{
				"list.test_child_resource.test_child[\"inst1\"]": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("resp_child-resp_parent-inst1_foo"),
					}),
				}),
			},
		},
		{
			name: "diagnostic from provider",
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
			variables: map[string]cty.Value{
				"input": cty.StringVal("error-trigger"),
			},
			providerCustomizer: func(p *testing_provider.MockProvider) {
				originalFn := p.ListResourceFn
				p.ListResourceFn = func(request providers.ListResourceRequest) error {
					filter := request.Config.GetAttr("filter")
					str := filter.GetAttr("attr").AsString()

					// Emit diagnostic for a specific filter value
					if str == "error-trigger" {
						err := fmt.Errorf("test error: resource listing failed for filter '%s'", str)
						diags := tfdiags.Diagnostics{}
						request.DiagEmitter(diags.Append(err))
						request.DoneCh <- struct{}{}
						return nil
					}

					return originalFn(request)
				}
			},
			assertErrorDiags: func(diags tfdiags.Diagnostics) bool {
				return strings.Contains(diags.Err().Error(),
					"test error: resource listing failed for filter 'error-trigger'")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
			gotAttr := map[string][]string{}
			p := getQueryTestProvider(gotAttr)

			if tc.providerCustomizer != nil {
				tc.providerCustomizer(p)
			}

			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
				},
			})

			qv := &MockQueryViews{
				ResourceAddrs: addrs.MakeMap[addrs.AbsResourceInstance, []*states.ResourceInstanceObjectSrc](),
			}

			plan, _, diags := ctx.PlanAndEval(m, states.NewState(), &PlanOpts{
				QueryViews: qv,
				SetVariables: func() InputValues {
					ret := InputValues{}
					for k, v := range tc.variables {
						ret[k] = &InputValue{Value: v}
					}
					return ret
				}(),
				Mode: plans.QueryMode,
			})

			// Check if diagnostics are expected
			if tc.assertErrorDiags != nil {
				if !diags.HasErrors() {
					t.Fatal("Expected diagnostics with errors but none were returned")
				}

				if !tc.assertErrorDiags(diags) {
					t.Fatal("Returned diagnostics did not match expected diagnostics")
				}

				// Skip the rest of the checks since we expected errors
				return
			}

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

			// Check the expected attributes
			for addr, expected := range tc.wantFilter {
				if got, ok := gotAttr[addr]; !ok {
					t.Fatalf("Expected %s to be in the query results", addr)
				} else {
					sort.Strings(got)
					sort.Strings(expected)
					if diff := cmp.Diff(got, expected); diff != "" {
						t.Fatalf("Unexpected query result for %s:\n%s", addr, diff)
					}
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
