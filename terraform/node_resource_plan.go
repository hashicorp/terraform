package terraform

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

/*
// GraphNodeDynamicExpandable
func (n *NodePlannableResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	state, lock := ctx.State()
	lock.RLock()
	defer lock.RUnlock()

	// Start creating the steps
	steps := make([]GraphTransformer, 0, 5)

	// Expand counts.
	steps = append(steps, &ResourceCountTransformer{
		Resource: n.Resource,
		Destroy:  n.Destroy,
		Targets:  n.Targets,
	})

	// Always end with the root being added
	steps = append(steps, &RootTransformer{})

	// Build the graph
	b := &BasicGraphBuilder{Steps: steps, Validate: true}
	return b.Build(ctx.Path())
}
*/
