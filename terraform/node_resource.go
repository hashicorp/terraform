package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// NodeApplyableResource represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodeApplyableResource struct {
	Addr          *ResourceAddress // Addr is the address for this resource
	Config        *config.Resource // Config is the resource in the config
	ResourceState *ResourceState   // ResourceState is the ResourceState for this
}

func (n *NodeApplyableResource) Name() string {
	return n.Addr.String()
}

// GraphNodeSubPath
func (n *NodeApplyableResource) Path() []string {
	return n.Addr.Path
}

// GraphNodeProviderConsumer
func (n *NodeApplyableResource) ProvidedBy() []string {
	// If we have a config we prefer that above all else
	if n.Config != nil {
		return []string{resourceProvider(n.Config.Type, n.Config.Provider)}
	}

	// If we have state, then we will use the provider from there
	if n.ResourceState != nil {
		return []string{n.ResourceState.Provider}
	}

	// Use our type
	return []string{resourceProvider(n.Addr.Type, "")}
}

// GraphNodeEvalable
func (n *NodeApplyableResource) EvalTree() EvalNode {
	// stateId is the ID to put into the state
	stateId := n.Addr.stateId()
	if n.Addr.Index > -1 {
		stateId = fmt.Sprintf("%s.%d", stateId, n.Addr.Index)
	}

	// Build the instance info. More of this will be populated during eval
	info := &InstanceInfo{
		Id:   stateId,
		Type: n.Addr.Type,
	}

	// Build the resource for eval
	resource := &Resource{
		Name:       n.Addr.Name,
		Type:       n.Addr.Type,
		CountIndex: n.Addr.Index,
	}
	if resource.CountIndex < 0 {
		resource.CountIndex = 0
	}

	// Determine the dependencies for the state. We use some older
	// code for this that we've used for a long time.
	var stateDeps []string
	{
		oldN := &graphNodeExpandedResource{Resource: n.Config}
		stateDeps = oldN.StateDependencies()
	}

	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var provider ResourceProvider
	var diff *InstanceDiff
	var state *InstanceState
	var resourceConfig *ResourceConfig
	var err error
	var createNew bool
	var createBeforeDestroyEnabled bool

	return &EvalSequence{
		Nodes: []EvalNode{
			// Build the instance info
			&EvalInstanceInfo{
				Info: info,
			},

			// Get the saved diff for apply
			&EvalReadDiff{
				Name: stateId,
				Diff: &diff,
			},

			// We don't want to do any destroys
			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					if diff == nil {
						return true, EvalEarlyExitError{}
					}

					if diff.GetDestroy() && diff.GetAttributesLen() == 0 {
						return true, EvalEarlyExitError{}
					}

					diff.SetDestroy(false)
					return true, nil
				},
				Then: EvalNoop{},
			},

			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					destroy := false
					if diff != nil {
						destroy = diff.GetDestroy() || diff.RequiresNew()
					}

					createBeforeDestroyEnabled =
						n.Config.Lifecycle.CreateBeforeDestroy &&
							destroy

					return createBeforeDestroyEnabled, nil
				},
				Then: &EvalDeposeState{
					Name: stateId,
				},
			},

			&EvalInterpolate{
				Config:   n.Config.RawConfig.Copy(),
				Resource: resource,
				Output:   &resourceConfig,
			},
			&EvalGetProvider{
				Name:   n.ProvidedBy()[0],
				Output: &provider,
			},
			&EvalReadState{
				Name:   stateId,
				Output: &state,
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
			&EvalDiff{
				Info:       info,
				Config:     &resourceConfig,
				Resource:   n.Config,
				Provider:   &provider,
				Diff:       &diff,
				State:      &state,
				OutputDiff: &diff,
			},

			// Get the saved diff
			&EvalReadDiff{
				Name: stateId,
				Diff: &diff,
			},

			// Compare the diffs
			&EvalCompareDiff{
				Info: info,
				One:  &diff,
				Two:  &diff,
			},

			&EvalGetProvider{
				Name:   n.ProvidedBy()[0],
				Output: &provider,
			},
			&EvalReadState{
				Name:   stateId,
				Output: &state,
			},
			&EvalApply{
				Info:      info,
				State:     &state,
				Diff:      &diff,
				Provider:  &provider,
				Output:    &state,
				Error:     &err,
				CreateNew: &createNew,
			},
			&EvalWriteState{
				Name:         stateId,
				ResourceType: n.Config.Type,
				Provider:     n.Config.Provider,
				Dependencies: stateDeps,
				State:        &state,
			},
			/*
				TODO: this has to work
				&EvalApplyProvisioners{
					Info:           info,
					State:          &state,
					Resource:       n.Config,
					InterpResource: resource,
					CreateNew:      &createNew,
					Error:          &err,
				},
			*/
			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					return createBeforeDestroyEnabled && err != nil, nil
				},
				Then: &EvalUndeposeState{
					Name:  stateId,
					State: &state,
				},
				Else: &EvalWriteState{
					Name:         stateId,
					ResourceType: n.Config.Type,
					Provider:     n.Config.Provider,
					Dependencies: stateDeps,
					State:        &state,
				},
			},

			// We clear the diff out here so that future nodes
			// don't see a diff that is already complete. There
			// is no longer a diff!
			&EvalWriteDiff{
				Name: stateId,
				Diff: nil,
			},

			&EvalApplyPost{
				Info:  info,
				State: &state,
				Error: &err,
			},
			&EvalUpdateStateHook{},
		},
	}
}
