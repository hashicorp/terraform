package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
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
	addr := n.ResourceInstanceAddr().Resource

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
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

	// In this particular call to EvalReadData we include our planned
	// change, which signals that we expect this read to complete fully
	// with no unknown values; it'll produce an error if not.
	var state *states.ResourceInstanceObject
	readDataApply := &evalReadDataApply{
		evalReadData{
			Addr:           addr,
			Config:         n.Config,
			Planned:        &change,
			Provider:       &provider,
			ProviderAddr:   n.ResolvedProvider,
			ProviderMetas:  n.ProviderMetas,
			ProviderSchema: &providerSchema,
			State:          &state,
		},
	}
	diags = diags.Append(readDataApply.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	// We don't write dependencies for datasources
	diags = diags.Append(n.writeResourceInstanceState(ctx, state, nil, workingState))
	if diags.HasErrors() {
		return diags
	}

	writeDiff := &EvalWriteDiff{
		Addr:           addr,
		ProviderSchema: &providerSchema,
		Change:         nil,
	}
	diags = diags.Append(writeDiff.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(UpdateStateHook(ctx))
	return diags
}

func (n *NodeApplyableResourceInstance) managedResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var state *states.ResourceInstanceObject
	var createNew bool
	var createBeforeDestroyEnabled bool
	var deposedKey states.DeposedKey

	addr := n.ResourceInstanceAddr().Resource
	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
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

	state, err = n.ReadResourceInstanceState(ctx, n.ResourceInstanceAddr())
	diags = diags.Append(err)
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
	evalDiff := &EvalDiff{
		Addr:           addr,
		Config:         n.Config,
		Provider:       &provider,
		ProviderAddr:   n.ResolvedProvider,
		ProviderMetas:  n.ProviderMetas,
		ProviderSchema: &providerSchema,
		State:          &state,
		PreviousDiff:   &diff,
		OutputChange:   &diffApply,
		OutputState:    &state,
	}
	diags = diags.Append(evalDiff.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	// Compare the diffs
	checkPlannedChange := &EvalCheckPlannedChange{
		Addr:           addr,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		Planned:        &diff,
		Actual:         &diffApply,
	}
	diags = diags.Append(checkPlannedChange.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	state, err = n.ReadResourceInstanceState(ctx, n.ResourceInstanceAddr())
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	reduceDiff := &EvalReduceDiff{
		Addr:      addr,
		InChange:  &diffApply,
		Destroy:   false,
		OutChange: &diffApply,
	}
	diags = diags.Append(reduceDiff.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	// EvalReduceDiff may have simplified our planned change
	// into a NoOp if it only requires destroying, since destroying
	// is handled by NodeDestroyResourceInstance.
	if diffApply == nil || diffApply.Action == plans.NoOp {
		return diags
	}

	diags = diags.Append(n.PreApplyHook(ctx, diffApply))
	if diags.HasErrors() {
		return diags
	}

	var applyError error
	evalApply := &EvalApply{
		Addr:                addr,
		Config:              n.Config,
		State:               &state,
		Change:              &diffApply,
		Provider:            &provider,
		ProviderAddr:        n.ResolvedProvider,
		ProviderMetas:       n.ProviderMetas,
		ProviderSchema:      &providerSchema,
		Output:              &state,
		Error:               &applyError,
		CreateNew:           &createNew,
		CreateBeforeDestroy: n.CreateBeforeDestroy(),
	}
	diags = diags.Append(evalApply.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	// We clear the change out here so that future nodes don't see a change
	// that is already complete.
	writeDiff := &EvalWriteDiff{
		Addr:           addr,
		ProviderSchema: &providerSchema,
		Change:         nil,
	}
	diags = diags.Append(writeDiff.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	evalMaybeTainted := &EvalMaybeTainted{
		Addr:   addr,
		State:  &state,
		Change: &diffApply,
		Error:  &applyError,
	}
	diags = diags.Append(evalMaybeTainted.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.writeResourceInstanceState(ctx, state, n.Dependencies, workingState))
	if diags.HasErrors() {
		return diags
	}

	applyProvisioners := &EvalApplyProvisioners{
		Addr:           addr,
		State:          &state, // EvalApplyProvisioners will skip if already tainted
		ResourceConfig: n.Config,
		CreateNew:      &createNew,
		Error:          &applyError,
		When:           configs.ProvisionerWhenCreate,
	}
	diags = diags.Append(applyProvisioners.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	evalMaybeTainted = &EvalMaybeTainted{
		Addr:   addr,
		State:  &state,
		Change: &diffApply,
		Error:  &applyError,
	}
	diags = diags.Append(evalMaybeTainted.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.writeResourceInstanceState(ctx, state, n.Dependencies, workingState))
	if diags.HasErrors() {
		return diags
	}

	if createBeforeDestroyEnabled && applyError != nil {
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
				log.Printf("[TRACE] EvalMaybeRestoreDeposedObject: %s deposed object %s was restored as the current object", addr, deposedKey)
			} else {
				log.Printf("[TRACE] EvalMaybeRestoreDeposedObject: %s deposed object %s remains deposed", addr, deposedKey)
			}
		}
	}
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.PostApplyHook(ctx, state, &applyError))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(UpdateStateHook(ctx))
	return diags
}
