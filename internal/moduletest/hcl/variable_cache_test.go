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
		Values       map[string]string
		GlobalValues map[string]string
		Variables    map[string]configs.VariableParsingMode
		Want         map[string]cty.Value
	}{
		"no_variables": {
			Want: make(map[string]cty.Value),
		},
		"string": {
			Values: map[string]string{
				"foo": `"bar"`,
			},
			Want: map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			},
		},
		"boolean": {
			Values: map[string]string{
				"foo": "true",
			},
			Want: map[string]cty.Value{
				"foo": cty.BoolVal(true),
			},
		},
		"reference": {
			Values: map[string]string{
				"foo": "var.bar",
			},
			GlobalValues: map[string]string{
				"bar": `"baz"`,
			},
			Variables: map[string]configs.VariableParsingMode{
				"foo": configs.VariableParseLiteral,
			},
			Want: map[string]cty.Value{
				"foo": cty.StringVal("baz"),
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			caches := NewVariableCaches(func(vc *VariableCaches) {
				vc.FileVariables = func() map[string]hcl.Expression {
					vars := make(map[string]hcl.Expression)
					for name, value := range tc.Values {
						expr, diags := hclsyntax.ParseExpression([]byte(value), "test.tf", hcl.Pos{Line: 0, Column: 0, Byte: 0})
						if len(diags) > 0 {
							t.Fatalf("unexpected errors: %v", diags)
						}
						vars[name] = expr
					}
					return vars
				}()
				vc.GlobalVariables = func() map[string]backendrun.UnparsedVariableValue {
					vars := make(map[string]backendrun.UnparsedVariableValue)
					for name, value := range tc.GlobalValues {
						vars[name] = &variable{name, value}
					}
					return vars
				}()
			})
			config := makeConfigWithVariables(tc.Variables)

			cache := caches.GetCache("test", config)
			got := make(map[string]cty.Value)
			for name := range tc.Want {
				value, diags := cache.GetFileVariable(name)
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

func TestGlobalVariables(t *testing.T) {

	tcs := map[string]struct {
		Values    map[string]string
		Variables map[string]configs.VariableParsingMode
		Want      map[string]cty.Value
	}{
		"no_variables": {
			Want: make(map[string]cty.Value),
		},
		"string": {
			Values: map[string]string{
				"foo": "bar",
			},
			Variables: map[string]configs.VariableParsingMode{
				"foo": configs.VariableParseLiteral,
			},
			Want: map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			},
		},
		"boolean_string": {
			Values: map[string]string{
				"foo": "true",
			},
			Variables: map[string]configs.VariableParsingMode{
				"foo": configs.VariableParseLiteral,
			},
			Want: map[string]cty.Value{
				"foo": cty.StringVal("true"),
			},
		},
		"boolean": {
			Values: map[string]string{
				"foo": "true",
			},
			Variables: map[string]configs.VariableParsingMode{
				"foo": configs.VariableParseHCL,
			},
			Want: map[string]cty.Value{
				"foo": cty.BoolVal(true),
			},
		},
		"string_hcl": {
			Values: map[string]string{
				"foo": `"bar"`,
			},
			Variables: map[string]configs.VariableParsingMode{
				"foo": configs.VariableParseHCL,
			},
			Want: map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			},
		},
		"missing_config": {
			Values: map[string]string{
				"foo": `"bar"`,
			},
			Want: map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			caches := NewVariableCaches(func(vc *VariableCaches) {
				vc.GlobalVariables = func() map[string]backendrun.UnparsedVariableValue {
					vars := make(map[string]backendrun.UnparsedVariableValue)
					for name, value := range tc.Values {
						vars[name] = &variable{name, value}
					}
					return vars
				}()
			})

			config := makeConfigWithVariables(tc.Variables)

			cache := caches.GetCache("test", config)
			got := make(map[string]cty.Value)
			for name := range tc.Want {
				value, diags := cache.GetGlobalVariable(name)
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

func makeConfigWithVariables(modes map[string]configs.VariableParsingMode) *configs.Config {
	return &configs.Config{
		Module: &configs.Module{
			Variables: func() map[string]*configs.Variable {
				vars := make(map[string]*configs.Variable)
				for name, mode := range modes {
					vars[name] = &configs.Variable{
						ParsingMode: mode,
					}
				}
				return vars
			}(),
		},
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
