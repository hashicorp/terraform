// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestEvaluateForEachExpression_valid(t *testing.T) {
	tests := map[string]struct {
		Expr       hcl.Expression
		ForEachMap map[string]cty.Value
	}{
		"empty set": {
			hcltest.MockExprLiteral(cty.SetValEmpty(cty.String)),
			map[string]cty.Value{},
		},
		"multi-value string set": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")})),
			map[string]cty.Value{
				"a": cty.StringVal("a"),
				"b": cty.StringVal("b"),
			},
		},
		"empty map": {
			hcltest.MockExprLiteral(cty.MapValEmpty(cty.Bool)),
			map[string]cty.Value{},
		},
		"map": {
			hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
				"a": cty.BoolVal(true),
				"b": cty.BoolVal(false),
			})),
			map[string]cty.Value{
				"a": cty.BoolVal(true),
				"b": cty.BoolVal(false),
			},
		},
		"map containing unknown values": {
			hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
				"a": cty.UnknownVal(cty.Bool),
				"b": cty.UnknownVal(cty.Bool),
			})),
			map[string]cty.Value{
				"a": cty.UnknownVal(cty.Bool),
				"b": cty.UnknownVal(cty.Bool),
			},
		},
		"map containing sensitive values, but strings are literal": {
			hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
				"a": cty.BoolVal(true).Mark(marks.Sensitive),
				"b": cty.BoolVal(false),
			})),
			map[string]cty.Value{
				"a": cty.BoolVal(true).Mark(marks.Sensitive),
				"b": cty.BoolVal(false),
			},
		},
		"map containing ephemeral values": {
			hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
				"a": cty.BoolVal(true).Mark(marks.Ephemeral),
				"b": cty.BoolVal(false),
			})),
			map[string]cty.Value{
				"a": cty.BoolVal(true).Mark(marks.Ephemeral),
				"b": cty.BoolVal(false),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := &MockEvalContext{}
			ctx.installSimpleEval()
			forEachMap, _, diags := evaluateForEachExpression(test.Expr, ctx, false)

			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
			}

			if !reflect.DeepEqual(forEachMap, test.ForEachMap) {
				t.Errorf(
					"wrong map value\ngot:  %swant: %s",
					spew.Sdump(forEachMap), spew.Sdump(test.ForEachMap),
				)
			}

		})
	}
}

func TestEvaluateForEachExpression_errors(t *testing.T) {
	tests := map[string]struct {
		Expr                                                  hcl.Expression
		Summary, DetailSubstring                              string
		CausedByUnknown, CausedByEphemeral, CausedBySensitive bool
	}{
		"null set": {
			hcltest.MockExprLiteral(cty.NullVal(cty.Set(cty.String))),
			"Invalid for_each argument",
			`the given "for_each" argument value is null`,
			false, false, false,
		},
		"string": {
			hcltest.MockExprLiteral(cty.StringVal("i am definitely a set")),
			"Invalid for_each argument",
			"must be a map, or set of strings, and you have provided a value of type string",
			false, false, false,
		},
		"list": {
			hcltest.MockExprLiteral(cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("a")})),
			"Invalid for_each argument",
			"must be a map, or set of strings, and you have provided a value of type list",
			false, false, false,
		},
		"tuple": {
			hcltest.MockExprLiteral(cty.TupleVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")})),
			"Invalid for_each argument",
			"must be a map, or set of strings, and you have provided a value of type tuple",
			false, false, false,
		},
		"unknown string set": {
			hcltest.MockExprLiteral(cty.UnknownVal(cty.Set(cty.String))),
			"Invalid for_each argument",
			"set includes values derived from resource attributes that cannot be determined until apply",
			true, false, false,
		},
		"unknown map": {
			hcltest.MockExprLiteral(cty.UnknownVal(cty.Map(cty.Bool))),
			"Invalid for_each argument",
			"map includes keys derived from resource attributes that cannot be determined until apply",
			true, false, false,
		},
		"sensitive map": {
			hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
				"a": cty.BoolVal(true),
				"b": cty.BoolVal(false),
			}).Mark(marks.Sensitive)),
			"Invalid for_each argument",
			"Sensitive values, or values derived from sensitive values, cannot be used as for_each arguments. If used, the sensitive value could be exposed as a resource instance key.",
			false, false, true,
		},
		"ephemeral map": {
			hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
				"a": cty.BoolVal(true),
				"b": cty.BoolVal(false),
			}).Mark(marks.Ephemeral)),
			"Invalid for_each argument",
			`The given "for_each" value is derived from an ephemeral value`,
			false, true, false,
		},
		"set containing booleans": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.BoolVal(true)})),
			"Invalid for_each set argument",
			"supports maps and sets of strings, but you have provided a set containing type bool",
			false, false, false,
		},
		"set containing null": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.NullVal(cty.String)})),
			"Invalid for_each set argument",
			"must not contain null values",
			false, false, false,
		},
		"set containing unknown value": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.UnknownVal(cty.String)})),
			"Invalid for_each argument",
			"set includes values derived from resource attributes that cannot be determined until apply",
			true, false, false,
		},
		"set containing dynamic unknown value": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.UnknownVal(cty.DynamicPseudoType)})),
			"Invalid for_each argument",
			"set includes values derived from resource attributes that cannot be determined until apply",
			true, false, false,
		},
		"set containing sensitive values": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.StringVal("beep").Mark(marks.Sensitive), cty.StringVal("boop")})),
			"Invalid for_each argument",
			"Sensitive values, or values derived from sensitive values, cannot be used as for_each arguments. If used, the sensitive value could be exposed as a resource instance key.",
			false, false, true,
		},
		"set containing ephemeral values": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.StringVal("beep").Mark(marks.Ephemeral), cty.StringVal("boop")})),
			"Invalid for_each argument",
			`The given "for_each" value is derived from an ephemeral value`,
			false, true, false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := &MockEvalContext{}
			ctx.installSimpleEval()
			_, _, diags := evaluateForEachExpression(test.Expr, ctx, false)

			if len(diags) != 1 {
				t.Fatalf("got %d diagnostics; want 1", len(diags))
			}
			if got, want := diags[0].Severity(), tfdiags.Error; got != want {
				t.Errorf("wrong diagnostic severity %#v; want %#v", got, want)
			}
			if got, want := diags[0].Description().Summary, test.Summary; got != want {
				t.Errorf("wrong diagnostic summary\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := diags[0].Description().Detail, test.DetailSubstring; !strings.Contains(got, want) {
				t.Errorf("wrong diagnostic detail\ngot: %s\nwant substring: %s", got, want)
			}
			if fromExpr := diags[0].FromExpr(); fromExpr != nil {
				if fromExpr.Expression == nil {
					t.Errorf("diagnostic does not refer to an expression")
				}
				if fromExpr.EvalContext == nil {
					t.Errorf("diagnostic does not refer to an EvalContext")
				}
			} else {
				t.Errorf("diagnostic does not support FromExpr\ngot: %s", spew.Sdump(diags[0]))
			}

			if got, want := tfdiags.DiagnosticCausedByUnknown(diags[0]), test.CausedByUnknown; got != want {
				t.Errorf("wrong result from tfdiags.DiagnosticCausedByUnknown\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := tfdiags.DiagnosticCausedByEphemeral(diags[0]), test.CausedByEphemeral; got != want {
				t.Errorf("wrong result from tfdiags.DiagnosticCausedByEphemeral\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := tfdiags.DiagnosticCausedBySensitive(diags[0]), test.CausedBySensitive; got != want {
				t.Errorf("wrong result from tfdiags.DiagnosticCausedBySensitive\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	}
}

func TestEvaluateForEachExpression_allowUnknown(t *testing.T) {
	tests := map[string]struct {
		Expr hcl.Expression
	}{
		"unknown string set": {
			hcltest.MockExprLiteral(cty.UnknownVal(cty.Set(cty.String))),
		},
		"unknown map": {
			hcltest.MockExprLiteral(cty.UnknownVal(cty.Map(cty.Bool))),
		},
		"set containing unknown value": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.UnknownVal(cty.String)})),
		},
		"set containing dynamic unknown value": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.UnknownVal(cty.DynamicPseudoType)})),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := &MockEvalContext{}
			ctx.installSimpleEval()
			_, known, diags := evaluateForEachExpression(test.Expr, ctx, true)

			// With allowUnknown set, all of these expressions should be treated
			// as valid for_each values.
			assertNoDiagnostics(t, diags)

			if known {
				t.Errorf("result is known; want unknown")
			}
		})
	}
}

func TestEvaluateForEachExpressionKnown(t *testing.T) {
	tests := map[string]hcl.Expression{
		"unknown string set": hcltest.MockExprLiteral(cty.UnknownVal(cty.Set(cty.String))),
		"unknown map":        hcltest.MockExprLiteral(cty.UnknownVal(cty.Map(cty.Bool))),
	}

	for name, expr := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := &MockEvalContext{}
			ctx.installSimpleEval()
			diags := newForEachEvaluator(expr, ctx, false).ValidateResourceValue()

			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
			}
		})
	}
}
