// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
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

				// First, apply any defaults that are declared in the
				// configuration.

				// Next, convert the value to the expected type.
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

				// TODO: check the value against any custom validation rules
				// declared in the configuration.
				return cfg.markValue(val), diags

			default:
				definedByCallInst, definedByRemovedCallInst := v.DefinedByStackCallInstance(ctx, phase)
				switch {
				case definedByCallInst != nil:
					allVals := definedByCallInst.InputVariableValues(ctx, phase)
					val := allVals.GetAttr(v.addr.Item.Name)

					// TODO: check the value against any custom validation rules
					// declared in the configuration.

					return cfg.markValue(val), diags
				case definedByRemovedCallInst != nil:
					allVals, _ := definedByRemovedCallInst.InputVariableValues(ctx, phase)
					val := allVals.GetAttr(v.addr.Item.Name)

					// TODO: check the value against any custom validation rules
					// declared in the configuration.

					return cfg.markValue(val), diags
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

// ExternalInputValue represents the value of an input variable provided
// from outside the stack configuration.
type ExternalInputValue struct {
	Value    cty.Value
	DefRange tfdiags.SourceRange
}
