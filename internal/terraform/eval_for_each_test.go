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
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := &MockEvalContext{}
			ctx.installSimpleEval()
			forEachMap, diags := evaluateForEachExpression(test.Expr, ctx)

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
		Expr                     hcl.Expression
		Summary, DetailSubstring string
	}{
		"null set": {
			hcltest.MockExprLiteral(cty.NullVal(cty.Set(cty.String))),
			"Invalid for_each argument",
			`the given "for_each" argument value is null`,
		},
		"string": {
			hcltest.MockExprLiteral(cty.StringVal("i am definitely a set")),
			"Invalid for_each argument",
			"must be a map, or set of strings, and you have provided a value of type string",
		},
		"list": {
			hcltest.MockExprLiteral(cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("a")})),
			"Invalid for_each argument",
			"must be a map, or set of strings, and you have provided a value of type list",
		},
		"tuple": {
			hcltest.MockExprLiteral(cty.TupleVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")})),
			"Invalid for_each argument",
			"must be a map, or set of strings, and you have provided a value of type tuple",
		},
		"unknown string set": {
			hcltest.MockExprLiteral(cty.UnknownVal(cty.Set(cty.String))),
			"Invalid for_each argument",
			"depends on resource attributes that cannot be determined until apply",
		},
		"unknown map": {
			hcltest.MockExprLiteral(cty.UnknownVal(cty.Map(cty.Bool))),
			"Invalid for_each argument",
			"depends on resource attributes that cannot be determined until apply",
		},
		"marked map": {
			hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
				"a": cty.BoolVal(true),
				"b": cty.BoolVal(false),
			}).Mark(marks.Sensitive)),
			"Invalid for_each argument",
			"Sensitive values, or values derived from sensitive values, cannot be used as for_each arguments. If used, the sensitive value could be exposed as a resource instance key.",
		},
		"set containing booleans": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.BoolVal(true)})),
			"Invalid for_each set argument",
			"supports maps and sets of strings, but you have provided a set containing type bool",
		},
		"set containing null": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.NullVal(cty.String)})),
			"Invalid for_each set argument",
			"must not contain null values",
		},
		"set containing unknown value": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.UnknownVal(cty.String)})),
			"Invalid for_each argument",
			"depends on resource attributes that cannot be determined until apply",
		},
		"set containing dynamic unknown value": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.UnknownVal(cty.DynamicPseudoType)})),
			"Invalid for_each argument",
			"depends on resource attributes that cannot be determined until apply",
		},
		"set containing marked values": {
			hcltest.MockExprLiteral(cty.SetVal([]cty.Value{cty.StringVal("beep").Mark(marks.Sensitive), cty.StringVal("boop")})),
			"Invalid for_each argument",
			"Sensitive values, or values derived from sensitive values, cannot be used as for_each arguments. If used, the sensitive value could be exposed as a resource instance key.",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := &MockEvalContext{}
			ctx.installSimpleEval()
			_, diags := evaluateForEachExpression(test.Expr, ctx)

			if len(diags) != 1 {
				t.Fatalf("got %d diagnostics; want 1", diags)
			}
			if got, want := diags[0].Severity(), tfdiags.Error; got != want {
				t.Errorf("wrong diagnostic severity %#v; want %#v", got, want)
			}
			if got, want := diags[0].Description().Summary, test.Summary; got != want {
				t.Errorf("wrong diagnostic summary %#v; want %#v", got, want)
			}
			if got, want := diags[0].Description().Detail, test.DetailSubstring; !strings.Contains(got, want) {
				t.Errorf("wrong diagnostic detail %#v; want %#v", got, want)
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
			forEachVal, diags := evaluateForEachExpressionValue(expr, ctx, true)

			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
			}

			if forEachVal.IsKnown() {
				t.Error("got known, want unknown")
			}
		})
	}
}
