// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var _ ExpressionScope = (*RemovedStackCallInstance)(nil)
var _ Plannable = (*RemovedStackCallInstance)(nil)
var _ Applyable = (*RemovedStackCallInstance)(nil)

type RemovedStackCallInstance struct {
	call     *RemovedStackCall
	from     stackaddrs.StackInstance
	deferred bool

	main *Main

	repetition instances.RepetitionData

	stack               perEvalPhase[promising.Once[*Stack]]
	inputVariableValues perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
}

func newRemovedStackCallInstance(call *RemovedStackCall, from stackaddrs.StackInstance, repetition instances.RepetitionData, deferred bool) *RemovedStackCallInstance {
	return &RemovedStackCallInstance{
		call:       call,
		from:       from,
		repetition: repetition,
		deferred:   deferred,
		main:       call.main,
	}
}

func (r *RemovedStackCallInstance) Stack(ctx context.Context, phase EvalPhase) *Stack {
	stack, err := r.stack.For(phase).Do(ctx, r.from.String()+" create", func(ctx context.Context) (*Stack, error) {

		mode := plans.DestroyMode
		if r.main.PlanningMode() == plans.RefreshOnlyMode {
			mode = plans.RefreshOnlyMode
		}

		return newStack(r.main, r.from, r.call.stack, r.call.config.TargetConfig(), r.call.GetExternalRemovedBlocks(), mode, r.deferred), nil
	})
	if err != nil {
		// we never return an error from within the once call, so this shouldn't
		// happen
		return nil
	}
	return stack
}

func (r *RemovedStackCallInstance) InputVariableValues(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(ctx, r.tracingName()+" inputs", r.inputVariableValues.For(phase),
		validateStackCallInstanceInputsFn(r.Stack(ctx, phase), r.call.config.config.Inputs, r.call.config.config.DeclRange.ToHCL().Ptr(), r, phase))
}

func (r *RemovedStackCallInstance) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	_, moreDiags := r.InputVariableValues(ctx, ApplyPhase)
	diags = diags.Append(moreDiags)

	return nil, diags
}

func (r *RemovedStackCallInstance) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	_, moreDiags := r.InputVariableValues(ctx, PlanPhase)
	diags = diags.Append(moreDiags)

	return nil, diags
}

func (r *RemovedStackCallInstance) tracingName() string {
	return r.from.String() + " (removed)"
}

func (r *RemovedStackCallInstance) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	return r.call.stack.resolveExpressionReference(ctx, ref, nil, r.repetition)
}

func (r *RemovedStackCallInstance) PlanTimestamp() time.Time {
	return r.call.stack.main.PlanTimestamp()
}

func (r *RemovedStackCallInstance) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return r.call.stack.main.ProviderFunctions(ctx, r.call.stack.config)
}
