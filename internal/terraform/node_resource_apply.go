// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

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
	_ GraphNodeDynamicExpandable    = (*nodeExpandApplyableResource)(nil)
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
	var diags tfdiags.Diagnostics
	expander := globalCtx.InstanceExpander()
	moduleInstances := expander.ExpandModule(n.Addr.Module, false)
	for _, module := range moduleInstances {
		moduleCtx := evalContextForModuleInstance(globalCtx, module)
		diags = diags.Append(n.writeResourceState(moduleCtx, n.Addr.Resource.Absolute(module)))
	}

	return diags
}

func (n *nodeExpandApplyableResource) DynamicExpand(globalCtx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	g := &Graph{}

	if n.Addr.Resource.Mode.PersistsPlanToApply() {
		// We don't need to do anything for resources of modes that use
		// plannable actions, because our toplevel apply graph already
		// includes the expanded instances of those based on the plan diff.
		addRootNodeToGraph(g)
		return g, diags
	}

	// For resources of modes that _don't_ persist from plan to apply, we'll
	// generate the nodes representing instances dynamically here to mimic
	// what we would've done during the plan walk.
	expander := globalCtx.InstanceExpander()
	for _, modInstAddr := range expander.ExpandModule(n.ModulePath(), false) {
		resourceAddr := n.Addr.Resource.Absolute(modInstAddr)
		for _, instAddr := range expander.ExpandResource(resourceAddr) {
			// FIXME: The code we use to do the similar thing in the plan phase
			// is not really shaped well to be reused here, so this is just a
			// bare-minimum thing to get close enough for the sake of prototyping
			// ephemeral resources. We should probably do this in a different
			// way if we make a real implementation.
			log.Printf("[TRACE] nodeExpandApplyableResource: adding node for %s", instAddr)
			instN := &NodeApplyableResourceInstance{
				NodeAbstractResourceInstance: NewNodeAbstractResourceInstance(instAddr),
			}
			instN.Config = n.Config
			instN.ResolvedProvider = n.ResolvedProvider
			instN.Schema = n.Schema
			instN.SchemaVersion = n.SchemaVersion
			g.Add(instN)
		}
	}

	addRootNodeToGraph(g)
	return g, diags
}
