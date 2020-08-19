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

	destroyNode      GraphNodeDestroyerCBD
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

// GraphNodeAttachDestroyer
func (n *NodeApplyableResourceInstance) AttachDestroyNode(d GraphNodeDestroyerCBD) {
	n.destroyNode = d
}

// CreateBeforeDestroy checks this nodes config status and the status af any
// companion destroy node for CreateBeforeDestroy.
func (n *NodeApplyableResourceInstance) CreateBeforeDestroy() bool {
	if n.ForceCreateBeforeDestroy {
		return n.ForceCreateBeforeDestroy
	}

	if n.Config != nil && n.Config.Managed != nil {
		return n.Config.Managed.CreateBeforeDestroy
	}

	if n.destroyNode != nil {
		return n.destroyNode.CreateBeforeDestroy()
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

func (n *NodeApplyableResourceInstance) Execute(ctx EvalContext) tfdiags.Diagnostics {
	addr := n.ResourceInstanceAddr()
	var diags tfdiags.Diagnostics

	if n.Config == nil {
		// This should not be possible, but we've got here in at least one
		// case as discussed in the following issue:
		//    https://github.com/hashicorp/terraform/issues/21258
		// To avoid an outright crash here, we'll instead return an explicit
		// error.
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Resource node has no configuration attached",
			fmt.Sprintf(
				"The graph node for %s has no configuration attached to it. This suggests a bug in Terraform's apply graph builder; please report it!",
				addr,
			),
		))
	}

	// Eval info is different depending on what kind of resource this is
	var err error
	switch n.Config.Mode {
	case addrs.ManagedResourceMode:
		err = n.execManagedResource(ctx, addr)
	case addrs.DataResourceMode:
		err = n.execDataResource(ctx, addr)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}

	if err != nil {
		if _, isEarlyExit := err.(EvalEarlyExitError); isEarlyExit {
			// In this path we abort early, losing any non-error
			// diagnostics we saw earlier.
			// var retDiags tfdiags.Diagnostics
			// return retDiags.Append(err)
			//
			// Eval.go swallows early exit errors, so for the nonce i'll do the same here
			return nil
		}
	}
	return diags.Append(err)
}

func (n *NodeApplyableResourceInstance) execDataResource(ctx EvalContext, addr addrs.AbsResourceInstance) error {
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	// Get the saved diff for apply
	readDiff := &EvalReadDiff{
		Addr:           addr.Resource,
		ProviderSchema: &providerSchema,
		Change:         &change,
	}
	_, err = readDiff.Eval(ctx)
	if err != nil {
		return err
	}

	// EvalIf{}
	if change == nil {
		return EvalEarlyExitError{}
	}
	// Q: Do we need to do anything with this, or is early exit sufficient?
	// Then: EvalNoop{} ...

	// In this particular call to EvalReadData we include our planned
	// change, which signals that we expect this read to complete fully
	// with no unknown values; it'll produce an error if not.
	evalRDA := &evalReadDataApply{
		evalReadData{
			Addr:           addr.Resource,
			Config:         n.Config,
			Planned:        &change,
			Provider:       &provider,
			ProviderAddr:   n.ResolvedProvider,
			ProviderMetas:  n.ProviderMetas,
			ProviderSchema: &providerSchema,
			State:          &state,
		},
	}
	_, err = evalRDA.Eval(ctx)
	if err != nil {
		return err
	}

	req := EvalWriteState{
		Addr:           addr.Resource,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		State:          &state,
		Dependencies:   &n.Dependencies,
	}
	_, err = ExecWriteState(ctx, req)
	if err != nil {
		return err
	}

	writeDiff := &EvalReadDiff{
		Addr:           addr.Resource,
		ProviderSchema: &providerSchema,
		Change:         &change,
	}
	_, err = writeDiff.Eval(ctx)
	if err != nil {
		return err
	}

	hook := &EvalUpdateStateHook{}
	_, err = hook.Eval(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (n *NodeApplyableResourceInstance) execManagedResource(ctx EvalContext, addr addrs.AbsResourceInstance) error {
	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var diff, diffApply *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject
	var applyErr error // the err from EvalApply must be passed to EvalMaybeTainted{}
	var createNew bool
	var createBeforeDestroyEnabled bool
	var deposedKey states.DeposedKey

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	// Get the saved diff for apply
	readDiff := &EvalReadDiff{
		Addr:           addr.Resource,
		ProviderSchema: &providerSchema,
		Change:         &diffApply,
	}
	_, err = readDiff.Eval(ctx)
	if err != nil {
		return err
	}

	// First EvalIf
	// We don't want to do any destroys
	// (these are handled by NodeDestroyResourceInstance instead)
	if diffApply == nil {
		return EvalEarlyExitError{}
	}
	if diffApply.Action == plans.Delete {
		return EvalEarlyExitError{}
	}

	// Second EvalIf
	destroy := false
	if diffApply != nil {
		destroy = (diffApply.Action == plans.Delete || diffApply.Action.IsReplace())
	}
	if destroy && n.CreateBeforeDestroy() {
		createBeforeDestroyEnabled = true
		evalDepose := &EvalDeposeState{
			Addr:      addr.Resource,
			ForceKey:  n.PreallocatedDeposedKey,
			OutputKey: &deposedKey,
		}
		_, err = evalDepose.Eval(ctx)
		if err != nil {
			return err
		}
	}

	evalReadState := &EvalReadState{
		Addr:           addr.Resource,
		Provider:       &provider,
		ProviderSchema: &providerSchema,

		Output: &state,
	}
	_, err = evalReadState.Eval(ctx)
	if err != nil {
		return err
	}

	// Get the saved diff
	readDiff = &EvalReadDiff{
		Addr:           addr.Resource,
		ProviderSchema: &providerSchema,
		Change:         &diff,
	}
	_, err = readDiff.Eval(ctx)
	if err != nil {
		return err
	}

	// Make a new diff, in case we've learned new values in the state
	// during apply which we can now incorporate.
	evalDiff := &EvalDiff{
		Addr:           addr.Resource,
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
	evalCheckPlannedChange := &EvalCheckPlannedChange{
		Addr:           addr.Resource,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		Planned:        &diff,
		Actual:         &diffApply,
	}
	_, err = evalCheckPlannedChange.Eval(ctx)
	if err != nil {
		return err
	}

	evalReadState = &EvalReadState{
		Addr:           addr.Resource,
		Provider:       &provider,
		ProviderSchema: &providerSchema,

		Output: &state,
	}
	_, err = evalReadState.Eval(ctx)
	if err != nil {
		return err
	}

	erd := &EvalReduceDiff{
		Addr:      addr.Resource,
		InChange:  &diffApply,
		Destroy:   false,
		OutChange: &diffApply,
	}
	_, err = erd.Eval(ctx)
	if err != nil {
		return err
	}

	// EvalReduceDiff may have simplified our planned change
	// into a NoOp if it only requires destroying, since destroying
	// is handled by NodeDestroyResourceInstance.
	if diffApply == nil || diffApply.Action == plans.NoOp {
		return EvalEarlyExitError{}
	}

	// Call pre-apply hook
	preApply := &EvalApplyPre{
		Addr:   addr.Resource,
		State:  &state,
		Change: &diffApply,
	}
	_, err = preApply.Eval(ctx)
	if err != nil {
		return err
	}

	apply := &EvalApply{
		Addr:                addr.Resource,
		Config:              n.Config,
		State:               &state,
		Change:              &diffApply,
		Provider:            &provider,
		ProviderAddr:        n.ResolvedProvider,
		ProviderMetas:       n.ProviderMetas,
		ProviderSchema:      &providerSchema,
		Output:              &state,
		Error:               &applyErr,
		CreateNew:           &createNew,
		CreateBeforeDestroy: n.CreateBeforeDestroy(),
	}
	_, err = apply.Eval(ctx)
	if err != nil {
		return err
	}

	evalTainted := &EvalMaybeTainted{
		Addr:   addr.Resource,
		State:  &state,
		Change: &diffApply,
		Error:  &applyErr,
	}
	_, err = evalTainted.Eval(ctx)
	if err != nil {
		return err
	}

	req := EvalWriteState{
		Addr:           addr.Resource,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		State:          &state,
		Dependencies:   &n.Dependencies,
	}
	state, err = ExecWriteState(ctx, req)
	if err != nil {
		return err
	}

	applyProvisioners := &EvalApplyProvisioners{
		Addr:           addr.Resource,
		State:          &state, // EvalApplyProvisioners will skip if already tainted
		ResourceConfig: n.Config,
		CreateNew:      &createNew,
		Error:          &applyErr,
		When:           configs.ProvisionerWhenCreate,
	}
	_, err = applyProvisioners.Eval(ctx)
	if err != nil {
		return err
	}
	// Check if the provisioning step failed & left a tainted resource
	evalTainted = &EvalMaybeTainted{
		Addr:   addr.Resource,
		State:  &state,
		Change: &diffApply,
		Error:  &applyErr,
	}
	_, err = evalTainted.Eval(ctx)
	if err != nil {
		return err
	}

	req = EvalWriteState{
		Addr:           addr.Resource,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		State:          &state,
		Dependencies:   &n.Dependencies,
	}
	state, err = ExecWriteState(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("createBeforeDestroyEnabled: %#v\n", createBeforeDestroyEnabled)
	fmt.Printf("applyErr: %#v\n", applyErr)

	if createBeforeDestroyEnabled && applyErr != nil {
		emrdo := &EvalMaybeRestoreDeposedObject{
			Addr:          addr.Resource,
			PlannedChange: &diffApply,
			Key:           &deposedKey,
		}
		_, err = emrdo.Eval(ctx)
		if err != nil {
			return err
		}
	}

	// We clear the diff out here so that future nodes
	// don't see a diff that is already complete. There
	// is no longer a diff!
	if !diff.Action.IsReplace() || !n.CreateBeforeDestroy() {
		evalWriteDiff := &EvalWriteDiff{
			Addr:           addr.Resource,
			ProviderSchema: &providerSchema,
			Change:         nil,
		}
		_, err := evalWriteDiff.Eval(ctx)
		if err != nil {
			return err
		}
	}

	postApply := &EvalApplyPost{
		Addr:  addr.Resource,
		State: &state,
		Error: &applyErr,
	}
	_, err = postApply.Eval(ctx)
	if err != nil {
		return err
	}

	hook := &EvalUpdateStateHook{}
	_, err = hook.Eval(ctx)
	if err != nil {
		return err
	}

	return nil
}
