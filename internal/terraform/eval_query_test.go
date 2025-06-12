// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"
)

func TestEvaluateLimitExpression(t *testing.T) {
	tests := map[string]struct {
		expr      hcl.Expression
		result    int64
		wantError bool
	}{
		"nil expression returns default": {
			expr:      nil,
			result:    100,
			wantError: false,
		},
		"valid integer": {
			expr:      hcltest.MockExprLiteral(cty.NumberIntVal(5)),
			result:    5,
			wantError: false,
		},
		"zero": {
			expr:      hcltest.MockExprLiteral(cty.NumberIntVal(0)),
			result:    0,
			wantError: false,
		},
		"negative integer": {
			expr:      hcltest.MockExprLiteral(cty.NumberIntVal(-1)),
			result:    100,
			wantError: true,
		},
		"null value": {
			expr:      hcltest.MockExprLiteral(cty.NullVal(cty.Number)),
			result:    100,
			wantError: true,
		},
		"unknown value": {
			expr:      hcltest.MockExprLiteral(cty.UnknownVal(cty.Number)),
			result:    100,
			wantError: true,
		},
		"wrong type": {
			expr:      hcltest.MockExprLiteral(cty.StringVal("foo")),
			result:    100,
			wantError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := &MockEvalContext{}
			ctx.installSimpleEval()

			got, diags := evaluateLimitExpression(tc.expr, ctx)
			if got != tc.result {
				t.Errorf("got %d, want %d", got, tc.result)
			}
			if tc.wantError && !diags.HasErrors() {
				t.Errorf("expected error but got none")
			}
			if !tc.wantError && diags.HasErrors() {
				t.Errorf("unexpected error: %v", diags.Err())
			}
		})
	}
}

func TestEvaluateIncludeResourceExpression(t *testing.T) {
	tests := map[string]struct {
		expr      hcl.Expression
		result    bool
		wantError bool
	}{
		"nil expression returns false": {
			expr:      nil,
			result:    false,
			wantError: false,
		},
		"true value": {
			expr:      hcltest.MockExprLiteral(cty.True),
			result:    true,
			wantError: false,
		},
		"false value": {
			expr:      hcltest.MockExprLiteral(cty.False),
			result:    false,
			wantError: false,
		},
		"null value": {
			expr:      hcltest.MockExprLiteral(cty.NullVal(cty.Bool)),
			result:    false,
			wantError: true,
		},
		"unknown value": {
			expr:      hcltest.MockExprLiteral(cty.UnknownVal(cty.Bool)),
			result:    false,
			wantError: true,
		},
		"wrong type": {
			expr:      hcltest.MockExprLiteral(cty.NumberIntVal(1)),
			result:    false,
			wantError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := &MockEvalContext{}
			ctx.installSimpleEval()

			got, diags := evaluateIncludeResourceExpression(tc.expr, ctx)
			if got != tc.result {
				t.Errorf("got %v, want %v", got, tc.result)
			}
			if tc.wantError && !diags.HasErrors() {
				t.Errorf("expected error but got none")
			}
			if !tc.wantError && diags.HasErrors() {
				t.Errorf("unexpected error: %v", diags.Err())
			}
		})
	}
}
