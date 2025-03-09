// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

// NodeForgetResourceInstance represents a resource instance that is to be
// removed from state.
type NodeForgetResourceInstance struct {
	*NodeAbstractResourceInstance
}

var (
	_ GraphNodeModuleInstance      = (*NodeForgetResourceInstance)(nil)
	_ GraphNodeConfigResource      = (*NodeForgetResourceInstance)(nil)
	_ GraphNodeResourceInstance    = (*NodeForgetResourceInstance)(nil)
	_ GraphNodeReferencer          = (*NodeForgetResourceInstance)(nil)
	_ GraphNodeExecutable          = (*NodeForgetResourceInstance)(nil)
	_ GraphNodeProviderConsumer    = (*NodeForgetResourceInstance)(nil)
	_ GraphNodeProvisionerConsumer = (*NodeForgetResourceInstance)(nil)
	_ GraphNodeDestroyer           = (*NodeForgetResourceInstance)(nil)
)

func (n *NodeForgetResourceInstance) DestroyAddr() *addrs.AbsResourceInstance {
	return &n.Addr
}

func (n *NodeForgetResourceInstance) Name() string {
	return n.ResourceInstanceAddr().String() + " (forget)"
}

func (n *NodeForgetResourceInstance) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
	if n.Addr.Resource.Resource.Mode == addrs.DataResourceMode {
		// Indicate that this node does not require a configured provider
		return nil, true
	}
	return n.NodeAbstractResourceInstance.ProvidedBy()
}

// GraphNodeExecutable
func (n *NodeForgetResourceInstance) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	is := n.instanceState
	if is == nil {
		log.Printf("[WARN] NodeForgetResourceInstance for %s with no state", addr)
	}

	var changeApply *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	changeApply, err = n.readDiff(ctx, providerSchema)
	diags = diags.Append(err)
	if changeApply == nil || diags.HasErrors() {
		return diags
	}

	state, readDiags := n.readResourceInstanceState(ctx, addr)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return diags
	}

	// Exit early if state is already null
	if state == nil || state.Value.IsNull() {
		return diags
	}

	ctx.State().ForgetResourceInstanceCurrent(n.Addr)

	diags = diags.Append(updateStateHook(ctx))
	return diags
}
