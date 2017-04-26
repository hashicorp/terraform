package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// NodeDestroyResource represents a resource that is to be destroyed.
type NodeDestroyResource struct {
	*NodeAbstractResource
}

func (n *NodeDestroyResource) Name() string {
	return n.NodeAbstractResource.Name() + " (destroy)"
}

// GraphNodeDestroyer
func (n *NodeDestroyResource) DestroyAddr() *ResourceAddress {
	return n.Addr
}

// GraphNodeDestroyerCBD
func (n *NodeDestroyResource) CreateBeforeDestroy() bool {
	// If we have no config, we just assume no
	if n.Config == nil {
		return false
	}

	return n.Config.Lifecycle.CreateBeforeDestroy
}

// GraphNodeDestroyerCBD
func (n *NodeDestroyResource) ModifyCreateBeforeDestroy(v bool) error {
	// If we have no config, do nothing since it won't affect the
	// create step anyways.
	if n.Config == nil {
		return nil
	}

	// Set CBD to true
	n.Config.Lifecycle.CreateBeforeDestroy = true

	return nil
}

// GraphNodeReferenceable, overriding NodeAbstractResource
func (n *NodeDestroyResource) ReferenceableName() []string {
	// We modify our referenceable name to have the suffix of ".destroy"
	// since depending on the creation side doesn't necessarilly mean
	// depending on destruction.
	suffix := ".destroy"

	// If we're CBD, we also append "-cbd". This is because CBD will setup
	// its own edges (in CBDEdgeTransformer). Depending on the "destroy"
	// side generally doesn't mean depending on CBD as well. See GH-11349
	if n.CreateBeforeDestroy() {
		suffix += "-cbd"
	}

	result := n.NodeAbstractResource.ReferenceableName()
	for i, v := range result {
		result[i] = v + suffix
	}

	return result
}

// GraphNodeReferencer, overriding NodeAbstractResource
func (n *NodeDestroyResource) References() []string {
	// If we have a config, then we need to include destroy-time dependencies
	if c := n.Config; c != nil {
		var result []string
		for _, p := range c.Provisioners {
			// We include conn info and config for destroy time provisioners
			// as dependencies that we have.
			if p.When == config.ProvisionerWhenDestroy {
				result = append(result, ReferencesFromConfig(p.ConnInfo)...)
				result = append(result, ReferencesFromConfig(p.RawConfig)...)
			}
		}

		return result
	}

	return nil
}

// GraphNodeDynamicExpandable
func (n *NodeDestroyResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	// If we have no config we do nothing
	if n.Addr == nil {
		return nil, nil
	}

	state, lock := ctx.State()
	lock.RLock()
	defer lock.RUnlock()

	// Start creating the steps
	steps := make([]GraphTransformer, 0, 5)

	// We want deposed resources in the state to be destroyed
	steps = append(steps, &DeposedTransformer{
		State: state,
		View:  n.Addr.stateId(),
	})

	// Target
	steps = append(steps, &TargetsTransformer{
		ParsedTargets: n.Targets,
	})

	// Always end with the root being added
	steps = append(steps, &RootTransformer{})

	// Build the graph
	b := &BasicGraphBuilder{
		Steps: steps,
		Name:  "NodeResourceDestroy",
	}
	return b.Build(ctx.Path())
}

// GraphNodeEvalable
func (n *NodeDestroyResource) EvalTree() EvalNode {
	// stateId is the ID to put into the state
	stateId := n.Addr.stateId()

	// Build the instance info. More of this will be populated during eval
	info := &InstanceInfo{
		Id:          stateId,
		Type:        n.Addr.Type,
		uniqueExtra: "destroy",
	}

	// Build the resource for eval
	addr := n.Addr
	resource := &Resource{
		Name:       addr.Name,
		Type:       addr.Type,
		CountIndex: addr.Index,
	}
	if resource.CountIndex < 0 {
		resource.CountIndex = 0
	}

	// Get our state
	rs := n.ResourceState
	if rs == nil {
		rs = &ResourceState{}
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

				// Load the instance info so we have the module path set
				&EvalInstanceInfo{Info: info},

				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
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
					Info:  info,
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
						Info:           info,
						State:          &state,
						Resource:       n.Config,
						InterpResource: resource,
						Error:          &err,
						When:           config.ProvisionerWhenDestroy,
					},
				},

				// If we have a provisioning error, then we just call
				// the post-apply hook now.
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						return err != nil, nil
					},

					Then: &EvalApplyPost{
						Info:  info,
						State: &state,
						Error: &err,
					},
				},

				// Make sure we handle data sources properly.
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						if n.Addr == nil {
							return false, fmt.Errorf("nil address")
						}

						if n.Addr.Mode == config.DataResourceMode {
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
					Name:         stateId,
					ResourceType: n.Addr.Type,
					Provider:     rs.Provider,
					Dependencies: rs.Dependencies,
					State:        &state,
				},
				&EvalApplyPost{
					Info:  info,
					State: &state,
					Error: &err,
				},
				&EvalUpdateStateHook{},
			},
		},
	}
}
