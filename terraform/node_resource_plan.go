package terraform

import (
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
)

// NodePlannableResource represents a resource that is "plannable":
// it is ready to be planned in order to create a diff.
type NodePlannableResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeSubPath              = (*NodePlannableResource)(nil)
	_ GraphNodeDynamicExpandable    = (*NodePlannableResource)(nil)
	_ GraphNodeReferenceable        = (*NodePlannableResource)(nil)
	_ GraphNodeReferencer           = (*NodePlannableResource)(nil)
	_ GraphNodeResource             = (*NodePlannableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlannableResource)(nil)
)

// GraphNodeDynamicExpandable
func (n *NodePlannableResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var diags tfdiags.Diagnostics

	count, countDiags := evaluateResourceCountExpression(n.Config.Count, ctx)
	diags = diags.Append(countDiags)
	if countDiags.HasErrors() {
		return nil, diags.Err()
	}

	// Next we need to potentially rename an instance address in the state
	// if we're transitioning whether "count" is set at all.
	fixResourceCountSetTransition(ctx, n.ResourceAddr().Resource, count != -1)

	// Grab the state which we read
	state, lock := ctx.State()
	lock.RLock()
	defer lock.RUnlock()

	// The concrete resource factory we'll use
	concreteResource := func(a *NodeAbstractResourceInstance) dag.Vertex {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider
		a.Schema = n.Schema
		a.ProvisionerSchemas = n.ProvisionerSchemas

		return &NodePlannableResourceInstance{
			NodeAbstractResourceInstance: a,
		}
	}

	// The concrete resource factory we'll use for orphans
	concreteResourceOrphan := func(a *NodeAbstractResourceInstance) dag.Vertex {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider
		a.Schema = n.Schema
		a.ProvisionerSchemas = n.ProvisionerSchemas

		return &NodePlannableResourceInstanceOrphan{
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

		// Add the count orphans
		&OrphanResourceCountTransformer{
			Concrete: concreteResourceOrphan,
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
		Name:     "NodePlannableResource",
	}
	graph, diags := b.Build(ctx.Path())
	return graph, diags.ErrWithWarnings()
}
