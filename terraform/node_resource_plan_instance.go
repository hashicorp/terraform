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
func (n *NodePlannableResourceInstance) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
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

func (n *NodePlannableResourceInstance) dataResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	config := n.Config
	addr := n.ResourceInstanceAddr()

	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	state, err = n.ReadResourceInstanceState(ctx, addr)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	validateSelfRef := &EvalValidateSelfRef{
		Addr:           addr.Resource,
		Config:         config.Config,
		ProviderSchema: &providerSchema,
	}
	diags = diags.Append(validateSelfRef.Eval(ctx))
	if diags.HasErrors() {
		return diags
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
	diags = diags.Append(readDataPlan.Eval(ctx))
	if diags.HasErrors() {
		return diags
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
	diags = diags.Append(writeRefreshState.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	writeState := &EvalWriteState{
		Addr:           addr.Resource,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		State:          &state,
	}
	diags = diags.Append(writeState.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	writeDiff := &EvalWriteDiff{
		Addr:           addr.Resource,
		ProviderSchema: &providerSchema,
		Change:         &change,
	}
	diags = diags.Append(writeDiff.Eval(ctx))
	return diags
}

func (n *NodePlannableResourceInstance) managedResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	config := n.Config
	addr := n.ResourceInstanceAddr()

	var change *plans.ResourceInstanceChange
	var instanceRefreshState *states.ResourceInstanceObject
	var instancePlanState *states.ResourceInstanceObject

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	validateSelfRef := &EvalValidateSelfRef{
		Addr:           addr.Resource,
		Config:         config.Config,
		ProviderSchema: &providerSchema,
	}
	diags = diags.Append(validateSelfRef.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	instanceRefreshState, err = n.ReadResourceInstanceState(ctx, addr)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}
	refreshLifecycle := &EvalRefreshLifecycle{
		Addr:                     addr,
		Config:                   n.Config,
		State:                    &instanceRefreshState,
		ForceCreateBeforeDestroy: n.ForceCreateBeforeDestroy,
	}
	diags = diags.Append(refreshLifecycle.Eval(ctx))
	if diags.HasErrors() {
		return diags
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
		diags := diags.Append(refresh.Eval(ctx))
		if diags.HasErrors() {
			return diags
		}

		writeRefreshState := &EvalWriteState{
			Addr:           addr.Resource,
			ProviderAddr:   n.ResolvedProvider,
			ProviderSchema: &providerSchema,
			State:          &instanceRefreshState,
			targetState:    refreshState,
			Dependencies:   &n.Dependencies,
		}
		diags = diags.Append(writeRefreshState.Eval(ctx))
		if diags.HasErrors() {
			return diags
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
	diags = diags.Append(diff.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.checkPreventDestroy(change))
	if diags.HasErrors() {
		return diags
	}

	writeState := &EvalWriteState{
		Addr:           addr.Resource,
		ProviderAddr:   n.ResolvedProvider,
		State:          &instancePlanState,
		ProviderSchema: &providerSchema,
	}
	diags = diags.Append(writeState.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	writeDiff := &EvalWriteDiff{
		Addr:           addr.Resource,
		ProviderSchema: &providerSchema,
		Change:         &change,
	}
	diags = diags.Append(writeDiff.Eval(ctx))
	return diags
}
