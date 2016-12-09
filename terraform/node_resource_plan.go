package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// NodePlannableResource represents a resource that is "plannable":
// it is ready to be planned in order to create a diff.
type NodePlannableResource struct {
	*NodeAbstractResource

	// Set by GraphNodeTargetable and used during DynamicExpand to
	// forward targets downwards.
	targets []ResourceAddress
}

// GraphNodeTargetable
func (n *NodePlannableResource) SetTargets(targets []ResourceAddress) {
	n.targets = targets
}

// GraphNodeEvalable
func (n *NodePlannableResource) EvalTree() EvalNode {
	return &EvalSequence{
		Nodes: []EvalNode{
			// The EvalTree for a plannable resource primarily involves
			// interpolating the count since it can contain variables
			// we only just received access to.
			//
			// With the interpolated count, we can then DynamicExpand
			// into the proper number of instances.
			&EvalInterpolate{Config: n.Config.RawCount},

			&EvalCountCheckComputed{Resource: n.Config},
			&EvalCountFixZeroOneBoundary{Resource: n.Config},
		},
	}
}

// GraphNodeDynamicExpandable
func (n *NodePlannableResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	// Grab the state which we read
	state, lock := ctx.State()
	lock.RLock()
	defer lock.RUnlock()

	// Expand the resource count which must be available by now from EvalTree
	count, err := n.Config.Count()
	if err != nil {
		return nil, err
	}

	// The concrete resource factory we'll use
	concreteResource := func(a *NodeAbstractResource) dag.Vertex {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config

		return &NodePlannableResourceInstance{
			NodeAbstractResource: a,
		}
	}

	// The concrete resource factory we'll use for oprhans
	concreteResourceOrphan := func(a *NodeAbstractResource) dag.Vertex {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config

		return &NodePlannableResourceOrphan{
			NodeAbstractResource: a,
		}
	}

	// Start creating the steps
	steps := []GraphTransformer{
		// Expand the count.
		&ResourceCountTransformer{
			Concrete: concreteResource,
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
		&TargetsTransformer{ParsedTargets: n.targets},

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
	return b.Build(ctx.Path())
}
