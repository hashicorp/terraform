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

// NodeRefreshableManagedResource represents a resource that is expanabled into
// NodeRefreshableManagedResourceInstance. Resource count orphans are also added.
type NodeRefreshableManagedResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeSubPath              = (*NodeRefreshableManagedResource)(nil)
	_ GraphNodeDynamicExpandable    = (*NodeRefreshableManagedResource)(nil)
	_ GraphNodeReferenceable        = (*NodeRefreshableManagedResource)(nil)
	_ GraphNodeReferencer           = (*NodeRefreshableManagedResource)(nil)
	_ GraphNodeResource             = (*NodeRefreshableManagedResource)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeRefreshableManagedResource)(nil)
)

// GraphNodeDynamicExpandable
func (n *NodeRefreshableManagedResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var diags tfdiags.Diagnostics

	count, countDiags := evaluateResourceCountExpression(n.Config.Count, ctx)
	diags = diags.Append(countDiags)
	if countDiags.HasErrors() {
		return nil, diags.Err()
	}

	// Next we need to potentially rename an instance address in the state
	// if we're transitioning whether "count" is set at all.
	fixResourceCountSetTransition(ctx, n.ResourceAddr(), count != -1)

	// Our graph transformers require access to the full state, so we'll
	// temporarily lock it while we work on this.
	state := ctx.State().Lock()
	defer ctx.State().Unlock()

	// The concrete resource factory we'll use
	concreteResource := func(a *NodeAbstractResourceInstance) dag.Vertex {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider

		return &NodeRefreshableManagedResourceInstance{
			NodeAbstractResourceInstance: a,
		}
	}

	// Start creating the steps
	steps := []GraphTransformer{
		// Expand the count.
		&ResourceCountTransformer{
			Concrete: concreteResource,
			Schema:   n.Schema,
			Count:    count,
			Addr:     n.ResourceAddr(),
		},

		// Add the count orphans to make sure these resources are accounted for
		// during a scale in.
		&OrphanResourceCountTransformer{
			Concrete: concreteResource,
			Count:    count,
			Addr:     n.ResourceAddr(),
			State:    state,
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

	graph, diags := b.Build(ctx.Path())
	return graph, diags.ErrWithWarnings()
}

// NodeRefreshableManagedResourceInstance represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodeRefreshableManagedResourceInstance struct {
	*NodeAbstractResourceInstance
}

var (
	_ GraphNodeSubPath              = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeReferencer           = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeDestroyer            = (*NodeRefreshableManagedResourceInstance)(nil)
	_ GraphNodeResource             = (*NodeRefreshableManagedResourceInstance)(nil)
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
		if n.ResourceState == nil {
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
			dn = &NodeDestroyableDataResource{
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
	if n.ResourceState == nil {
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

			&EvalRefresh{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				Provider:       &provider,
				ProviderSchema: &providerSchema,
				State:          &state,
				Output:         &state,
			},

			&EvalWriteState{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
				State:          &state,
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
			},
		},
	}
}
