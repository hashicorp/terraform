package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// NodeRefreshableManagedResource represents a resource that is expanabled into
// NodeRefreshableManagedResourceInstance. Resource count orphans are also added.
type NodeRefreshableManagedResource struct {
	*NodeAbstractCountResource
}

// GraphNodeDynamicExpandable
func (n *NodeRefreshableManagedResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
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
		a.ResolvedProvider = n.ResolvedProvider

		return &NodeRefreshableManagedResourceInstance{
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
		Name:     "NodeRefreshableManagedResource",
	}

	return b.Build(ctx.Path())
}

// NodeRefreshableManagedResourceInstance represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodeRefreshableManagedResourceInstance struct {
	*NodeAbstractResource
}

// GraphNodeDestroyer
func (n *NodeRefreshableManagedResourceInstance) DestroyAddr() *ResourceAddress {
	return n.Addr
}

// GraphNodeEvalable
func (n *NodeRefreshableManagedResourceInstance) EvalTree() EvalNode {
	// Eval info is different depending on what kind of resource this is
	switch mode := n.Addr.Mode; mode {
	case config.ManagedResourceMode:
		if n.ResourceState == nil {
			return n.evalTreeManagedResourceNoState()
		}
		return n.evalTreeManagedResource()

	case config.DataResourceMode:
		// Get the data source node. If we don't have a configuration
		// then it is an orphan so we destroy it (remove it from the state).
		var dn GraphNodeEvalable
		if n.Config != nil {
			dn = &NodeRefreshableDataResourceInstance{
				NodeAbstractResource: n.NodeAbstractResource,
			}
		} else {
			dn = &NodeDestroyableDataResource{
				NodeAbstractResource: n.NodeAbstractResource,
			}
		}

		return dn.EvalTree()
	default:
		panic(fmt.Errorf("unsupported resource mode %s", mode))
	}
}

func (n *NodeRefreshableManagedResourceInstance) evalTreeManagedResource() EvalNode {
	addr := n.NodeAbstractResource.Addr

	// stateId is the ID to put into the state
	stateId := addr.stateId()

	// Build the instance info. More of this will be populated during eval
	info := &InstanceInfo{
		Id:   stateId,
		Type: addr.Type,
	}

	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var provider ResourceProvider
	var state *InstanceState

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
				Name:   n.ResolvedProvider,
				Output: &provider,
			},
			&EvalReadState{
				Name:   stateId,
				Output: &state,
			},
			&EvalRefresh{
				Info:     info,
				Provider: &provider,
				State:    &state,
				Output:   &state,
			},
			&EvalWriteState{
				Name:         stateId,
				ResourceType: n.ResourceState.Type,
				Provider:     n.ResourceState.Provider,
				Dependencies: n.ResourceState.Dependencies,
				State:        &state,
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
	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var provider ResourceProvider
	var state *InstanceState
	var resourceConfig *ResourceConfig

	addr := n.NodeAbstractResource.Addr
	stateID := addr.stateId()
	info := &InstanceInfo{
		Id:         stateID,
		Type:       addr.Type,
		ModulePath: normalizeModulePath(addr.Path),
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

	// Determine the dependencies for the state.
	stateDeps := n.StateReferences()

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalInterpolate{
				Config:   n.Config.RawConfig.Copy(),
				Resource: resource,
				Output:   &resourceConfig,
			},
			&EvalGetProvider{
				Name:   n.ResolvedProvider,
				Output: &provider,
			},
			// Re-run validation to catch any errors we missed, e.g. type
			// mismatches on computed values.
			&EvalValidateResource{
				Provider:       &provider,
				Config:         &resourceConfig,
				ResourceName:   n.Config.Name,
				ResourceType:   n.Config.Type,
				ResourceMode:   n.Config.Mode,
				IgnoreWarnings: true,
			},
			&EvalReadState{
				Name:   stateID,
				Output: &state,
			},
			&EvalDiff{
				Name:        stateID,
				Info:        info,
				Config:      &resourceConfig,
				Resource:    n.Config,
				Provider:    &provider,
				State:       &state,
				OutputState: &state,
				Stub:        true,
			},
			&EvalWriteState{
				Name:         stateID,
				ResourceType: n.Config.Type,
				Provider:     n.Config.Provider,
				Dependencies: stateDeps,
				State:        &state,
			},
		},
	}
}
