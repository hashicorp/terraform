// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type PlanDataSourceStep struct {
}

func (s *PlanDataSourceStep) Execute(ctx EvalContext, node *NodePlannableResourceInstance, data *ResourceData) (ResourceState[*NodePlannableResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	addr := node.ResourceInstanceAddr()
	deferrals := ctx.Deferrals()
	change, state, deferred, repeatData, planDiags := node.planDataSource(ctx, data.CheckRuleSeverity, data.SkipPlanning, deferrals.ShouldDeferResourceInstanceChanges(addr, node.Dependencies))
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	// A nil change here indicates that Terraform is deciding NOT to make a
	// change at all. In which case even if we wanted to try and defer it
	// (because of a dependency) we can't as there is no change to defer.
	//
	// The most common case for this is when the data source is being refreshed
	// but depends on unknown values or dependencies which means we just skip
	// refreshing the data source. We maintain that behaviour here.
	if change != nil && deferred != nil {
		// Then this data source got deferred by the provider during planning.
		deferrals.ReportDataSourceInstanceDeferred(addr, deferred.Reason, change)
	} else {
		// Not deferred; business as usual.

		// write the data source into both the refresh state and the
		// working state
		diags = diags.Append(node.writeResourceInstanceState(ctx, state, refreshState))
		if diags.HasErrors() {
			return nil, diags
		}
		diags = diags.Append(node.writeResourceInstanceState(ctx, state, workingState))
		if diags.HasErrors() {
			return nil, diags
		}

		diags = diags.Append(node.writeChange(ctx, change, ""))

		// Post-conditions might block further progress. We intentionally do this
		// _after_ writing the state/diff because we want to check against
		// the result of the operation, and to fail on future operations
		// until the user makes the condition succeed.
		checkDiags := evalCheckRules(
			addrs.ResourcePostcondition,
			node.Config.Postconditions,
			ctx, addr, repeatData,
			data.CheckRuleSeverity,
		)
		diags = diags.Append(checkDiags)
	}

	return nil, diags
}
