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

			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					// If the config explicitly has a depends_on for this
					// data source, assume the intention is to prevent
					// refreshing ahead of that dependency, and therefore
					// we need to deal with this resource during the apply
					// phase..
					if len(n.Config.DependsOn) > 0 {
						return true, EvalEarlyExitError{}
					}

					return true, nil
				},
				Then: EvalNoop{},
			},

			// EvalReadData will _attempt_ to read the data source, but may
			// generate an incomplete planned object if the configuration
			// includes values that won't be known until apply.
			&EvalReadData{
				Addr:              addr.Resource,
				Config:            n.Config,
				Dependencies:      n.StateReferences(),
				Provider:          &provider,
				ProviderAddr:      n.ResolvedProvider,
				ProviderSchema:    &providerSchema,
				OutputChange:      &change,
				OutputConfigValue: &configVal,
				OutputState:       &state,
			},

			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					return (*state).Status != states.ObjectPlanned, nil
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
