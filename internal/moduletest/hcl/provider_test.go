// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hcl

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestProviderConfig(t *testing.T) {

	tcs := map[string]struct {
		content         string
		schema          *hcl.BodySchema
		variables       map[string]cty.Value
		runBlockOutputs map[string]map[string]cty.Value
		validate        func(t *testing.T, content *hcl.BodyContent)
		expectedErrors  []string
	}{
		"simple_no_vars": {
			content: "attribute = \"string\"",
			schema: &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name: "attribute",
					},
				},
			},
			validate: func(t *testing.T, content *hcl.BodyContent) {
				equals(t, content, "attribute", cty.StringVal("string"))
			},
		},
		"simple_var_ref": {
			content: "attribute = var.input",
			schema: &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name: "attribute",
					},
				},
			},
			variables: map[string]cty.Value{
				"input": cty.StringVal("string"),
			},
			validate: func(t *testing.T, content *hcl.BodyContent) {
				equals(t, content, "attribute", cty.StringVal("string"))
			},
		},
		"missing_var_ref": {
			content: "attribute = var.missing",
			schema: &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name: "attribute",
					},
				},
			},
			variables: map[string]cty.Value{
				"input": cty.StringVal("string"),
			},
			expectedErrors: []string{
				"The input variable \"missing\" is not available to the current provider configuration. You can only reference variables defined at the file or global levels.",
			},
			validate: func(t *testing.T, content *hcl.BodyContent) {
				if len(content.Attributes) > 0 {
					t.Errorf("should have excluded the invalid attribute but found %d", len(content.Attributes))
				}
			},
		},
		"simple_run_block": {
			content: "attribute = run.setup.value",
			schema: &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name: "attribute",
					},
				},
			},
			runBlockOutputs: map[string]map[string]cty.Value{
				"setup": {
					"value": cty.StringVal("string"),
				},
			},
			validate: func(t *testing.T, content *hcl.BodyContent) {
				equals(t, content, "attribute", cty.StringVal("string"))
			},
		},
		"missing_run_block": {
			content: "attribute = run.missing.value",
			schema: &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name: "attribute",
					},
				},
			},
			runBlockOutputs: map[string]map[string]cty.Value{
				"setup": {
					"value": cty.StringVal("string"),
				},
			},
			expectedErrors: []string{
				"The run block \"missing\" does not exist within this test file. You can only reference run blocks that are in the same test file and will execute before the provider is required.",
			},
			validate: func(t *testing.T, content *hcl.BodyContent) {
				if len(content.Attributes) > 0 {
					t.Errorf("should have excluded the invalid attribute but found %d", len(content.Attributes))
				}
			},
		},
		"late_run_block": {
			content: "attribute = run.setup.value",
			schema: &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name: "attribute",
					},
				},
			},
			runBlockOutputs: map[string]map[string]cty.Value{
				"setup": nil,
			},
			expectedErrors: []string{
				"The run block \"setup\" has not executed yet. You can only reference run blocks that are in the same test file and will execute before the provider is required.",
			},
			validate: func(t *testing.T, content *hcl.BodyContent) {
				if len(content.Attributes) > 0 {
					t.Errorf("should have excluded the invalid attribute but found %d", len(content.Attributes))
				}
			},
		},
		"invalid_ref": {
			content: "attribute = data.type.name.value",
			schema: &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name: "attribute",
					},
				},
			},
			runBlockOutputs: map[string]map[string]cty.Value{
				"setup": nil,
			},
			expectedErrors: []string{
				"You can only reference run blocks, file level, and global variables while defining variables from inside provider configurations.",
			},
			validate: func(t *testing.T, content *hcl.BodyContent) {
				if len(content.Attributes) > 0 {
					t.Errorf("should have excluded the invalid attribute but found %d", len(content.Attributes))
				}
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			file, diags := hclsyntax.ParseConfig([]byte(tc.content), "main.tf", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatalf("failed to parse hcl: %s", diags.Error())
			}

			outputs := make(map[addrs.Run]cty.Value)
			for name, values := range tc.runBlockOutputs {
				addr := addrs.Run{Name: name}
				if values == nil {
					outputs[addr] = cty.NilVal
					continue
				}

				attrs := make(map[string]cty.Value)
				for name, value := range values {
					attrs[name] = value
				}

				outputs[addr] = cty.ObjectVal(attrs)
			}

			variableCaches := NewVariableCaches(func(vc *VariableCaches) {
				vc.FileVariables = func() map[string]hcl.Expression {
					variables := make(map[string]hcl.Expression)
					for name, value := range tc.variables {
						variables[name] = hcl.StaticExpr(value, hcl.Range{})
					}
					return variables
				}()
			})

			config := ProviderConfig{
				Original:            file.Body,
				VariableCache:       variableCaches.GetCache("test", nil),
				AvailableRunOutputs: outputs,
			}

			content, diags := config.Content(tc.schema)

			var actualErrs []string
			for _, diag := range diags {
				actualErrs = append(actualErrs, diag.Detail)
			}
			if diff := cmp.Diff(actualErrs, tc.expectedErrors); len(diff) > 0 {
				t.Errorf("unmatched errors\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", strings.Join(tc.expectedErrors, "\n"), strings.Join(actualErrs, "\n"), diff)
			}

			tc.validate(t, content)
		})
	}
}

func equals(t *testing.T, content *hcl.BodyContent, attribute string, expected cty.Value) {
	value, diags := content.Attributes[attribute].Expr.Value(nil)
	if diags.HasErrors() {
		t.Errorf("failed to get value from attribute %s: %s", attribute, diags.Error())
	}
	if !value.RawEquals(expected) {
		t.Errorf("expected:\n%s\nbut got:\n%s", expected.GoString(), value.GoString())
	}
}
