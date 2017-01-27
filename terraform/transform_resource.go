package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
)

type graphNodeExpandedResource struct {
	Index    int
	Resource *config.Resource
	Path     []string
}

func (n *graphNodeExpandedResource) Name() string {
	if n.Index == -1 {
		return n.Resource.Id()
	}

	return fmt.Sprintf("%s #%d", n.Resource.Id(), n.Index)
}

// GraphNodeAddressable impl.
func (n *graphNodeExpandedResource) ResourceAddress() *ResourceAddress {
	// We want this to report the logical index properly, so we must undo the
	// special case from the expand
	index := n.Index
	if index == -1 {
		index = 0
	}
	return &ResourceAddress{
		Path:         n.Path[1:],
		Index:        index,
		InstanceType: TypePrimary,
		Name:         n.Resource.Name,
		Type:         n.Resource.Type,
		Mode:         n.Resource.Mode,
	}
}

// GraphNodeDependable impl.
func (n *graphNodeExpandedResource) DependableName() []string {
	return []string{
		n.Resource.Id(),
		n.stateId(),
	}
}

// GraphNodeDependent impl.
func (n *graphNodeExpandedResource) DependentOn() []string {
	configNode := &GraphNodeConfigResource{Resource: n.Resource}
	result := configNode.DependentOn()

	// Walk the variables to find any count-specific variables we depend on.
	configNode.VarWalk(func(v config.InterpolatedVariable) {
		rv, ok := v.(*config.ResourceVariable)
		if !ok {
			return
		}

		// We only want ourselves
		if rv.ResourceId() != n.Resource.Id() {
			return
		}

		// If this isn't a multi-access (which shouldn't be allowed but
		// is verified elsewhere), then we depend on the specific count
		// of this resource, ignoring ourself (which again should be
		// validated elsewhere).
		if rv.Index > -1 {
			id := fmt.Sprintf("%s.%d", rv.ResourceId(), rv.Index)
			if id != n.stateId() && id != n.stateId()+".0" {
				result = append(result, id)
			}
		}
	})

	return result
}

// GraphNodeProviderConsumer
func (n *graphNodeExpandedResource) ProvidedBy() []string {
	return []string{resourceProvider(n.Resource.Type, n.Resource.Provider)}
}

func (n *graphNodeExpandedResource) StateDependencies() []string {
	depsRaw := n.DependentOn()
	deps := make([]string, 0, len(depsRaw))
	for _, d := range depsRaw {
		// Ignore any variable dependencies
		if strings.HasPrefix(d, "var.") {
			continue
		}

		// This is sad. The dependencies are currently in the format of
		// "module.foo.bar" (the full field). This strips the field off.
		if strings.HasPrefix(d, "module.") {
			parts := strings.SplitN(d, ".", 3)
			d = strings.Join(parts[0:2], ".")
		}
		deps = append(deps, d)
	}

	return deps
}

// instanceInfo is used for EvalTree.
func (n *graphNodeExpandedResource) instanceInfo() *InstanceInfo {
	return &InstanceInfo{Id: n.stateId(), Type: n.Resource.Type}
}

// stateId is the name used for the state key
func (n *graphNodeExpandedResource) stateId() string {
	if n.Index == -1 {
		return n.Resource.Id()
	}

	return fmt.Sprintf("%s.%d", n.Resource.Id(), n.Index)
}

// GraphNodeStateRepresentative impl.
func (n *graphNodeExpandedResource) StateId() []string {
	return []string{n.stateId()}
}

// graphNodeExpandedResourceDestroy represents an expanded resource that
// is to be destroyed.
type graphNodeExpandedResourceDestroy struct {
	*graphNodeExpandedResource
}

func (n *graphNodeExpandedResourceDestroy) Name() string {
	return fmt.Sprintf("%s (destroy)", n.graphNodeExpandedResource.Name())
}

// GraphNodeEvalable impl.
func (n *graphNodeExpandedResourceDestroy) EvalTree() EvalNode {
	info := n.instanceInfo()
	info.uniqueExtra = "destroy"

	var diffApply *InstanceDiff
	var provider ResourceProvider
	var state *InstanceState
	var err error
	return &EvalOpFilter{
		Ops: []walkOperation{walkApply, walkDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				// Get the saved diff for apply
				&EvalReadDiff{
					Name: n.stateId(),
					Diff: &diffApply,
				},

				// Filter the diff so we only get the destroy
				&EvalFilterDiff{
					Diff:    &diffApply,
					Output:  &diffApply,
					Destroy: true,
				},

				// If we're not destroying, then compare diffs
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						if diffApply != nil && diffApply.GetDestroy() {
							return true, nil
						}

						return true, EvalEarlyExitError{}
					},
					Then: EvalNoop{},
				},

				// Load the instance info so we have the module path set
				&EvalInstanceInfo{Info: info},

				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadState{
					Name:   n.stateId(),
					Output: &state,
				},
				&EvalRequireState{
					State: &state,
				},
				// Make sure we handle data sources properly.
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						if n.Resource.Mode == config.DataResourceMode {
							return true, nil
						}

						return false, nil
					},

					Then: &EvalReadDataApply{
						Info:     info,
						Diff:     &diffApply,
						Provider: &provider,
						Output:   &state,
					},
					Else: &EvalApply{
						Info:     info,
						State:    &state,
						Diff:     &diffApply,
						Provider: &provider,
						Output:   &state,
						Error:    &err,
					},
				},
				&EvalWriteState{
					Name:         n.stateId(),
					ResourceType: n.Resource.Type,
					Provider:     n.Resource.Provider,
					Dependencies: n.StateDependencies(),
					State:        &state,
				},
				&EvalApplyPost{
					Info:  info,
					State: &state,
					Error: &err,
				},
			},
		},
	}
}
