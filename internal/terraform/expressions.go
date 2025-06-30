// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// ExprEvaluator is a generic struct that exposes methods for evaluating
// single HCL expressions of a primitive cty type (boolean, number, or string) T,
// and converting the result to a primitive Go type U. It also includes validation logic
// for the evaluated expression, such as checking for null or unknown values.
type ExprEvaluator[T cty.Type, U comparable] struct {
	cType           T
	defaultValue    U
	argName         string
	allowUnknown    bool
	allowEphemeral  bool
	validateGoValue func(hcl.Expression, U) tfdiags.Diagnostics
}

// EvaluateExpr evaluates the HCL expression and produces the cty.Value and the final Go value U.
// The cty value may be unknown if the constructor is configured to allow unknown values. The marks
// on the cty value are preserved.
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

	// Unmark the value so that it can be decoded into a Go type.
	unmarked, _ := result.Unmark()

	// derive the Go value from the cty.Value
	var goVal U
	err := gocty.FromCtyValue(unmarked, &goVal)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invalid %q argument", e.argName),
			Detail:   fmt.Sprintf(`The given %q argument value is unsuitable: %s.`, e.argName, err),
			Subject:  expression.Range().Ptr(),
		})
		return result, e.defaultValue, diags
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
			Detail:   fmt.Sprintf(`The given %q argument type must be a primitive type, got %s. This is a bug in Terraform.`, e.argName, cType.FriendlyName()),
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
			Detail:   fmt.Sprintf(`The given %q argument value is null. A %s is required.`, e.argName, cType.FriendlyName()),
			Subject:  expression.Range().Ptr(),
		})
		return val, diags
	case !val.IsKnown():
		if e.allowUnknown {
			return cty.UnknownVal(cType).WithMarks(val.Marks()), diags
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invalid %q argument", e.argName),
			Detail:   fmt.Sprintf(`The given %q argument value is unknown. A known %s is required.`, e.argName, cType.FriendlyName()),
			Subject:  expression.Range().Ptr(),
			Extra:    diagnosticCausedByUnknown(true),
		})
		return val, diags
	case val.HasMark(marks.Ephemeral) && !e.allowEphemeral:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invalid %q argument", e.argName),
			Detail:   fmt.Sprintf(`The given %q is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values here.`, e.argName),
			Subject:  expression.Range().Ptr(),

			// TODO: Also populate Expression and EvalContext in here, but
			// we can't easily do that right now because the hcl.EvalContext
			// (which is not the same as the ctx we have in scope here) is
			// hidden away inside ctx.EvaluateExpr.
			Extra: DiagnosticCausedByEphemeral(true),
		})
		return val, diags
	}

	return val, diags
}
