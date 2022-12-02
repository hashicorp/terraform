package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
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
		// If there is no config, and there is no change, then we have nothing
		// to do and the change was left in the plan for informational
		// purposes only.
		changes := ctx.Changes()
		csrc := changes.GetResourceInstanceChange(n.ResourceInstanceAddr(), states.CurrentGen)
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
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodeApplyableResourceInstance) dataResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {

	// smokeTest will be non-nil if and only if this data resource belongs to
	// a smoke_test block. If so, that activates some special behavior:
	// - The plan phase would've forced deferring the read to the apply step,
	//   so we'll always end up in here.
	// - If the provider returns any errors when we ask it to read the data
	//   source, we force them to be treated as warnings instead so that
	//   the apply step can still run to completion.
	// - We'll report the success or failure of the read as a check condition
	//   status in the checks state, so that it'll contribute to the final
	//   status of this smoke test.
	smokeTest := n.Config.SmokeTest
	reportResult := func(diags tfdiags.Diagnostics, checkStatusForErrors checks.Status) tfdiags.Diagnostics {
		// Just a direct passthrough unless we're dealing with a smoke test.
		return diags
	}
	if smokeTest != nil {
		smokeTestAddr := addrs.SmokeTest{Name: smokeTest.Name}.Absolute(n.Addr.Module)
		// If we belong to a smoke test then we should be able to find this
		// data resource as one of the entries in the smoke test's data
		// resources, and its index will be our condition index.
		// FIXME: This is a pretty clumsy and inefficient way to determine this,
		// though it'll be okay as long as smoke tests only have a reasonable
		// number of data resources.
		conditionIdx := -1 // -1 means "none"
		for i, found := range smokeTest.DataResources {
			if found == n.Config {
				conditionIdx = i
				break
			}
		}
		if conditionIdx == -1 {
			// If we get here then the configuration decoder did something
			// wrong, because there should always be a back-reference from
			// the smoke test to the data resource.
			panic(fmt.Sprintf("data resource %s is not tracked by its containing smoke_test %q", n.Addr, smokeTest.Name))
		}
		// We have a more elaborate reportResult implementation when we're in
		// smoke testing mode.
		checkState := ctx.Checks()
		reportResult = func(diags tfdiags.Diagnostics, checkStatusForErrors checks.Status) tfdiags.Diagnostics {
			if !diags.HasErrors() {
				checkState.ReportCheckResult(smokeTestAddr, addrs.SmokeTestDataResource, conditionIdx, checks.StatusPass)
				return diags
			}
			if checkStatusForErrors == checks.StatusFail {
				// If this is a situation where a read error signals check
				// failure rather than check error then we need to downgrade
				// all of the error diagnostics into warning diagnostics.
				diags = tfdiags.WithErrorsAsWarnings(diags)
				checkState.ReportCheckFailure(smokeTestAddr, addrs.SmokeTestDataResource, conditionIdx, fmt.Sprintf("Failed to read %s.", n.Addr.Resource.String()))
				return diags
			}
			// For any other status we'll just report it and return the
			// diagnostics. checks.StatusError is the only sensible status for
			// us to use here.
			checkState.ReportCheckResult(smokeTestAddr, addrs.SmokeTestDataResource, conditionIdx, checkStatusForErrors)
			return diags
		}

		// If this smoke test also has preconditions then one of them might
		// have failed already, in which case we won't read the data source
		// at all. This is safe because only other data resources and
		// postconditions from the same smoke test can refer to this and
		// both of those will be inhibited by this existing failure.
		precondStatus := checkState.ObjectCheckStatusByConditionType(smokeTestAddr, addrs.SmokeTestPrecondition)
		if precondStatus == checks.StatusFail || precondStatus == checks.StatusError {
			// If it's already failing then it can't get any more successful.
			log.Printf("[TRACE] nodeSmokeTestPost: Skipping %s because %s preconditions failed", n.Addr, smokeTestAddr)
			return reportResult(diags, checks.StatusError)
		}
	}

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return reportResult(diags, checks.StatusError)
	}

	change, err := n.readDiff(ctx, providerSchema)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return reportResult(diags, checks.StatusError)
	}
	// Stop early if we don't actually have a diff
	if change == nil {
		return reportResult(diags, checks.StatusError)
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
		return reportResult(diags, checks.StatusFail)
	}

	if state != nil {
		// If n.applyDataSource returned a nil state object with no accompanying
		// errors then it determined that the given change doesn't require
		// actually reading the data (e.g. because it was already read during
		// the plan phase) and so we're only running through here to get the
		// extra details like precondition/postcondition checks.
		diags = diags.Append(n.writeResourceInstanceState(ctx, state, workingState))
		if diags.HasErrors() {
			return reportResult(diags, checks.StatusError)
		}
	}

	diags = diags.Append(n.writeChange(ctx, nil, ""))

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

	return reportResult(diags, checks.StatusError)
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
	diff, err := n.readDiff(ctx, providerSchema)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// Make a new diff, in case we've learned new values in the state
	// during apply which we can now incorporate.
	diffApply, _, repeatData, planDiags := n.plan(ctx, diff, state, false, n.forceReplace)
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
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
	// re-write the state, but we do need to re-evaluate postconditions.
	if diffApply.Action == plans.NoOp {
		return diags.Append(n.managedResourcePostconditions(ctx, repeatData))
	}

	state, applyDiags := n.apply(ctx, state, diffApply, n.Config, repeatData, n.CreateBeforeDestroy())
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

	// Post-conditions might block further progress. We intentionally do this
	// _after_ writing the state because we want to check against
	// the result of the operation, and to fail on future operations
	// until the user makes the condition succeed.
	return diags.Append(n.managedResourcePostconditions(ctx, repeatData))
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
