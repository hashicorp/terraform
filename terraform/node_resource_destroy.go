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

				// Make sure we handle data sources properly.
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						return addr.Resource.Resource.Mode == addrs.DataResourceMode, nil
					},

					Then: &EvalReadDataApply{
						Addr:           addr.Resource,
						Config:         n.Config,
						Change:         &changeApply,
						Provider:       &provider,
						ProviderAddr:   n.ResolvedProvider,
						ProviderSchema: &providerSchema,
						Output:         &state,
					},
					Else: &EvalApply{
						Addr:           addr.Resource,
						Config:         nil, // No configuration because we are destroying
						State:          &state,
						Change:         &changeApply,
						Provider:       &provider,
						ProviderAddr:   n.ResolvedProvider,
						ProviderSchema: &providerSchema,
						Output:         &state,
						Error:          &err,
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

// NodeDestroyResourceInstance represents a resource that is to be destroyed.
//
// Destroying a resource is a state-only operation: it is the individual
// instances being destroyed that affects remote objects. During graph
// construction, NodeDestroyResource should always depend on any other node
// related to the given resource, since it's just a final cleanup to avoid
// leaving skeleton resource objects in state after their instances have
// all been destroyed.
type NodeDestroyResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeResource      = (*NodeDestroyResource)(nil)
	_ GraphNodeReferenceable = (*NodeDestroyResource)(nil)
	_ GraphNodeReferencer    = (*NodeDestroyResource)(nil)
	_ GraphNodeEvalable      = (*NodeDestroyResource)(nil)

	// FIXME: this is here to document that this node is both
	// GraphNodeProviderConsumer by virtue of the embedded
	// NodeAbstractResource, but that behavior is not desired and we skip it by
	// checking for GraphNodeNoProvider.
	_ GraphNodeProviderConsumer = (*NodeDestroyResource)(nil)
	_ GraphNodeNoProvider       = (*NodeDestroyResource)(nil)
)

func (n *NodeDestroyResource) Name() string {
	return n.ResourceAddr().String() + " (clean up state)"
}

// GraphNodeReferenceable, overriding NodeAbstractResource
func (n *NodeDestroyResource) ReferenceableAddrs() []addrs.Referenceable {
	// NodeDestroyResource doesn't participate in references: the graph
	// builder that created it should ensure directly that it already depends
	// on every other node related to its resource, without relying on
	// references.
	return nil
}

// GraphNodeReferencer, overriding NodeAbstractResource
func (n *NodeDestroyResource) References() []*addrs.Reference {
	// NodeDestroyResource doesn't participate in references: the graph
	// builder that created it should ensure directly that it already depends
	// on every other node related to its resource, without relying on
	// references.
	return nil
}

// GraphNodeEvalable
func (n *NodeDestroyResource) EvalTree() EvalNode {
	// This EvalNode will produce an error if the resource isn't already
	// empty by the time it is called, since it should just be pruning the
	// leftover husk of a resource in state after all of the child instances
	// and their objects were destroyed.
	return &EvalForgetResourceState{
		Addr: n.ResourceAddr().Resource,
	}
}

// GraphNodeResource
func (n *NodeDestroyResource) ResourceAddr() addrs.AbsResource {
	return n.NodeAbstractResource.ResourceAddr()
}

// GraphNodeSubpath
func (n *NodeDestroyResource) Path() addrs.ModuleInstance {
	return n.NodeAbstractResource.Path()
}

// GraphNodeNoProvider
// FIXME: this should be removed once the node can be separated from the
// Internal NodeAbstractResource behavior.
func (n *NodeDestroyResource) NoProvider() {
}

type GraphNodeNoProvider interface {
	NoProvider()
}
