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
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// InputVariable represents an input variable belonging to a [Stack].
type InputVariable struct {
	addr stackaddrs.AbsInputVariable

	main *Main

	value perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
}

var _ Plannable = (*InputVariable)(nil)
var _ Referenceable = (*InputVariable)(nil)

func newInputVariable(main *Main, addr stackaddrs.AbsInputVariable) *InputVariable {
	return &InputVariable{
		addr: addr,
		main: main,
	}
}

func (v *InputVariable) Addr() stackaddrs.AbsInputVariable {
	return v.addr
}

func (v *InputVariable) Config(ctx context.Context) *InputVariableConfig {
	configAddr := stackaddrs.ConfigForAbs(v.Addr())
	stackCfg := v.main.StackConfig(ctx, configAddr.Stack)
	return stackCfg.InputVariable(ctx, configAddr.Item)
}

func (v *InputVariable) Declaration(ctx context.Context) *stackconfig.InputVariable {
	return v.Config(ctx).Declaration()
}

// DefinedByStackCallInstance returns the stack call which ought to provide
// the definition (i.e. the final value) of this input variable.
//
// Returns nil if this input variable belongs to the main stack, because
// the main stack's input variables come from the planning options instead.
// Also returns nil if the reciever belongs to a stack config instance
// that isn't actually declared in the configuration, which typically suggests
// that we don't yet know the number of instances of one of the stack calls
// along the chain.
func (v *InputVariable) DefinedByStackCallInstance(ctx context.Context, phase EvalPhase) *StackCallInstance {
	declarerAddr := v.Addr().Stack
	if declarerAddr.IsRoot() {
		return nil
	}

	callAddr := declarerAddr.Call()
	callerAddr := callAddr.Stack
	callerStack := v.main.Stack(ctx, callerAddr, phase)
	if callerStack == nil {
		// Suggests that we are beneath a stack call whose instances
		// aren't known yet.
		return nil
	}

	callerCalls := callerStack.EmbeddedStackCalls(ctx)
	call := callerCalls[callAddr.Item]
	if call == nil {
		// Suggests that we're descended from a stack call that doesn't
		// actually exist, which is odd but we'll tolerate it.
		return nil
	}
	callInsts, unknown := call.Instances(ctx, phase)
	if unknown {
		// Return our static unknown instance for this variable.
		return call.UnknownInstance(ctx, phase)
	}
	if callInsts == nil {
		// Could get here if the call's for_each is invalid.
		return nil
	}

	lastStep := declarerAddr[len(declarerAddr)-1]
	instKey := lastStep.Key
	return callInsts[instKey]
}

func (v *InputVariable) Value(ctx context.Context, phase EvalPhase) cty.Value {
	val, _ := v.CheckValue(ctx, phase)
	return val
}

func (v *InputVariable) CheckValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(
		ctx, v.value.For(phase), v.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			cfg := v.Config(ctx)
			decl := v.Declaration(ctx)

			switch {
			case v.Addr().Stack.IsRoot():
				var err error

				wantTy := decl.Type.Constraint
				extVal := v.main.RootVariableValue(ctx, v.Addr().Item, phase)

				val := extVal.Value
				if val.IsNull() {
					// A null value is equivalent to an unspecified value, so
					// we'll replace it with the variable's default value.
					val = cfg.DefaultValue(ctx)
					if val == cty.NilVal {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "No value for required variable",
							Detail:   fmt.Sprintf("The root input variable %q is not set, and has no default value.", v.Addr()),
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
							v.Addr().Item.Name, err,
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
				definedByCallInst := v.DefinedByStackCallInstance(ctx, phase)
				if definedByCallInst == nil {
					// We seem to belong to a call instance that doesn't actually
					// exist in the configuration. That either means that
					// something's gone wrong or we are descended from a stack
					// call whose instances aren't known yet; we'll assume
					// the latter and return a placeholder.
					return cfg.markValue(cty.UnknownVal(v.Declaration(ctx).Type.Constraint)), diags
				}

				allVals := definedByCallInst.InputVariableValues(ctx, phase)
				val := allVals.GetAttr(v.Addr().Item.Name)

				// TODO: check the value against any custom validation rules
				// declared in the configuration.

				return cfg.markValue(val), diags
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
	if !v.Addr().Stack.IsRoot() {
		return nil, diags
	}

	destroy := v.main.PlanningOpts().PlanningMode == plans.DestroyMode

	before := v.main.PlanPrevState().RootInputVariable(v.Addr().Item)

	decl := v.Declaration(ctx)
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
			Addr:            v.Addr().Item,
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
	addr := v.Addr()
	if addr.Stack.IsRoot() {
		// Variables declared in the root module can't refer to anything,
		// because they are defined outside of the stack configuration by
		// our caller.
		return nil
	}
	stackAddr := addr.Stack
	parentStack := v.main.StackUnchecked(ctx, stackAddr.Parent())
	if parentStack == nil {
		// Weird, but we'll tolerate it for robustness.
		return nil
	}
	callAddr := stackAddr.Call()
	call := parentStack.EmbeddedStackCall(ctx, callAddr.Item)
	if call == nil {
		// Weird, but we'll tolerate it for robustness.
		return nil
	}
	return call.References(ctx)
}

// CheckApply implements Applyable.
func (v *InputVariable) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	if !v.Addr().Stack.IsRoot() {
		return nil, v.checkValid(ctx, ApplyPhase)
	}

	diags := v.checkValid(ctx, ApplyPhase)
	if diags.HasErrors() {
		return nil, diags
	}

	if v.main.PlanBeingApplied().DeletedInputVariables.Has(v.Addr().Item) {
		// If the plan being applied has this variable as being deleted, then
		// we won't handle it here. This is usually the case during a destroy
		// only plan in which we wanted to both capture the value for an input
		// as we still need it, while also noting that everything is being
		// destroyed.
		return nil, diags
	}

	decl := v.Declaration(ctx)
	value := v.Value(ctx, ApplyPhase)
	if decl.Ephemeral {
		value = cty.NullVal(value.Type())
	}

	return []stackstate.AppliedChange{
		&stackstate.AppliedChangeInputVariable{
			Addr:  v.Addr().Item,
			Value: value,
		},
	}, diags
}

func (v *InputVariable) tracingName() string {
	return v.Addr().String()
}

// reportNamedPromises implements namedPromiseReporter.
func (v *InputVariable) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	name := v.Addr().String()
	v.value.Each(func(ep EvalPhase, o *promising.Once[withDiagnostics[cty.Value]]) {
		cb(o.PromiseID(), name)
	})
}

// ExternalInputValue represents the value of an input variable provided
// from outside the stack configuration.
type ExternalInputValue struct {
	Value    cty.Value
	DefRange tfdiags.SourceRange
}
