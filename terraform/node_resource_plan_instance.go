package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/addrs"
)

// NodePlannableResourceInstance represents a _single_ resource
// instance that is plannable. This means this represents a single
// count index, for example.
type NodePlannableResourceInstance struct {
	*NodeAbstractResourceInstance
	ForceCreateBeforeDestroy bool
	skipRefresh              bool
}

var (
	_ GraphNodeModuleInstance       = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeReferencer           = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeConfigResource       = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeResourceInstance     = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeAttachResourceState  = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeExecutable           = (*NodePlannableResourceInstance)(nil)
)

// GraphNodeEvalable
func (n *NodePlannableResourceInstance) Execute(ctx EvalContext, op walkOperation) error {
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

func (n *NodePlannableResourceInstance) dataResourceExecute(ctx EvalContext) error {
	var diags tfdiags.Diagnostics
	config := n.Config
	addr := n.ResourceInstanceAddr()

	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	state, err = n.ReadResourceInstanceState(ctx, addr)
	if err != nil {
		return err
	}

	validateSelfRef := &EvalValidateSelfRef{
		Addr:           addr.Resource,
		Config:         config.Config,
		ProviderSchema: &providerSchema,
	}
	diags = validateSelfRef.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	readDataPlan := &evalReadDataPlan{
		evalReadData: evalReadData{
			Addr:           addr.Resource,
			Config:         n.Config,
			Provider:       &provider,
			ProviderAddr:   n.ResolvedProvider,
			ProviderMetas:  n.ProviderMetas,
			ProviderSchema: &providerSchema,
			OutputChange:   &change,
			State:          &state,
			dependsOn:      n.dependsOn,
		},
	}
	diags = readDataPlan.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	// write the data source into both the refresh state and the
	// working state
	writeRefreshState := &EvalWriteState{
		Addr:           addr.Resource,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		State:          &state,
		targetState:    refreshState,
	}
	diags = writeRefreshState.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	writeState := &EvalWriteState{
		Addr:           addr.Resource,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		State:          &state,
	}
	diags = writeState.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	writeDiff := &EvalWriteDiff{
		Addr:           addr.Resource,
		ProviderSchema: &providerSchema,
		Change:         &change,
	}
	diags = writeDiff.Eval(ctx)
	return diags.ErrWithWarnings()
}

func (n *NodePlannableResourceInstance) managedResourceExecute(ctx EvalContext) error {
	var diags tfdiags.Diagnostics
	config := n.Config
	addr := n.ResourceInstanceAddr()

	var change *plans.ResourceInstanceChange
	var instanceRefreshState *states.ResourceInstanceObject
	var instancePlanState *states.ResourceInstanceObject

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	validateSelfRef := &EvalValidateSelfRef{
		Addr:           addr.Resource,
		Config:         config.Config,
		ProviderSchema: &providerSchema,
	}
	diags = validateSelfRef.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	instanceRefreshState, err = n.ReadResourceInstanceState(ctx, addr)
	if err != nil {
		return err
	}
	refreshLifecycle := &EvalRefreshLifecycle{
		Addr:                     addr,
		Config:                   n.Config,
		State:                    &instanceRefreshState,
		ForceCreateBeforeDestroy: n.ForceCreateBeforeDestroy,
	}
	diags = refreshLifecycle.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	// Refresh, maybe
	if !n.skipRefresh {
		refresh := &EvalRefresh{
			Addr:           addr.Resource,
			ProviderAddr:   n.ResolvedProvider,
			Provider:       &provider,
			ProviderMetas:  n.ProviderMetas,
			ProviderSchema: &providerSchema,
			State:          &instanceRefreshState,
			Output:         &instanceRefreshState,
		}
		diags := refresh.Eval(ctx)
		if diags.HasErrors() {
			return diags.ErrWithWarnings()
		}

		writeRefreshState := &EvalWriteState{
			Addr:           addr.Resource,
			ProviderAddr:   n.ResolvedProvider,
			ProviderSchema: &providerSchema,
			State:          &instanceRefreshState,
			targetState:    refreshState,
			Dependencies:   &n.Dependencies,
		}
		diags = writeRefreshState.Eval(ctx)
		if diags.HasErrors() {
			return diags.ErrWithWarnings()
		}
	}

	// Plan the instance
	diff := &EvalDiff{
		Addr:                addr.Resource,
		Config:              n.Config,
		CreateBeforeDestroy: n.ForceCreateBeforeDestroy,
		Provider:            &provider,
		ProviderAddr:        n.ResolvedProvider,
		ProviderMetas:       n.ProviderMetas,
		ProviderSchema:      &providerSchema,
		State:               &instanceRefreshState,
		OutputChange:        &change,
		OutputState:         &instancePlanState,
	}
	diags = diff.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	err = n.checkPreventDestroy(change)
	if err != nil {
		return err
	}

	writeState := &EvalWriteState{
		Addr:           addr.Resource,
		ProviderAddr:   n.ResolvedProvider,
		State:          &instancePlanState,
		ProviderSchema: &providerSchema,
	}
	diags = writeState.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	writeDiff := &EvalWriteDiff{
		Addr:           addr.Resource,
		ProviderSchema: &providerSchema,
		Change:         &change,
	}
	diags = writeDiff.Eval(ctx)
	return diags.ErrWithWarnings()
}
