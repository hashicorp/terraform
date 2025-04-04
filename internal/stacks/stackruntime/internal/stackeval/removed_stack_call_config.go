// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ Validatable     = (*RemovedStackCallConfig)(nil)
	_ Plannable       = (*RemovedStackCallConfig)(nil)
	_ ExpressionScope = (*RemovedStackCallConfig)(nil)
)

type RemovedStackCallConfig struct {
	target stackaddrs.ConfigStackCall // relative to stack
	config *stackconfig.Removed
	stack  *StackConfig

	main *Main

	forEachValue        perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	inputVariableValues perEvalPhase[promising.Once[withDiagnostics[map[stackaddrs.InputVariable]cty.Value]]]
}

func newRemovedStackCallConfig(main *Main, target stackaddrs.ConfigStackCall, stack *StackConfig, config *stackconfig.Removed) *RemovedStackCallConfig {
	return &RemovedStackCallConfig{
		target: target,
		config: config,
		stack:  stack,
		main:   main,
	}
}

func (r *RemovedStackCallConfig) TargetConfig() *StackConfig {
	current := r.stack
	for _, step := range r.target.Stack {
		current = current.ChildConfig(step)
	}
	return current.ChildConfig(stackaddrs.StackStep{Name: r.target.Item.Name})
}

func (r *RemovedStackCallConfig) InputVariableValues(ctx context.Context, phase EvalPhase) (map[stackaddrs.InputVariable]cty.Value, tfdiags.Diagnostics) {

	return doOnceWithDiags(ctx, r.tracingName()+" inputs", r.inputVariableValues.For(phase), validateStackCallInputsFn(r.config.Inputs, r.config.DeclRange.ToHCL(), r.TargetConfig(), r, phase))
}

func (r *RemovedStackCallConfig) ForEachValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(ctx, r.tracingName()+" for_each", r.forEachValue.For(phase), func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
		if r.config.ForEach == nil {
			// This stack config isn't even using for_each.
			return cty.NilVal, nil
		}

		var diags tfdiags.Diagnostics
		result, moreDiags := evaluateForEachExpr(ctx, r.config.ForEach, ValidatePhase, r.stack, "stack")
		diags = diags.Append(moreDiags)
		return result.Value, diags
	})
}

func (r *RemovedStackCallConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	_, moreDiags := r.ForEachValue(ctx, ValidatePhase)
	diags = diags.Append(moreDiags)
	_, moreDiags = r.InputVariableValues(ctx, ValidatePhase)
	diags = diags.Append(moreDiags)
	return diags
}

func (r *RemovedStackCallConfig) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	_, moreDiags := r.ForEachValue(ctx, PlanPhase)
	diags = diags.Append(moreDiags)
	_, moreDiags = r.InputVariableValues(ctx, PlanPhase)
	diags = diags.Append(moreDiags)
	return nil, diags
}

func (r *RemovedStackCallConfig) tracingName() string {
	return fmt.Sprintf("%s -> %s (removed)", r.stack.addr, r.target)
}

func (r *RemovedStackCallConfig) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	repetition := instances.RepetitionData{}
	if r.config.ForEach != nil {
		// We're producing an approximation across all eventual instances
		// of this call, so we'll set each.key and each.value to unknown
		// values.
		repetition.EachKey = cty.UnknownVal(cty.String).RefineNotNull()
		repetition.EachValue = cty.DynamicVal
	}
	ret, diags := r.stack.resolveExpressionReference(ctx, ref, nil, repetition)

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

func (r *RemovedStackCallConfig) PlanTimestamp() time.Time {
	return r.main.PlanTimestamp()
}

func (r *RemovedStackCallConfig) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return r.main.ProviderFunctions(ctx, r.stack)
}
