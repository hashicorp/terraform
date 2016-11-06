package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// NodePlannableResource represents a resource that is "plannable":
// it is ready to be planned in order to create a diff.
type NodePlannableResource struct {
	*NodeAbstractResource
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
		},
	}
}

// GraphNodeDynamicExpandable
func (n *NodePlannableResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	// Expand the resource count which must be available by now from EvalTree
	count, err := n.Config.Count()
	if err != nil {
		return nil, err
	}

	// The concrete resource factory we'll use
	concreteResource := func(a *NodeAbstractResource) dag.Vertex {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config
		a.ResourceState = n.ResourceState

		return &NodePlannableResourceInstance{
			NodeAbstractResource: a,
		}
	}

	// Start creating the steps
	steps := make([]GraphTransformer, 0, 5)

	// Expand counts.
	steps = append(steps, &ResourceCountTransformer{
		Concrete: concreteResource,
		Count:    count,
		Addr:     n.ResourceAddr(),
	})

	// Always end with the root being added
	steps = append(steps, &RootTransformer{})

	// Build the graph
	b := &BasicGraphBuilder{Steps: steps, Validate: true}
	return b.Build(ctx.Path())
}
