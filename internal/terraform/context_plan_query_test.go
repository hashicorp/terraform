// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Plan_queryList(t *testing.T) {
	listResourceFn := func(request providers.ListResourceRequest) providers.ListResourceResponse {
		instanceTypes := []string{"ami-123456", "ami-654321", "ami-789012"}
		madeUp := []cty.Value{}
		for i := range len(instanceTypes) {
			madeUp = append(madeUp, cty.ObjectVal(map[string]cty.Value{
				"instance_type": cty.StringVal(instanceTypes[i]),
				"id":            cty.StringVal(fmt.Sprint(i)),
			}))
		}

		ids := []cty.Value{}
		for i := range madeUp {
			ids = append(ids, cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(fmt.Sprintf("i-v%d", i+1)),
			}))
		}

		resp := []cty.Value{}
		for i, v := range madeUp {
			mp := map[string]cty.Value{
				"identity":     ids[i],
				"display_name": cty.StringVal(fmt.Sprintf("Instance %d", i+1)),
			}
			if request.IncludeResourceObject {
				mp["state"] = v
			}
			resp = append(resp, cty.ObjectVal(mp))
		}

		ret := map[string]cty.Value{
			"data": cty.TupleVal(resp),
		}
		for k, v := range request.Config.AsValueMap() {
			if k != "data" {
				ret[k] = v
			}
		}

		return providers.ListResourceResponse{Result: cty.ObjectVal(ret)}
	}

	type resources struct {
		list    map[string]bool // map of list resource addresses to whether they want the resource state included in the response
		managed []string
	}

	cases := []struct {
		name                string
		mainConfig          string
		queryConfig         string
		generatedPath       string
		transformSchema     func(*providers.GetProviderSchemaResponse)
		assertValidateDiags func(t *testing.T, diags tfdiags.Diagnostics)
		assertPlanDiags     func(t *testing.T, diags tfdiags.Diagnostics)
		expectedResources   resources
	}{
		{
			name: "valid list reference - generates config",
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
					provider = test
					include_resource = true

					config {
						filter = {
							attr = var.input
						}
					}
				}

				list "test_resource" "test2" {
					provider = test

					config {
						filter = {
							attr = list.test_resource.test.data[0].state.instance_type
						}
					}
				}
				`,
			generatedPath: t.TempDir(),
			expectedResources: resources{
				list: map[string]bool{
					"list.test_resource.test":  true,
					"list.test_resource.test2": false,
				},
				managed: []string{},
			},
		},
		{
			name: "valid list instance reference",
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
					include_resource = true

					config {
						filter = {
							attr = var.input
						}
					}
				}

				list "test_resource" "test2" {
					provider = test
					include_resource = true

					config {
						filter = {
							attr = list.test_resource.test[0].data[0].state.instance_type
						}
					}
				}
				`,
			expectedResources: resources{
				list:    map[string]bool{"list.test_resource.test[0]": true, "list.test_resource.test2": true},
				managed: []string{},
			},
		},
		{
			name: "with empty config when it is required",
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
					provider = test
				}
				`,

			transformSchema: func(schema *providers.GetProviderSchemaResponse) {
				schema.ListResourceTypes["test_resource"].Body.BlockTypes = map[string]*configschema.NestedBlock{
					"config": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"filter": {
									Required: true,
									NestedType: &configschema.Object{
										Nesting: configschema.NestingSingle,
										Attributes: map[string]*configschema.Attribute{
											"attr": {
												Type:     cty.String,
												Optional: true,
											},
										},
									},
								},
							},
						},
						Nesting:  configschema.NestingSingle,
						MinItems: 1,
						MaxItems: 1,
					},
				}

			},
			assertValidateDiags: func(t *testing.T, diags tfdiags.Diagnostics) {
				tfdiags.AssertDiagnosticCount(t, diags, 1)
				var exp tfdiags.Diagnostics
				exp = exp.Append(&hcl.Diagnostic{
					Summary: "Missing config block",
					Detail:  "A block of type \"config\" is required here.",
					Subject: diags[0].Source().Subject.ToHCL().Ptr(),
				})
				tfdiags.AssertDiagnosticsMatch(t, diags, exp)
			},
		},
		{
			name: "with empty optional config",
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
					provider = test
				}
				`,
			transformSchema: func(schema *providers.GetProviderSchemaResponse) {
				schema.ListResourceTypes["test_resource"].Body.BlockTypes = map[string]*configschema.NestedBlock{
					"config": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"filter": {
									Optional: true,
									NestedType: &configschema.Object{
										Nesting: configschema.NestingSingle,
										Attributes: map[string]*configschema.Attribute{
											"attr": {
												Type:     cty.String,
												Optional: true,
											},
										},
									},
								},
							},
						},
						Nesting: configschema.NestingSingle,
					},
				}

			},
			expectedResources: resources{
				list:    map[string]bool{"list.test_resource.test": false},
				managed: []string{},
			},
		},
		{
			name: "invalid list result's attribute reference",
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
					provider = test
					include_resource = true

					config {
						filter = {
							attr = var.input
						}
					}
				}

				list "test_resource" "test2" {
					provider = test
					include_resource = true

					config {
						filter = {
							attr = list.test_resource.test.state.instance_type
						}
					}
				}
				`,
			assertValidateDiags: func(t *testing.T, diags tfdiags.Diagnostics) {
				tfdiags.AssertDiagnosticCount(t, diags, 1)
				var exp tfdiags.Diagnostics
				exp = exp.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid list resource traversal",
					Detail:   "The first step in the traversal for a list resource must be an attribute \"data\".",
					Subject:  diags[0].Source().Subject.ToHCL().Ptr(),
				})

				tfdiags.AssertDiagnosticsMatch(t, diags, exp)
			},
		},
		{
			// Test referencing a non-existent list resource
			name: "reference non-existent list resource",
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
				list "test_resource" "test" {
					provider = test
					include_resource = true

					config {
						filter = {
							attr = list.non_existent.attr
						}
					}
				}
				`,
			assertValidateDiags: func(t *testing.T, diags tfdiags.Diagnostics) {
				tfdiags.AssertDiagnosticCount(t, diags, 1)
				var exp tfdiags.Diagnostics

				exp = exp.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to undeclared resource",
					Detail:   "A list resource \"non_existent\" \"attr\" has not been declared in the root module.",
					Subject:  diags[0].Source().Subject.ToHCL().Ptr(),
				})

				tfdiags.AssertDiagnosticsMatch(t, diags, exp)
			},
		},
		{
			// Test referencing a list resource with invalid attribute
			name: "reference list resource with invalid attribute",
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
				list "test_resource" "test" {
					provider = test
					include_resource = true

					config {
						filter = {
							attr = "valid"
						}
					}
				}

				list "test_resource" "another" {
					provider = test
					include_resource = true

					config {
						filter = {
							attr = list.test_resource.test.data[0].state.invalid_attr
						}
					}
				}
				`,
			assertValidateDiags: func(t *testing.T, diags tfdiags.Diagnostics) {
				tfdiags.AssertDiagnosticCount(t, diags, 1)
				var exp tfdiags.Diagnostics

				exp = exp.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported attribute",
					Detail:   "This object has no argument, nested block, or exported attribute named \"invalid_attr\".",
					Subject:  diags[0].Source().Subject.ToHCL().Ptr(),
				})

				tfdiags.AssertDiagnosticsMatch(t, diags, exp)
			},
		},
		{
			name: "circular reference between list resources",
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
				list "test_resource" "test1" {
					provider = test
					include_resource = true

					config {
						filter = {
							attr = list.test_resource.test2.data[0].state.id
						}
					}
				}

				list "test_resource" "test2" {
					provider = test
					include_resource = true

					config {
						filter = {
							attr = list.test_resource.test1.data[0].state.id
						}
					}
				}
				`,
			assertValidateDiags: func(t *testing.T, diags tfdiags.Diagnostics) {
				tfdiags.AssertDiagnosticCount(t, diags, 1)
				if !strings.Contains(diags[0].Description().Summary, "Cycle: list.test_resource") {
					t.Errorf("Expected error message to contain 'Cycle: list.test_resource', got %q", diags[0].Description().Summary)
				}
				if diags[0].Severity() != tfdiags.Error {
					t.Errorf("Expected error severity to be Error, got %s", diags[0].Severity())
				}
			},
		},
		{
			// Test complex expression with list reference
			name: "complex expression with list reference",
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
				variable "test_var" {
					type = string
					default = "default"
				}

				list "test_resource" "test1" {
					provider = test
					include_resource = true

					config {
						filter = {
							attr = var.test_var
						}
					}
				}

				list "test_resource" "test2" {
					provider = test
					include_resource = true

					config {
						filter = {
							attr = length(list.test_resource.test1.data) > 0 ? list.test_resource.test1.data[0].state.instance_type : var.test_var
						}
					}
				}
				`,
			expectedResources: resources{
				list:    map[string]bool{"list.test_resource.test1": true, "list.test_resource.test2": true},
				managed: []string{},
			},
		},
		{
			name: "list reference as for_each",
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
				list "test_resource" "test1" {
					for_each = toset(["foo", "bar"])
					provider = test
					include_resource = true

					config {
						filter = {
							attr = each.value
						}
					}
				}

				list "test_resource" "test2" {
					provider = test
					include_resource = true
					for_each = list.test_resource.test1

					config {
						filter = {
							attr = each.value.data[0].state.instance_type
						}
					}
				}
				`,
			expectedResources: resources{
				list: map[string]bool{
					"list.test_resource.test1[\"foo\"]": true,
					"list.test_resource.test1[\"bar\"]": true,
					"list.test_resource.test2[\"foo\"]": true,
					"list.test_resource.test2[\"bar\"]": true,
				},
				managed: []string{},
			},
		},
		{
			name: ".tf file blocks should not be evaluated in query mode unless in path of list resources",
			mainConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}
				
				locals {
					foo = "bar"
					// This local variable is not evaluated in query mode, but it is still validated
					bar = resource.test_resource.example.instance_type
				}
				
				// This would produce a plan error if triggered, but we expect it to be ignored in query mode
				// because no list resource depends on it
				resource "test_resource" "example" {
					instance_type = "ami-123456"
					
					lifecycle {
						precondition {
							condition = local.foo != "bar"
							error_message = "This should not be executed"
						}
					}
				}
				
				`,
			queryConfig: `
				list "test_resource" "test" {
					provider = test
					include_resource = true

					config {
						filter = {
							attr = "foo"
						}
					}
				}
				`,
			expectedResources: resources{
				list: map[string]bool{
					"list.test_resource.test": true,
				},
				managed: []string{},
			},
		},
		{
			name: "when list provider depends on managed resource",
			mainConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}
				
				locals {
					foo = "bar"
					bar = resource.test_resource.example.instance_type
				}
				
				provider "test" {
					alias = "example"
					region = resource.test_resource.example.instance_type
				}
				
				// The list resource depends on this via the provider,
				// so this resource will be evaluated as well.
				resource "test_resource" "example" {
					instance_type = "ami-123456"
				}
				
				`,
			queryConfig: `
				list "test_resource" "test" {
					provider = test.example
					include_resource = true

					config {
						filter = {
							attr = "foo"
						}
					}
				}
				`,
			expectedResources: resources{
				list: map[string]bool{
					"list.test_resource.test": true,
				},
				managed: []string{"test_resource.example"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configFiles := map[string]string{"main.tf": tc.mainConfig}
			if tc.queryConfig != "" {
				configFiles["main.tfquery.hcl"] = tc.queryConfig
			}

			mod := testModuleInline(t, configFiles, configs.MatchQueryFiles())
			providerAddr := addrs.NewDefaultProvider("test")
			provider := testProvider("test")
			provider.ConfigureProvider(providers.ConfigureProviderRequest{})
			provider.GetProviderSchemaResponse = getListProviderSchemaResp()
			if tc.transformSchema != nil {
				tc.transformSchema(provider.GetProviderSchemaResponse)
			}
			var requestConfigs = make(map[string]cty.Value)
			provider.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
				if request.Config.IsNull() || request.Config.GetAttr("config").IsNull() {
					t.Fatalf("config should never be null, got null for %s", request.TypeName)
				}
				requestConfigs[request.TypeName] = request.Config
				return listResourceFn(request)
			}

			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					providerAddr: testProviderFuncFixed(provider),
				},
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			diags = ctx.Validate(mod, &ValidateOpts{
				Query: true,
			})
			if tc.assertValidateDiags != nil {
				tc.assertValidateDiags(t, diags)
				return
			} else {
				tfdiags.AssertNoDiagnostics(t, diags)
			}

			plan, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
				Mode:               plans.NormalMode,
				SetVariables:       testInputValuesUnset(mod.Module.Variables),
				Query:              true,
				GenerateConfigPath: tc.generatedPath,
			})
			if tc.assertPlanDiags != nil {
				tc.assertPlanDiags(t, diags)
				return
			} else {
				tfdiags.AssertNoDiagnostics(t, diags)
			}

			// If no diags expected, assert that the plan is valid
			if tc.assertValidateDiags == nil && tc.assertPlanDiags == nil {
				sch, err := ctx.Schemas(mod, states.NewState())
				if err != nil {
					t.Fatalf("failed to get schemas: %s", err)
				}
				expectedResources := slices.Collect(maps.Keys(tc.expectedResources.list))
				actualResources := make([]string, 0)
				generatedCfgs := make([]string, 0)
				for _, change := range plan.Changes.Queries {
					actualResources = append(actualResources, change.Addr.String())
					schema := sch.Providers[providerAddr].ListResourceTypes[change.Addr.Resource.Resource.Type]
					cs, err := change.Decode(schema)
					if err != nil {
						t.Fatalf("failed to decode change: %s", err)
					}

					// Verify data. If the state is included, we check that, otherwise we check the id.
					expectedData := []string{"ami-123456", "ami-654321", "ami-789012"}
					includeState := tc.expectedResources.list[change.Addr.String()]
					if !includeState {
						expectedData = []string{"i-v1", "i-v2", "i-v3"}
					}
					actualData := make([]string, 0)
					obj := cs.Results.Value.GetAttr("data")
					if obj.IsNull() {
						t.Fatalf("Expected 'data' attribute to be present, but it is null")
					}
					obj.ForEachElement(func(key cty.Value, val cty.Value) bool {
						if includeState {
							val = val.GetAttr("state")
							if val.IsNull() {
								t.Fatalf("Expected 'state' attribute to be present, but it is null")
							}
							if val.GetAttr("instance_type").IsNull() {
								t.Fatalf("Expected 'instance_type' attribute to be present, but it is missing")
							}
							actualData = append(actualData, val.GetAttr("instance_type").AsString())
						} else {
							val = val.GetAttr("identity")
							if val.IsNull() {
								t.Fatalf("Expected 'identity' attribute to be present, but it is null")
							}
							if val.GetAttr("id").IsNull() {
								t.Fatalf("Expected 'id' attribute to be present, but it is missing")
							}
							actualData = append(actualData, val.GetAttr("id").AsString())
						}
						return false
					})
					sort.Strings(actualData)
					sort.Strings(expectedData)
					if diff := cmp.Diff(expectedData, actualData); diff != "" {
						t.Fatalf("Expected instance types to match, but they differ: %s", diff)
					}

					if tc.generatedPath != "" {
						generatedCfgs = append(generatedCfgs, change.Generated.String())
					}
				}
				sort.Strings(actualResources)
				sort.Strings(expectedResources)
				if diff := cmp.Diff(expectedResources, actualResources); diff != "" {
					t.Fatalf("Expected resources to match, but they differ: %s", diff)
				}

				expectedManagedResources := tc.expectedResources.managed
				actualResources = make([]string, 0)
				for _, change := range plan.Changes.Resources {
					actualResources = append(actualResources, change.Addr.String())
				}
				if diff := cmp.Diff(expectedManagedResources, actualResources); diff != "" {
					t.Fatalf("Expected resources to match, but they differ: %s", diff)
				}

				if tc.generatedPath != "" {
					if diff := cmp.Diff([]string{testResourceCfg, testResourceCfg2}, generatedCfgs); diff != "" {
						t.Fatalf("Expected generated configs to match, but they differ: %s", diff)
					}
				}
			}
		})
	}
}

func TestContext2Plan_queryListArgs(t *testing.T) {
	mainConfig := `
	terraform {
		required_providers {
			test = {
				source = "hashicorp/test"
				version = "1.0.0"
			}
		}
	}`

	cases := []struct {
		name           string
		queryConfig    string
		diagCount      int
		expectedErrMsg []string
		assertRequest  providers.ListResourceRequest
	}{
		{
			name: "simple list, no args",
			queryConfig: `
				list "test_resource" "test1" {
					provider = test
				}
			`,
			assertRequest: providers.ListResourceRequest{
				TypeName: "test_resource",
				Limit:    100,
			},
		},
		{
			name: "simple list, with args",
			queryConfig: `
				list "test_resource" "test1" {
					provider = test
					limit = 1000
					include_resource = true
				}
			`,
			assertRequest: providers.ListResourceRequest{
				TypeName:              "test_resource",
				Limit:                 1000,
				IncludeResourceObject: true,
			},
		},
		{
			name: "args with local references",
			queryConfig: `
				list "test_resource" "test1" {
					provider = test
					limit = local.test_limit
					include_resource = local.test_include
				}
				locals {
					test_limit = 500
					test_include = true
				}
			`,
			assertRequest: providers.ListResourceRequest{
				TypeName:              "test_resource",
				Limit:                 500,
				IncludeResourceObject: true,
			},
		},
		{
			name: "args with variable references",
			queryConfig: `
				list "test_resource" "test1" {
					provider = test
					limit = var.test_limit
					include_resource = var.test_include
				}
				variable "test_limit" {
					default = 500
					type = number
				}
				variable "test_include" {
					default = true
					type = bool
				}
			`,
			assertRequest: providers.ListResourceRequest{
				TypeName:              "test_resource",
				Limit:                 500,
				IncludeResourceObject: true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configFiles := map[string]string{"main.tf": mainConfig}
			configFiles["main.tfquery.hcl"] = tc.queryConfig

			mod := testModuleInline(t, configFiles, configs.MatchQueryFiles())

			providerAddr := addrs.NewDefaultProvider("test")
			provider := testProvider("test")
			provider.ConfigureProvider(providers.ConfigureProviderRequest{})
			provider.GetProviderSchemaResponse = getListProviderSchemaResp()
			var recordedRequest providers.ListResourceRequest
			provider.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
				if request.Config.IsNull() || request.Config.GetAttr("config").IsNull() {
					t.Fatalf("config should never be null, got null for %s", request.TypeName)
				}
				recordedRequest = request
				return provider.ListResourceResponse
			}

			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					providerAddr: testProviderFuncFixed(provider),
				},
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			_, diags = ctx.Plan(mod, states.NewState(), &PlanOpts{
				Mode:         plans.NormalMode,
				SetVariables: testInputValuesUnset(mod.Module.Variables),
				Query:        true,
			})
			if len(diags) != tc.diagCount {
				t.Fatalf("expected %d diagnostics, got %d \n -diags: %s", tc.diagCount, len(diags), diags)
			}

			if diff := cmp.Diff(tc.assertRequest, recordedRequest, ctydebug.CmpOptions, cmpopts.IgnoreFields(providers.ListResourceRequest{}, "Config")); diff != "" {
				t.Fatalf("unexpected request: %s", diff)
			}

			if tc.diagCount > 0 {
				for _, err := range tc.expectedErrMsg {
					if !strings.Contains(diags.Err().Error(), err) {
						t.Fatalf("expected error message %q, but got %q", err, diags.Err().Error())
					}
				}
			}

		})
	}
}

// getListProviderSchemaResp returns a mock provider schema response for testing list resources.
// THe schema returned here is a mock of what the internal protobuf layer would return
// for a provider that supports list resources.
func getListProviderSchemaResp() *providers.GetProviderSchemaResponse {
	listSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"data": {
				Type:     cty.DynamicPseudoType,
				Computed: true,
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"config": {
				Block: configschema.Block{
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
				},
				Nesting: configschema.NestingSingle,
			},
		},
	}

	return getProviderSchemaResponseFromProviderSchema(&providerSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"region": {
					Type:     cty.String,
					Optional: true,
				},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"list": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"instance_type": {
						Type:     cty.String,
						Computed: true,
						Optional: true,
					},
					"id": {
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
			"test_resource":       listSchema,
			"test_child_resource": listSchema,
		},
		IdentityTypes: map[string]*configschema.Object{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Required: true,
					},
				},
				Nesting: configschema.NestingSingle,
			},
			"test_child_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Required: true,
					},
				},
				Nesting: configschema.NestingSingle,
			},
		},
	})
}

func TestContext2Plan_queryListConfigGeneration(t *testing.T) {
	listResourceFn := func(request providers.ListResourceRequest) providers.ListResourceResponse {
		instanceTypes := []string{"ami-123456", "ami-654321", "ami-789012"}
		madeUp := []cty.Value{}
		for i := range len(instanceTypes) {
			madeUp = append(madeUp, cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal(instanceTypes[i])}))
		}

		ids := []cty.Value{}
		for i := range madeUp {
			ids = append(ids, cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(fmt.Sprintf("i-v%d", i+1)),
			}))
		}

		resp := []cty.Value{}
		for i, v := range madeUp {
			mp := map[string]cty.Value{
				"identity":     ids[i],
				"display_name": cty.StringVal(fmt.Sprintf("Instance %d", i+1)),
			}
			if request.IncludeResourceObject {
				mp["state"] = v
			}
			resp = append(resp, cty.ObjectVal(mp))
		}

		ret := map[string]cty.Value{
			"data": cty.TupleVal(resp),
		}
		for k, v := range request.Config.AsValueMap() {
			if k != "data" {
				ret[k] = v
			}
		}

		return providers.ListResourceResponse{Result: cty.ObjectVal(ret)}
	}

	mainConfig := `
		terraform {
			required_providers {
				test = {
					source = "hashicorp/test"
					version = "1.0.0"
				}
			}
		}
		`
	queryConfig := `
		variable "input" {
			type = string
			default = "foo"
		}
		
		list "test_resource" "test2" {
			for_each = toset(["§us-east-2", "§us-west-1"])
			provider = test

			config {
				filter = {
					attr = var.input
				}
			}
		}
		`

	configFiles := map[string]string{"main.tf": mainConfig}
	configFiles["main.tfquery.hcl"] = queryConfig

	mod := testModuleInline(t, configFiles, configs.MatchQueryFiles())
	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")
	provider.ConfigureProvider(providers.ConfigureProviderRequest{})
	provider.GetProviderSchemaResponse = getListProviderSchemaResp()

	var requestConfigs = make(map[string]cty.Value)
	provider.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
		if request.Config.IsNull() || request.Config.GetAttr("config").IsNull() {
			t.Fatalf("config should never be null, got null for %s", request.TypeName)
		}
		requestConfigs[request.TypeName] = request.Config
		return listResourceFn(request)
	}

	ctx, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: testProviderFuncFixed(provider),
		},
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	diags = ctx.Validate(mod, &ValidateOpts{
		Query: true,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	generatedPath := t.TempDir()
	plan, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:               plans.NormalMode,
		SetVariables:       testInputValuesUnset(mod.Module.Variables),
		Query:              true,
		GenerateConfigPath: generatedPath,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	sch, err := ctx.Schemas(mod, states.NewState())
	if err != nil {
		t.Fatalf("failed to get schemas: %s", err)
	}

	expectedResources := []string{
		`list.test_resource.test2["§us-east-2"]`,
		`list.test_resource.test2["§us-west-1"]`,
	}
	actualResources := make([]string, 0)
	generatedCfgs := make([]string, 0)
	uniqCfgs := make(map[string]struct{})

	for _, change := range plan.Changes.Queries {
		actualResources = append(actualResources, change.Addr.String())
		schema := sch.Providers[providerAddr].ListResourceTypes[change.Addr.Resource.Resource.Type]
		cs, err := change.Decode(schema)
		if err != nil {
			t.Fatalf("failed to decode change: %s", err)
		}

		// Verify data. If the state is included, we check that, otherwise we check the id.
		expectedData := []string{"ami-123456", "ami-654321", "ami-789012"}
		includeState := change.Addr.String() == "list.test_resource.test"
		if !includeState {
			expectedData = []string{"i-v1", "i-v2", "i-v3"}
		}
		actualData := make([]string, 0)
		obj := cs.Results.Value.GetAttr("data")
		if obj.IsNull() {
			t.Fatalf("Expected 'data' attribute to be present, but it is null")
		}
		obj.ForEachElement(func(key cty.Value, val cty.Value) bool {
			if includeState {
				val = val.GetAttr("state")
				if val.IsNull() {
					t.Fatalf("Expected 'state' attribute to be present, but it is null")
				}
				if val.GetAttr("instance_type").IsNull() {
					t.Fatalf("Expected 'instance_type' attribute to be present, but it is missing")
				}
				actualData = append(actualData, val.GetAttr("instance_type").AsString())
			} else {
				val = val.GetAttr("identity")
				if val.IsNull() {
					t.Fatalf("Expected 'identity' attribute to be present, but it is null")
				}
				if val.GetAttr("id").IsNull() {
					t.Fatalf("Expected 'id' attribute to be present, but it is missing")
				}
				actualData = append(actualData, val.GetAttr("id").AsString())
			}
			return false
		})
		sort.Strings(actualData)
		sort.Strings(expectedData)
		if diff := cmp.Diff(expectedData, actualData); diff != "" {
			t.Fatalf("Expected instance types to match, but they differ: %s", diff)
		}

		generatedCfgs = append(generatedCfgs, change.Generated.String())
		uniqCfgs[change.Addr.String()] = struct{}{}
	}

	sort.Strings(actualResources)
	sort.Strings(expectedResources)
	if diff := cmp.Diff(expectedResources, actualResources); diff != "" {
		t.Fatalf("Expected resources to match, but they differ: %s", diff)
	}

	// Verify no managed resources are created
	if len(plan.Changes.Resources) != 0 {
		t.Fatalf("Expected no managed resources, but got %d", len(plan.Changes.Resources))
	}

	// Verify generated configs match expected
	expected := `resource "test_resource" "test2_0_0" {
  provider      = test
  instance_type = "ami-123456"
}

import {
  to       = test_resource.test2_0_0
  provider = test
  identity = {
    id = "i-v1"
  }
}

resource "test_resource" "test2_0_1" {
  provider      = test
  instance_type = "ami-654321"
}

import {
  to       = test_resource.test2_0_1
  provider = test
  identity = {
    id = "i-v2"
  }
}

resource "test_resource" "test2_0_2" {
  provider      = test
  instance_type = "ami-789012"
}

import {
  to       = test_resource.test2_0_2
  provider = test
  identity = {
    id = "i-v3"
  }
}
`
	joinedCfgs := strings.Join(generatedCfgs, "\n")
	if !strings.Contains(joinedCfgs, expected) {
		t.Fatalf("Expected config to contain expected resource, but it does not: %s", cmp.Diff(expected, joinedCfgs))
	}

	// Verify that the generated config is valid.
	// The function panics if the config is invalid.
	testModuleInline(t, map[string]string{
		"main.tf": strings.Join(generatedCfgs, "\n"),
	})
}

var (
	testResourceCfg = `resource "test_resource" "test_0" {
  provider      = test
  instance_type = "ami-123456"
}

import {
  to       = test_resource.test_0
  provider = test
  identity = {
    id = "i-v1"
  }
}

resource "test_resource" "test_1" {
  provider      = test
  instance_type = "ami-654321"
}

import {
  to       = test_resource.test_1
  provider = test
  identity = {
    id = "i-v2"
  }
}

resource "test_resource" "test_2" {
  provider      = test
  instance_type = "ami-789012"
}

import {
  to       = test_resource.test_2
  provider = test
  identity = {
    id = "i-v3"
  }
}

`

	testResourceCfg2 = `resource "test_resource" "test2_0" {
  provider      = test
  instance_type = "ami-123456"
}

import {
  to       = test_resource.test2_0
  provider = test
  identity = {
    id = "i-v1"
  }
}

resource "test_resource" "test2_1" {
  provider      = test
  instance_type = "ami-654321"
}

import {
  to       = test_resource.test2_1
  provider = test
  identity = {
    id = "i-v2"
  }
}

resource "test_resource" "test2_2" {
  provider      = test
  instance_type = "ami-789012"
}

import {
  to       = test_resource.test2_2
  provider = test
  identity = {
    id = "i-v3"
  }
}

`
)
