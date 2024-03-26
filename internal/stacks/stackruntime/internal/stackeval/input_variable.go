// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/collections"
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
	callInsts := call.Instances(ctx, phase)
	if callInsts == nil {
		// Could get here if the call's for_each is unknown or invalid,
		// in which case we'll assume unknown.
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
	return withCtyDynamicValPlaceholder(doOnceWithDiags(
		ctx, v.value.For(phase), v.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			switch {
			case v.Addr().Stack.IsRoot():
				wantTy := v.Declaration(ctx).Type.Constraint

				extVal := v.main.RootVariableValue(ctx, v.Addr().Item, phase)

				// We treat a null value as equivalent to an unspecified value,
				// and replace it with the variable's default value. This is
				// consistent with how embedded stacks handle defaults.
				if extVal.Value.IsNull() {
					cfg := v.Config(ctx)

					// A separate code path will validate the default value, so
					// we don't need to do that here.
					defVal := cfg.DefaultValue(ctx)
					if defVal == cty.NilVal {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "No value for required variable",
							Detail:   fmt.Sprintf("The root input variable %q is not set, and has no default value.", v.Addr()),
							Subject:  cfg.config.DeclRange.ToHCL().Ptr(),
						})
						return cty.UnknownVal(wantTy), diags
					}

					extVal = ExternalInputValue{
						Value:    defVal,
						DefRange: cfg.Declaration().DeclRange,
					}
				}

				val, err := convert.Convert(extVal.Value, wantTy)
				const errSummary = "Invalid value for root input variable"
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  errSummary,
						Detail: fmt.Sprintf(
							"Cannot use the given value for input variable %q: %s.",
							v.Addr().Item.Name, err,
						),
					})
					val = cty.UnknownVal(wantTy)
					return val, diags
				}

				// TODO: check the value against any custom validation rules
				// declared in the configuration.
				return val, diags

			default:
				definedByCallInst := v.DefinedByStackCallInstance(ctx, phase)
				if definedByCallInst == nil {
					// We seem to belong to a call instance that doesn't actually
					// exist in the configuration. That either means that
					// something's gone wrong or we are descended from a stack
					// call whose instances aren't known yet; we'll assume
					// the latter and return a placeholder.
					return cty.UnknownVal(v.Declaration(ctx).Type.Constraint), diags
				}

				allVals := definedByCallInst.InputVariableValues(ctx, phase)
				val := allVals.GetAttr(v.Addr().Item.Name)

				// TODO: check the value against any custom validation rules
				// declared in the configuration.

				return val, diags
			}
		},
	))
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

	val := v.Value(ctx, PlanPhase)
	return []stackplan.PlannedChange{
		&stackplan.PlannedChangeRootInputValue{
			Addr:  v.Addr().Item,
			Value: val,
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

// RequiredComponents implements Applyable
func (v *InputVariable) RequiredComponents(ctx context.Context) collections.Set[stackaddrs.AbsComponent] {
	return v.main.requiredComponentsForReferrer(ctx, v, PlanPhase)
}

// CheckApply implements Applyable.
func (v *InputVariable) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, v.checkValid(ctx, ApplyPhase)
}

func (v *InputVariable) tracingName() string {
	return v.Addr().String()
}

// ExternalInputValue represents the value of an input variable provided
// from outside the stack configuration.
type ExternalInputValue struct {
	Value    cty.Value
	DefRange tfdiags.SourceRange
}
