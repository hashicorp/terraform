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

	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StackCallConfig represents a "stack" block in a stack configuration,
// representing a call to an embedded stack.
type StackCallConfig struct {
	addr   stackaddrs.ConfigStackCall
	config *stackconfig.EmbeddedStack

	main *Main

	forEachValue        perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	inputVariableValues perEvalPhase[promising.Once[withDiagnostics[map[stackaddrs.InputVariable]cty.Value]]]
	resultValue         perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
}

var _ Validatable = (*StackCallConfig)(nil)
var _ Referenceable = (*StackCallConfig)(nil)
var _ ExpressionScope = (*StackCallConfig)(nil)
var _ namedPromiseReporter = (*StackCallConfig)(nil)

func newStackCallConfig(main *Main, addr stackaddrs.ConfigStackCall, config *stackconfig.EmbeddedStack) *StackCallConfig {
	return &StackCallConfig{
		addr:   addr,
		config: config,
		main:   main,
	}
}

func (s *StackCallConfig) Addr() stackaddrs.ConfigStackCall {
	return s.addr
}

func (s *StackCallConfig) tracingName() string {
	return s.Addr().String()
}

// CallerConfig returns the object representing the stack configuration that this
// stack call was declared within.
func (s *StackCallConfig) CallerConfig(ctx context.Context) *StackConfig {
	return s.main.mustStackConfig(ctx, s.Addr().Stack)
}

// CalleeConfig returns the object representing the child stack configuration
// that this stack call is referring to.
func (s *StackCallConfig) CalleeConfig(ctx context.Context) *StackConfig {
	return s.main.mustStackConfig(ctx, s.Addr().Stack.Child(s.addr.Item.Name))
}

// Declaration returns the [stackconfig.EmbeddedStack] that declared this object.
func (s *StackCallConfig) Declaration(ctx context.Context) *stackconfig.EmbeddedStack {
	return s.config
}

// ResultType returns the type of the overall result value for this call.
//
// If this call uses for_each then the result type is a map of object types.
// If it has no repetition then it's just a naked object type.
func (s *StackCallConfig) ResultType(ctx context.Context) cty.Type {
	// The result type of each of our instances is an object type constructed
	// from all of the declared output values in the child stack.
	calleeStack := s.CalleeConfig(ctx)
	calleeOutputs := calleeStack.OutputValues(ctx)
	atys := make(map[string]cty.Type, len(calleeOutputs))
	for addr, ov := range calleeOutputs {
		atys[addr.Name] = ov.ValueTypeConstraint(ctx)
	}
	instTy := cty.Object(atys)

	switch {
	case s.config.ForEach != nil:
		return cty.Map(instTy)
	default:
		// No repetition
		return instTy
	}
}

// ValidateForEachValue validates and returns the value from this stack call's
// for_each argument, or returns [cty.NilVal] if it doesn't use for_each.
//
// If the for_each expression is invalid in some way then the returned
// diagnostics will contain errors and the returned value will be a placeholder
// unknown value.
func (s *StackCallConfig) ValidateForEachValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return withCtyDynamicValPlaceholder(doOnceWithDiags(
		ctx, s.forEachValue.For(phase), s.main,
		s.validateForEachValueInner,
	))
}

func (s *StackCallConfig) validateForEachValueInner(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if s.config.ForEach == nil {
		// This stack config isn't even using for_each.
		return cty.NilVal, diags
	}

	result, moreDiags := evaluateForEachExpr(ctx, s.config.ForEach, ValidatePhase, s.CallerConfig(ctx), "stack")
	diags = diags.Append(moreDiags)
	return result.Value, diags
}

// ValidateInputVariableValues evaluates the "inputs" argument inside the
// configuration block, ensure that it's valid per the expectations of the
// child stack config, and then returns the resulting values.
//
// A [StackCallConfig] represents the not-yet-expanded stack call, so the
// result is an approximation of the input variables for all instances of
// this call. To get the final values for a particular instance, use
// [StackCall.InputVariableValues] instead.
//
// If the returned diagnostics contains errors then the returned values may
// be incomplete, but should at least be of the types specified in the
// variable declarations.
func (s *StackCallConfig) ValidateInputVariableValues(ctx context.Context, phase EvalPhase) (map[stackaddrs.InputVariable]cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(
		ctx, s.inputVariableValues.For(phase), s.main,
		s.validateInputVariableValuesInner,
	)
}

func (s *StackCallConfig) validateInputVariableValuesInner(ctx context.Context) (map[stackaddrs.InputVariable]cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	callee := s.CalleeConfig(ctx)
	vars := callee.InputVariables(ctx)

	atys := make(map[string]cty.Type, len(vars))
	var optional []string
	defs := make(map[string]cty.Value, len(vars))
	for addr, v := range vars {
		aty := v.TypeConstraint()

		atys[addr.Name] = aty
		if def := v.DefaultValue(ctx); def != cty.NilVal {
			optional = append(optional, addr.Name)
			defs[addr.Name] = def
		}
	}

	oty := cty.ObjectWithOptionalAttrs(atys, optional)

	var varsObj cty.Value
	var hclCtx *hcl.EvalContext // NOTE: remains nil when h.config.Inputs is unset
	if s.config.Inputs != nil {
		result, moreDiags := EvalExprAndEvalContext(ctx, s.config.Inputs, ValidatePhase, s)
		v := result.Value
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			v = cty.UnknownVal(oty.WithoutOptionalAttributesDeep())
		}
		varsObj = v
		hclCtx = result.EvalContext
	} else {
		varsObj = cty.EmptyObjectVal
	}

	// FIXME: TODO: We need to apply the nested optional attribute defaults
	// somewhere in here too, but it isn't clear where we should do that since
	// we're supposed to do that before type conversion but we don't yet have
	// the isolated variable values to apply the defaults to.

	varsObj, err := convert.Convert(varsObj, oty)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid input variable definitions",
			Detail: fmt.Sprintf(
				"Unsuitable input variable definitions: %s.",
				tfdiags.FormatError(err),
			),
			Subject: s.config.Inputs.Range().Ptr(),

			// NOTE: The following two will be nil if the author didn't
			// actually define the "inputs" argument, but that's okay
			// because these fields are both optional anyway.
			Expression:  s.config.Inputs,
			EvalContext: hclCtx,
		})
		varsObj = cty.UnknownVal(oty.WithoutOptionalAttributesDeep())
	}

	ret := make(map[stackaddrs.InputVariable]cty.Value, len(vars))

	for addr := range vars {
		val := varsObj.GetAttr(addr.Name)
		if val.IsNull() {
			if def, ok := defs[addr.Name]; ok {
				ret[addr] = def
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing definition for required input variable",
					Detail:   fmt.Sprintf("The input variable %q is required, so cannot be omitted.", addr.Name),
					Subject:  s.config.Inputs.Range().Ptr(),

					// NOTE: The following two will be nil if the author didn't
					// actually define the "inputs" argument, but that's okay
					// because these fields are both optional anyway.
					Expression:  s.config.Inputs,
					EvalContext: hclCtx,
				})
				ret[addr] = cty.UnknownVal(atys[addr.Name])
			}
		} else {
			ret[addr] = val
		}
	}

	return ret, diags
}

// InputVariableValues returns the effective input variable values specified
// in this call, or correctly-typed placeholders if any values are invalid.
//
// This is intended to support downstream evaluation of other objects during
// the validate phase, rather than for direct validation of this object. If you
// are intending to report problems directly to the user, use
// [StackCallConfig.ValidateInputVariableValues] instead.
func (s *StackCallConfig) InputVariableValues(ctx context.Context, phase EvalPhase) map[stackaddrs.InputVariable]cty.Value {
	ret, _ := s.ValidateInputVariableValues(ctx, phase)
	return ret
}

// ResultValue returns a suitable placeholder value to use to approximate the
// result of this call during the validation phase, where we typically don't
// yet have access to all necessary information.
//
// If the stack configuration is itself invalid then this will still return
// a suitably-typed unknown value, to permit partial validation downstream.
//
// The result is a good value to use for resolving "stack.foo" references
// in expressions elsewhere while running in validation mode.
func (s *StackCallConfig) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	v, _ := s.ValidateResultValue(ctx, phase)
	return v
}

// ValidateResultValue returns a validation-time approximation of the overall
// result of the embedded stack call, along with diagnostics describing any
// problems with the stack call itself (NOT with the child stack that was called)
// that we discover in the process of building it.
//
// During validation we don't perform instance expansion of any embedded stacks
// and so the validation-time approximation of a multi-instance embedded stack
// is always an unknown value with a suitable type constraint, allowing
// downstream references to detect type-related errors but not value-related
// errors.
func (s *StackCallConfig) ValidateResultValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return withCtyDynamicValPlaceholder(doOnceWithDiags(
		ctx, s.resultValue.For(phase), s.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			// Our result is really just all of the output values of all of our
			// instances aggregated together into a single data structure, but
			// we do need to do this a little differently depending on what
			// kind of repetition (if any) this stack call is using.
			switch {
			case s.config.ForEach != nil:
				// The call uses for_each, and so we can't actually build a known
				// result just yet because we don't know yet how many instances
				// there will be and what their keys will be. We'll just construct
				// an unknown value of a suitable type instead.
				return cty.UnknownVal(s.ResultType(ctx)), diags
			default:
				// No repetition at all, then. In this case we _can_ attempt to
				// construct at least a partial result, because we already know
				// there will be exactly one instance and can assume that
				// the output value implementation will provide a suitable
				// approximation of the final value.
				calleeStack := s.CalleeConfig(ctx)
				calleeOutputs := calleeStack.OutputValues(ctx)
				attrs := make(map[string]cty.Value, len(calleeOutputs))
				for addr, ov := range calleeOutputs {
					attrs[addr.Name] = ov.Value(ctx, phase)
				}
				return cty.ObjectVal(attrs), diags
			}
		},
	))
}

// ResolveExpressionReference implements ExpressionScope for evaluating
// expressions within a "stack" block during the validation phase.
//
// Note that the "stack" block lives in the caller scope rather than the
// callee scope, so this scope is not appropriate for evaluating anything
// inside the child variable declarations: they belong to the callee
// scope.
//
// This scope produces an approximation of expression results that is true
// for all instances of the stack call, not final results for a specific
// instance of a stack call. This is not the right scope to use during the
// plan and apply phases.
func (s *StackCallConfig) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	repetition := instances.RepetitionData{}
	if s.config.ForEach != nil {
		// We're producing an approximation across all eventual instances
		// of this call, so we'll set each.key and each.value to unknown
		// values.
		repetition.EachKey = cty.UnknownVal(cty.String).RefineNotNull()
		repetition.EachValue = cty.DynamicVal
	}
	ret, diags := s.main.
		mustStackConfig(ctx, s.Addr().Stack).
		resolveExpressionReference(ctx, ref, nil, repetition)

	if _, ok := ret.(*ProviderConfig); ok {
		// We can't reference other providers from anywhere inside an embedded
		// stack call - they should define their own providers.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   fmt.Sprintf("The object %s is not in scope at this location.", ref.Target.String()),
			Subject:  ref.SourceRange.ToHCL().Ptr(),
		})
	}

	return ret, diags
}

// ExternalFunctions implements ExpressionScope.
func (s *StackCallConfig) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return s.main.ProviderFunctions(ctx, s.main.StackConfig(ctx, s.Addr().Stack))
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (s *StackCallConfig) PlanTimestamp() time.Time {
	return s.main.PlanTimestamp()
}

func (s *StackCallConfig) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	_, moreDiags := s.ValidateForEachValue(ctx, phase)
	diags = diags.Append(moreDiags)
	_, moreDiags = s.ValidateInputVariableValues(ctx, phase)
	diags = diags.Append(moreDiags)
	_, moreDiags = s.ValidateResultValue(ctx, phase)
	diags = diags.Append(moreDiags)
	moreDiags = ValidateDependsOn(ctx, s.CallerConfig(ctx), s.config.DependsOn)
	diags = diags.Append(moreDiags)
	return diags
}

// Validate implements Validatable
func (s *StackCallConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	return s.checkValid(ctx, ValidatePhase)
}

// PlanChanges implements Plannable.
func (s *StackCallConfig) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, s.checkValid(ctx, PlanPhase)
}

// ExprReferenceValue implements Referenceable.
func (s *StackCallConfig) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	return s.ResultValue(ctx, phase)
}

// reportNamedPromises implements namedPromiseReporter.
func (s *StackCallConfig) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	// We'll report the same names for each promise in a given category
	// because promises from different phases should not typically interact
	// with one another and so mentioning phase here will typically just
	// make error messages more confusing.
	forEachName := s.Addr().String() + " for_each"
	s.forEachValue.Each(func(ep EvalPhase, once *promising.Once[withDiagnostics[cty.Value]]) {
		cb(once.PromiseID(), forEachName)
	})
	inputsName := s.Addr().String() + " inputs"
	s.inputVariableValues.Each(func(ep EvalPhase, once *promising.Once[withDiagnostics[map[stackaddrs.InputVariable]cty.Value]]) {
		cb(once.PromiseID(), inputsName)
	})
	resultName := s.Addr().String() + " collected outputs"
	s.resultValue.Each(func(ep EvalPhase, once *promising.Once[withDiagnostics[cty.Value]]) {
		cb(once.PromiseID(), resultName)
	})
}
