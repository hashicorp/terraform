// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Plan_queryList(t *testing.T) {
	cases := []struct {
		name           string
		mainConfig     string
		queryConfig    string
		diagCount      int
		expectedErrMsg []string
		assertState    func(*states.State)
		assertChanges  func(providers.ProviderSchema, *plans.ChangesSrc)
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
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				madeUp := []cty.Value{
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-123456")}),
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-654321")}),
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-789012")}),
				}
				ids := []cty.Value{}
				for i := range madeUp {
					ids = append(ids, cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal(fmt.Sprintf("i-v%d", i+1)),
					}))
				}

				resp := []cty.Value{}
				if request.IncludeResourceObject {
					for i, v := range madeUp {
						resp = append(resp, cty.ObjectVal(map[string]cty.Value{
							"state":        v,
							"identity":     ids[i],
							"display_name": cty.StringVal(fmt.Sprintf("Instance %d", i+1)),
						}))
					}
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
			},
			assertChanges: func(sch providers.ProviderSchema, changes *plans.ChangesSrc) {
				expectedResources := map[string][]string{
					"list.test_resource.test":  {"ami-123456", "ami-654321", "ami-789012"},
					"list.test_resource.test2": {},
				}
				actualResources := map[string][]string{}
				for _, change := range changes.Resources {
					schema := sch.ListResourceTypes[change.Addr.Resource.Resource.Type]
					cs, err := change.Decode(schema)
					if err != nil {
						t.Fatalf("failed to decode change: %s", err)
					}

					if cs.Change.Action != plans.Read {
						t.Fatalf("expected action to be Read, got %s", cs.Change.Action)
					}

					// Verify instance types
					actualTypes := make([]string, 0)
					obj := cs.After.GetAttr("data")
					if obj.IsNull() {
						t.Fatalf("Expected 'data' attribute to be present, but it is null")
					}
					obj.ForEachElement(func(key cty.Value, val cty.Value) bool {
						if !val.Type().HasAttribute("state") {
							t.Fatalf("Expected 'state' attribute to be present, but it is missing")
						}

						val = val.GetAttr("state")
						if !val.IsNull() {
							if val.GetAttr("instance_type").IsNull() {
								t.Fatalf("Expected 'instance_type' attribute to be present, but it is missing")
							}
							actualTypes = append(actualTypes, val.GetAttr("instance_type").AsString())
						}

						return false
					})
					sort.Strings(actualTypes)
					actualResources[change.Addr.String()] = actualTypes
				}

				if diff := cmp.Diff(expectedResources, actualResources); diff != "" {
					t.Fatalf("Expected resources to match, but they differ: %s", diff)
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
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				madeUp := []cty.Value{
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-123456")}),
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-654321")}),
				}
				ids := []cty.Value{}
				for i := range madeUp {
					ids = append(ids, cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal(fmt.Sprintf("i-v%d", i+1)),
					}))
				}

				resp := []cty.Value{}
				for i, v := range madeUp {
					resp = append(resp, cty.ObjectVal(map[string]cty.Value{
						"state":        v,
						"identity":     ids[i],
						"display_name": cty.StringVal(fmt.Sprintf("Instance %d", i+1)),
					}))
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
			},
			assertChanges: func(sch providers.ProviderSchema, changes *plans.ChangesSrc) {
				expectedResources := []string{"list.test_resource.test[0]", "list.test_resource.test2"}
				actualResources := make([]string, 0)
				for _, change := range changes.Resources {
					actualResources = append(actualResources, change.Addr.String())
					schema := sch.ListResourceTypes[change.Addr.Resource.Resource.Type]
					cs, err := change.Decode(schema)
					if err != nil {
						t.Fatalf("failed to decode change: %s", err)
					}

					if cs.Change.Action != plans.Read {
						t.Fatalf("expected action to be Read, got %s", cs.Change.Action)
					}

					// Verify instance types
					expectedTypes := []string{"ami-123456", "ami-654321"}
					actualTypes := make([]string, 0)
					obj := cs.After.GetAttr("data")
					if obj.IsNull() {
						t.Fatalf("Expected 'data' attribute to be present, but it is null")
					}
					obj.ForEachElement(func(key cty.Value, val cty.Value) bool {
						val = val.GetAttr("state")
						if val.IsNull() {
							t.Fatalf("Expected 'state' attribute to be present, but it is null")
						}
						if val.GetAttr("instance_type").IsNull() {
							t.Fatalf("Expected 'instance_type' attribute to be present, but it is missing")
						}
						actualTypes = append(actualTypes, val.GetAttr("instance_type").AsString())
						return false
					})
					sort.Strings(actualTypes)
					sort.Strings(expectedTypes)
					if diff := cmp.Diff(expectedTypes, actualTypes); diff != "" {
						t.Fatalf("Expected instance types to match, but they differ: %s", diff)
					}
				}
				sort.Strings(actualResources)
				sort.Strings(expectedResources)
				if diff := cmp.Diff(expectedResources, actualResources); diff != "" {
					t.Fatalf("Expected resources to match, but they differ: %s", diff)
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
			diagCount: 1,
			expectedErrMsg: []string{
				"Invalid list resource traversal",
				"The first step in the traversal for a list resource must be an attribute \"data\"",
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
			diagCount: 1,
			expectedErrMsg: []string{
				"A list resource \"non_existent\" \"attr\" has not been declared in the root module.",
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
			diagCount: 1,
			expectedErrMsg: []string{
				"Unsupported attribute: This object has no argument, nested block, or exported attribute named \"invalid_attr\".",
			},
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				madeUp := []cty.Value{
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-123456")}),
				}
				ids := []cty.Value{}
				for i := range madeUp {
					ids = append(ids, cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal(fmt.Sprintf("i-v%d", i+1)),
					}))
				}

				resp := []cty.Value{}
				for i, v := range madeUp {
					resp = append(resp, cty.ObjectVal(map[string]cty.Value{
						"state":        v,
						"identity":     ids[i],
						"display_name": cty.StringVal(fmt.Sprintf("Instance %d", i+1)),
					}))
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
			diagCount: 1,
			expectedErrMsg: []string{
				"Cycle: list.test_resource",
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
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				madeUp := []cty.Value{
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-123456")}),
				}
				ids := []cty.Value{}
				for i := range madeUp {
					ids = append(ids, cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal(fmt.Sprintf("i-v%d", i+1)),
					}))
				}

				resp := []cty.Value{}
				for i, v := range madeUp {
					resp = append(resp, cty.ObjectVal(map[string]cty.Value{
						"state":        v,
						"identity":     ids[i],
						"display_name": cty.StringVal(fmt.Sprintf("Instance %d", i+1)),
					}))
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
			},
			assertChanges: func(sch providers.ProviderSchema, changes *plans.ChangesSrc) {
				expectedResources := []string{"list.test_resource.test1", "list.test_resource.test2"}
				actualResources := make([]string, 0)
				for _, change := range changes.Resources {
					actualResources = append(actualResources, change.Addr.String())
					schema := sch.ListResourceTypes[change.Addr.Resource.Resource.Type]
					cs, err := change.Decode(schema)
					if err != nil {
						t.Fatalf("failed to decode change: %s", err)
					}

					if cs.Change.Action != plans.Read {
						t.Fatalf("expected action to be Read, got %s", cs.Change.Action)
					}

					// Verify instance types
					expectedTypes := []string{"ami-123456"}
					actualTypes := make([]string, 0)
					obj := cs.After.GetAttr("data")
					if obj.IsNull() {
						t.Fatalf("Expected 'data' attribute to be present, but it is null")
					}
					obj.ForEachElement(func(key cty.Value, val cty.Value) bool {
						val = val.GetAttr("state")
						if val.IsNull() {
							t.Fatalf("Expected 'state' attribute to be present, but it is null")
						}
						if val.GetAttr("instance_type").IsNull() {
							t.Fatalf("Expected 'instance_type' attribute to be present, but it is missing")
						}
						actualTypes = append(actualTypes, val.GetAttr("instance_type").AsString())
						return false
					})
					sort.Strings(actualTypes)
					sort.Strings(expectedTypes)
					if diff := cmp.Diff(expectedTypes, actualTypes); diff != "" {
						t.Fatalf("Expected instance types to match, but they differ: %s", diff)
					}
				}
				sort.Strings(actualResources)
				sort.Strings(expectedResources)
				if diff := cmp.Diff(expectedResources, actualResources); diff != "" {
					t.Fatalf("Expected resources to match, but they differ: %s", diff)
				}
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
			listResourceFn: func(request providers.ListResourceRequest) providers.ListResourceResponse {
				madeUp := []cty.Value{
					cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-123456")}),
				}
				ids := []cty.Value{}
				for i := range madeUp {
					ids = append(ids, cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal(fmt.Sprintf("i-v%d", i+1)),
					}))
				}

				resp := []cty.Value{}
				for i, v := range madeUp {
					resp = append(resp, cty.ObjectVal(map[string]cty.Value{
						"state":        v,
						"identity":     ids[i],
						"display_name": cty.StringVal(fmt.Sprintf("Instance %d", i+1)),
					}))
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
			},
			assertChanges: func(sch providers.ProviderSchema, changes *plans.ChangesSrc) {
				expectedResources := []string{"list.test_resource.test1[\"foo\"]", "list.test_resource.test1[\"bar\"]", "list.test_resource.test2[\"foo\"]", "list.test_resource.test2[\"bar\"]"}
				actualResources := make([]string, 0)
				for _, change := range changes.Resources {
					actualResources = append(actualResources, change.Addr.String())
					schema := sch.ListResourceTypes[change.Addr.Resource.Resource.Type]
					cs, err := change.Decode(schema)
					if err != nil {
						t.Fatalf("failed to decode change: %s", err)
					}

					if cs.Change.Action != plans.Read {
						t.Fatalf("expected action to be Read, got %s", cs.Change.Action)
					}

					// Verify instance types
					expectedTypes := []string{"ami-123456"}
					actualTypes := make([]string, 0)
					obj := cs.After.GetAttr("data")
					if obj.IsNull() {
						t.Fatalf("Expected 'data' attribute to be present, but it is null")
					}
					obj.ForEachElement(func(key cty.Value, val cty.Value) bool {
						val = val.GetAttr("state")
						if val.IsNull() {
							t.Fatalf("Expected 'state' attribute to be present, but it is null")
						}
						if val.GetAttr("instance_type").IsNull() {
							t.Fatalf("Expected 'instance_type' attribute to be present, but it is missing")
						}
						actualTypes = append(actualTypes, val.GetAttr("instance_type").AsString())
						return false
					})
					sort.Strings(actualTypes)
					sort.Strings(expectedTypes)
					if diff := cmp.Diff(expectedTypes, actualTypes); diff != "" {
						t.Fatalf("Expected instance types to match, but they differ: %s", diff)
					}
				}
				sort.Strings(actualResources)
				sort.Strings(expectedResources)
				if diff := cmp.Diff(expectedResources, actualResources); diff != "" {
					t.Fatalf("Expected resources to match, but they differ: %s", diff)
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

			mod := testModuleInline(t, configs)
			providerAddr := addrs.NewDefaultProvider("test")
			provider := testProvider("test")
			provider.ConfigureProvider(providers.ConfigureProviderRequest{})
			provider.GetProviderSchemaResponse = getListProviderSchemaResp()
			var requestConfigs = make(map[string]cty.Value)
			provider.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
				requestConfigs[request.TypeName] = request.Config
				fn := tc.listResourceFn
				if fn == nil {
					return provider.ListResourceResponse
				}
				return fn(request)
			}

			ctx, diags := NewContext(&ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					providerAddr: testProviderFuncFixed(provider),
				},
			})
			tfdiags.AssertNoDiagnostics(t, diags)

			plan, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
				Mode:         plans.NormalMode,
				SetVariables: testInputValuesUnset(mod.Module.Variables),
				Query:        true,
			})
			if len(diags) != tc.diagCount {
				t.Fatalf("expected %d diagnostics, got %d \n -diags: %s", tc.diagCount, len(diags), diags)
			}

			if tc.assertChanges != nil {
				sch, err := ctx.Schemas(mod, states.NewState())
				if err != nil {
					t.Fatalf("failed to get schemas: %s", err)
				}
				tc.assertChanges(sch.Providers[providerAddr], plan.Changes)
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
			configs := map[string]string{"main.tf": mainConfig}
			configs["main.tfquery.hcl"] = tc.queryConfig

			mod := testModuleInline(t, configs)

			providerAddr := addrs.NewDefaultProvider("test")
			provider := testProvider("test")
			provider.ConfigureProvider(providers.ConfigureProviderRequest{})
			provider.GetProviderSchemaResponse = getListProviderSchemaResp()
			var recordedRequest providers.ListResourceRequest
			provider.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
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
