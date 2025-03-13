// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodePlanDestroyableResourceInstance represents a resource that is ready
// to be planned for destruction.
type NodePlanDestroyableResourceInstance struct {
	*NodeAbstractResourceInstance

	// skipRefresh indicates that we should skip refreshing
	skipRefresh bool
}

var (
	_ GraphNodeModuleInstance       = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeReferencer           = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeDestroyer            = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeConfigResource       = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeResourceInstance     = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeAttachResourceState  = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeExecutable           = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeProviderConsumer     = (*NodePlanDestroyableResourceInstance)(nil)
)

func (n *NodePlanDestroyableResourceInstance) Name() string {
	return n.NodeAbstractResource.Name() + " (destroy)"
}

// GraphNodeDestroyer
func (n *NodePlanDestroyableResourceInstance) DestroyAddr() *addrs.AbsResourceInstance {
	addr := n.ResourceInstanceAddr()
	return &addr
}

// GraphNodeEvalable
func (n *NodePlanDestroyableResourceInstance) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	switch addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		return n.managedResourceExecute(ctx, op)
	case addrs.DataResourceMode:
		return n.dataResourceExecute(ctx, op)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodePlanDestroyableResourceInstance) managedResourceExecute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	// Declare a bunch of variables that are used for state during
	// evaluation. These are written to by address in the EvalNodes we
	// declare below.
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	state, err := n.readResourceInstanceState(ctx, addr)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// If we are in the "skip refresh" mode then we will have skipped over our
	// usual opportunity to update the previous run state and refresh state
	// with the result of any provider schema upgrades, so we'll compensate
	// by doing that here.
	//
	// NOTE: this is coupled with logic in Context.destroyPlan which skips
	// running a normal plan walk when refresh is enabled. These two
	// conditionals must agree (be exactly opposite) in order to get the
	// correct behavior in both cases.
	if n.skipRefresh {
		diags = diags.Append(n.writeResourceInstanceState(ctx, state, prevRunState))
		if diags.HasErrors() {
			return diags
		}
		diags = diags.Append(n.writeResourceInstanceState(ctx, state, refreshState))
		if diags.HasErrors() {
			return diags
		}
	}

	change, deferred, destroyPlanDiags := n.planDestroy(ctx, state, "")
	diags = diags.Append(destroyPlanDiags)
	if diags.HasErrors() {
		return diags
	}

	if deferred != nil {
		ctx.Deferrals().ReportResourceInstanceDeferred(n.Addr, deferred.Reason, change)
		return diags
	} else if ctx.Deferrals().ShouldDeferResourceInstanceChanges(n.Addr, n.Dependencies) {
		ctx.Deferrals().ReportResourceInstanceDeferred(n.Addr, providers.DeferredReasonDeferredPrereq, change)
		return diags
	}

	// We intentionally write the change before the subsequent checks, because
	// all of the checks below this point are for problems caused by the
	// context surrounding the change, rather than the change itself, and
	// so it's helpful to still include the valid-in-isolation change as
	// part of the plan as additional context in our error output.
	diags = diags.Append(n.writeChange(ctx, change, ""))

	diags = diags.Append(n.checkPreventDestroy(change))
	if diags.HasErrors() {
		return diags
	}

	return diags
}

func (n *NodePlanDestroyableResourceInstance) dataResourceExecute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {

	// We may not be able to read a prior data source from the state if the
	// schema was upgraded and we are destroying before ever refreshing that
	// data source. Regardless, a data source  "destroy" is simply writing a
	// null state, which we can do with a null prior state too.
	change := &plans.ResourceInstanceChange{
		Addr:        n.ResourceInstanceAddr(),
		PrevRunAddr: n.prevRunAddr(ctx),
		Change: plans.Change{
			Action: plans.Delete,
			Before: cty.NullVal(cty.DynamicPseudoType),
			After:  cty.NullVal(cty.DynamicPseudoType),
		},
		ProviderAddr: n.ResolvedProvider,
	}
	return diags.Append(n.writeChange(ctx, change, ""))
}
