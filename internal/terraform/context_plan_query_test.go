// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Plan_queryList(t *testing.T) {
	schemaResp := getProviderSchemaResponseFromProviderSchema(&providerSchema{
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
		extraConfig    map[string]string
		queryConfig    string
		diagCount      int
		expectedErrMsg []string
		assertState    func(*states.State)
		InputVariables InputValues
		listResourceFn func(request providers.ListResourceRequest) providers.ListResourceResponse
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
					provider = test

					filter = {
						attr = var.input
					}
				}

				list "test_resource" "test2" {
					provider = test

					filter = {
						attr = list.test_resource.test.data[0].instance_type
					}
				}
				`,
			InputVariables: InputValues{
				"input": &InputValue{
					Value: cty.StringVal("foo"),
				},
			},
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
			assertState: func(state *states.State) {
				// Verify test list resource
				testInst := state.GetListResource(addrs.RootModuleInstance.Resource(addrs.ListResourceMode, "test_resource", "test"))
				if testInst.Len() == 0 {
					t.Fatalf("Expected list resource test_resource.test to exist in state, but it doesn't")
				}

				// Verify test2 list resource
				test2Inst := state.GetListResource(addrs.RootModuleInstance.Resource(addrs.ListResourceMode, "test_resource", "test2"))
				if test2Inst.Len() == 0 {
					t.Fatalf("Expected list resource test_child_resource.test2 to exist in state, but it doesn't")
				}

				// Verify instance types
				expectedTypes := []string{"ami-123456", "ami-654321", "ami-789012"}
				actualTypes := make([]string, 0)
				for _, obj := range testInst.Elements() {
					val := obj.Value
					val.Value.ForEachElement(func(key cty.Value, val cty.Value) bool {
						actualTypes = append(actualTypes, val.GetAttr("instance_type").AsString())
						return false
					})
				}

				if diff := cmp.Diff(expectedTypes, actualTypes); diff != "" {
					t.Fatalf("Expected instance types to match, but they differ: %s", diff)
				}
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

					filter = {
						attr = var.input
					}
				}

				list "test_resource" "test2" {
					provider = test

					filter = {
						attr = list.test_resource.test[0].data[0].instance_type
					}
				}
				`,
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				madeUp := []cty.Value{
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-123456")}),
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-654321")}),
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
			InputVariables: InputValues{
				"input": &InputValue{
					Value: cty.StringVal("foo"),
				},
			},
			assertState: func(state *states.State) {
				// Check that the plan state contains the list resources
				// We need to check if the list resources exist by iterating through all list resources
				allLists := state.AllListResourceInstances()

				// Check for test_resource.test with count
				testKey := "list.test_resource.test"
				testInst, ok := allLists[testKey]
				if !ok || testInst.Len() == 0 {
					t.Fatalf("Expected list resource test_resource.test with count to exist in state, but it doesn't")
				}

				// Check for test_resource.test2
				test2Key := "list.test_resource.test2"
				test2Inst, ok := allLists[test2Key]
				if !ok || test2Inst.Len() == 0 {
					t.Fatalf("Expected list resource test_resource.test2 to exist in state, but it doesn't")
				}
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

					filter = {
						attr = var.input
					}
				}

				list "test_resource" "test2" {
					provider = test

					filter = {
						attr = list.test_resource.test.instance_type
					}
				}
				`,
			diagCount: 1,
			expectedErrMsg: []string{
				"Invalid list resource traversal",
				"The first step in the traversal for a list resource must be an attribute \"data\"",
			},
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				return func(yield func(providers.ListResourceEvent, error) bool) {
					return
				}
			},
			InputVariables: InputValues{
				"input": &InputValue{
					Value: cty.StringVal("foo"),
				},
			},
		},
		{
			// We tried to reference a resource of type list without using the fully-qualified name.
			// The error contains a hint to help the user.
			name: "reference list block from resource",
			mainConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}

				resource "list" "test_resource1" {
					provider = test
				}

				resource "list" "test_resource2" {
					provider = test
					attr = list.test_resource1.attr
				}
				`,
			queryConfig: `
				variable "input" {
					type = string
					default = "foo"
				}

				list "test_resource" "test" {
					provider = test

					filter = {
						attr = var.input
					}
				}
				`,
			diagCount: 1,
			expectedErrMsg: []string{
				"Reference to undeclared resource",
				"A list resource \"test_resource1\" \"attr\" has not been declared in the root module.",
				"Did you mean the managed resource list.test_resource1? If so, please use the fully qualified name of the resource, e.g. resource.list.test_resource1",
			},
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				return func(yield func(providers.ListResourceEvent, error) bool) {
					return
				}
			},
			InputVariables: InputValues{
				"input": &InputValue{
					Value: cty.StringVal("foo"),
				},
			},
		},
		{
			// We are referencing a managed resource
			// of type list using the resource.<block>.<name> syntax. This should be allowed.
			name: "reference managed resource of type list using resource.<block>.<name>",
			mainConfig: `
				terraform {
					required_providers {
						test = {
							source = "hashicorp/test"
							version = "1.0.0"
						}
					}
				}

				resource "list" "test_resource" {
					provider = test
					attr = "bar"
				}

				resource "list" "normal_resource" {
					provider = test
					attr = resource.list.test_resource.attr
				}
				`,
			queryConfig: `
				list "test_resource" "test" {
					provider = test

					filter = {
						attr = resource.list.test_resource.attr
					}
				}
				`,
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				madeUp := []cty.Value{
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-123456")}),
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
			assertState: func(state *states.State) {
				// Verify test list resource
				testInst := state.GetListResource(addrs.RootModuleInstance.Resource(addrs.ListResourceMode, "test_resource", "test"))
				if testInst.Len() == 0 {
					t.Fatalf("Expected list resource test_resource.test to exist in state, but it doesn't")
				}
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

					filter = {
						attr = list.non_existent.attr
					}
				}
				`,
			diagCount: 1,
			expectedErrMsg: []string{
				"A list resource \"non_existent\" \"attr\" has not been declared in the root module.",
			},
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				return func(yield func(providers.ListResourceEvent, error) bool) {
					return
				}
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

					filter = {
						attr = "valid"
					}
				}

				list "test_resource" "another" {
					provider = test

					filter = {
						attr = list.test_resource.test.data[0].invalid_attr
					}
				}
				`,
			diagCount: 1,
			expectedErrMsg: []string{
				"Unsupported attribute: This object has no argument, nested block, or exported attribute named \"invalid_attr\".",
			},
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				madeUp := []cty.Value{
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-123456")}),
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

					filter = {
						attr = list.test_resource.test2.data[0].id
					}
				}

				list "test_resource" "test2" {
					provider = test

					filter = {
						attr = list.test_resource.test1.data[0].id
					}
				}
				`,
			diagCount: 1,
			expectedErrMsg: []string{
				"Cycle: list.test_resource",
			},
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				return func(yield func(providers.ListResourceEvent, error) bool) {
					return
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

					filter = {
						attr = var.test_var
					}
				}

				list "test_resource" "test2" {
					provider = test

					filter = {
						attr = length(list.test_resource.test1.data) > 0 ? list.test_resource.test1.data[0].instance_type : var.test_var
					}
				}
				`,
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				madeUp := []cty.Value{
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-123456")}),
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
			assertState: func(state *states.State) {
				// Verify test1 list resource
				test1Inst := state.GetListResource(addrs.RootModuleInstance.Resource(addrs.ListResourceMode, "test_resource", "test1"))
				if test1Inst.Len() == 0 {
					t.Fatalf("Expected list resource test_resource.test1 to exist in state, but it doesn't")
				}

				// Verify test2 list resource
				test2Inst := state.GetListResource(addrs.RootModuleInstance.Resource(addrs.ListResourceMode, "test_resource", "test2"))
				if test2Inst.Len() == 0 {
					t.Fatalf("Expected list resource test_resource.test2 to exist in state, but it doesn't")
				}
			},
			InputVariables: InputValues{
				"test_var": &InputValue{
					Value: cty.StringVal("foo"),
				},
			},
		},
		{
			// Test list reference with index but without data field
			name: "list reference with index but without data field",
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

					filter = {
						attr = each.value
					}
				}

				list "test_resource" "test2" {
					provider = test
					for_each = list.test_resource.test1

					filter = {
						attr = each.value.data[0].instance_type
					}
				}
				`,
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				madeUp := []cty.Value{
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-123456")}),
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
			assertState: func(state *states.State) {
				// Check that the plan state contains the list resources with for_each
				allLists := state.AllListResourceInstances()

				// Check for test_resource.test1 with for_each - should have instances for "foo" and "bar"
				test1Key := "list.test_resource.test1"
				test1Inst, ok := allLists[test1Key]
				if !ok {
					t.Fatalf("Expected list resource test_resource.test1 with for_each to exist in state, but it doesn't")
				}

				// We expect 2 instances of test1 (for "foo" and "bar")
				if test1Inst.Len() < 2 {
					t.Fatalf("Expected at least 2 instances of test_resource.test1 with for_each to exist in state, but found %d", test1Inst.Len())
				}

				// Check for test_resource.test2 with for_each from test1
				test2Key := "list.test_resource.test2"
				test2Inst, ok := allLists[test2Key]
				if !ok || test2Inst.Len() == 0 {
					t.Fatalf("Expected instances of test_resource.test2 with for_each to exist in state, but found none")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configs := map[string]string{"main.tf": tc.mainConfig}
			if tc.queryConfig != "" {
				configs["main.tfquery.hcl"] = tc.queryConfig
			}
			maps.Copy(configs, tc.extraConfig)

			m := testModuleInline(t, configs)

			providerAddr := addrs.NewDefaultProvider("test")
			provider := testProvider("test")
			provider.ConfigureProvider(providers.ConfigureProviderRequest{})
			provider.GetProviderSchemaResponse = schemaResp
			var requestConfigs = make(map[string]cty.Value)
			provider.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
				requestConfigs[request.TypeName] = request.Config
				return tc.listResourceFn(request)
			}

			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					providerAddr: testProviderFuncFixed(provider),
				},
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
				Mode:         plans.NormalMode,
				SetVariables: tc.InputVariables,
			})
			if len(diags) != tc.diagCount {
				t.Fatalf("expected %d diagnostics, got %d \n -diags: %s", tc.diagCount, len(diags), diags)
			}

			if tc.assertState != nil {
				tc.assertState(plan.PostPlanState)
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
