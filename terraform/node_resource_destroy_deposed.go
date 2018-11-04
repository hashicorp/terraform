package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
)

// ConcreteResourceInstanceDeposedNodeFunc is a callback type used to convert
// an abstract resource instance to a concrete one of some type that has
// an associated deposed object key.
type ConcreteResourceInstanceDeposedNodeFunc func(*NodeAbstractResourceInstance, states.DeposedKey) dag.Vertex

type GraphNodeDeposedResourceInstanceObject interface {
	DeposedInstanceObjectKey() states.DeposedKey
}

// NodePlanDeposedResourceInstanceObject represents deposed resource
// instance objects during plan. These are distinct from the primary object
// for each resource instance since the only valid operation to do with them
// is to destroy them.
//
// This node type is also used during the refresh walk to ensure that the
// record of a deposed object is up-to-date before we plan to destroy it.
type NodePlanDeposedResourceInstanceObject struct {
	*NodeAbstractResourceInstance
	DeposedKey states.DeposedKey
}

var (
	_ GraphNodeDeposedResourceInstanceObject = (*NodePlanDeposedResourceInstanceObject)(nil)
	_ GraphNodeResource                      = (*NodePlanDeposedResourceInstanceObject)(nil)
	_ GraphNodeResourceInstance              = (*NodePlanDeposedResourceInstanceObject)(nil)
	_ GraphNodeReferenceable                 = (*NodePlanDeposedResourceInstanceObject)(nil)
	_ GraphNodeReferencer                    = (*NodePlanDeposedResourceInstanceObject)(nil)
	_ GraphNodeEvalable                      = (*NodePlanDeposedResourceInstanceObject)(nil)
	_ GraphNodeProviderConsumer              = (*NodePlanDeposedResourceInstanceObject)(nil)
	_ GraphNodeProvisionerConsumer           = (*NodePlanDeposedResourceInstanceObject)(nil)
)

func (n *NodePlanDeposedResourceInstanceObject) Name() string {
	return fmt.Sprintf("%s (deposed %s)", n.ResourceInstanceAddr().String(), n.DeposedKey)
}

func (n *NodePlanDeposedResourceInstanceObject) DeposedInstanceObjectKey() states.DeposedKey {
	return n.DeposedKey
}

// GraphNodeReferenceable implementation, overriding the one from NodeAbstractResourceInstance
func (n *NodePlanDeposedResourceInstanceObject) ReferenceableAddrs() []addrs.Referenceable {
	// Deposed objects don't participate in references.
	return nil
}

// GraphNodeReferencer implementation, overriding the one from NodeAbstractResourceInstance
func (n *NodePlanDeposedResourceInstanceObject) References() []*addrs.Reference {
	// We don't evaluate configuration for deposed objects, so they effectively
	// make no references.
	return nil
}

// GraphNodeEvalable impl.
func (n *NodePlanDeposedResourceInstanceObject) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	var provider providers.Interface
	var providerSchema *ProviderSchema
	var state *states.ResourceInstanceObject

	seq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}

	// During the refresh walk we will ensure that our record of the deposed
	// object is up-to-date. If it was already deleted outside of Terraform
	// then this will remove it from state and thus avoid us planning a
	// destroy for it during the subsequent plan walk.
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Addr:   n.ResolvedProvider,
					Output: &provider,
					Schema: &providerSchema,
				},
				&EvalReadStateDeposed{
					Addr:           addr.Resource,
					ProviderSchema: &providerSchema,
					Key:            n.DeposedKey,
					Output:         &state,
				},
				&EvalRefresh{
					Addr:           addr.Resource,
					ProviderAddr:   n.ResolvedProvider,
					Provider:       &provider,
					ProviderSchema: &providerSchema,
					State:          &state,
					Output:         &state,
				},
				&EvalWriteStateDeposed{
					Addr:           addr.Resource,
					Key:            n.DeposedKey,
					ProviderAddr:   n.ResolvedProvider,
					ProviderSchema: &providerSchema,
					State:          &state,
				},
			},
		},
	})

	// During the plan walk we always produce a planned destroy change, because
	// destroying is the only supported action for deposed objects.
	var change *plans.ResourceInstanceChange
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkPlan, walkPlanDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Addr:   n.ResolvedProvider,
					Output: &provider,
					Schema: &providerSchema,
				},
				&EvalReadStateDeposed{
					Addr:           addr.Resource,
					Output:         &state,
					Key:            n.DeposedKey,
					Provider:       &provider,
					ProviderSchema: &providerSchema,
				},
				&EvalDiffDestroy{
					Addr:         addr.Resource,
					ProviderAddr: n.ResolvedProvider,
					DeposedKey:   n.DeposedKey,
					State:        &state,
					Output:       &change,
				},
				&EvalWriteDiff{
					Addr:           addr.Resource,
					DeposedKey:     n.DeposedKey,
					ProviderSchema: &providerSchema,
					Change:         &change,
				},
				// Since deposed objects cannot be referenced by expressions
				// elsewhere, we don't need to also record the planned new
				// state in this case.
			},
		},
	})

	return seq
}

// NodeDestroyDeposedResourceInstanceObject represents deposed resource
// instance objects during apply. Nodes of this type are inserted by
// DiffTransformer when the planned changeset contains "delete" changes for
// deposed instance objects, and its only supported operation is to destroy
// and then forget the associated object.
type NodeDestroyDeposedResourceInstanceObject struct {
	*NodeAbstractResourceInstance
	DeposedKey states.DeposedKey
}

var (
	_ GraphNodeDeposedResourceInstanceObject = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeResource                      = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeResourceInstance              = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeDestroyer                     = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeDestroyerCBD                  = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeReferenceable                 = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeReferencer                    = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeEvalable                      = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeProviderConsumer              = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeProvisionerConsumer           = (*NodeDestroyDeposedResourceInstanceObject)(nil)
)

func (n *NodeDestroyDeposedResourceInstanceObject) Name() string {
	return fmt.Sprintf("%s (destroy deposed %s)", n.Addr.String(), n.DeposedKey)
}

func (n *NodeDestroyDeposedResourceInstanceObject) DeposedInstanceObjectKey() states.DeposedKey {
	return n.DeposedKey
}

// GraphNodeReferenceable implementation, overriding the one from NodeAbstractResourceInstance
func (n *NodeDestroyDeposedResourceInstanceObject) ReferenceableAddrs() []addrs.Referenceable {
	// Deposed objects don't participate in references.
	return nil
}

// GraphNodeReferencer implementation, overriding the one from NodeAbstractResourceInstance
func (n *NodeDestroyDeposedResourceInstanceObject) References() []*addrs.Reference {
	// We don't evaluate configuration for deposed objects, so they effectively
	// make no references.
	return nil
}

// GraphNodeDestroyer
func (n *NodeDestroyDeposedResourceInstanceObject) DestroyAddr() *addrs.AbsResourceInstance {
	addr := n.ResourceInstanceAddr()
	return &addr
}

// GraphNodeDestroyerCBD
func (n *NodeDestroyDeposedResourceInstanceObject) CreateBeforeDestroy() bool {
	// A deposed instance is always CreateBeforeDestroy by definition, since
	// we use deposed only to handle create-before-destroy.
	return true
}

// GraphNodeDestroyerCBD
func (n *NodeDestroyDeposedResourceInstanceObject) ModifyCreateBeforeDestroy(v bool) error {
	if !v {
		// Should never happen: deposed instances are _always_ create_before_destroy.
		return fmt.Errorf("can't deactivate create_before_destroy for a deposed instance")
	}
	return nil
}

// GraphNodeEvalable impl.
func (n *NodeDestroyDeposedResourceInstanceObject) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	var provider providers.Interface
	var providerSchema *ProviderSchema
	var state *states.ResourceInstanceObject
	var change *plans.ResourceInstanceChange
	var err error

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},
			&EvalReadStateDeposed{
				Addr:           addr.Resource,
				Output:         &state,
				Key:            n.DeposedKey,
				Provider:       &provider,
				ProviderSchema: &providerSchema,
			},
			&EvalDiffDestroy{
				Addr:         addr.Resource,
				ProviderAddr: n.ResolvedProvider,
				State:        &state,
				Output:       &change,
			},
			// Call pre-apply hook
			&EvalApplyPre{
				Addr:   addr.Resource,
				State:  &state,
				Change: &change,
			},
			&EvalApply{
				Addr:           addr.Resource,
				Config:         nil, // No configuration because we are destroying
				State:          &state,
				Change:         &change,
				Provider:       &provider,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
				Output:         &state,
				Error:          &err,
			},
			// Always write the resource back to the state deposed... if it
			// was successfully destroyed it will be pruned. If it was not, it will
			// be caught on the next run.
			&EvalWriteStateDeposed{
				Addr:           addr.Resource,
				Key:            n.DeposedKey,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
				State:          &state,
			},
			&EvalApplyPost{
				Addr:  addr.Resource,
				State: &state,
				Error: &err,
			},
			&EvalReturnError{
				Error: &err,
			},
			&EvalUpdateStateHook{},
		},
	}
}

// GraphNodeDeposer is an optional interface implemented by graph nodes that
// might create a single new deposed object for a specific associated resource
// instance, allowing a caller to optionally pre-allocate a DeposedKey for
// it.
type GraphNodeDeposer interface {
	// SetPreallocatedDeposedKey will be called during graph construction
	// if a particular node must use a pre-allocated deposed key if/when it
	// "deposes" the current object of its associated resource instance.
	SetPreallocatedDeposedKey(key states.DeposedKey)
}

// graphNodeDeposer is an embeddable implementation of GraphNodeDeposer.
// Embed it in a node type to get automatic support for it, and then access
// the field PreallocatedDeposedKey to access any pre-allocated key.
type graphNodeDeposer struct {
	PreallocatedDeposedKey states.DeposedKey
}

func (n *graphNodeDeposer) SetPreallocatedDeposedKey(key states.DeposedKey) {
	n.PreallocatedDeposedKey = key
}
