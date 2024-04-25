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

func (n *nodeExpandApplyableResource) Execute(globalCtx EvalContext, op walkOperation) tfdiags.Diagnostics {

	// TODO: When validating support for modules (TF-13952), we should check
	// here if the whole module is partially expanded and skip the .ExpandModule
	// call below.
	//
	//  for _, per := range n.PartialExpansions {
	//    if _, ok := per.PartialExpandedModule(); ok {
	//      return nil // don't even try to expand the modules
	//    }
	//  }
	//
	//  The above checks if the module is partially expanded and if it is, it
	//  skips the expansion of the module. This isn't implemented yet, because
	//  partial module expansion is not implemented properly yet.

	var diags tfdiags.Diagnostics
	expander := globalCtx.InstanceExpander()
	moduleInstances := expander.ExpandModule(n.Addr.Module, false)
Insts:
	for _, module := range moduleInstances {

		// First check if this resource in this module instance in part of a
		// partial expansion. If it is, we can't and don't need to expand it.
		for _, per := range n.PartialExpansions {
			if per.MatchesResource(n.Addr.Absolute(module)) {
				continue Insts
			}
		}

		moduleCtx := evalContextForModuleInstance(globalCtx, module)
		diags = diags.Append(n.writeResourceState(moduleCtx, n.Addr.Resource.Absolute(module)))
	}

	return diags
}
