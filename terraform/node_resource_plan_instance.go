package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// NodePlannableResourceInstance represents a _single_ resource
// instance that is plannable. This means this represents a single
// count index, for example.
type NodePlannableResourceInstance struct {
	*NodeAbstractResource
}

// GraphNodeEvalable
func (n *NodePlannableResourceInstance) EvalTree() EvalNode {
	addr := n.NodeAbstractResource.Addr

	// stateId is the ID to put into the state
	stateId := addr.stateId()

	// Build the instance info. More of this will be populated during eval
	info := &InstanceInfo{
		Id:         stateId,
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

	// Eval info is different depending on what kind of resource this is
	switch n.Config.Mode {
	case config.ManagedResourceMode:
		return n.evalTreeManagedResource(
			stateId, info, resource, stateDeps,
		)
	case config.DataResourceMode:
		return n.evalTreeDataResource(
			stateId, info, resource, stateDeps)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodePlannableResourceInstance) evalTreeDataResource(
	stateId string, info *InstanceInfo,
	resource *Resource, stateDeps []string) EvalNode {
	var provider ResourceProvider
	var config *ResourceConfig
	var diff *InstanceDiff
	var state *InstanceState

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalReadState{
				Name:   stateId,
				Output: &state,
			},

			// We need to re-interpolate the config here because some
			// of the attributes may have become computed during
			// earlier planning, due to other resources having
			// "requires new resource" diffs.
			&EvalInterpolate{
				Config:   n.Config.RawConfig.Copy(),
				Resource: resource,
				Output:   &config,
			},

			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					computed := config.ComputedKeys != nil && len(config.ComputedKeys) > 0

					// If the configuration is complete and we
					// already have a state then we don't need to
					// do any further work during apply, because we
					// already populated the state during refresh.
					if !computed && state != nil {
						return true, EvalEarlyExitError{}
					}

					return true, nil
				},
				Then: EvalNoop{},
			},

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

			&EvalWriteState{
				Name:         stateId,
				ResourceType: n.Config.Type,
				Provider:     n.Config.Provider,
				Dependencies: stateDeps,
				State:        &state,
			},

			&EvalWriteDiff{
				Name: stateId,
				Diff: &diff,
			},
		},
	}
}

func (n *NodePlannableResourceInstance) evalTreeManagedResource(
	stateId string, info *InstanceInfo,
	resource *Resource, stateDeps []string) EvalNode {
	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var provider ResourceProvider
	var diff *InstanceDiff
	var state *InstanceState
	var resourceConfig *ResourceConfig

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalInterpolate{
				Config:   n.Config.RawConfig.Copy(),
				Resource: resource,
				Output:   &resourceConfig,
			},
			&EvalGetProvider{
				Name:   n.ProvidedBy()[0],
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
				Name:   stateId,
				Output: &state,
			},
			&EvalDiff{
				Name:        stateId,
				Info:        info,
				Config:      &resourceConfig,
				Resource:    n.Config,
				Provider:    &provider,
				State:       &state,
				OutputDiff:  &diff,
				OutputState: &state,
			},
			&EvalCheckPreventDestroy{
				Resource: n.Config,
				Diff:     &diff,
			},
			&EvalWriteState{
				Name:         stateId,
				ResourceType: n.Config.Type,
				Provider:     n.Config.Provider,
				Dependencies: stateDeps,
				State:        &state,
			},
			&EvalWriteDiff{
				Name: stateId,
				Diff: &diff,
			},
		},
	}
}
