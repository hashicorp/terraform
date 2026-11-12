// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestAddressExpr(t *testing.T) {
	// evalKey evaluates each key with a fixed value for each.key.
	evalKey := func(eachKey cty.Value) func(hcl.Expression) (cty.Value, hcl.Diagnostics) {
		return func(expr hcl.Expression) (cty.Value, hcl.Diagnostics) {
			return expr.Value(&hcl.EvalContext{
				Variables: map[string]cty.Value{
					"each": cty.ObjectVal(map[string]cty.Value{"key": eachKey}),
				},
			})
		}
	}

	for name, tc := range map[string]struct {
		expr    string
		evalKey func(hcl.Expression) (cty.Value, hcl.Diagnostics)
		want    string
		wantErr string
	}{
		"static address": {
			expr: `module.m[0].test_instance.foo["a"]`,
			want: `module.m[0].test_instance.foo["a"]`,
		},
		"evaluated keys": {
			expr:    `module.m[each.key].test_instance.foo[each.key]`,
			evalKey: evalKey(cty.StringVal("a")),
			want:    `module.m["a"].test_instance.foo["a"]`,
		},
		"unknown keys": {
			expr:    `module.m[each.key].test_instance.foo`,
			evalKey: evalKey(cty.UnknownVal(cty.String)),
			want:    `module.m[*].test_instance.foo`,
		},
		"keys without an evaluator": {
			expr: `module.m[each.key].module.n[0].test_instance.foo[each.key]`,
			want: `module.m[*].module.n[0].test_instance.foo[*]`,
		},
		"key evaluation error": {
			expr:    `test_instance.foo[local.key]`,
			evalKey: evalKey(cty.StringVal("a")),
			wantErr: "Unknown variable",
		},
		"not an address": {
			expr:    `upper(test_instance.foo)`,
			wantErr: "Invalid expression",
		},
	} {
		t.Run(name, func(t *testing.T) {
			expr, hclDiags := hclsyntax.ParseExpression([]byte(tc.expr), "", hcl.InitialPos)
			if hclDiags.HasErrors() {
				t.Fatal(hclDiags.Error())
			}

			addrExpr, diags := ParseAddressExpr(expr)
			var traversal hcl.Traversal
			if !diags.HasErrors() {
				traversal, diags = addrExpr.Traversal(tc.evalKey)
			}
			if tc.wantErr != "" {
				if !diags.HasErrors() {
					t.Fatalf("unexpected success\nwant error: %s", tc.wantErr)
				}
				if got := diags[0].Summary; got != tc.wantErr {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, tc.wantErr)
				}
				return
			}
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}

			target, targetDiags := ParseTarget(traversal)
			if targetDiags.HasErrors() {
				t.Fatal(targetDiags.Err())
			}
			if got := target.String(); got != tc.want {
				t.Errorf("wrong address\ngot:  %s\nwant: %s", got, tc.want)
			}
		})
	}
}
