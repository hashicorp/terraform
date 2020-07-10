package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"

	"github.com/hashicorp/terraform/states"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
)

// nodeExpandRefreshableResource handles the first layer of resource
// expansion durin refresh. We need this extra layer so DynamicExpand is called
// twice for the resource, the first to expand the Resource for each module
// instance, and the second to expand each ResourceInstance for the expanded
// Resources.
type nodeExpandRefreshableManagedResource struct {
	*NodeAbstractResource

	// We attach dependencies to the Resource during refresh, since the
	// instances are instantiated during DynamicExpand.
	Dependencies []addrs.ConfigResource
}

var (
	_ GraphNodeDynamicExpandable    = (*nodeExpandRefreshableManagedResource)(nil)
	_ GraphNodeReferenceable        = (*nodeExpandRefreshableManagedResource)(nil)
	_ GraphNodeReferencer           = (*nodeExpandRefreshableManagedResource)(nil)
	_ GraphNodeConfigResource       = (*nodeExpandRefreshableManagedResource)(nil)
	_ GraphNodeAttachResourceConfig = (*nodeExpandRefreshableManagedResource)(nil)
	_ GraphNodeAttachDependencies   = (*nodeExpandRefreshableManagedResource)(nil)
)

func (n *nodeExpandRefreshableManagedResource) Name() string {
	return n.NodeAbstractResource.Name() + " (expand)"
}

// GraphNodeAttachDependencies
func (n *nodeExpandRefreshableManagedResource) AttachDependencies(deps []addrs.ConfigResource) {
	n.Dependencies = deps
}

func (n *nodeExpandRefreshableManagedResource) References() []*addrs.Reference {
	return (&NodeRefreshableManagedResource{NodeAbstractResource: n.NodeAbstractResource}).References()
}

func (n *nodeExpandRefreshableManagedResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph

	expander := ctx.InstanceExpander()
	for _, module := range expander.ExpandModule(n.Addr.Module) {
		g.Add(&NodeRefreshableManagedResource{
			NodeAbstractResource: n.NodeAbstractResource,
			Addr:                 n.Addr.Resource.Absolute(module),
			Dependencies:         n.Dependencies,
		})
	}

	return &g, nil
}

// NodeRefreshableManagedResource represents a resource that is expandable into
// NodeRefreshableManagedResourceInstance. Resource count orphans are also added.
type NodeRefreshableManagedResource struct {
	*NodeAbstractResource

	Addr addrs.AbsResource

	// We attach dependencies to the Resource during refresh, since the
	// instances are instantiated during DynamicExpand.
	Dependencies []addrs.ConfigResource
}

var (
	_ GraphNodeModuleInstance       = (*NodeRefreshableManagedResource)(nil)
	_ GraphNodeDynamicExpandable    = (*NodeRefreshableManagedResource)(nil)
	_ GraphNodeReferenceable        = (*NodeRefreshableManagedResource)(nil)
	_ GraphNodeReferencer           = (*NodeRefreshableManagedResource)(nil)
	_ GraphNodeConfigResource       = (*NodeRefreshableManagedResource)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeRefreshableManagedResource)(nil)
)

func (n *NodeRefreshableManagedResource) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// GraphNodeDynamicExpandable
func (n *NodeRefreshableManagedResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var diags tfdiags.Diagnostics

	expander := ctx.InstanceExpander()
	// Inform our instance expander about our expansion results, and then use
	// it to calculate the instance addresses we'll expand for.
	switch {
	case n.Config.Count != nil:
		count, countDiags := evaluateCountExpression(n.Config.Count, ctx)
		diags = diags.Append(countDiags)
		if countDiags.HasErrors() {
			return nil, diags.Err()
		}

		expander.SetResourceCount(n.Addr.Module, n.Addr.Resource, count)

	case n.Config.ForEach != nil:
		forEachMap, forEachDiags := evaluateForEachExpression(n.Config.ForEach, ctx)
		if forEachDiags.HasErrors() {
			return nil, diags.Err()
		}

		expander.SetResourceForEach(n.Addr.Module, n.Addr.Resource, forEachMap)

	default:
		expander.SetResourceSingle(n.Addr.Module, n.Addr.Resource)
	}

	// Next we need to potentially rename an instance address in the state
	// if we're transitioning whether "count" is set at all.
	fixResourceCountSetTransition(ctx, n.Addr.Config(), n.Config.Count != nil)
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
		a.Dependencies = n.Dependencies
		a.ProviderMetas = n.ProviderMetas

		return &NodeRefreshableManagedResourceInstance{
			NodeAbstractResourceInstance: a,
		}
	}

	// Start creating the steps
	steps := []GraphTransformer{
		// Expand the count.
		&ResourceCountTransformer{
			Concrete:      concreteResource,
			Schema:        n.Schema,
			Addr:          n.Addr.Config(),
			InstanceAddrs: instanceAddrs,
		},

		// Add the count orphans to make sure these resources are accounted for
		// during a scale in.
		&OrphanResourceInstanceCountTransformer{
			Concrete:      concreteResource,
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
		Name:     "NodeRefreshableManagedResource",
	}

	graph, diags := b.Build(nil)
	return graph, diags.ErrWithWarnings()
}

// NodeRefreshableManagedResourceInstance represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodeRefreshableManagedResourceInstance struct {
	*NodeAbstractResourceInstance
}

var (
	_ GraphNodeModuleInstance       = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeReferencer           = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeDestroyer            = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeConfigResource       = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeResourceInstance     = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeAttachResourceState  = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeEvalable             = (*NodeRefreshableManagedResourceInstance)(nil)
)

// GraphNodeDestroyer
func (n *NodeRefreshableManagedResourceInstance) DestroyAddr() *addrs.AbsResourceInstance {
	addr := n.ResourceInstanceAddr()
	return &addr
}

// GraphNodeEvalable
func (n *NodeRefreshableManagedResourceInstance) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	// Eval info is different depending on what kind of resource this is
	switch addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		if n.instanceState == nil {
			log.Printf("[TRACE] NodeRefreshableManagedResourceInstance: %s has no existing state to refresh", addr)
			return n.evalTreeManagedResourceNoState()
		}
		log.Printf("[TRACE] NodeRefreshableManagedResourceInstance: %s will be refreshed", addr)
		return n.evalTreeManagedResource()

	case addrs.DataResourceMode:
		// Get the data source node. If we don't have a configuration
		// then it is an orphan so we destroy it (remove it from the state).
		var dn GraphNodeEvalable
		if n.Config != nil {
			dn = &NodeRefreshableDataResourceInstance{
				NodeAbstractResourceInstance: n.NodeAbstractResourceInstance,
			}
		} else {
			dn = &NodeDestroyableDataResourceInstance{
				NodeAbstractResourceInstance: n.NodeAbstractResourceInstance,
			}
		}

		return dn.EvalTree()
	default:
		panic(fmt.Errorf("unsupported resource mode %s", addr.Resource.Resource.Mode))
	}
}

func (n *NodeRefreshableManagedResourceInstance) evalTreeManagedResource() EvalNode {
	addr := n.ResourceInstanceAddr()

	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var provider providers.Interface
	var providerSchema *ProviderSchema
	var state *states.ResourceInstanceObject

	// This happened during initial development. All known cases were
	// fixed and tested but as a sanity check let's assert here.
	if n.instanceState == nil {
		err := fmt.Errorf(
			"No resource state attached for addr: %s\n\n"+
				"This is a bug. Please report this to Terraform with your configuration\n"+
				"and state attached. Please be careful to scrub any sensitive information.",
			addr)
		return &EvalReturnError{Error: &err}
	}

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

				Output: &state,
			},

			&EvalRefreshDependencies{
				State:        &state,
				Dependencies: &n.Dependencies,
			},

			&EvalRefresh{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				Provider:       &provider,
				ProviderMetas:  n.ProviderMetas,
				ProviderSchema: &providerSchema,
				State:          &state,
				Output:         &state,
			},

			&EvalWriteState{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
				State:          &state,
				Dependencies:   &n.Dependencies,
			},
		},
	}
}

// evalTreeManagedResourceNoState produces an EvalSequence for refresh resource
// nodes that don't have state attached. An example of where this functionality
// is useful is when a resource that already exists in state is being scaled
// out, ie: has its resource count increased. In this case, the scaled out node
// needs to be available to other nodes (namely data sources) that may depend
// on it for proper interpolation, or confusing "index out of range" errors can
// occur.
//
// The steps in this sequence are very similar to the steps carried out in
// plan, but nothing is done with the diff after it is created - it is dropped,
// and its changes are not counted in the UI.
func (n *NodeRefreshableManagedResourceInstance) evalTreeManagedResourceNoState() EvalNode {
	addr := n.ResourceInstanceAddr()

	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
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

				Output: &state,
			},

			&EvalDiff{
				Addr:           addr.Resource,
				Config:         n.Config,
				Provider:       &provider,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
				State:          &state,
				OutputChange:   &change,
				OutputState:    &state,
				Stub:           true,
			},

			&EvalWriteState{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
				State:          &state,
				Dependencies:   &n.Dependencies,
			},

			// We must also save the planned change, so that expressions in
			// other nodes, such as provider configurations and data resources,
			// can work with the planned new value.
			//
			// This depends on the fact that Context.Refresh creates a
			// temporary new empty changeset for the duration of its graph
			// walk, and so this recorded change will be discarded immediately
			// after the refresh walk completes.
			&EvalWriteDiff{
				Addr:           addr.Resource,
				Change:         &change,
				ProviderSchema: &providerSchema,
			},
		},
	}
}
