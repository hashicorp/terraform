// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// ExprEvaluator is a generic struct that exposes methods for evaluating
// single HCL expressions of a primitive type (boolean, number, or string) T,
// and converting the result to a Go type U. It also includes validation logic
// for the evaluated expression, such as checking for null or unknown values.
type ExprEvaluator[T cty.Type, U any] struct {
	cType           T
	defaultValue    U
	argName         string
	allowUnknown    bool
	validateGoValue func(hcl.Expression, U) tfdiags.Diagnostics
}

// EvaluateExpr evaluates the HCL expression and produces the cty.Value and the final Go value U.
// The cty value may be unknown if the constructor is configured to allow unknown values.
func (e *ExprEvaluator[T, U]) EvaluateExpr(ctx EvalContext, expression hcl.Expression) (cty.Value, U, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result, diags := e.evaluateExpr(ctx, expression)
	if diags.HasErrors() || result.IsNull() {
		return result, e.defaultValue, diags
	}

	// if the result is unknown, we can stop here and just return the default value
	// alongside the unknown cty.Value
	if !result.IsKnown() {
		return result, e.defaultValue, diags
	}

	// derive the Go value from the cty.Value
	var goVal U
	err := gocty.FromCtyValue(result, &goVal)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invalid %q argument", e.argName),
			Detail:   fmt.Sprintf(`The given %q argument value is unsuitable: %s.`, e.argName, err),
			Subject:  expression.Range().Ptr(),
		})
	}

	if e.validateGoValue != nil {
		diags = diags.Append(e.validateGoValue(expression, goVal))
		if diags.HasErrors() {
			return result, e.defaultValue, diags
		}
	}

	return result, goVal, diags
}

// evaluateExpr evaluates a given HCL expression within the provided EvalContext.
// It returns the evaluated cty.Value.
func (e *ExprEvaluator[T, U]) evaluateExpr(ctx EvalContext, expression hcl.Expression) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	cType := cty.Type(e.cType)

	if expression == nil {
		return cty.NullVal(cType), diags
	}

	// only primitive types are allowed
	if !cType.IsPrimitiveType() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invalid %q argument", e.argName),
			Detail:   fmt.Sprintf(`The given %q argument value is unsuitable: expected a primitive type, got %s.`, e.argName, cType.FriendlyName()),
			Subject:  expression.Range().Ptr(),
		})
		return cty.NullVal(cType), diags
	}

	val, exprDiags := ctx.EvaluateExpr(expression, cType, nil)
	diags = diags.Append(exprDiags)
	if diags.HasErrors() {
		return cty.NullVal(cType), diags
	}

	switch {
	case val.IsNull():
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invalid %q argument", e.argName),
			Detail:   fmt.Sprintf(`The given %q argument value is null. An %s is required.`, cType.FriendlyName(), e.argName),
			Subject:  expression.Range().Ptr(),
		})
		return val, diags
	case !val.IsKnown() && !e.allowUnknown:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invalid %q argument", e.argName),
			Detail:   fmt.Sprintf(`The given %q argument value is unknown. A known %s is required.`, cType.FriendlyName(), e.argName),
			Subject:  expression.Range().Ptr(),
		})
		return val, diags
	case !val.IsKnown() && e.allowUnknown:
		return cty.UnknownVal(cType), diags
	}

	return val, diags
}
