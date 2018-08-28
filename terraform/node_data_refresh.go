package terraform

import (
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// NodeRefreshableDataResource represents a resource that is "refreshable".
type NodeRefreshableDataResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeSubPath              = (*NodeRefreshableDataResource)(nil)
	_ GraphNodeDynamicExpandable    = (*NodeRefreshableDataResource)(nil)
	_ GraphNodeReferenceable        = (*NodeRefreshableDataResource)(nil)
	_ GraphNodeReferencer           = (*NodeRefreshableDataResource)(nil)
	_ GraphNodeResource             = (*NodeRefreshableDataResource)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeRefreshableDataResource)(nil)
)

// GraphNodeDynamicExpandable
func (n *NodeRefreshableDataResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
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

		return &NodeRefreshableDataResourceInstance{
			NodeAbstractResourceInstance: a,
		}
	}

	// We also need a destroyable resource for orphans that are a result of a
	// scaled-in count.
	concreteResourceDestroyable := func(a *NodeAbstractResourceInstance) dag.Vertex {
		// Add the config since we don't do that via transforms
		a.Config = n.Config

		return &NodeDestroyableDataResource{
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

		// Add the count orphans. As these are orphaned refresh nodes, we add them
		// directly as NodeDestroyableDataResource.
		&OrphanResourceCountTransformer{
			Concrete: concreteResourceDestroyable,
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
		Name:     "NodeRefreshableDataResource",
	}

	graph, diags := b.Build(ctx.Path())
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
	var configVal cty.Value

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},

			// Always destroy the existing state first, since we must
			// make sure that values from a previous read will not
			// get interpolated if we end up needing to defer our
			// loading until apply time.
			&EvalWriteState{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				State:          &state, // a pointer to nil, here
				ProviderSchema: &providerSchema,
			},

			&EvalReadDataDiff{
				Addr:              addr.Resource,
				Config:            n.Config,
				ProviderAddr:      n.ResolvedProvider,
				ProviderSchema:    &providerSchema,
				Output:            &change,
				OutputConfigValue: &configVal,
				OutputState:       &state,
			},

			// The rest of this pass can proceed only if there are no
			// computed values in our config.
			// (If there are, we'll deal with this during the plan and
			// apply phases.)
			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					if !configVal.IsWhollyKnown() {
						return true, EvalEarlyExitError{}
					}

					// If the config explicitly has a depends_on for this
					// data source, assume the intention is to prevent
					// refreshing ahead of that dependency.
					if len(n.Config.DependsOn) > 0 {
						return true, EvalEarlyExitError{}
					}

					return true, nil
				},
				Then: EvalNoop{},
			},

			&EvalReadDataApply{
				Addr:     addr.Resource,
				Change:   &change,
				Provider: &provider,
				Output:   &state,
			},

			&EvalWriteState{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				State:          &state,
				ProviderSchema: &providerSchema,
			},

			&EvalUpdateStateHook{},
		},
	}
}
