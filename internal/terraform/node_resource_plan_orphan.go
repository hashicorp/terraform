// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodePlannableResourceInstanceOrphan represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodePlannableResourceInstanceOrphan struct {
	*NodeAbstractResourceInstance

	// skipRefresh indicates that we should skip refreshing individual instances
	skipRefresh bool

	// skipPlanChanges indicates we should skip trying to plan change actions
	// for any instances.
	skipPlanChanges bool

	// forgetResources lists resources that should not be destroyed, only removed
	// from state.
	forgetResources []addrs.ConfigResource

	// forgetModules lists modules that should not be destroyed, only removed
	// from state.
	forgetModules []addrs.Module
}

var (
	_ GraphNodeModuleInstance       = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeReferenceable        = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeReferencer           = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeConfigResource       = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeResourceInstance     = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeAttachResourceState  = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeExecutable           = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeProviderConsumer     = (*NodePlannableResourceInstanceOrphan)(nil)
)

func (n *NodePlannableResourceInstanceOrphan) Name() string {
	return n.ResourceInstanceAddr().String() + " (orphan)"
}

// GraphNodeExecutable
func (n *NodePlannableResourceInstanceOrphan) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	addr := n.ResourceInstanceAddr()

	// Eval info is different depending on what kind of resource this is
	switch addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		return n.managedResourceExecute(ctx)
	case addrs.DataResourceMode:
		return n.dataResourceExecute(ctx)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodePlannableResourceInstanceOrphan) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
	if n.Addr.Resource.Resource.Mode == addrs.DataResourceMode {
		// indicate that this node does not require a configured provider
		return nil, true
	}
	return n.NodeAbstractResourceInstance.ProvidedBy()
}

func (n *NodePlannableResourceInstanceOrphan) dataResourceExecute(ctx EvalContext) tfdiags.Diagnostics {
	// A data source that is no longer in the config is removed from the state
	log.Printf("[TRACE] NodePlannableResourceInstanceOrphan: removing state object for %s", n.Addr)

	// we need to update both the refresh state to refresh the current data
	// source, and the working state for plan-time evaluations.
	refreshState := ctx.RefreshState()
	refreshState.SetResourceInstanceCurrent(n.Addr, nil, n.ResolvedProvider)

	workingState := ctx.State()
	workingState.SetResourceInstanceCurrent(n.Addr, nil, n.ResolvedProvider)
	return nil
}

func (n *NodePlannableResourceInstanceOrphan) managedResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	oldState, readDiags := n.readResourceInstanceState(ctx, addr)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return diags
	}

	// Note any upgrades that readResourceInstanceState might've done in the
	// prevRunState, so that it'll conform to current schema.
	diags = diags.Append(n.writeResourceInstanceState(ctx, oldState, prevRunState))
	if diags.HasErrors() {
		return diags
	}
	// Also the refreshState, because that should still reflect schema upgrades
	// even if not refreshing.
	diags = diags.Append(n.writeResourceInstanceState(ctx, oldState, refreshState))
	if diags.HasErrors() {
		return diags
	}

	if !n.skipRefresh {
		// Refresh this instance even though it is going to be destroyed, in
		// order to catch missing resources. If this is a normal plan,
		// providers expect a Read request to remove missing resources from the
		// plan before apply, and may not handle a missing resource during
		// Delete correctly.  If this is a simple refresh, Terraform is
		// expected to remove the missing resource from the state entirely
		refreshedState, deferred, refreshDiags := n.refresh(ctx, states.NotDeposed, oldState)
		diags = diags.Append(refreshDiags)
		if diags.HasErrors() {
			return diags
		}

		diags = diags.Append(n.writeResourceInstanceState(ctx, refreshedState, refreshState))
		if diags.HasErrors() {
			return diags
		}

		// If we refreshed then our subsequent planning should be in terms of
		// the new object, not the original object.
		if deferred == nil {
			oldState = refreshedState
		} else {
			ctx.Deferrals().ReportResourceInstanceDeferred(n.Addr, deferred.Reason, &plans.ResourceInstanceChange{
				Addr: n.Addr,
				Change: plans.Change{
					Action: plans.Read,
					Before: oldState.Value,
					After:  refreshedState.Value,
				},
			})
		}
	}

	// If we're skipping planning, all we need to do is write the state. If the
	// refresh indicates the instance no longer exists, there is also nothing
	// to plan because there is no longer any state and it doesn't exist in the
	// config.
	if n.skipPlanChanges || oldState == nil || oldState.Value.IsNull() {
		return diags.Append(n.writeResourceInstanceState(ctx, oldState, workingState))
	}

	var forget bool
	for _, ft := range n.forgetResources {
		if ft.Equal(n.ResourceAddr()) {
			forget = true
		}
	}
	for _, fm := range n.forgetModules {
		if fm.TargetContains(n.Addr) {
			forget = true
		}
	}
	var change *plans.ResourceInstanceChange
	var pDiags tfdiags.Diagnostics
	if forget {
		change, pDiags = n.planForget(ctx, oldState, "")
		diags = diags.Append(pDiags)
	} else {
		change, pDiags = n.planDestroy(ctx, oldState, "")
		diags = diags.Append(pDiags)
		if diags.HasErrors() {
			return diags
		}
	}
	if diags.HasErrors() {
		return diags
	}

	// We might be able to offer an approximate reason for why we are
	// planning to delete this object. (This is best-effort; we might
	// sometimes not have a reason.)
	// The change can be nil in case of deferred destroys.
	if change != nil {
		change.ActionReason = n.deleteActionReason(ctx)
	}

	// We intentionally write the change before the subsequent checks, because
	// all of the checks below this point are for problems caused by the
	// context surrounding the change, rather than the change itself, and
	// so it's helpful to still include the valid-in-isolation change as
	// part of the plan as additional context in our error output.
	diags = diags.Append(n.writeChange(ctx, change, ""))
	if diags.HasErrors() {
		return diags
	}

	if !forget {
		diags = diags.Append(n.checkPreventDestroy(change))
		if diags.HasErrors() {
			return diags
		}
	}

	return diags.Append(n.writeResourceInstanceState(ctx, nil, workingState))
}

func (n *NodePlannableResourceInstanceOrphan) deleteActionReason(ctx EvalContext) plans.ResourceInstanceChangeActionReason {
	cfg := n.Config
	if cfg == nil {
		if !n.Addr.Equal(n.prevRunAddr(ctx)) {
			// This means the resource was moved - see also
			// ResourceInstanceChange.Moved() which calculates
			// this the same way.
			return plans.ResourceInstanceDeleteBecauseNoMoveTarget
		}

		return plans.ResourceInstanceDeleteBecauseNoResourceConfig
	}

	// If this is a resource instance inside a module instance that's no
	// longer declared then we will have a config (because config isn't
	// instance-specific) but the expander will know that our resource
	// address's module path refers to an undeclared module instance.
	if expander := ctx.InstanceExpander(); expander != nil { // (sometimes nil in MockEvalContext in tests)
		validModuleAddr := expander.GetDeepestExistingModuleInstance(n.Addr.Module)
		if len(validModuleAddr) != len(n.Addr.Module) {
			// If we get here then at least one step in the resource's module
			// path is to a module instance that doesn't exist at all, and
			// so a missing module instance is the delete reason regardless
			// of whether there might _also_ be a change to the resource
			// configuration inside the module. (Conceptually the configurations
			// inside the non-existing module instance don't exist at all,
			// but they end up existing just as an artifact of the
			// implementation detail that we detect module instance orphans
			// only dynamically.)
			return plans.ResourceInstanceDeleteBecauseNoModule
		}
	}

	switch n.Addr.Resource.Key.(type) {
	case nil: // no instance key at all
		if cfg.Count != nil || cfg.ForEach != nil {
			return plans.ResourceInstanceDeleteBecauseWrongRepetition
		}
	case addrs.IntKey:
		if cfg.Count == nil {
			// This resource isn't using "count" at all, then
			return plans.ResourceInstanceDeleteBecauseWrongRepetition
		}

		expander := ctx.InstanceExpander()
		if expander == nil {
			break // only for tests that produce an incomplete MockEvalContext
		}
		insts := expander.ExpandResource(n.Addr.ContainingResource())

		declared := false
		for _, inst := range insts {
			if n.Addr.Equal(inst) {
				declared = true
			}
		}
		if !declared {
			// This instance key is outside of the configured range
			return plans.ResourceInstanceDeleteBecauseCountIndex
		}
	case addrs.StringKey:
		if cfg.ForEach == nil {
			// This resource isn't using "for_each" at all, then
			return plans.ResourceInstanceDeleteBecauseWrongRepetition
		}

		expander := ctx.InstanceExpander()
		if expander == nil {
			break // only for tests that produce an incomplete MockEvalContext
		}
		insts := expander.ExpandResource(n.Addr.ContainingResource())

		declared := false
		for _, inst := range insts {
			if n.Addr.Equal(inst) {
				declared = true
			}
		}
		if !declared {
			// This instance key is outside of the configured range
			return plans.ResourceInstanceDeleteBecauseEachKey
		}
	}

	// If we get here then the instance key type matches the configured
	// repetition mode, and so we need to consider whether the key itself
	// is within the range of the repetition construct.
	if expander := ctx.InstanceExpander(); expander != nil { // (sometimes nil in MockEvalContext in tests)
		// First we'll check whether our containing module instance still
		// exists, so we can talk about that differently in the reason.
		declared := false
		for _, inst := range expander.ExpandModule(n.Addr.Module.Module()) {
			if n.Addr.Module.Equal(inst) {
				declared = true
				break
			}
		}
		if !declared {
			return plans.ResourceInstanceDeleteBecauseNoModule
		}

		// Now we've proven that we're in a still-existing module instance,
		// we'll see if our instance key matches something actually declared.
		declared = false
		for _, inst := range expander.ExpandResource(n.Addr.ContainingResource()) {
			if n.Addr.Equal(inst) {
				declared = true
				break
			}
		}
		if !declared {
			// Because we already checked that the key _type_ was correct
			// above, we can assume that any mismatch here is a range error,
			// and thus we just need to decide which of the two range
			// errors we're going to return.
			switch n.Addr.Resource.Key.(type) {
			case addrs.IntKey:
				return plans.ResourceInstanceDeleteBecauseCountIndex
			case addrs.StringKey:
				return plans.ResourceInstanceDeleteBecauseEachKey
			}
		}
	}

	// If we didn't find any specific reason to report, we'll report "no reason"
	// as a fallback, which means the UI should just state it'll be deleted
	// without any explicit reasoning.
	return plans.ResourceInstanceChangeNoReason
}
