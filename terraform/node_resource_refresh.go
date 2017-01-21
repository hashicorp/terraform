package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// NodeRefreshableResource represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodeRefreshableResource struct {
	*NodeAbstractResource
}

// GraphNodeDestroyer
func (n *NodeRefreshableResource) DestroyAddr() *ResourceAddress {
	return n.Addr
}

// GraphNodeEvalable
func (n *NodeRefreshableResource) EvalTree() EvalNode {
	// Eval info is different depending on what kind of resource this is
	switch mode := n.Addr.Mode; mode {
	case config.ManagedResourceMode:
		return n.evalTreeManagedResource()
	case config.DataResourceMode:
		return n.evalTreeDataResource()
	default:
		panic(fmt.Errorf("unsupported resource mode %s", mode))
	}
}

func (n *NodeRefreshableResource) evalTreeManagedResource() EvalNode {
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
				Name:   n.ProvidedBy()[0],
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

func (n *NodeRefreshableResource) evalTreeDataResource() EvalNode {
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
		// Determine the dependencies for the state. We use some older
		// code for this that we've used for a long time.
		var stateDeps []string
		{
			oldN := &graphNodeExpandedResource{
				Resource: n.Config,
				Index:    addr.Index,
			}
			stateDeps = oldN.StateDependencies()
		}

		rs = &ResourceState{
			Type:         n.Config.Type,
			Provider:     n.Config.Provider,
			Dependencies: stateDeps,
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
