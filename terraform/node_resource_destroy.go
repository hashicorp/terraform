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

// NodeDestroyResourceInstance represents a resource instance that is to be
// destroyed.
type NodeDestroyResourceInstance struct {
	*NodeAbstractResourceInstance

	// If DeposedKey is set to anything other than states.NotDeposed then
	// this node destroys a deposed object of the associated instance
	// rather than its current object.
	DeposedKey states.DeposedKey

	CreateBeforeDestroyOverride *bool
}

var (
	_ GraphNodeModuleInstance      = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeConfigResource      = (*NodeDestroyResourceInstance)(nil)
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
	if n.DeposedKey != states.NotDeposed {
		return fmt.Sprintf("%s (destroy deposed %s)", n.ResourceInstanceAddr(), n.DeposedKey)
	}
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

	// Config takes precedence
	if n.Config != nil && n.Config.Managed != nil {
		return n.Config.Managed.CreateBeforeDestroy
	}

	// Otherwise check the state for a stored destroy order
	if s := n.instanceState; s != nil {
		if s.Current != nil {
			return s.Current.CreateBeforeDestroy
		}
	}

	return false
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

// GraphNodeEvalable
func (n *NodeDestroyResourceInstance) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	// Get our state
	is := n.instanceState
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
				&EvalGetProvider{
					Addr:   n.ResolvedProvider,
					Output: &provider,
					Schema: &providerSchema,
				},

				// Get the saved diff for apply
				&EvalReadDiff{
					Addr:           addr.Resource,
					ProviderSchema: &providerSchema,
					Change:         &changeApply,
				},

				&EvalReduceDiff{
					Addr:      addr.Resource,
					InChange:  &changeApply,
					Destroy:   true,
					OutChange: &changeApply,
				},

				// EvalReduceDiff may have simplified our planned change
				// into a NoOp if it does not require destroying.
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						if changeApply == nil || changeApply.Action == plans.NoOp {
							return true, EvalEarlyExitError{}
						}
						return true, nil
					},
					Then: EvalNoop{},
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

				// Managed resources need to be destroyed, while data sources
				// are only removed from state.
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						return addr.Resource.Resource.Mode == addrs.ManagedResourceMode, nil
					},

					Then: &EvalSequence{
						Nodes: []EvalNode{
							&EvalApply{
								Addr:           addr.Resource,
								Config:         nil, // No configuration because we are destroying
								State:          &state,
								Change:         &changeApply,
								Provider:       &provider,
								ProviderAddr:   n.ResolvedProvider,
								ProviderMetas:  n.ProviderMetas,
								ProviderSchema: &providerSchema,
								Output:         &state,
								Error:          &err,
							},
							&EvalWriteState{
								Addr:           addr.Resource,
								ProviderAddr:   n.ResolvedProvider,
								ProviderSchema: &providerSchema,
								State:          &state,
							},
						},
					},
					Else: &evalWriteEmptyState{
						EvalWriteState{
							Addr:           addr.Resource,
							ProviderAddr:   n.ResolvedProvider,
							ProviderSchema: &providerSchema,
						},
					},
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
