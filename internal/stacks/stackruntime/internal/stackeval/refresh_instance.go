// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// RefreshInstance is different kind of node in the graph. Rather than being
// instantiated by the configuration, it is loaded dynamically by a relevant
// component or removed block. It represents the refresh action of a given
// instance within state.
//
// This is only ever called during a destroy operation, and is used to refresh
// the state of the component before it is destroyed. If this changes, then
// the PreDestroyRefresh option should be removed from the plan options.
type RefreshInstance struct {
	component *ComponentInstance

	result         promising.Once[map[string]cty.Value]
	moduleTreePlan promising.Once[withDiagnostics[*plans.Plan]]
}

func newRefreshInstance(component *ComponentInstance) *RefreshInstance {
	return &RefreshInstance{
		component: component,
	}
}

// reportNamedPromises implements namedPromiseReporter.
func (r *RefreshInstance) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	cb(r.moduleTreePlan.PromiseID(), r.component.Addr().String()+" instance")
	cb(r.result.PromiseID(), r.component.Addr().String()+" result")
}

// Result returns the outputs of the refresh action for this instance.
func (r *RefreshInstance) Result(ctx context.Context) map[string]cty.Value {
	result, err := r.result.Do(ctx, func(ctx context.Context) (map[string]cty.Value, error) {
		config := r.component.ModuleTree(ctx)

		plan, _ := r.Plan(ctx)
		if plan == nil {
			// Then we'll return dynamic values for all outputs, and the error
			// from the plan will be raised elsewhere.
			outputs := make(map[string]cty.Value, len(config.Module.Outputs))
			for output := range config.Module.Outputs {
				outputs[output] = cty.DynamicVal
			}
			return outputs, nil
		}
		return stackplan.OutputsFromPlan(config, plan), nil
	})
	if err != nil {
		// This should never happen as we do not return an error from within
		// the function literal passed to Do. But, if somehow we do this, then
		// it means we will skip the refresh for this component.
		return nil
	}
	return result
}

func (r *RefreshInstance) Plan(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
	return doOnceWithDiags(ctx, &r.moduleTreePlan, r, func(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
		opts, diags := r.component.PlanOpts(ctx, plans.RefreshOnlyMode, false)
		if opts == nil {
			return nil, diags
		}

		// For now, the refresh option is only used to separate the refresh
		// from the apply during a destroy operation. So, we want to use that
		// option here to ensure that the refresh is done in a way that is
		// compatible with the destroy operation.
		opts.PreDestroyRefresh = true

		plan, moreDiags := PlanComponentInstance(ctx, r.component.main, r.component.PlanPrevState(ctx), opts, r.component)
		return plan, diags.Append(moreDiags)
	})
}
