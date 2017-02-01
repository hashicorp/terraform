package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// NodeRefreshableDataResource represents a resource that is "plannable":
// it is ready to be planned in order to create a diff.
type NodeRefreshableDataResource struct {
	*NodeAbstractCountResource
}

// GraphNodeDynamicExpandable
func (n *NodeRefreshableDataResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
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

		return &NodeRefreshableDataResourceInstance{
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

		// Attach the state
		&AttachStateTransformer{State: state},

		// Targeting
		&TargetsTransformer{ParsedTargets: n.Targets},

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

	return b.Build(ctx.Path())
}

// NodeRefreshableDataResourceInstance represents a _single_ resource instance
// that is refreshable.
type NodeRefreshableDataResourceInstance struct {
	*NodeAbstractResource
}

// GraphNodeEvalable
func (n *NodeRefreshableDataResourceInstance) EvalTree() EvalNode {
	addr := n.NodeAbstractResource.Addr

	// stateId is the ID to put into the state
	stateId := addr.stateId()

	// Build the instance info. More of this will be populated during eval
	info := &InstanceInfo{
		Id:   stateId,
		Type: addr.Type,
	}

	// Get the state if we have it, if not we build it
	rs := n.ResourceState
	if rs == nil {
		rs = &ResourceState{}
	}

	// If the config isn't empty we update the state
	if n.Config != nil {
		rs = &ResourceState{
			Type:         n.Config.Type,
			Provider:     n.Config.Provider,
			Dependencies: n.StateReferences(),
		}
	}

	// Build the resource for eval
	resource := &Resource{
		Name:       addr.Name,
		Type:       addr.Type,
		CountIndex: addr.Index,
	}
	if resource.CountIndex < 0 {
		resource.CountIndex = 0
	}

	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var config *ResourceConfig
	var diff *InstanceDiff
	var provider ResourceProvider
	var state *InstanceState

	return &EvalSequence{
		Nodes: []EvalNode{
			// Always destroy the existing state first, since we must
			// make sure that values from a previous read will not
			// get interpolated if we end up needing to defer our
			// loading until apply time.
			&EvalWriteState{
				Name:         stateId,
				ResourceType: rs.Type,
				Provider:     rs.Provider,
				Dependencies: rs.Dependencies,
				State:        &state, // state is nil here
			},

			&EvalInterpolate{
				Config:   n.Config.RawConfig.Copy(),
				Resource: resource,
				Output:   &config,
			},

			// The rest of this pass can proceed only if there are no
			// computed values in our config.
			// (If there are, we'll deal with this during the plan and
			// apply phases.)
			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					if config.ComputedKeys != nil && len(config.ComputedKeys) > 0 {
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

			// The remainder of this pass is the same as running
			// a "plan" pass immediately followed by an "apply" pass,
			// populating the state early so it'll be available to
			// provider configurations that need this data during
			// refresh/plan.
			&EvalGetProvider{
				Name:   n.ProvidedBy()[0],
				Output: &provider,
			},

			&EvalReadDataDiff{
				Info:        info,
				Config:      &config,
				Provider:    &provider,
				Output:      &diff,
				OutputState: &state,
			},

			&EvalReadDataApply{
				Info:     info,
				Diff:     &diff,
				Provider: &provider,
				Output:   &state,
			},

			&EvalWriteState{
				Name:         stateId,
				ResourceType: rs.Type,
				Provider:     rs.Provider,
				Dependencies: rs.Dependencies,
				State:        &state,
			},

			&EvalUpdateStateHook{},
		},
	}
}
