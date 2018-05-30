package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
)

// NodeDestroyResource represents a resource that is to be destroyed.
type NodeDestroyResourceInstance struct {
	*NodeAbstractResourceInstance

	CreateBeforeDestroyOverride *bool
}

var (
	_ GraphNodeResource            = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeResourceInstance    = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeDestroyer           = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeDestroyerCBD        = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeReferenceable       = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeReferencer          = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeEvalable            = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeProviderConsumer    = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeProvisionerConsumer = (*NodeDestroyResourceInstance)(nil)
)

func (n *NodeDestroyResourceInstance) Name() string {
	return n.ResourceInstanceAddr().String() + " (destroy)"
}

// GraphNodeDestroyer
func (n *NodeDestroyResourceInstance) DestroyAddr() *addrs.AbsResourceInstance {
	addr := n.ResourceInstanceAddr()
	return &addr
}

// GraphNodeDestroyerCBD
func (n *NodeDestroyResourceInstance) CreateBeforeDestroy() bool {
	if n.CreateBeforeDestroyOverride != nil {
		return *n.CreateBeforeDestroyOverride
	}

	// If we have no config, we just assume no
	if n.Config == nil || n.Config.Managed == nil {
		return false
	}

	return n.Config.Managed.CreateBeforeDestroy
}

// GraphNodeDestroyerCBD
func (n *NodeDestroyResourceInstance) ModifyCreateBeforeDestroy(v bool) error {
	n.CreateBeforeDestroyOverride = &v
	return nil
}

// GraphNodeReferenceable, overriding NodeAbstractResource
func (n *NodeDestroyResourceInstance) ReferenceableAddrs() []addrs.Referenceable {
	normalAddrs := n.NodeAbstractResourceInstance.ReferenceableAddrs()
	destroyAddrs := make([]addrs.Referenceable, len(normalAddrs))

	phaseType := addrs.ResourceInstancePhaseDestroy
	if n.CreateBeforeDestroy() {
		phaseType = addrs.ResourceInstancePhaseDestroyCBD
	}

	for i, normalAddr := range normalAddrs {
		switch ta := normalAddr.(type) {
		case addrs.Resource:
			destroyAddrs[i] = ta.Phase(phaseType)
		case addrs.ResourceInstance:
			destroyAddrs[i] = ta.Phase(phaseType)
		default:
			destroyAddrs[i] = normalAddr
		}
	}

	return destroyAddrs
}

// GraphNodeReferencer, overriding NodeAbstractResource
func (n *NodeDestroyResourceInstance) References() []*addrs.Reference {
	// If we have a config, then we need to include destroy-time dependencies
	if c := n.Config; c != nil && c.Managed != nil {
		var result []*addrs.Reference

		// We include conn info and config for destroy time provisioners
		// as dependencies that we have.
		for _, p := range c.Managed.Provisioners {
			schema := n.ProvisionerSchemas[p.Type]

			if p.When == configs.ProvisionerWhenDestroy {
				if p.Connection != nil {
					result = append(result, ReferencesFromConfig(p.Connection.Config, connectionBlockSupersetSchema)...)
				}
				result = append(result, ReferencesFromConfig(p.Config, schema)...)
			}
		}

		return result
	}

	return nil
}

// GraphNodeDynamicExpandable
func (n *NodeDestroyResourceInstance) DynamicExpand(ctx EvalContext) (*Graph, error) {
	// stateId is the legacy-style ID to put into the state
	stateId := NewLegacyResourceInstanceAddress(n.ResourceInstanceAddr()).stateId()

	state, lock := ctx.State()
	lock.RLock()
	defer lock.RUnlock()

	// Start creating the steps
	steps := make([]GraphTransformer, 0, 5)

	// We want deposed resources in the state to be destroyed
	steps = append(steps, &DeposedTransformer{
		State:            state,
		View:             stateId,
		ResolvedProvider: n.ResolvedProvider,
	})

	// Target
	steps = append(steps, &TargetsTransformer{
		Targets: n.Targets,
	})

	// Always end with the root being added
	steps = append(steps, &RootTransformer{})

	// Build the graph
	b := &BasicGraphBuilder{
		Steps: steps,
		Name:  "NodeResourceDestroy",
	}
	g, diags := b.Build(ctx.Path())
	return g, diags.ErrWithWarnings()
}

// GraphNodeEvalable
func (n *NodeDestroyResourceInstance) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	// stateId is the legacy-style ID to put into the state
	stateId := NewLegacyResourceInstanceAddress(n.ResourceInstanceAddr()).stateId()

	// Get our state
	rs := n.ResourceState
	if rs == nil {
		rs = &ResourceState{
			Provider: n.ResolvedProvider.String(),
		}
	}

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
					Name: stateId,
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

				&EvalGetProvider{
					Addr:   n.ResolvedProvider,
					Output: &provider,
				},
				&EvalReadState{
					Name:   stateId,
					Output: &state,
				},
				&EvalRequireState{
					State: &state,
				},

				// Call pre-apply hook
				&EvalApplyPre{
					Addr:  addr.Resource,
					State: &state,
					Diff:  &diffApply,
				},

				// Run destroy provisioners if not tainted
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						if state != nil && state.Tainted {
							return false, nil
						}

						return true, nil
					},

					Then: &EvalApplyProvisioners{
						Addr:           addr.Resource,
						State:          &state,
						ResourceConfig: n.Config,
						Error:          &err,
						When:           configs.ProvisionerWhenDestroy,
					},
				},

				// If we have a provisioning error, then we just call
				// the post-apply hook now.
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						return err != nil, nil
					},

					Then: &EvalApplyPost{
						Addr:  addr.Resource,
						State: &state,
						Error: &err,
					},
				},

				// Make sure we handle data sources properly.
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						return addr.Resource.Resource.Mode == addrs.DataResourceMode, nil
					},

					Then: &EvalReadDataApply{
						Addr:     addr.Resource,
						Diff:     &diffApply,
						Provider: &provider,
						Output:   &state,
					},
					Else: &EvalApply{
						Addr:     addr.Resource,
						State:    &state,
						Diff:     &diffApply,
						Provider: &provider,
						Output:   &state,
						Error:    &err,
					},
				},
				&EvalWriteState{
					Name:         stateId,
					ResourceType: addr.Resource.Resource.Type,
					Provider:     n.ResolvedProvider,
					Dependencies: rs.Dependencies,
					State:        &state,
				},
				&EvalApplyPost{
					Addr:  addr.Resource,
					State: &state,
					Error: &err,
				},
				&EvalUpdateStateHook{},
			},
		},
	}
}
