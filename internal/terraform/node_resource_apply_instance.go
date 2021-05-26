package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
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

	// If this node is forced to be CreateBeforeDestroy, we need to record that
	// in the state to.
	ForceCreateBeforeDestroy bool

	// forceReplace are resource instance addresses where the user wants to
	// force generating a replace action. This set isn't pre-filtered, so
	// it might contain addresses that have nothing to do with the resource
	// that this node represents, which the node itself must therefore ignore.
	forceReplace []addrs.AbsResourceInstance
}

var (
	_ GraphNodeConfigResource     = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeResourceInstance   = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeCreator            = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeReferencer         = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeDeposer            = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeExecutable         = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeAttachDependencies = (*NodeApplyableResourceInstance)(nil)
)

// CreateBeforeDestroy returns this node's CreateBeforeDestroy status.
func (n *NodeApplyableResourceInstance) CreateBeforeDestroy() bool {
	if n.ForceCreateBeforeDestroy {
		return n.ForceCreateBeforeDestroy
	}

	if n.Config != nil && n.Config.Managed != nil {
		return n.Config.Managed.CreateBeforeDestroy
	}

	return false
}

func (n *NodeApplyableResourceInstance) ModifyCreateBeforeDestroy(v bool) error {
	n.ForceCreateBeforeDestroy = v
	return nil
}

// GraphNodeCreator
func (n *NodeApplyableResourceInstance) CreateAddr() *addrs.AbsResourceInstance {
	addr := n.ResourceInstanceAddr()
	return &addr
}

// GraphNodeReferencer, overriding NodeAbstractResourceInstance
func (n *NodeApplyableResourceInstance) References() []*addrs.Reference {
	// Start with the usual resource instance implementation
	ret := n.NodeAbstractResourceInstance.References()

	// Applying a resource must also depend on the destruction of any of its
	// dependencies, since this may for example affect the outcome of
	// evaluating an entire list of resources with "count" set (by reducing
	// the count).
	//
	// However, we can't do this in create_before_destroy mode because that
	// would create a dependency cycle. We make a compromise here of requiring
	// changes to be updated across two applies in this case, since the first
	// plan will use the old values.
	if !n.CreateBeforeDestroy() {
		for _, ref := range ret {
			switch tr := ref.Subject.(type) {
			case addrs.ResourceInstance:
				newRef := *ref // shallow copy so we can mutate
				newRef.Subject = tr.Phase(addrs.ResourceInstancePhaseDestroy)
				newRef.Remaining = nil // can't access attributes of something being destroyed
				ret = append(ret, &newRef)
			case addrs.Resource:
				newRef := *ref // shallow copy so we can mutate
				newRef.Subject = tr.Phase(addrs.ResourceInstancePhaseDestroy)
				newRef.Remaining = nil // can't access attributes of something being destroyed
				ret = append(ret, &newRef)
			}
		}
	}

	return ret
}

// GraphNodeAttachDependencies
func (n *NodeApplyableResourceInstance) AttachDependencies(deps []addrs.ConfigResource) {
	n.Dependencies = deps
}

// GraphNodeExecutable
func (n *NodeApplyableResourceInstance) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	if n.Config == nil {
		// This should not be possible, but we've got here in at least one
		// case as discussed in the following issue:
		//    https://github.com/hashicorp/terraform/issues/21258
		// To avoid an outright crash here, we'll instead return an explicit
		// error.
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
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodeApplyableResourceInstance) dataResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	change, err := n.readDiff(ctx, providerSchema)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}
	// Stop early if we don't actually have a diff
	if change == nil {
		return diags
	}

	// In this particular call to applyDataSource we include our planned
	// change, which signals that we expect this read to complete fully
	// with no unknown values; it'll produce an error if not.
	state, applyDiags := n.applyDataSource(ctx, change)
	diags = diags.Append(applyDiags)
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.writeResourceInstanceState(ctx, state, workingState))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.writeChange(ctx, nil, ""))

	diags = diags.Append(updateStateHook(ctx))
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
	diffApply, err := n.readDiff(ctx, providerSchema)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// We don't want to do any destroys
	// (these are handled by NodeDestroyResourceInstance instead)
	if diffApply == nil || diffApply.Action == plans.Delete {
		return diags
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
	diff, err := n.readDiff(ctx, providerSchema)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// Make a new diff, in case we've learned new values in the state
	// during apply which we can now incorporate.
	diffApply, _, planDiags := n.plan(ctx, diff, state, false, n.forceReplace)
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
		return diags
	}

	// Compare the diffs
	diags = diags.Append(n.checkPlannedChange(ctx, diff, diffApply, providerSchema))
	if diags.HasErrors() {
		return diags
	}

	state, readDiags = n.readResourceInstanceState(ctx, n.ResourceInstanceAddr())
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return diags
	}

	diffApply = reducePlan(addr, diffApply, false)
	// reducePlan may have simplified our planned change
	// into a NoOp if it only requires destroying, since destroying
	// is handled by NodeDestroyResourceInstance.
	if diffApply == nil || diffApply.Action == plans.NoOp {
		return diags
	}

	diags = diags.Append(n.preApplyHook(ctx, diffApply))
	if diags.HasErrors() {
		return diags
	}

	state, applyDiags := n.apply(ctx, state, diffApply, n.Config, n.CreateBeforeDestroy())
	diags = diags.Append(applyDiags)

	// We clear the change out here so that future nodes don't see a change
	// that is already complete.
	err = n.writeChange(ctx, nil, "")
	if err != nil {
		return diags.Append(err)
	}

	state = maybeTainted(addr.Absolute(ctx.Path()), state, diffApply, diags.Err())

	if state != nil {
		// dependencies are always updated to match the configuration during apply
		state.Dependencies = n.Dependencies
	}
	err = n.writeResourceInstanceState(ctx, state, workingState)
	if err != nil {
		return diags.Append(err)
	}

	// Run Provisioners
	createNew := (diffApply.Action == plans.Create || diffApply.Action.IsReplace())
	applyProvisionersDiags := n.evalApplyProvisioners(ctx, state, createNew, configs.ProvisionerWhenCreate)
	// the provisioner errors count as port of the apply error, so we can bundle the diags
	diags = diags.Append(applyProvisionersDiags)

	state = maybeTainted(addr.Absolute(ctx.Path()), state, diffApply, diags.Err())

	err = n.writeResourceInstanceState(ctx, state, workingState)
	if err != nil {
		return diags.Append(err)
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
	return diags
}

// checkPlannedChange produces errors if the _actual_ expected value is not
// compatible with what was recorded in the plan.
//
// Errors here are most often indicative of a bug in the provider, so our error
// messages will report with that in mind. It's also possible that there's a bug
// in Terraform's Core's own "proposed new value" code in EvalDiff.
func (n *NodeApplyableResourceInstance) checkPlannedChange(ctx EvalContext, plannedChange, actualChange *plans.ResourceInstanceChange, providerSchema *ProviderSchema) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	addr := n.ResourceInstanceAddr().Resource

	schema, _ := providerSchema.SchemaForResourceAddr(addr.ContainingResource())
	if schema == nil {
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

	errs := objchange.AssertObjectCompatible(schema, plannedChange.After, actualChange.After)
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

// maybeTainted takes the resource addr, new value, planned change, and possible
// error from an apply operation and return a new instance object marked as
// tainted if it appears that a create operation has failed.
func maybeTainted(addr addrs.AbsResourceInstance, state *states.ResourceInstanceObject, change *plans.ResourceInstanceChange, err error) *states.ResourceInstanceObject {
	if state == nil || change == nil || err == nil {
		return state
	}
	if state.Status == states.ObjectTainted {
		log.Printf("[TRACE] maybeTainted: %s was already tainted, so nothing to do", addr)
		return state
	}
	if change.Action == plans.Create {
		// If there are errors during a _create_ then the object is
		// in an undefined state, and so we'll mark it as tainted so
		// we can try again on the next run.
		//
		// We don't do this for other change actions because errors
		// during updates will often not change the remote object at all.
		// If there _were_ changes prior to the error, it's the provider's
		// responsibility to record the effect of those changes in the
		// object value it returned.
		log.Printf("[TRACE] maybeTainted: %s encountered an error during creation, so it is now marked as tainted", addr)
		return state.AsTainted()
	}
	return state
}
