// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hcl

import (
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"

	"testing"
)

func TestFileVariables(t *testing.T) {

	tcs := map[string]struct {
		TestFileVariableExpressions map[string]string
		ExternalVariableValues      map[string]string
		TestFileVariableDefinitions map[string]*configs.Variable
		Want                        map[string]cty.Value
	}{
		"no_variables": {
			Want: make(map[string]cty.Value),
		},
		"string": {
			TestFileVariableExpressions: map[string]string{
				"foo": `"bar"`,
			},
			Want: map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			},
		},
		"boolean": {
			TestFileVariableExpressions: map[string]string{
				"foo": "true",
			},
			Want: map[string]cty.Value{
				"foo": cty.BoolVal(true),
			},
		},
		"reference": {
			TestFileVariableExpressions: map[string]string{
				"foo": "var.bar",
			},
			ExternalVariableValues: map[string]string{
				"bar": `"baz"`,
			},
			TestFileVariableDefinitions: map[string]*configs.Variable{
				"bar": {
					ParsingMode:    configs.VariableParseHCL,
					ConstraintType: cty.String,
				},
			},
			Want: map[string]cty.Value{
				"foo": cty.StringVal("baz"),
			},
		},
		"reference to missing external": {
			TestFileVariableExpressions: map[string]string{
				"foo": "var.bar",
			},
			ExternalVariableValues: map[string]string{
				"bar": `"baz"`,
			},
			Want: map[string]cty.Value{
				"foo": cty.StringVal("baz"),
			},
		},
		"reference with default": {
			TestFileVariableExpressions: map[string]string{
				"foo": "var.bar",
			},
			TestFileVariableDefinitions: map[string]*configs.Variable{
				"bar": {
					ParsingMode:    configs.VariableParseLiteral,
					ConstraintType: cty.String,
					Default:        cty.StringVal("baz"),
				},
			},
			Want: map[string]cty.Value{
				"foo": cty.StringVal("baz"),
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			cache := &VariableCache{
				TestFileVariableExpressions: func() map[string]hcl.Expression {
					vars := make(map[string]hcl.Expression)
					for name, value := range tc.TestFileVariableExpressions {
						expr, diags := hclsyntax.ParseExpression([]byte(value), "test.tf", hcl.Pos{Line: 0, Column: 0, Byte: 0})
						if len(diags) > 0 {
							t.Fatalf("unexpected errors: %v", diags)
						}
						vars[name] = expr
					}
					return vars
				}(),
				ExternalVariableValues: func() map[string]backendrun.UnparsedVariableValue {
					vars := make(map[string]backendrun.UnparsedVariableValue)
					for name, value := range tc.ExternalVariableValues {
						vars[name] = &variable{name, value}
					}
					return vars
				}(),
				TestFileVariableDefinitions: tc.TestFileVariableDefinitions,
			}

			got := make(map[string]cty.Value)
			for name := range tc.Want {
				value, diags := cache.GetVariableValue(name)
				if diags.HasErrors() {
					t.Fatalf("unexpected errors: %v", diags)
				}
				got[name] = value.Value
			}

			if diff := cmp.Diff(tc.Want, got, ctydebug.CmpOptions); len(diff) > 0 {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

var _ backendrun.UnparsedVariableValue = (*variable)(nil)

type variable struct {
	name  string
	value string
}

func (v *variable) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	value, valueDiags := mode.Parse(v.name, v.value)
	diags = diags.Append(valueDiags)
	return &terraform.InputValue{
		Value:      value,
		SourceType: terraform.ValueFromUnknown,
	}, diags
}
