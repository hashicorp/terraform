// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
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
		return n.dynamicExpandEphemeral(ctx)
	}

	var diags tfdiags.Diagnostics
	expander := ctx.InstanceExpander()
	moduleInstances := expander.ExpandModule(n.Addr.Module, false)
	for _, module := range moduleInstances {
		moduleCtx := evalContextForModuleInstance(ctx, module)
		diags = diags.Append(n.recordResourceData(moduleCtx, n.Addr.Resource.Absolute(module)))
	}

	return nil, diags
}

// We need to expand the ephemeral resources mostly the same as we do during
// planning. There a lot of options than happen during planning which aren't
// applicable to apply however, and we have to make sure we don't re-register
// checks which already recorded in the plan, so we create a pared down version
// of the plan expansion here.
func (n *nodeExpandApplyableResource) dynamicExpandEphemeral(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var g Graph

	expander := ctx.InstanceExpander()
	moduleInstances := expander.ExpandModule(n.Addr.Module, false)

	for _, module := range moduleInstances {
		resAddr := n.Addr.Resource.Absolute(module)
		expDiags := n.expandEphemeralResourceInstances(ctx, resAddr, &g)
		diags = diags.Append(expDiags)
	}

	return &g, diags
}

func (n *nodeExpandApplyableResource) expandEphemeralResourceInstances(globalCtx EvalContext, resAddr addrs.AbsResource, g *Graph) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// The rest of our work here needs to know which module instance it's
	// working in, so that it can evaluate expressions in the appropriate scope.
	moduleCtx := evalContextForModuleInstance(globalCtx, resAddr.Module)

	// writeResourceState is responsible for informing the expander of what
	// repetition mode this resource has, which allows expander.ExpandResource
	// to work below.
	moreDiags := n.recordResourceData(moduleCtx, resAddr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	expander := moduleCtx.InstanceExpander()
	instanceAddrs := expander.ExpandResource(resAddr)

	instG, instDiags := n.ephemeralResourceInstanceSubgraph(resAddr, instanceAddrs)
	if instDiags.HasErrors() {
		diags = diags.Append(instDiags)
		return diags
	}
	g.Subsume(&instG.AcyclicGraph.Graph)
	return diags
}

func (n *nodeExpandApplyableResource) ephemeralResourceInstanceSubgraph(addr addrs.AbsResource, instanceAddrs []addrs.AbsResourceInstance) (*Graph, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	concreteEphemeral := func(a *NodeAbstractResourceInstance) dag.Vertex {
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider
		a.Schema = n.Schema
		a.ProvisionerSchemas = n.ProvisionerSchemas
		a.ProviderMetas = n.ProviderMetas
		a.dependsOn = n.dependsOn

		// we still need the Plannable resource instance
		return &NodeApplyableResourceInstance{
			NodeAbstractResourceInstance: a,
		}
	}

	// Start creating the steps
	steps := []GraphTransformer{
		// Expand the count or for_each (if present)
		&ResourceCountTransformer{
			Concrete:      concreteEphemeral,
			Schema:        n.Schema,
			Addr:          n.ResourceAddr(),
			InstanceAddrs: instanceAddrs,
		},

		// Targeting
		&TargetsTransformer{Targets: n.Targets},

		// Connect references so ordering is correct
		&ReferenceTransformer{},

		// Make sure there is a single root
		&RootTransformer{},
	}

	// Build the graph
	b := &BasicGraphBuilder{
		Steps: steps,
		Name:  "nodeExpandApplyEphemeralResource",
	}
	graph, graphDiags := b.Build(addr.Module)
	diags = diags.Append(graphDiags)

	return graph, diags
}
