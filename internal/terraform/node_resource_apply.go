// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeExpandApplyableResource handles the first layer of resource
// expansion during apply. Even though the resource instances themselves are
// already expanded from the plan, we still need to expand the
// NodeApplyableResource nodes into their respective modules.
type nodeExpandApplyableResource struct {
	*NodeAbstractResource

	PartialExpansions []addrs.PartialExpandedResource
}

var (
	_ GraphNodeReferenceable        = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeReferencer           = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeConfigResource       = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*nodeExpandApplyableResource)(nil)
	_ graphNodeExpandsInstances     = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeTargetable           = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeDynamicExpandable    = (*nodeExpandApplyableResource)(nil)
)

func (n *nodeExpandApplyableResource) expandsInstances() {
}

func (n *nodeExpandApplyableResource) References() []*addrs.Reference {
	refs := n.NodeAbstractResource.References()

	// The expand node needs to connect to the individual resource instances it
	// references, but cannot refer to it's own instances without causing
	// cycles. It would be preferable to entirely disallow self references
	// without the `self` identifier, but those were allowed in provisioners
	// for compatibility with legacy configuration. We also can't always just
	// filter them out for all resource node types, because the only method we
	// have for catching certain invalid configurations are the cycles that
	// result from these inter-instance references.
	return filterSelfRefs(n.Addr.Resource, refs)
}

func (n *nodeExpandApplyableResource) Name() string {
	return n.NodeAbstractResource.Name() + " (expand)"
}

func (n *nodeExpandApplyableResource) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	if n.Addr.Resource.Mode == addrs.EphemeralResourceMode {
		// FIXME: we need to expand the ephemeral resources the same as we do
		// during planning, so we convert this into the plannable node on the
		// fly, and skip Execute later on.
		return (&nodeExpandPlannableResource{
			NodeAbstractResource: n.NodeAbstractResource,
		}).DynamicExpand(ctx)
	}
	return nil, nil
}

func (n *nodeExpandApplyableResource) Execute(globalCtx EvalContext, op walkOperation) tfdiags.Diagnostics {
	if n.Addr.Resource.Mode == addrs.EphemeralResourceMode {
		return nil
	}

	var diags tfdiags.Diagnostics
	expander := globalCtx.InstanceExpander()
	moduleInstances := expander.ExpandModule(n.Addr.Module, false)
	for _, module := range moduleInstances {
		moduleCtx := evalContextForModuleInstance(globalCtx, module)
		diags = diags.Append(n.recordResourceData(moduleCtx, n.Addr.Resource.Absolute(module)))
	}

	return diags
}
