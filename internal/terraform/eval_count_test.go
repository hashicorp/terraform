// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestEvaluateCountExpression(t *testing.T) {
	tests := map[string]struct {
		Expr  hcl.Expression
		Count int
		Known bool
	}{
		"zero": {
			hcltest.MockExprLiteral(cty.NumberIntVal(0)),
			0,
			true,
		},
		"expression with marked value": {
			hcltest.MockExprLiteral(cty.NumberIntVal(8).Mark(marks.Sensitive)),
			8,
			true,
		},
		"unknown value": {
			hcltest.MockExprLiteral(cty.UnknownVal(cty.Number)),
			-1,
			false,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := &MockEvalContext{}
			ctx.installSimpleEval()
			countVal, known, diags := evaluateCountExpression(test.Expr, ctx)

			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
			}

			if !reflect.DeepEqual(countVal, test.Count) {
				t.Errorf(
					"wrong map value\ngot:  %swant: %s",
					spew.Sdump(countVal), spew.Sdump(test.Count),
				)
			}
			if known != test.Known {
				t.Errorf("wrong 'knownness'\ngot:  %#v\nwant: %#v", known, test.Known)
			}
		})
	}
}
