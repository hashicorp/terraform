// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodeApplyableResourceInstance represents a resource instance that is
// "applyable": it is ready to be applied and is represented by a diff.
//
// This node is for a specific instance of a resource. It will usually be
// accompanied in the graph by a NodeApplyableResource representing its
// containing resource, and should depend on that node to ensure that the
// state is properly prepared to receive changes to instances.
type NodeApplyableResourceInstance struct {
	*NodeAbstractResourceInstance

	graphNodeDeposer // implementation of GraphNodeDeposerConfig

	// forceReplace indicates that this resource is being replaced for external
	// reasons, like a -replace flag or via replace_triggered_by.
	forceReplace bool
}

var (
	_ GraphNodeConfigResource         = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeResourceInstance       = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeCreator                = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeReferencer             = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeDeposer                = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeExecutable             = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeAttachDependencies     = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeActionProviderConsumer = (*NodeApplyableResourceInstance)(nil)
)

// GraphNodeCreator
func (n *NodeApplyableResourceInstance) CreateAddr() *addrs.AbsResourceInstance {
	addr := n.ResourceInstanceAddr()
	return &addr
}

// GraphNodeReferencer, overriding NodeAbstractResourceInstance
func (n *NodeApplyableResourceInstance) References() []*addrs.Reference {
	return n.NodeAbstractResourceInstance.References()
}

// GraphNodeAttachDependencies
func (n *NodeApplyableResourceInstance) AttachDependencies(deps []addrs.ConfigResource) {
	n.Dependencies = deps
}

// GraphNodeExecutable
func (n *NodeApplyableResourceInstance) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	if n.Config == nil {
		// If there is no config, and there is no change, then we have nothing
		// to do and the change was left in the plan for informational
		// purposes only.
		changes := ctx.Changes()
		csrc := changes.GetResourceInstanceChange(n.ResourceInstanceAddr(), addrs.NotDeposed)
		if csrc == nil || csrc.Action == plans.NoOp {
			log.Printf("[DEBUG] NodeApplyableResourceInstance: No config or planned change recorded for %s", n.Addr)
			return nil
		}

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Resource node has no configuration attached",
			fmt.Sprintf(
				"The graph node for %s has no configuration attached to it. This suggests a bug in Terraform's apply graph builder; please report it!",
				addr,
			),
		))
		return diags
	}

	// Eval info is different depending on what kind of resource this is
	switch n.Config.Mode {
	case addrs.ManagedResourceMode:
		return n.managedResourceExecute(ctx)
	case addrs.DataResourceMode:
		return n.dataResourceExecute(ctx)
	case addrs.EphemeralResourceMode:
		return n.ephemeralResourceExecute(ctx)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodeApplyableResourceInstance) ephemeralResourceExecute(ctx EvalContext) tfdiags.Diagnostics {
	_, diags := ephemeralResourceOpen(ctx, ephemeralResourceInput{
		addr:           n.Addr,
		config:         n.Config,
		providerConfig: n.ResolvedProvider,
		override:       n.override,
	})

	return diags
}

func (n *NodeApplyableResourceInstance) dataResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	change := ctx.Changes().GetResourceInstanceChange(n.Addr, addrs.NotDeposed)
	// Stop early if we don't actually have a diff
	if change == nil {
		return diags
	}
	if change.Action != plans.Read && change.Action != plans.NoOp {
		diags = diags.Append(fmt.Errorf("nonsensical planned action %#v for %s; this is a bug in Terraform", change.Action, n.Addr))
	}

	// In this particular call to applyDataSource we include our planned
	// change, which signals that we expect this read to complete fully
	// with no unknown values; it'll produce an error if not.
	state, repeatData, applyDiags := n.applyDataSource(ctx, change)
	diags = diags.Append(applyDiags)
	if diags.HasErrors() {
		return diags
	}

	if state != nil {
		// If n.applyDataSource returned a nil state object with no accompanying
		// errors then it determined that the given change doesn't require
		// actually reading the data (e.g. because it was already read during
		// the plan phase) and so we're only running through here to get the
		// extra details like precondition/postcondition checks.
		diags = diags.Append(n.writeResourceInstanceState(ctx, state, workingState))
		if diags.HasErrors() {
			return diags
		}
	}

	diags = diags.Append(n.writeChange(ctx, nil, states.NotDeposed))

	diags = diags.Append(updateStateHook(ctx))

	// Post-conditions might block further progress. We intentionally do this
	// _after_ writing the state/diff because we want to check against
	// the result of the operation, and to fail on future operations
	// until the user makes the condition succeed.
	checkDiags := evalCheckRules(
		addrs.ResourcePostcondition,
		n.Config.Postconditions,
		ctx, n.ResourceInstanceAddr(),
		repeatData,
		tfdiags.Error,
	)
	diags = diags.Append(checkDiags)

	return diags
}

func (n *NodeApplyableResourceInstance) managedResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var state *states.ResourceInstanceObject
	var createBeforeDestroyEnabled bool
	var deposedKey states.DeposedKey

	addr := n.ResourceInstanceAddr().Resource
	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// Get the saved diff for apply
	diffApply := ctx.Changes().GetResourceInstanceChange(n.Addr, addrs.NotDeposed)

	// We don't want to do any destroys
	// (these are handled by NodeDestroyResourceInstance instead)
	if diffApply == nil || diffApply.Action == plans.Delete {
		return diags
	}
	if diffApply.Action == plans.Read {
		diags = diags.Append(fmt.Errorf("nonsensical planned action %#v for %s; this is a bug in Terraform", diffApply.Action, n.Addr))
	}

	destroy := (diffApply.Action == plans.Delete || diffApply.Action.IsReplace())
	// Get the stored action for CBD if we have a plan already
	createBeforeDestroyEnabled = diffApply.Change.Action == plans.CreateThenDelete

	if destroy && n.CreateBeforeDestroy() {
		createBeforeDestroyEnabled = true
	}

	if createBeforeDestroyEnabled {
		state := ctx.State()
		if n.PreallocatedDeposedKey == states.NotDeposed {
			deposedKey = state.DeposeResourceInstanceObject(n.Addr)
		} else {
			deposedKey = n.PreallocatedDeposedKey
			state.DeposeResourceInstanceObjectForceKey(n.Addr, deposedKey)
		}
		log.Printf("[TRACE] managedResourceExecute: prior object for %s now deposed with key %s", n.Addr, deposedKey)
	}

	state, readDiags := n.readResourceInstanceState(ctx, n.ResourceInstanceAddr())
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return diags
	}

	// Get the saved diff
	diff := ctx.Changes().GetResourceInstanceChange(n.Addr, addrs.NotDeposed)

	var forEach map[string]cty.Value
	if n.Config != nil {
		// these diagnostics would be caught earlier, and adding them here only
		// causes duplicates
		forEach, _, _ = evaluateForEachExpression(n.Config.ForEach, ctx, false)
	}

	repData := EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, forEach)

	// Make a new diff, in case we've learned new values in the state
	// during apply which we can now incorporate.
	diffApply, _, deferred, planDiags := n.plan(ctx, diff, state, false, n.forceReplace, repData)
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
		return diags
	}

	if deferred != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Resource deferred during apply, but not during plan",
			Detail: fmt.Sprintf(
				"Terraform has encountered a bug where a provider would mark the resource %q as deferred during apply, but not during plan. This is most likely a bug in the provider. Please file an issue with the provider.", n.Addr,
			),
			Subject: n.Config.DeclRange.Ptr(),
		})
		return diags
	}

	// Compare the diffs
	diags = diags.Append(n.checkPlannedChange(ctx, diff, diffApply, providerSchema))
	if diags.HasErrors() {
		return diags
	}

	diffApply = reducePlan(addr, diffApply, false)
	// reducePlan may have simplified our planned change
	// into a NoOp if it only requires destroying, since destroying
	// is handled by NodeDestroyResourceInstance. If so, we'll
	// still run through most of the logic here because we do still
	// need to deal with other book-keeping such as marking the
	// change as "complete", and running the author's postconditions.

	diags = diags.Append(n.preApplyHook(ctx, diffApply))
	if diags.HasErrors() {
		return diags
	}

	// If there is no change, there was nothing to apply, and we don't need to
	// re-write the state, but we do need to re-evaluate postconditions and
	// still evaluate policy against the final state.
	if diffApply.Action == plans.NoOp {
		n.addPolicyNode(ctx, diffApply, state)

		return diags.Append(n.managedResourcePostconditions(ctx, repData))
	}

	// run any before_* actions
	log.Printf("[DEBUG] NodeApplyableResourceInstance: invoking before actions for %s", n.Addr)
	_, actionDiags := n.invokeActions(ctx, repData, configs.BeforeEvents, diffApply.After)
	diags = diags.Append(actionDiags)
	if diags.HasErrors() {
		log.Printf("[ERROR] NodeApplyableResourceInstance encountered errors while invoking actions, apply aborted.")
		return diags
	}

	state, applyDiags := n.apply(ctx, state, diffApply, n.Config, repData, n.CreateBeforeDestroy())
	diags = diags.Append(applyDiags)
	if diags.HasErrors() {
		// apply errors might need to taint the state
		if err := n.taintInstanceState(ctx, state, diffApply.Action); err != nil {
			return diags.Append(err)
		}
	} else {
		// We only add a policy node if the apply was successful.
		n.addPolicyNode(ctx, diffApply, state)
	}

	// We clear the change out here so that future nodes don't see a change
	// that is already complete.
	err = n.writeChange(ctx, nil, states.NotDeposed)
	if err != nil {
		return diags.Append(err)
	}

	if state != nil {
		// dependencies are always updated to match the configuration during apply
		state.Dependencies = n.Dependencies
	}
	if err = n.writeResourceInstanceState(ctx, state, workingState); err != nil {
		return diags.Append(err)
	}

	///////////
	// Run provisioners and actions. We don't try to invoke either if the
	// resource was already tainted, because apply must have failed.
	if state.Status != states.ObjectTainted {
		createNew := (diffApply.Action == plans.Create || diffApply.Action.IsReplace())
		applyProvisionersDiags := n.evalApplyProvisioners(ctx, state, createNew, configs.ProvisionerWhenCreate)
		// the provisioner errors count as port of the apply error, so we can bundle the diags
		diags = diags.Append(applyProvisionersDiags)
		// provisioners always tainted on error
		if diags.HasErrors() {
			if err := n.taintInstanceState(ctx, state, diffApply.Action); err != nil {
				// we always return immediately if we can't update state
				return diags.Append(err)
			}
		}

		// Actions will directly tell us if the resource is tainted separately from
		// diagnostics.
		taintInstance, actionDiags := n.invokeActions(ctx, repData, configs.AfterEvents, state.Value)
		diags = diags.Append(actionDiags)
		if taintInstance {
			if err := n.taintInstanceState(ctx, state, diffApply.Action); err != nil {
				// we always return immediately if we can't update state
				return diags.Append(err)
			}
		}
	} else {
		log.Printf("[TRACE] evalApply: %s is tainted, so skipping provisioners and actions", n.Addr)
	}

	if createBeforeDestroyEnabled && diags.HasErrors() {
		if deposedKey == states.NotDeposed {
			// This should never happen, and so it always indicates a bug.
			// We should evaluate this node only if we've previously deposed
			// an object as part of the same operation.
			if diffApply != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Attempt to restore non-existent deposed object",
					fmt.Sprintf(
						"Terraform has encountered a bug where it would need to restore a deposed object for %s without knowing a deposed object key for that object. This occurred during a %s action. This is a bug in Terraform; please report it!",
						addr, diffApply.Action,
					),
				))
			} else {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Attempt to restore non-existent deposed object",
					fmt.Sprintf(
						"Terraform has encountered a bug where it would need to restore a deposed object for %s without knowing a deposed object key for that object. This is a bug in Terraform; please report it!",
						addr,
					),
				))
			}
		} else {
			restored := ctx.State().MaybeRestoreResourceInstanceDeposed(addr.Absolute(ctx.Path()), deposedKey)
			if restored {
				log.Printf("[TRACE] managedResourceExecute: %s deposed object %s was restored as the current object", addr, deposedKey)
			} else {
				log.Printf("[TRACE] managedResourceExecute: %s deposed object %s remains deposed", addr, deposedKey)
			}
		}
	}

	diags = diags.Append(n.postApplyHook(ctx, state, diags.Err()))
	diags = diags.Append(updateStateHook(ctx))

	// Post-conditions might block further progress. We intentionally do this
	// _after_ writing the state because we want to check against
	// the result of the operation, and to fail on future operations
	// until the user makes the condition succeed.
	diags = diags.Append(n.managedResourcePostconditions(ctx, repData))

	return diags
}

func (n *NodeApplyableResourceInstance) managedResourcePostconditions(ctx EvalContext, repeatData instances.RepetitionData) (diags tfdiags.Diagnostics) {

	checkDiags := evalCheckRules(
		addrs.ResourcePostcondition,
		n.Config.Postconditions,
		ctx, n.ResourceInstanceAddr(), repeatData,
		tfdiags.Error,
	)
	return diags.Append(checkDiags)
}

// checkPlannedChange produces errors if the _actual_ expected value is not
// compatible with what was recorded in the plan.
//
// Errors here are most often indicative of a bug in the provider, so our error
// messages will report with that in mind. It's also possible that there's a bug
// in Terraform's Core's own "proposed new value" code in EvalDiff.
func (n *NodeApplyableResourceInstance) checkPlannedChange(ctx EvalContext, plannedChange, actualChange *plans.ResourceInstanceChange, providerSchema providers.ProviderSchema) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	addr := n.ResourceInstanceAddr().Resource

	schema := providerSchema.SchemaForResourceAddr(addr.ContainingResource())
	if schema.Body == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support %q", addr.Resource.Type))
		return diags
	}

	absAddr := addr.Absolute(ctx.Path())

	log.Printf("[TRACE] checkPlannedChange: Verifying that actual change (action %s) matches planned change (action %s)", actualChange.Action, plannedChange.Action)

	if plannedChange.Action != actualChange.Action {
		switch {
		case plannedChange.Action == plans.Update && actualChange.Action == plans.NoOp:
			// It's okay for an update to become a NoOp once we've filled in
			// all of the unknown values, since the final values might actually
			// match what was there before after all.
			log.Printf("[DEBUG] After incorporating new values learned so far during apply, %s change has become NoOp", absAddr)

		case (plannedChange.Action == plans.CreateThenDelete && actualChange.Action == plans.DeleteThenCreate) ||
			(plannedChange.Action == plans.DeleteThenCreate && actualChange.Action == plans.CreateThenDelete):
			// If the order of replacement changed, then that is a bug in terraform
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Terraform produced inconsistent final plan",
				fmt.Sprintf(
					"When expanding the plan for %s to include new values learned so far during apply, the planned action changed from %s to %s.\n\nThis is a bug in Terraform and should be reported.",
					absAddr, plannedChange.Action, actualChange.Action,
				),
			))
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced inconsistent final plan",
				fmt.Sprintf(
					"When expanding the plan for %s to include new values learned so far during apply, provider %q changed the planned action from %s to %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					absAddr, n.ResolvedProvider.Provider.String(),
					plannedChange.Action, actualChange.Action,
				),
			))
		}
	}

	errs := objchange.AssertObjectCompatible(schema.Body, plannedChange.After, actualChange.After)
	for _, err := range errs {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced inconsistent final plan",
			fmt.Sprintf(
				"When expanding the plan for %s to include new values learned so far during apply, provider %q produced an invalid new value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				absAddr, n.ResolvedProvider.Provider.String(), tfdiags.FormatError(err),
			),
		))
	}
	return diags
}

// taintInstanceState takes the state object error from an apply operation and
// writes the instance object to the global stated marked as tainted, but only
// if the instance was being created.
//
// TODO: Tainted was invented for failed create events, and provisioners which
// could only be associated with those create events. If actions ever need to be
// rerun for other event types, something more specific than `Tainted` needs to
// be added to the resource state.
func (n *NodeApplyableResourceInstance) taintInstanceState(ctx EvalContext, state *states.ResourceInstanceObject, action plans.Action) error {
	if action != plans.Create {
		return nil
	}

	log.Printf("[TRACE] taintState: %s encountered an error during creation, so it is now marked as tainted", n.Addr)
	return n.writeResourceInstanceState(ctx, state.AsTainted(), workingState)

}
