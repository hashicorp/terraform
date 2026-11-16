// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// InputVariable represents an input variable belonging to a [Stack].
type InputVariable struct {
	addr   stackaddrs.AbsInputVariable
	stack  *Stack
	config *InputVariableConfig

	main *Main

	value perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
}

var _ Plannable = (*InputVariable)(nil)
var _ Referenceable = (*InputVariable)(nil)

func newInputVariable(main *Main, addr stackaddrs.AbsInputVariable, stack *Stack, config *InputVariableConfig) *InputVariable {
	return &InputVariable{
		addr:   addr,
		stack:  stack,
		config: config,
		main:   main,
	}
}

// DefinedByStackCallInstance returns the stack call which ought to provide
// the definition (i.e. the final value) of this input variable. The source
// of the stack could either be a regular stack call instance or a removed
// stack call instance. One of the two will be returned. They are mutually
// exclusive as it is an error for two blocks to create the same stack instance.
//
// Returns nil if this input variable belongs to the main stack, because
// the main stack's input variables come from the planning options instead.
//
// Also returns nil if the receiver belongs to a stack config instance
// that isn't actually declared in the configuration, which typically suggests
// that we don't yet know the number of instances of one of the stack calls
// along the chain.
func (v *InputVariable) DefinedByStackCallInstance(ctx context.Context, phase EvalPhase) (*StackCallInstance, *RemovedStackCallInstance) {
	declarerAddr := v.addr.Stack
	if declarerAddr.IsRoot() {
		return nil, nil
	}

	callAddr := declarerAddr.Call()

	if call := v.stack.parent.EmbeddedStackCall(callAddr.Item); call != nil {
		lastStep := declarerAddr[len(declarerAddr)-1]
		instKey := lastStep.Key

		callInsts, unknown := call.Instances(ctx, phase)
		if unknown {
			// Return our static unknown instance for this variable.
			return call.UnknownInstance(ctx, instKey, phase), nil
		}
		if inst, ok := callInsts[instKey]; ok {
			return inst, nil
		}

		// otherwise, let's check if we have any removed calls that match the
		// target instance
	}

	if calls := v.stack.parent.RemovedEmbeddedStackCall(callAddr.Item); calls != nil {
		for _, call := range calls {
			callInsts, unknown := call.InstancesFor(ctx, v.stack.addr, phase)
			if unknown {
				return nil, call.UnknownInstance(ctx, v.stack.addr, phase)
			}
			for _, inst := range callInsts {
				// because we used the exact v.stack.addr in InstancesFor above
				// then we should have at most one entry here if there were any
				// matches.
				return nil, inst
			}
		}
	}

	return nil, nil
}

func (v *InputVariable) Value(ctx context.Context, phase EvalPhase) cty.Value {
	val, _ := v.CheckValue(ctx, phase)
	return val
}

func (v *InputVariable) CheckValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(
		ctx, v.tracingName(), v.value.For(phase),
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			cfg := v.config
			decl := cfg.config

			switch {
			case v.addr.Stack.IsRoot():
				var err error

				wantTy := decl.Type.Constraint
				extVal := v.main.RootVariableValue(v.addr.Item, phase)

				val := extVal.Value
				if val.IsNull() {
					// A null value is equivalent to an unspecified value, so
					// we'll replace it with the variable's default value.
					val = cfg.DefaultValue()
					if val == cty.NilVal {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "No value for required variable",
							Detail:   fmt.Sprintf("The root input variable %q is not set, and has no default value.", v.addr),
							Subject:  cfg.config.DeclRange.ToHCL().Ptr(),
						})
						return cty.UnknownVal(wantTy), diags
					}
				} else {
					// The DefaultValue function already validated the default
					// value, and applied the defaults, so we only apply the
					// defaults to a user supplied value.
					if defaults := decl.Type.Defaults; defaults != nil {
						val = defaults.Apply(val)
					}
				}

				// Convert the value to the expected type.
				val, err = convert.Convert(val, wantTy)
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid value for root input variable",
						Detail: fmt.Sprintf(
							"Cannot use the given value for input variable %q: %s.",
							v.addr.Item.Name, err,
						),
					})
					val = cfg.markValue(cty.UnknownVal(wantTy))
					return val, diags
				}

				if phase == ApplyPhase && !cfg.config.Ephemeral {
					// Now, we're just going to check the apply time value
					// against the plan time value. It is expected that
					// ephemeral variables will have different values between
					// plan and apply time, so these are not checked here.
					plan := v.main.PlanBeingApplied()
					planValue := plan.RootInputValues[v.addr.Item]
					if errs := objchange.AssertValueCompatible(planValue, val); errs != nil {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Inconsistent value for input variable during apply",
							Detail:   fmt.Sprintf("The value for non-ephemeral input variable %q was set to a different value during apply than was set during plan. Only ephemeral input variables can change between the plan and apply phases.", v.addr.Item.Name),
							Subject:  cfg.config.DeclRange.ToHCL().Ptr(),
						})
						// Return a solidly invalid value to prevent further
						// processing of this variable. This is a rare case and
						// a bug in Terraform so it's okay that might cause
						// additional errors to be raised later. We just want
						// to make sure we don't continue when something has
						// gone wrong elsewhere.
						return cty.NilVal, diags
					}
				}

				// Mark the value before validation so that validation can detect
				// sensitive/ephemeral marks and avoid leaking protected values in error messages.
				val = cfg.markValue(val)

				// Evaluate custom validation rules against the input value.
				// Validation is skipped during ValidatePhase because actual input values
				// are not yet available at that phase — only plan/apply provide them.
				// This matches the behavior of core Terraform's variable validation.
				if phase != ValidatePhase {
					moreDiags := v.evalVariableValidations(ctx, val, phase)
					diags = diags.Append(moreDiags)
				}

				return val, diags

			default:
				definedByCallInst, definedByRemovedCallInst := v.DefinedByStackCallInstance(ctx, phase)
				switch {
				case definedByCallInst != nil:
					allVals := definedByCallInst.InputVariableValues(ctx, phase)
					val := allVals.GetAttr(v.addr.Item.Name)

					// Mark the value before validation to prevent leaking sensitive/ephemeral data.
					val = cfg.markValue(val)

					// Evaluate custom validation rules for values from stack call instances.
					// Skip during ValidatePhase as values are not yet available.
					if phase != ValidatePhase {
						moreDiags := v.evalVariableValidations(ctx, val, phase)
						diags = diags.Append(moreDiags)
					}

					return val, diags
				case definedByRemovedCallInst != nil:
					allVals, _ := definedByRemovedCallInst.InputVariableValues(ctx, phase)
					val := allVals.GetAttr(v.addr.Item.Name)

					// Mark the value before validation to prevent leaking sensitive/ephemeral data.
					val = cfg.markValue(val)

					// Evaluate validation rules for removed stack instances.
					// Skip during ValidatePhase as values are not yet available.
					if phase != ValidatePhase {
						moreDiags := v.evalVariableValidations(ctx, val, phase)
						diags = diags.Append(moreDiags)
					}

					return val, diags
				default:
					// We seem to belong to a call instance that doesn't actually
					// exist in the configuration. That either means that
					// something's gone wrong or we are descended from a stack
					// call whose instances aren't known yet; we'll assume
					// the latter and return a placeholder.
					return cfg.markValue(cty.UnknownVal(v.config.config.Type.Constraint)), diags
				}
			}
		},
	)
}

// ExprReferenceValue implements Referenceable.
func (v *InputVariable) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	return v.Value(ctx, phase)
}

func (v *InputVariable) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, moreDiags := v.CheckValue(ctx, phase)
	diags = diags.Append(moreDiags)

	return diags
}

// PlanChanges implements Plannable as a plan-time validation of the variable's
// declaration and of the caller's definition of the variable.
func (v *InputVariable) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	diags := v.checkValid(ctx, PlanPhase)
	if diags.HasErrors() {
		return nil, diags
	}

	// Only the root stack's input values can contribute directly to the plan.
	// Embedded stack inputs will be recalculated during the apply phase
	// because the values might be derived from component outputs that aren't
	// known yet during planning.
	if !v.addr.Stack.IsRoot() {
		return nil, diags
	}

	destroy := v.main.PlanningOpts().PlanningMode == plans.DestroyMode

	before := v.main.PlanPrevState().RootInputVariable(v.addr.Item)

	decl := v.config.config
	after := v.Value(ctx, PlanPhase)
	requiredOnApply := false
	if decl.Ephemeral {
		// we don't persist the value for an ephemeral variable, but we
		// do need to remember whether it was set.
		requiredOnApply = !after.IsNull()

		// we'll set the after value to null now that we've captured the
		// requiredOnApply flag.
		after = cty.NullVal(after.Type())
	}

	var action plans.Action
	if before != cty.NilVal {
		if decl.Ephemeral {
			// if the value is ephemeral, we always mark is as an update
			action = plans.Update
		} else {
			unmarkedBefore, beforePaths := before.UnmarkDeepWithPaths()
			unmarkedAfter, afterPaths := after.UnmarkDeepWithPaths()
			result := unmarkedBefore.Equals(unmarkedAfter)
			if result.IsKnown() && result.True() && marks.MarksEqual(beforePaths, afterPaths) {
				action = plans.NoOp
			} else {
				// If we don't know for sure that the values are equal, then we'll
				// call this an update.
				action = plans.Update
			}
		}
	} else {
		action = plans.Create
		before = cty.NullVal(cty.DynamicPseudoType)
	}

	return []stackplan.PlannedChange{
		&stackplan.PlannedChangeRootInputValue{
			Addr:            v.addr.Item,
			Action:          action,
			Before:          before,
			After:           after,
			RequiredOnApply: requiredOnApply,
			DeleteOnApply:   destroy,
		},
	}, diags
}

// References implements Referrer
func (v *InputVariable) References(ctx context.Context) []stackaddrs.AbsReference {
	// The references for an input variable actually come from the
	// call that defines it, in the parent stack.
	if v.addr.Stack.IsRoot() {
		// Variables declared in the root module can't refer to anything,
		// because they are defined outside of the stack configuration by
		// our caller.
		return nil
	}
	if v.stack.parent == nil {
		// Weird, but we'll tolerate it for robustness.
		return nil
	}
	callAddr := v.addr.Stack.Call()
	call := v.stack.parent.EmbeddedStackCall(callAddr.Item)
	if call == nil {
		// Weird, but we'll tolerate it for robustness.
		return nil
	}
	return call.References(ctx)
}

// CheckApply implements Applyable.
func (v *InputVariable) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	if !v.addr.Stack.IsRoot() {
		return nil, v.checkValid(ctx, ApplyPhase)
	}

	diags := v.checkValid(ctx, ApplyPhase)
	if diags.HasErrors() {
		return nil, diags
	}

	if v.main.PlanBeingApplied().DeletedInputVariables.Has(v.addr.Item) {
		// If the plan being applied has this variable as being deleted, then
		// we won't handle it here. This is usually the case during a destroy
		// only plan in which we wanted to both capture the value for an input
		// as we still need it, while also noting that everything is being
		// destroyed.
		return nil, diags
	}

	decl := v.config.config
	value := v.Value(ctx, ApplyPhase)
	if decl.Ephemeral {
		value = cty.NullVal(value.Type())
	}

	return []stackstate.AppliedChange{
		&stackstate.AppliedChangeInputVariable{
			Addr:  v.addr.Item,
			Value: value,
		},
	}, diags
}

func (v *InputVariable) tracingName() string {
	return v.addr.String()
}

// evalVariableValidations evaluates all custom validation rules for this input variable
// against the given value, returning diagnostics if any validations fail.
//
// This function implements runtime validation checking, which is distinct from the
// config-time parsing done in stackconfig. The validation rules were parsed and stored
// during config loading; this function evaluates those rules against actual input values.
//
// The validation process:
//  1. Creates an HCL evaluation context with the variable's value and available functions
//  2. Evaluates each validation rule's condition and error_message expressions
//  3. Always validates the error_message structure (sensitive/ephemeral marks are flagged
//     regardless of whether the condition passes or fails)
//  4. If the condition is false, reports an "Invalid value for variable" diagnostic
//
// This follows the same approach as core Terraform's evalVariableValidations, including
// handling of sensitive values, unknown values, and error message evaluation.
func (v *InputVariable) evalVariableValidations(ctx context.Context, val cty.Value, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	rules := v.config.config.Validations
	if len(rules) == 0 {
		// No validation rules defined, nothing to check
		return diags
	}

	// Get provider-defined functions from the stack scope.
	// These will be combined with built-in functions below.
	functions, moreDiags := v.stack.ExternalFunctions(ctx)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		// If we can't get the function table, we can't evaluate validation expressions
		// that depend on functions. Return early to avoid confusing downstream errors.
		return diags
	}

	// Create a scope to get the complete function table (provider-defined + built-in).
	// fakeScope.Functions() will combine the provider functions with built-in functions
	// like length(), regex(), etc. We don't need a full evaluation context, just the functions.
	fakeScope := &lang.Scope{
		Data:          nil, // not a real scope; can't actually make an evalcontext
		BaseDir:       ".",
		PureOnly:      phase != ApplyPhase,
		PlanTimestamp: v.stack.PlanTimestamp(),
		ExternalFuncs: functions,
	}

	// Create an HCL evaluation context with the variable value and functions.
	// The variable is made available as var.<name> within validation expressions.
	hclCtx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(map[string]cty.Value{
				v.addr.Item.Name: val,
			}),
		},
		Functions: fakeScope.Functions(),
	}

	// Evaluate each validation rule independently.
	// Multiple validation failures will all be reported.
	for _, validation := range rules {
		moreDiags := evalVariableValidation(validation, hclCtx, v.config.config.DeclRange.ToHCL())
		diags = diags.Append(moreDiags)
	}

	return diags
}

// evalVariableValidation evaluates a single validation rule against a variable value.
//
// This function handles the evaluation of one validation block's condition and error_message.
// It follows the same logic as core Terraform's variable validation:
//
//  1. Evaluates the condition and error_message expressions up front
//  2. Handles unknown/null/invalid condition results appropriately
//  3. Always validates the error_message structure — sensitive/ephemeral marks
//     in the message are flagged even when the condition passes
//  4. Returns early if the condition passes (after reporting any message issues)
//  5. Otherwise constructs an "Invalid value for variable" diagnostic
//
// Parameters:
//   - validation: The validation rule to evaluate (contains condition and error_message expressions)
//   - hclCtx: The HCL evaluation context with the variable value and functions
//   - valueRng: The source range of the variable declaration (for diagnostic reporting)
func evalVariableValidation(validation *stackconfig.CheckRule, hclCtx *hcl.EvalContext, valueRng hcl.Range) tfdiags.Diagnostics {
	const errInvalidCondition = "Invalid variable validation result"
	const errInvalidValue = "Invalid value for variable"
	var diags tfdiags.Diagnostics

	result, moreDiags := validation.Condition.Value(hclCtx)
	diags = diags.Append(moreDiags)

	if moreDiags.HasErrors() {
		// If we couldn't evaluate the condition at all (syntax error, etc.),
		// return early. The error is already in diags.
		return diags
	}

	// If the condition result is unknown, we can't determine validity yet.
	// This can happen when the condition references computed values.
	// Skip validation for now - it will be checked during apply if needed.
	if !result.IsKnown() {
		return diags
	}

	// Check if the result is null
	if result.IsNull() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     errInvalidCondition,
			Detail:      "Validation condition expression must return either true or false, not null.",
			Subject:     validation.Condition.Range().Ptr(),
			Expression:  validation.Condition,
			EvalContext: hclCtx,
		})
		return diags
	}

	// Convert result to boolean
	var err error
	result, err = convert.Convert(result, cty.Bool)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     errInvalidCondition,
			Detail:      fmt.Sprintf("Invalid validation condition result value: %s.", tfdiags.FormatError(err)),
			Subject:     validation.Condition.Range().Ptr(),
			Expression:  validation.Condition,
			EvalContext: hclCtx,
		})
		return diags
	}

	// Remove any marks (sensitive, ephemeral) before checking the boolean value.
	// The marks don't affect the validation result, only how we handle the error message.
	result, _ = result.Unmark()

	// Always evaluate the error_message expression, even when the condition passes —
	// unknown, sensitive, or ephemeral values in the message are structural problems
	// regardless of whether the check succeeds or fails.
	errorValue, errorDiags := validation.ErrorMessage.Value(hclCtx)
	diags = diags.Append(errorDiags)

	var errorMessage string
	if !errorDiags.HasErrors() {
		if !errorValue.IsKnown() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid error message",
				Detail:      "Unsuitable value for error message: expression refers to values that won't be known until the apply phase.",
				Subject:     validation.ErrorMessage.Range().Ptr(),
				Expression:  validation.ErrorMessage,
				EvalContext: hclCtx,
			})
			return diags
		} else if !errorValue.IsNull() {
			errorValue, err = convert.Convert(errorValue, cty.String)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     "Invalid error message",
					Detail:      fmt.Sprintf("Unsuitable value for error message: %s.", tfdiags.FormatError(err)),
					Subject:     validation.ErrorMessage.Range().Ptr(),
					Expression:  validation.ErrorMessage,
					EvalContext: hclCtx,
				})
			} else {
				// Check for sensitive/ephemeral marks; these are flagged even when
				// the condition passes, since the error message is structurally invalid.
				if marks.Has(errorValue, marks.Sensitive) {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Error message refers to sensitive values",
						Detail: `The error expression used to explain this condition refers to sensitive values. Terraform will not display the resulting message.

You can correct this by removing references to sensitive values, or by carefully using the nonsensitive() function if the expression will not reveal the sensitive data.`,
						Subject:     validation.ErrorMessage.Range().Ptr(),
						Expression:  validation.ErrorMessage,
						EvalContext: hclCtx,
					})
					errorMessage = "The error message included a sensitive value, so it will not be displayed."
				} else if marks.Has(errorValue, marks.Ephemeral) {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Error message refers to ephemeral values",
						Detail: `The error expression used to explain this condition refers to ephemeral values. Terraform will not display the resulting message.

You can correct this by removing references to ephemeral values, or by carefully using the ephemeralasnull() function if the expression will not reveal the ephemeral data.`,
						Subject:     validation.ErrorMessage.Range().Ptr(),
						Expression:  validation.ErrorMessage,
						EvalContext: hclCtx,
					})
					errorMessage = "The error message included an ephemeral value, so it will not be displayed."
				} else {
					errorMessage = strings.TrimSpace(errorValue.AsString())
				}
			}
		}
	}
	if errorMessage == "" {
		errorMessage = "Failed to evaluate condition error message."
	}

	// If the condition evaluated to true, the validation passed. We've validated
	// the error message above, so any structural issues are already reported.
	if result.True() {
		return diags
	}

	// Construct the validation failure diagnostic.
	// The detail includes both the custom error message and a reference to where
	// the validation rule is defined, helping users locate the validation in their config.
	diags = diags.Append(&hcl.Diagnostic{
		Severity:    hcl.DiagError,
		Summary:     errInvalidValue,
		Detail:      fmt.Sprintf("%s\n\nThis was checked by the validation rule at %s.", errorMessage, validation.DeclRange.String()),
		Subject:     &valueRng,
		Expression:  validation.Condition,
		EvalContext: hclCtx,
	})

	return diags
}

// ExternalInputValue represents the value of an input variable provided
// from outside the stack configuration.
type ExternalInputValue struct {
	Value    cty.Value
	DefRange tfdiags.SourceRange
}
