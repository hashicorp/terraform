// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StackCallInstance represents an instance of a [StackCall], acting as
// an expression scope for evaluating the expressions inside the configuration
// block.
//
// This does not represent the child stack object itself, although if you
// are holding a valid [StackCallInstance] then you can call
// [StackCallInstance.CalledStack] to get that stack.
type StackCallInstance struct {
	call     *StackCall
	key      addrs.InstanceKey
	deferred bool
	mode     plans.Mode

	main *Main

	repetition instances.RepetitionData

	stack               perEvalPhase[promising.Once[*Stack]]
	inputVariableValues perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
}

var _ ExpressionScope = (*StackCallInstance)(nil)
var _ Plannable = (*StackCallInstance)(nil)

func newStackCallInstance(call *StackCall, key addrs.InstanceKey, repetition instances.RepetitionData, mode plans.Mode, deferred bool) *StackCallInstance {
	return &StackCallInstance{
		call:       call,
		key:        key,
		deferred:   deferred,
		mode:       mode,
		main:       call.main,
		repetition: repetition,
	}
}

func (c *StackCallInstance) RepetitionData() instances.RepetitionData {
	return c.repetition
}

func (c *StackCallInstance) CalledStackAddr() stackaddrs.StackInstance {
	callAddr := c.call.addr
	callerAddr := callAddr.Stack
	return callerAddr.Child(callAddr.Item.Name, c.key)

}

func (c *StackCallInstance) Stack(ctx context.Context, phase EvalPhase) *Stack {
	stack, err := c.stack.For(phase).Do(ctx, c.tracingName(), func(ctx context.Context) (*Stack, error) {
		return newStack(c.main, c.CalledStackAddr(), c.call.stack, c.call.config.TargetConfig(), c.call.GetExternalRemovedBlocks(), c.mode, c.deferred), nil
	})
	if err != nil {
		// we don't have cycles in here, and we don't return an error so this
		// should never happen.
		panic(err)
	}
	return stack
}

// InputVariableValues returns the [cty.Value] representing the input variable
// values to pass to the child stack.
//
// If the definition of the input variable values is invalid then the result
// is [cty.DynamicVal] to represent that the values aren't known.
func (c *StackCallInstance) InputVariableValues(ctx context.Context, phase EvalPhase) cty.Value {
	v, _ := c.CheckInputVariableValues(ctx, phase)
	return v
}

// CheckInputVariableValues returns the [cty.Value] representing the input
// variable values to pass to the child stack.
//
// If the configuration is valid then the resulting value is always of an
// object type derived from the child stack's input variable declarations.
// The resulting object type is guaranteed to have an attribute for each of
// the child stack's input variables, whose type conforms to the input
// variable's declared type constraint.
//
// If the configuration is invalid then the returned diagnostics will have
// errors and the result value will be [cty.DynamicVal] representing that
// we don't actually know the input variable values.
//
// CheckInputVariableValues checks whether the given object conforms to
// the input variables' type constraints and inserts default values where
// appropriate, but it doesn't check other details such as whether the
// values pass any author-defined custom validation rules. Those other details
// must be handled by the [InputVariable] objects representing each individual
// child stack input variable declaration, as part of preparing the individual
// attributes of the result for their appearance in downstream expressions.
func (c *StackCallInstance) CheckInputVariableValues(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(ctx, c.tracingName()+" inputs", c.inputVariableValues.For(phase),
		validateStackCallInstanceInputsFn(c.Stack(ctx, phase), c.call.config.config.Inputs, c.call.config.config.DeclRange.ToHCL().Ptr(), c, phase))
}

// ResolveExpressionReference implements ExpressionScope for the arguments
// inside an embedded stack call block, evaluated in the context of a
// particular instance of that call.
func (c *StackCallInstance) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	return c.call.stack.resolveExpressionReference(ctx, ref, nil, c.repetition)
}

// ExternalFunctions implements ExpressionScope.
func (c *StackCallInstance) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return c.main.ProviderFunctions(ctx, c.call.stack.config)
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (c *StackCallInstance) PlanTimestamp() time.Time {
	return c.main.PlanTimestamp()
}

func (c *StackCallInstance) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, moreDiags := c.CheckInputVariableValues(ctx, phase)
	diags = diags.Append(moreDiags)

	return diags
}

// PlanChanges implements Plannable by performing plan-time validation of
// all of the per-instance arguments in the stack call configuration.
//
// This does not check the child stack instance implied by the call, so the
// plan walk driver must call [StackCallInstance.CalledStack] and explore
// it and all of its contents too.
func (c *StackCallInstance) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	// This is really just a "plan-time validation" behavior, since stack
	// calls never contribute directly to the planned changes.
	return nil, c.checkValid(ctx, PlanPhase)
}

// CheckApply implements Applyable by confirming that the input variable
// values are still valid after resolving any upstream changes.
func (c *StackCallInstance) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, ApplyPhase)
}

// tracingName implements Plannable.
func (c *StackCallInstance) tracingName() string {
	return fmt.Sprintf("%s call", c.CalledStackAddr())
}

func validateStackCallInstanceInputsFn(stack *Stack, expr hcl.Expression, rng *hcl.Range, scope ExpressionScope, phase EvalPhase) func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
	return func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
		var diags tfdiags.Diagnostics
		wantTy, defs := stack.InputsType()

		v := cty.EmptyObjectVal
		var hclCtx *hcl.EvalContext
		if expr != nil {
			result, moreDiags := EvalExprAndEvalContext(ctx, expr, phase, scope)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return cty.DynamicVal, diags
			}
			expr = result.Expression
			hclCtx = result.EvalContext
			v = result.Value
		}

		v = defs.Apply(v)
		v, err := convert.Convert(v, wantTy)
		if err != nil {
			// A conversion failure here could either be caused by an author-provided
			// expression that's invalid or by the author omitting the argument
			// altogether when there's at least one required attribute, so we'll
			// return slightly different messages in each case.
			if expr != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     "Invalid inputs for embedded stack",
					Detail:      fmt.Sprintf("Invalid input variable definition object: %s.", tfdiags.FormatError(err)),
					Subject:     rng,
					Expression:  expr,
					EvalContext: hclCtx,
				})
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing required inputs for embedded stack",
					Detail:   fmt.Sprintf("Must provide \"inputs\" argument to define the embedded stack's input variables: %s.", tfdiags.FormatError(err)),
					Subject:  rng,
				})
			}
			return cty.DynamicVal, diags
		}

		if v.IsKnown() && !v.IsNull() {
			var markDiags tfdiags.Diagnostics
			for varAddr, variable := range stack.InputVariables() {
				varVal := v.GetAttr(varAddr.Name)
				varDecl := variable.config.config

				if !varDecl.Ephemeral {
					// If the variable isn't declared as being ephemeral then we
					// cannot allow ephemeral values to be assigned to it.
					_, markses := varVal.UnmarkDeepWithPaths()
					ephemeralPaths, _ := marks.PathsWithMark(markses, marks.Ephemeral)
					for _, path := range ephemeralPaths {
						if len(path) == 0 {
							// The entire value is ephemeral, then.
							markDiags = markDiags.Append(&hcl.Diagnostic{
								Severity:    hcl.DiagError,
								Summary:     "Ephemeral value not allowed",
								Detail:      fmt.Sprintf("The input variable %q does not accept ephemeral values.", varAddr.Name),
								Subject:     rng,
								Expression:  expr,
								EvalContext: hclCtx,
								Extra:       diagnosticCausedByEphemeral(true),
							})
						} else {
							// Something nested inside is ephemeral, so we'll be
							// more specific.
							markDiags = markDiags.Append(&hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  "Ephemeral value not allowed",
								Detail: fmt.Sprintf(
									"The input variable %q does not accept ephemeral values, so the value for %s is not compatible.",
									varAddr.Name, tfdiags.FormatCtyPath(path),
								),
								Subject:     rng,
								Expression:  expr,
								EvalContext: hclCtx,
								Extra:       diagnosticCausedByEphemeral(true),
							})
						}
					}
				}
			}
			diags = diags.Append(markDiags)
			if markDiags.HasErrors() {
				// If we have an ephemeral value in a place where there shouldn't
				// be one then we'll return an entirely-unknown value to make sure
				// that downstreams that aren't checking the errors can't leak the
				// value into somewhere it ought not to be. We'll still preserve
				// the type constraint so that we can do type checking downstream.
				return cty.UnknownVal(v.Type()), diags
			}
		}

		return v, diags
	}
}
