package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

type nodeExpandRefreshableDataResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeDynamicExpandable    = (*nodeExpandRefreshableDataResource)(nil)
	_ GraphNodeReferenceable        = (*nodeExpandRefreshableDataResource)(nil)
	_ GraphNodeReferencer           = (*nodeExpandRefreshableDataResource)(nil)
	_ GraphNodeConfigResource       = (*nodeExpandRefreshableDataResource)(nil)
	_ GraphNodeAttachResourceConfig = (*nodeExpandRefreshableDataResource)(nil)
)

func (n *nodeExpandRefreshableDataResource) Name() string {
	return n.NodeAbstractResource.Name() + " (expand)"
}

func (n *nodeExpandRefreshableDataResource) References() []*addrs.Reference {
	return (&NodeRefreshableManagedResource{NodeAbstractResource: n.NodeAbstractResource}).References()
}

func (n *nodeExpandRefreshableDataResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph

	expander := ctx.InstanceExpander()
	for _, module := range expander.ExpandModule(n.Addr.Module) {
		g.Add(&NodeRefreshableDataResource{
			NodeAbstractResource: n.NodeAbstractResource,
			Addr:                 n.Addr.Resource.Absolute(module),
		})
	}

	return &g, nil
}

// NodeRefreshableDataResource represents a resource that is "refreshable".
type NodeRefreshableDataResource struct {
	*NodeAbstractResource

	Addr addrs.AbsResource
}

var (
	_ GraphNodeModuleInstance            = (*NodeRefreshableDataResource)(nil)
	_ GraphNodeDynamicExpandable         = (*NodeRefreshableDataResource)(nil)
	_ GraphNodeReferenceable             = (*NodeRefreshableDataResource)(nil)
	_ GraphNodeReferencer                = (*NodeRefreshableDataResource)(nil)
	_ GraphNodeConfigResource            = (*NodeRefreshableDataResource)(nil)
	_ GraphNodeAttachResourceConfig      = (*NodeRefreshableDataResource)(nil)
	_ GraphNodeAttachProviderMetaConfigs = (*NodeAbstractResource)(nil)
)

func (n *NodeRefreshableDataResource) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// GraphNodeDynamicExpandable
func (n *NodeRefreshableDataResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var diags tfdiags.Diagnostics

	expander := ctx.InstanceExpander()

	switch {
	case n.Config.Count != nil:
		count, countDiags := evaluateCountExpressionValue(n.Config.Count, ctx)
		diags = diags.Append(countDiags)
		if countDiags.HasErrors() {
			return nil, diags.Err()
		}
		if !count.IsKnown() {
			// If the count isn't known yet, we'll skip refreshing and try expansion
			// again during the plan walk.
			return nil, nil
		}

		c, _ := count.AsBigFloat().Int64()
		expander.SetResourceCount(n.Addr.Module, n.Addr.Resource, int(c))

	case n.Config.ForEach != nil:
		forEachVal, forEachDiags := evaluateForEachExpressionValue(n.Config.ForEach, ctx)
		diags = diags.Append(forEachDiags)
		if forEachDiags.HasErrors() {
			return nil, diags.Err()
		}
		if !forEachVal.IsKnown() {
			// If the for_each isn't known yet, we'll skip refreshing and try expansion
			// again during the plan walk.
			return nil, nil
		}

		expander.SetResourceForEach(n.Addr.Module, n.Addr.Resource, forEachVal.AsValueMap())

	default:
		expander.SetResourceSingle(n.Addr.Module, n.Addr.Resource)
	}

	// Next we need to potentially rename an instance address in the state
	// if we're transitioning whether "count" is set at all.
	fixResourceCountSetTransition(ctx, n.ResourceAddr(), n.Config.Count != nil)

	instanceAddrs := expander.ExpandResource(n.Addr)

	// Our graph transformers require access to the full state, so we'll
	// temporarily lock it while we work on this.
	state := ctx.State().Lock()
	defer ctx.State().Unlock()

	// The concrete resource factory we'll use
	concreteResource := func(a *NodeAbstractResourceInstance) dag.Vertex {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider
		a.ProviderMetas = n.ProviderMetas
		a.dependsOn = n.dependsOn
		a.forceDependsOn = n.forceDependsOn
		a.Targets = n.Targets

		return &NodeRefreshableDataResourceInstance{
			NodeAbstractResourceInstance: a,
		}
	}

	// We also need a destroyable resource for orphans that are a result of a
	// scaled-in count.
	concreteResourceDestroyable := func(a *NodeAbstractResourceInstance) dag.Vertex {
		// Add the config and provider since we don't do that via transforms
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider

		return &NodeDestroyableDataResourceInstance{
			NodeAbstractResourceInstance: a,
		}
	}

	// Start creating the steps
	steps := []GraphTransformer{
		// Expand the count.
		&ResourceCountTransformer{
			Concrete:      concreteResource,
			Schema:        n.Schema,
			Addr:          n.ResourceAddr(),
			InstanceAddrs: instanceAddrs,
		},

		// Add the count orphans. As these are orphaned refresh nodes, we add them
		// directly as NodeDestroyableDataResource.
		&OrphanResourceInstanceCountTransformer{
			Concrete:      concreteResourceDestroyable,
			Addr:          n.Addr,
			InstanceAddrs: instanceAddrs,
			State:         state,
		},

		// Attach the state
		&AttachStateTransformer{State: state},

		// Targeting
		&TargetsTransformer{Targets: n.Targets},

		// Connect references so ordering is correct
		&ReferenceTransformer{},

		// Make sure there is a single root
		&RootTransformer{},
	}

	// Build the graph
	b := &BasicGraphBuilder{
		Steps:    steps,
		Validate: true,
		Name:     "NodeRefreshableDataResource",
	}

	graph, diags := b.Build(nil)
	return graph, diags.ErrWithWarnings()
}

// NodeRefreshableDataResourceInstance represents a single resource instance
// that is refreshable.
type NodeRefreshableDataResourceInstance struct {
	*NodeAbstractResourceInstance
}

// GraphNodeEvalable
func (n *NodeRefreshableDataResourceInstance) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	// These variables are the state for the eval sequence below, and are
	// updated through pointers.
	var provider providers.Interface
	var providerSchema *ProviderSchema
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},

			&EvalReadState{
				Addr:           addr.Resource,
				Provider:       &provider,
				ProviderSchema: &providerSchema,
				Output:         &state,
			},

			// EvalReadDataRefresh will _attempt_ to read the data source, but
			// may generate an incomplete planned object if the configuration
			// includes values that won't be known until apply.
			&evalReadDataRefresh{
				evalReadData{
					Addr:           addr.Resource,
					Config:         n.Config,
					Provider:       &provider,
					ProviderAddr:   n.ResolvedProvider,
					ProviderMetas:  n.ProviderMetas,
					ProviderSchema: &providerSchema,
					OutputChange:   &change,
					State:          &state,
					dependsOn:      n.dependsOn,
					forceDependsOn: n.forceDependsOn,
				},
			},

			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					return change == nil, nil

				},
				Then: &EvalSequence{
					Nodes: []EvalNode{
						&EvalWriteState{
							Addr:           addr.Resource,
							ProviderAddr:   n.ResolvedProvider,
							State:          &state,
							ProviderSchema: &providerSchema,
						},
						&EvalUpdateStateHook{},
					},
				},
				Else: &EvalSequence{
					// We can't deal with this yet, so we'll repeat this step
					// during the plan walk to produce a planned change to read
					// this during the apply walk. However, we do still need to
					// save the generated change and partial state so that
					// results from it can be included in other data resources
					// or provider configurations during the refresh walk.
					// (The planned object we save in the state here will be
					// pruned out at the end of the refresh walk, returning
					// it back to being unset again for subsequent walks.)
					Nodes: []EvalNode{
						&EvalWriteDiff{
							Addr:           addr.Resource,
							Change:         &change,
							ProviderSchema: &providerSchema,
						},
						&EvalWriteState{
							Addr:           addr.Resource,
							ProviderAddr:   n.ResolvedProvider,
							State:          &state,
							ProviderSchema: &providerSchema,
						},
					},
				},
			},
		},
	}
}
