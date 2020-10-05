package terraform

import (
	"fmt"

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
func (n *NodeApplyableResourceInstance) Execute(ctx EvalContext, op walkOperation) error {
	addr := n.ResourceInstanceAddr()

	if n.Config == nil {
		// This should not be possible, but we've got here in at least one
		// case as discussed in the following issue:
		//    https://github.com/hashicorp/terraform/issues/21258
		// To avoid an outright crash here, we'll instead return an explicit
		// error.
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Resource node has no configuration attached",
			fmt.Sprintf(
				"The graph node for %s has no configuration attached to it. This suggests a bug in Terraform's apply graph builder; please report it!",
				addr,
			),
		))
		return diags.Err()
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

func (n *NodeApplyableResourceInstance) dataResourceExecute(ctx EvalContext) error {
	addr := n.ResourceInstanceAddr().Resource

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	change, err := n.readDiff(ctx, providerSchema)
	if err != nil {
		return err
	}
	// Stop early if we don't actually have a diff
	if change == nil {
		return EvalEarlyExitError{}
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
	_, err = readDataApply.Eval(ctx)
	if err != nil {
		return err
	}

	writeState := &EvalWriteState{
		Addr:           addr,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		State:          &state,
	}
	_, err = writeState.Eval(ctx)
	if err != nil {
		return err
	}

	writeDiff := &EvalWriteDiff{
		Addr:           addr,
		ProviderSchema: &providerSchema,
		Change:         nil,
	}
	_, err = writeDiff.Eval(ctx)
	if err != nil {
		return err
	}

	UpdateStateHook(ctx)
	return nil
}

func (n *NodeApplyableResourceInstance) managedResourceExecute(ctx EvalContext) error {
	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var state *states.ResourceInstanceObject
	var createNew bool
	var createBeforeDestroyEnabled bool
	var deposedKey states.DeposedKey

	addr := n.ResourceInstanceAddr().Resource
	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	// Get the saved diff for apply
	diffApply, err := n.readDiff(ctx, providerSchema)
	if err != nil {
		return err
	}

	// We don't want to do any destroys
	// (these are handled by NodeDestroyResourceInstance instead)
	if diffApply == nil || diffApply.Action == plans.Delete {
		return EvalEarlyExitError{}
	}

	destroy := (diffApply.Action == plans.Delete || diffApply.Action.IsReplace())
	// Get the stored action for CBD if we have a plan already
	createBeforeDestroyEnabled = diffApply.Change.Action == plans.CreateThenDelete

	if destroy && n.CreateBeforeDestroy() {
		createBeforeDestroyEnabled = true
	}

	if createBeforeDestroyEnabled {
		deposeState := &EvalDeposeState{
			Addr:      addr,
			ForceKey:  n.PreallocatedDeposedKey,
			OutputKey: &deposedKey,
		}
		_, err = deposeState.Eval(ctx)
		if err != nil {
			return err
		}
	}

	readState := &EvalReadState{
		Addr:           addr,
		Provider:       &provider,
		ProviderSchema: &providerSchema,

		Output: &state,
	}
	_, err = readState.Eval(ctx)
	if err != nil {
		return err
	}

	// Get the saved diff
	diff, err := n.readDiff(ctx, providerSchema)
	if err != nil {
		return err
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
	_, err = evalDiff.Eval(ctx)
	if err != nil {
		return err
	}

	// Compare the diffs
	checkPlannedChange := &EvalCheckPlannedChange{
		Addr:           addr,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		Planned:        &diff,
		Actual:         &diffApply,
	}
	_, err = checkPlannedChange.Eval(ctx)
	if err != nil {
		return err
	}

	readState = &EvalReadState{
		Addr:           addr,
		Provider:       &provider,
		ProviderSchema: &providerSchema,

		Output: &state,
	}
	_, err = readState.Eval(ctx)
	if err != nil {
		return err
	}

	reduceDiff := &EvalReduceDiff{
		Addr:      addr,
		InChange:  &diffApply,
		Destroy:   false,
		OutChange: &diffApply,
	}
	_, err = reduceDiff.Eval(ctx)
	if err != nil {
		return err
	}

	// EvalReduceDiff may have simplified our planned change
	// into a NoOp if it only requires destroying, since destroying
	// is handled by NodeDestroyResourceInstance.
	if diffApply == nil || diffApply.Action == plans.NoOp {
		return EvalEarlyExitError{}
	}

	evalApplyPre := &EvalApplyPre{
		Addr:   addr,
		State:  &state,
		Change: &diffApply,
	}
	_, err = evalApplyPre.Eval(ctx)
	if err != nil {
		return err
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
	_, err = evalApply.Eval(ctx)
	if err != nil {
		return err
	}

	evalMaybeTainted := &EvalMaybeTainted{
		Addr:   addr,
		State:  &state,
		Change: &diffApply,
		Error:  &applyError,
	}
	_, err = evalMaybeTainted.Eval(ctx)
	if err != nil {
		return err
	}

	writeState := &EvalWriteState{
		Addr:           addr,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		State:          &state,
		Dependencies:   &n.Dependencies,
	}
	_, err = writeState.Eval(ctx)
	if err != nil {
		return err
	}

	applyProvisioners := &EvalApplyProvisioners{
		Addr:           addr,
		State:          &state, // EvalApplyProvisioners will skip if already tainted
		ResourceConfig: n.Config,
		CreateNew:      &createNew,
		Error:          &applyError,
		When:           configs.ProvisionerWhenCreate,
	}
	_, err = applyProvisioners.Eval(ctx)
	if err != nil {
		return err
	}

	evalMaybeTainted = &EvalMaybeTainted{
		Addr:   addr,
		State:  &state,
		Change: &diffApply,
		Error:  &applyError,
	}
	_, err = evalMaybeTainted.Eval(ctx)
	if err != nil {
		return err
	}

	writeState = &EvalWriteState{
		Addr:           addr,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		State:          &state,
		Dependencies:   &n.Dependencies,
	}
	_, err = writeState.Eval(ctx)
	if err != nil {
		return err
	}

	if createBeforeDestroyEnabled && applyError != nil {
		maybeRestoreDesposedObject := &EvalMaybeRestoreDeposedObject{
			Addr:          addr,
			PlannedChange: &diffApply,
			Key:           &deposedKey,
		}
		_, err := maybeRestoreDesposedObject.Eval(ctx)
		if err != nil {
			return err
		}
	}

	// We clear the diff out here so that future nodes don't see a diff that is
	// already complete. There is no longer a diff!
	if !diff.Action.IsReplace() || !n.CreateBeforeDestroy() {
		writeDiff := &EvalWriteDiff{
			Addr:           addr,
			ProviderSchema: &providerSchema,
			Change:         nil,
		}
		_, err := writeDiff.Eval(ctx)
		if err != nil {
			return err
		}
	}

	applyPost := &EvalApplyPost{
		Addr:  addr,
		State: &state,
		Error: &applyError,
	}
	_, err = applyPost.Eval(ctx)
	if err != nil {
		return err
	}

	UpdateStateHook(ctx)
	return nil
}
