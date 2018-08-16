package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/states"
)

// NodeDestroyResource represents a resource that is to be destroyed.
type NodeDestroyResourceInstance struct {
	*NodeAbstractResourceInstance

	// If DeposedKey is set to anything other than states.NotDeposed then
	// this node destroys a deposed object of the associated instance
	// rather than its current object.
	DeposedKey states.DeposedKey

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
	if n.DeposedKey != states.NotDeposed {
		return nil, fmt.Errorf("NodeDestroyResourceInstance not yet updated to deal with explicit DeposedKey")
	}

	// Our graph transformers require direct access to read the entire state
	// structure, so we'll lock the whole state for the duration of this work.
	state := ctx.State().Lock()
	defer ctx.State().Unlock()

	// Start creating the steps
	steps := make([]GraphTransformer, 0, 5)

	// We want deposed resources in the state to be destroyed
	steps = append(steps, &DeposedTransformer{
		State:            state,
		InstanceAddr:     n.ResourceInstanceAddr(),
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

	// Get our state
	rs := n.ResourceState
	var is *states.ResourceInstance
	if rs != nil {
		is = rs.Instance(n.InstanceKey)
	}
	if is == nil {
		log.Printf("[WARN] NodeDestroyResourceInstance for %s with no state", addr)
	}

	var changeApply *plans.ResourceInstanceChange
	var provider providers.Interface
	var providerSchema *ProviderSchema
	var state *states.ResourceInstanceObject
	var err error
	return &EvalOpFilter{
		Ops: []walkOperation{walkApply, walkDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				// Get the saved diff for apply
				&EvalReadDiff{
					Addr:   addr.Resource,
					Change: &changeApply,
				},

				// If we're not destroying, then compare diffs
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						if changeApply != nil {
							if changeApply.Action == plans.Delete || changeApply.Action == plans.Replace {
								return true, nil
							}
						}

						return true, EvalEarlyExitError{}
					},
					Then: EvalNoop{},
				},

				&EvalGetProvider{
					Addr:   n.ResolvedProvider,
					Output: &provider,
					Schema: &providerSchema,
				},
				&EvalReadState{
					Addr:           addr.Resource,
					Output:         &state,
					Provider:       &provider,
					ProviderSchema: &providerSchema,
				},
				&EvalRequireState{
					State: &state,
				},

				// Call pre-apply hook
				&EvalApplyPre{
					Addr:   addr.Resource,
					State:  &state,
					Change: &changeApply,
				},

				// Run destroy provisioners if not tainted
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						if state != nil && state.Status == states.ObjectTainted {
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
						Change:   &changeApply,
						Provider: &provider,
						Output:   &state,
					},
					Else: &EvalApply{
						Addr:     addr.Resource,
						State:    &state,
						Change:   &changeApply,
						Provider: &provider,
						Output:   &state,
						Error:    &err,
					},
				},
				&EvalWriteState{
					Addr:           addr.Resource,
					ProviderAddr:   n.ResolvedProvider,
					ProviderSchema: &providerSchema,
					State:          &state,
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
