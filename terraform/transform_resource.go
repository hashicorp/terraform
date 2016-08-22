package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// ResourceCountTransformer is a GraphTransformer that expands the count
// out for a specific resource.
type ResourceCountTransformer struct {
	Resource *config.Resource
	Destroy  bool
	Targets  []ResourceAddress
}

func (t *ResourceCountTransformer) Transform(g *Graph) error {
	// Expand the resource count
	count, err := t.Resource.Count()
	if err != nil {
		return err
	}

	// Don't allow the count to be negative
	if count < 0 {
		return fmt.Errorf("negative count: %d", count)
	}

	// For each count, build and add the node
	nodes := make([]dag.Vertex, 0, count)
	for i := 0; i < count; i++ {
		// Set the index. If our count is 1 we special case it so that
		// we handle the "resource.0" and "resource" boundary properly.
		index := i
		if count == 1 {
			index = -1
		}

		// Save the node for later so we can do connections. Make the
		// proper node depending on if we're just a destroy node or if
		// were a regular node.
		var node dag.Vertex = &graphNodeExpandedResource{
			Index:    index,
			Resource: t.Resource,
			Path:     g.Path,
		}
		if t.Destroy {
			node = &graphNodeExpandedResourceDestroy{
				graphNodeExpandedResource: node.(*graphNodeExpandedResource),
			}
		}

		// Skip nodes if targeting excludes them
		if !t.nodeIsTargeted(node) {
			continue
		}

		// Add the node now
		nodes = append(nodes, node)
		g.Add(node)
	}

	// Make the dependency connections
	for _, n := range nodes {
		// Connect the dependents. We ignore the return value for missing
		// dependents since that should've been caught at a higher level.
		g.ConnectDependent(n)
	}

	return nil
}

func (t *ResourceCountTransformer) nodeIsTargeted(node dag.Vertex) bool {
	// no targets specified, everything stays in the graph
	if len(t.Targets) == 0 {
		return true
	}
	addressable, ok := node.(GraphNodeAddressable)
	if !ok {
		return false
	}

	addr := addressable.ResourceAddress()
	for _, targetAddr := range t.Targets {
		if targetAddr.Equals(addr) {
			return true
		}
	}
	return false
}

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

// graphNodeConfig impl.
func (n *graphNodeExpandedResource) ConfigType() GraphNodeConfigType {
	return GraphNodeConfigTypeResource
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

// GraphNodeEvalable impl.
func (n *graphNodeExpandedResource) EvalTree() EvalNode {
	var provider ResourceProvider
	var resourceConfig *ResourceConfig

	// Build the resource. If we aren't part of a multi-resource, then
	// we still consider ourselves as count index zero.
	index := n.Index
	if index < 0 {
		index = 0
	}
	resource := &Resource{
		Name:       n.Resource.Name,
		Type:       n.Resource.Type,
		CountIndex: index,
	}

	seq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}

	// Validate the resource
	vseq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}
	vseq.Nodes = append(vseq.Nodes, &EvalGetProvider{
		Name:   n.ProvidedBy()[0],
		Output: &provider,
	})
	vseq.Nodes = append(vseq.Nodes, &EvalInterpolate{
		Config:   n.Resource.RawConfig.Copy(),
		Resource: resource,
		Output:   &resourceConfig,
	})
	vseq.Nodes = append(vseq.Nodes, &EvalValidateResource{
		Provider:     &provider,
		Config:       &resourceConfig,
		ResourceName: n.Resource.Name,
		ResourceType: n.Resource.Type,
		ResourceMode: n.Resource.Mode,
	})

	// Validate all the provisioners
	for _, p := range n.Resource.Provisioners {
		var provisioner ResourceProvisioner
		vseq.Nodes = append(vseq.Nodes, &EvalGetProvisioner{
			Name:   p.Type,
			Output: &provisioner,
		}, &EvalInterpolate{
			Config:   p.RawConfig.Copy(),
			Resource: resource,
			Output:   &resourceConfig,
		}, &EvalValidateProvisioner{
			Provisioner: &provisioner,
			Config:      &resourceConfig,
		})
	}

	// Add the validation operations
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops:  []walkOperation{walkValidate},
		Node: vseq,
	})

	// Build instance info
	info := n.instanceInfo()
	seq.Nodes = append(seq.Nodes, &EvalInstanceInfo{Info: info})

	// Each resource mode has its own lifecycle
	switch n.Resource.Mode {
	case config.ManagedResourceMode:
		seq.Nodes = append(
			seq.Nodes,
			n.managedResourceEvalNodes(resource, info, resourceConfig)...,
		)
	case config.DataResourceMode:
		seq.Nodes = append(
			seq.Nodes,
			n.dataResourceEvalNodes(resource, info, resourceConfig)...,
		)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Resource.Mode))
	}

	return seq
}

func (n *graphNodeExpandedResource) managedResourceEvalNodes(resource *Resource, info *InstanceInfo, resourceConfig *ResourceConfig) []EvalNode {
	var diff *InstanceDiff
	var provider ResourceProvider
	var state *InstanceState

	nodes := make([]EvalNode, 0, 5)

	// Refresh the resource
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadState{
					Name:   n.stateId(),
					Output: &state,
				},
				&EvalRefresh{
					Info:     info,
					Provider: &provider,
					State:    &state,
					Output:   &state,
				},
				&EvalWriteState{
					Name:         n.stateId(),
					ResourceType: n.Resource.Type,
					Provider:     n.Resource.Provider,
					Dependencies: n.StateDependencies(),
					State:        &state,
				},
			},
		},
	})

	// Diff the resource
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkPlan},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalInterpolate{
					Config:   n.Resource.RawConfig.Copy(),
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
					ResourceName:   n.Resource.Name,
					ResourceType:   n.Resource.Type,
					ResourceMode:   n.Resource.Mode,
					IgnoreWarnings: true,
				},
				&EvalReadState{
					Name:   n.stateId(),
					Output: &state,
				},
				&EvalDiff{
					Info:        info,
					Config:      &resourceConfig,
					Resource:    n.Resource,
					Provider:    &provider,
					State:       &state,
					OutputDiff:  &diff,
					OutputState: &state,
				},
				&EvalCheckPreventDestroy{
					Resource: n.Resource,
					Diff:     &diff,
				},
				&EvalWriteState{
					Name:         n.stateId(),
					ResourceType: n.Resource.Type,
					Provider:     n.Resource.Provider,
					Dependencies: n.StateDependencies(),
					State:        &state,
				},
				&EvalWriteDiff{
					Name: n.stateId(),
					Diff: &diff,
				},
			},
		},
	})

	// Diff the resource for destruction
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkPlanDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalReadState{
					Name:   n.stateId(),
					Output: &state,
				},
				&EvalDiffDestroy{
					Info:   info,
					State:  &state,
					Output: &diff,
				},
				&EvalCheckPreventDestroy{
					Resource: n.Resource,
					Diff:     &diff,
				},
				&EvalWriteDiff{
					Name: n.stateId(),
					Diff: &diff,
				},
			},
		},
	})

	// Apply
	var diffApply *InstanceDiff
	var err error
	var createNew bool
	var createBeforeDestroyEnabled bool
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkApply, walkDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				// Get the saved diff for apply
				&EvalReadDiff{
					Name: n.stateId(),
					Diff: &diffApply,
				},

				// We don't want to do any destroys
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						if diffApply == nil {
							return true, EvalEarlyExitError{}
						}

						if diffApply.GetDestroy() && diffApply.GetAttributesLen() == 0 {
							return true, EvalEarlyExitError{}
						}

						diffApply.SetDestroy(false)
						return true, nil
					},
					Then: EvalNoop{},
				},

				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						destroy := false
						if diffApply != nil {
							destroy = diffApply.GetDestroy() || diffApply.RequiresNew()
						}

						createBeforeDestroyEnabled =
							n.Resource.Lifecycle.CreateBeforeDestroy &&
								destroy

						return createBeforeDestroyEnabled, nil
					},
					Then: &EvalDeposeState{
						Name: n.stateId(),
					},
				},

				&EvalInterpolate{
					Config:   n.Resource.RawConfig.Copy(),
					Resource: resource,
					Output:   &resourceConfig,
				},
				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadState{
					Name:   n.stateId(),
					Output: &state,
				},
				// Re-run validation to catch any errors we missed, e.g. type
				// mismatches on computed values.
				&EvalValidateResource{
					Provider:       &provider,
					Config:         &resourceConfig,
					ResourceName:   n.Resource.Name,
					ResourceType:   n.Resource.Type,
					ResourceMode:   n.Resource.Mode,
					IgnoreWarnings: true,
				},
				&EvalDiff{
					Info:       info,
					Config:     &resourceConfig,
					Resource:   n.Resource,
					Provider:   &provider,
					Diff:       &diffApply,
					State:      &state,
					OutputDiff: &diffApply,
				},

				// Get the saved diff
				&EvalReadDiff{
					Name: n.stateId(),
					Diff: &diff,
				},

				// Compare the diffs
				&EvalCompareDiff{
					Info: info,
					One:  &diff,
					Two:  &diffApply,
				},

				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadState{
					Name:   n.stateId(),
					Output: &state,
				},
				&EvalApply{
					Info:      info,
					State:     &state,
					Diff:      &diffApply,
					Provider:  &provider,
					Output:    &state,
					Error:     &err,
					CreateNew: &createNew,
				},
				&EvalWriteState{
					Name:         n.stateId(),
					ResourceType: n.Resource.Type,
					Provider:     n.Resource.Provider,
					Dependencies: n.StateDependencies(),
					State:        &state,
				},
				&EvalApplyProvisioners{
					Info:           info,
					State:          &state,
					Resource:       n.Resource,
					InterpResource: resource,
					CreateNew:      &createNew,
					Error:          &err,
				},
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						return createBeforeDestroyEnabled && err != nil, nil
					},
					Then: &EvalUndeposeState{
						Name:  n.stateId(),
						State: &state,
					},
					Else: &EvalWriteState{
						Name:         n.stateId(),
						ResourceType: n.Resource.Type,
						Provider:     n.Resource.Provider,
						Dependencies: n.StateDependencies(),
						State:        &state,
					},
				},

				// We clear the diff out here so that future nodes
				// don't see a diff that is already complete. There
				// is no longer a diff!
				&EvalWriteDiff{
					Name: n.stateId(),
					Diff: nil,
				},

				&EvalApplyPost{
					Info:  info,
					State: &state,
					Error: &err,
				},
				&EvalUpdateStateHook{},
			},
		},
	})

	return nodes
}

func (n *graphNodeExpandedResource) dataResourceEvalNodes(resource *Resource, info *InstanceInfo, resourceConfig *ResourceConfig) []EvalNode {
	//var diff *InstanceDiff
	var provider ResourceProvider
	var config *ResourceConfig
	var diff *InstanceDiff
	var state *InstanceState

	nodes := make([]EvalNode, 0, 5)

	// Refresh the resource
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalSequence{
			Nodes: []EvalNode{

				// Always destroy the existing state first, since we must
				// make sure that values from a previous read will not
				// get interpolated if we end up needing to defer our
				// loading until apply time.
				&EvalWriteState{
					Name:         n.stateId(),
					ResourceType: n.Resource.Type,
					Provider:     n.Resource.Provider,
					Dependencies: n.StateDependencies(),
					State:        &state, // state is nil here
				},

				&EvalInterpolate{
					Config:   n.Resource.RawConfig.Copy(),
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
					Name:         n.stateId(),
					ResourceType: n.Resource.Type,
					Provider:     n.Resource.Provider,
					Dependencies: n.StateDependencies(),
					State:        &state,
				},

				&EvalUpdateStateHook{},
			},
		},
	})

	// Diff the resource
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkPlan},
		Node: &EvalSequence{
			Nodes: []EvalNode{

				&EvalReadState{
					Name:   n.stateId(),
					Output: &state,
				},

				// We need to re-interpolate the config here because some
				// of the attributes may have become computed during
				// earlier planning, due to other resources having
				// "requires new resource" diffs.
				&EvalInterpolate{
					Config:   n.Resource.RawConfig.Copy(),
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
					Name:         n.stateId(),
					ResourceType: n.Resource.Type,
					Provider:     n.Resource.Provider,
					Dependencies: n.StateDependencies(),
					State:        &state,
				},

				&EvalWriteDiff{
					Name: n.stateId(),
					Diff: &diff,
				},
			},
		},
	})

	// Diff the resource for destruction
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkPlanDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{

				&EvalReadState{
					Name:   n.stateId(),
					Output: &state,
				},

				// Since EvalDiffDestroy doesn't interact with the
				// provider at all, we can safely share the same
				// implementation for data vs. managed resources.
				&EvalDiffDestroy{
					Info:   info,
					State:  &state,
					Output: &diff,
				},

				&EvalWriteDiff{
					Name: n.stateId(),
					Diff: &diff,
				},
			},
		},
	})

	// Apply
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkApply, walkDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				// Get the saved diff for apply
				&EvalReadDiff{
					Name: n.stateId(),
					Diff: &diff,
				},

				// Stop here if we don't actually have a diff
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						if diff == nil {
							return true, EvalEarlyExitError{}
						}

						if diff.GetAttributesLen() == 0 {
							return true, EvalEarlyExitError{}
						}

						return true, nil
					},
					Then: EvalNoop{},
				},

				// We need to re-interpolate the config here, rather than
				// just using the diff's values directly, because we've
				// potentially learned more variable values during the
				// apply pass that weren't known when the diff was produced.
				&EvalInterpolate{
					Config:   n.Resource.RawConfig.Copy(),
					Resource: resource,
					Output:   &config,
				},

				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},

				// Make a new diff with our newly-interpolated config.
				&EvalReadDataDiff{
					Info:     info,
					Config:   &config,
					Previous: &diff,
					Provider: &provider,
					Output:   &diff,
				},

				&EvalReadDataApply{
					Info:     info,
					Diff:     &diff,
					Provider: &provider,
					Output:   &state,
				},

				&EvalWriteState{
					Name:         n.stateId(),
					ResourceType: n.Resource.Type,
					Provider:     n.Resource.Provider,
					Dependencies: n.StateDependencies(),
					State:        &state,
				},

				// Clear the diff now that we've applied it, so
				// later nodes won't see a diff that's now a no-op.
				&EvalWriteDiff{
					Name: n.stateId(),
					Diff: nil,
				},

				&EvalUpdateStateHook{},
			},
		},
	})

	return nodes
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

// graphNodeConfig impl.
func (n *graphNodeExpandedResourceDestroy) ConfigType() GraphNodeConfigType {
	return GraphNodeConfigTypeResource
}

// GraphNodeEvalable impl.
func (n *graphNodeExpandedResourceDestroy) EvalTree() EvalNode {
	info := n.instanceInfo()

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
