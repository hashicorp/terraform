// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// LocalValueConfig represents a "locals" block in a stack configuration.
type LocalValueConfig struct {
	addr   stackaddrs.ConfigLocalValue
	config *stackconfig.LocalValue
	stack  *StackConfig

	main *Main

	validatedValue perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
}

var (
	_ Validatable   = (*LocalValueConfig)(nil)
	_ Referenceable = (*LocalValueConfig)(nil)
)

func newLocalValueConfig(main *Main, addr stackaddrs.ConfigLocalValue, stack *StackConfig, config *stackconfig.LocalValue) *LocalValueConfig {
	if config == nil {
		panic("newLocalValueConfig with nil configuration")
	}
	return &LocalValueConfig{
		addr:   addr,
		config: config,
		stack:  stack,
		main:   main,
	}
}

func (v *LocalValueConfig) tracingName() string {
	return v.addr.String()
}

// ExprReferenceValue implements Referenceable
func (v *LocalValueConfig) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	out, _ := v.ValidateValue(ctx, phase)
	return out
}

// ValidateValue validates that the value expression is evaluatable and that
// its result can convert to the declared type constraint, returning the
// resulting value.
//
// If the returned diagnostics has errors then the returned value might be
// just an approximation of the result, such as an unknown value with the
// declared type constraint.
func (v *LocalValueConfig) ValidateValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return withCtyDynamicValPlaceholder(doOnceWithDiags(
		ctx, v.tracingName(), v.validatedValue.For(phase),
		v.validateValueInner(phase),
	))
}

// validateValueInner is the real implementation of ValidateValue, which runs
// in the background only once per instance of [OutputValueConfig] and then
// provides the result for all ValidateValue callers simultaneously.
func (v *LocalValueConfig) validateValueInner(phase EvalPhase) func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
	return func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
		var diags tfdiags.Diagnostics

		result, moreDiags := EvalExprAndEvalContext(ctx, v.config.Value, phase, v.stack)
		value := result.Value
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			value = cty.UnknownVal(cty.DynamicPseudoType)
		}

		return value, diags
	}
}

func (v *LocalValueConfig) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	_, moreDiags := v.ValidateValue(ctx, phase)
	diags = diags.Append(moreDiags)
	return diags
}

// Validate implements Validatable
func (v *LocalValueConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	return v.checkValid(ctx, ValidatePhase)
}

// PlanChanges implements Plannable.
func (v *LocalValueConfig) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, v.checkValid(ctx, PlanPhase)
}
