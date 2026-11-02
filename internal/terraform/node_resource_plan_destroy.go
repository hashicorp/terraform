// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodePlanDestroyableResourceInstance represents a resource that is ready
// to be planned for destruction.
type NodePlanDestroyableResourceInstance struct {
	*NodeAbstractResourceInstance

	// skipRefresh indicates that we should skip refreshing
	skipRefresh bool

	// refresh is the node which upgrades and refreshes the object before this
	// node plans to destroy it.
	refresh *nodePlanDestroyRefreshResourceInstance
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
	_ GraphNodeActionCaller         = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeAttachActionTriggers = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeAttachDependencies   = (*NodePlanDestroyableResourceInstance)(nil)
)

func (n *NodePlanDestroyableResourceInstance) Name() string {
	return n.NodeAbstractResource.Name() + " (destroy)"
}

// GraphNodeDestroyer
func (n *NodePlanDestroyableResourceInstance) DestroyAddr() addrs.AbsResourceInstance {
	return n.ResourceInstanceAddr()
}

func (n *NodePlanDestroyableResourceInstance) ReferenceableAddrs() []addrs.Referenceable {
	// destroy nodes are not referenceable, so we must override the method from
	// the abstract layers
	return nil
}

func (n *NodePlanDestroyableResourceInstance) References() (refs []*addrs.Reference) {
	return n.destroyActionPlanReferences()
}

// AttachDependencies implements GraphNodeAttachDependencies. These are
// attached before any destroy edges are added, so they are the resources that
// destroy-time evaluation for this instance depends on in the current
// configuration, such as those referenced by its provider configuration or
// destroy actions. The stored dependencies are still the source of truth for
// the dependencies between objects.
func (n *NodePlanDestroyableResourceInstance) AttachDependencies(deps []addrs.ConfigResource) {
	if n.refresh != nil {
		n.refresh.destroyDependencies = deps
	}
}

func (n *NodePlanDestroyableResourceInstance) AttachActionTriggers(triggers []*resourceActionTrigger) {
	n.actionTriggers = triggers
}

// GraphNodeEvalable
func (n *NodePlanDestroyableResourceInstance) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	switch addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		return n.managedResourceExecute(ctx)
	case addrs.DataResourceMode:
		return n.dataResourceExecute(ctx)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodePlanDestroyableResourceInstance) managedResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	// Declare a bunch of variables that are used for state during
	// evaluation. These are written to by address in the EvalNodes we
	// declare below.
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	if n.refresh != nil {
		state = n.refresh.result
	} else {
		var err tfdiags.Diagnostics
		state, _, err = n.readResourceInstanceState(ctx, addr)
		diags = diags.Append(err)
		if diags.HasErrors() {
			return diags
		}
	}

	// The object was refreshed by the corresponding
	// nodePlanDestroyRefreshResourceInstance, and if it no longer exists there
	// is nothing to destroy.
	if state == nil || state.Value.IsNull() {
		return diags
	}

	change, deferred, destroyPlanDiags := n.planDestroy(ctx, state, "")
	diags = diags.Append(destroyPlanDiags)
	if diags.HasErrors() {
		return diags
	}

	if n.deferPlannedChange(ctx, deferred, change) {
		return diags
	}

	// We intentionally write the change before the subsequent checks, because
	// all of the checks below this point are for problems caused by the
	// context surrounding the change, rather than the change itself, and
	// so it's helpful to still include the valid-in-isolation change as
	// part of the plan as additional context in our error output.
	diags = diags.Append(n.writeChange(ctx, change, states.NotDeposed))

	// Plan the triggered actions before anything else is recorded, because
	// the resource must be deferred instead if any of those actions are
	// deferred.
	actionInvocations, actionsDeferred, actionDiags := n.planActionTriggers(ctx, EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, nil), change)
	if actionsDeferred {
		n.deferForActions(ctx, change)
		return diags.Append(actionDiags)
	}

	diags = diags.Append(n.checkPreventDestroy(change))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(actionDiags)
	recordActionInvocations(ctx, actionInvocations)

	return diags
}

func (n *NodePlanDestroyableResourceInstance) dataResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {

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
	return diags.Append(n.writeChange(ctx, change, states.NotDeposed))
}
