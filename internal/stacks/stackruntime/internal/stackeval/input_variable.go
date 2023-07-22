package stackeval

//lint:file-ignore U1000 This package is still WIP so not everything is here yet.

import (
	"context"

	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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
				// TODO: Take input variables from the plan options, then.
				return cty.UnknownVal(v.Declaration(ctx).Type.Constraint), diags

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

// PlanChanges implements Plannable as a plan-time validation of the variable's
// declaration and of the caller's definition of the variable.
func (v *InputVariable) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	_, moreDiags := v.CheckValue(ctx, PlanPhase)
	diags = diags.Append(moreDiags)

	return nil, diags
}

func (v *InputVariable) tracingName() string {
	return v.Addr().String()
}
