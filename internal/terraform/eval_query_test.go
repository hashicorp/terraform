// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestEvaluateLimitExpression(t *testing.T) {
	tests := map[string]struct {
		expr         hcl.Expression
		result       int64
		wantError    bool
		allowUnknown bool
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
		"ephemeral": {
			expr:      hcltest.MockExprLiteral(cty.NumberIntVal(5).Mark(marks.Ephemeral)),
			result:    5,
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
		"unknown value (allowed)": {
			expr:         hcltest.MockExprLiteral(cty.UnknownVal(cty.Number)),
			result:       100,
			wantError:    false,
			allowUnknown: true,
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

			_, derived, diags := newLimitEvaluator(tc.allowUnknown).EvaluateExpr(ctx, tc.expr)
			if !tc.wantError && diags.HasErrors() {
				t.Errorf("unexpected error: %v", diags.Err())
				return
			}

			if derived != tc.result {
				t.Errorf("got %v, want %v", derived, tc.result)
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
		expr         hcl.Expression
		result       bool
		wantError    bool
		allowUnknown bool
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
		"ephemeral true value": {
			expr:      hcltest.MockExprLiteral(cty.True.Mark(marks.Ephemeral)),
			result:    true,
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
		"unknown value (allowed)": {
			expr:         hcltest.MockExprLiteral(cty.UnknownVal(cty.Bool)),
			result:       false,
			wantError:    false,
			allowUnknown: true,
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

			_, derived, diags := newIncludeRscEvaluator(tc.allowUnknown).EvaluateExpr(ctx, tc.expr)
			if !tc.wantError && diags.HasErrors() {
				t.Errorf("unexpected error: %v", diags.Err())
				return
			}
			if derived != tc.result {
				t.Errorf("got %v, want %v", derived, tc.result)
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
