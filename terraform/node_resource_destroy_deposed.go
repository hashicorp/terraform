package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
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
	_ GraphNodeConfigResource                = (*NodePlanDeposedResourceInstanceObject)(nil)
	_ GraphNodeResourceInstance              = (*NodePlanDeposedResourceInstanceObject)(nil)
	_ GraphNodeReferenceable                 = (*NodePlanDeposedResourceInstanceObject)(nil)
	_ GraphNodeReferencer                    = (*NodePlanDeposedResourceInstanceObject)(nil)
	_ GraphNodeExecutable                    = (*NodePlanDeposedResourceInstanceObject)(nil)
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
func (n *NodePlanDeposedResourceInstanceObject) Execute(ctx EvalContext, op walkOperation) error {
	addr := n.ResourceInstanceAddr()

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	// During the plan walk we always produce a planned destroy change, because
	// destroying is the only supported action for deposed objects.
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	readStateDeposed := &EvalReadStateDeposed{
		Addr:           addr.Resource,
		Output:         &state,
		Key:            n.DeposedKey,
		Provider:       &provider,
		ProviderSchema: &providerSchema,
	}
	_, err = readStateDeposed.Eval(ctx)
	if err != nil {
		return err
	}

	diffDestroy := &EvalDiffDestroy{
		Addr:         addr.Resource,
		ProviderAddr: n.ResolvedProvider,
		DeposedKey:   n.DeposedKey,
		State:        &state,
		Output:       &change,
	}
	diags := diffDestroy.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	writeDiff := &EvalWriteDiff{
		Addr:           addr.Resource,
		DeposedKey:     n.DeposedKey,
		ProviderSchema: &providerSchema,
		Change:         &change,
	}
	diags = writeDiff.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	return nil
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
	_ GraphNodeConfigResource                = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeResourceInstance              = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeDestroyer                     = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeDestroyerCBD                  = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeReferenceable                 = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeReferencer                    = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeExecutable                    = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeProviderConsumer              = (*NodeDestroyDeposedResourceInstanceObject)(nil)
	_ GraphNodeProvisionerConsumer           = (*NodeDestroyDeposedResourceInstanceObject)(nil)
)

func (n *NodeDestroyDeposedResourceInstanceObject) Name() string {
	return fmt.Sprintf("%s (destroy deposed %s)", n.ResourceInstanceAddr(), n.DeposedKey)
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

// GraphNodeExecutable impl.
func (n *NodeDestroyDeposedResourceInstanceObject) Execute(ctx EvalContext, op walkOperation) error {
	var diags tfdiags.Diagnostics

	addr := n.ResourceInstanceAddr().Resource

	var state *states.ResourceInstanceObject
	var change *plans.ResourceInstanceChange
	var applyError error

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	readStateDeposed := &EvalReadStateDeposed{
		Addr:           addr,
		Output:         &state,
		Key:            n.DeposedKey,
		Provider:       &provider,
		ProviderSchema: &providerSchema,
	}
	_, err = readStateDeposed.Eval(ctx)
	if err != nil {
		return err
	}

	diffDestroy := &EvalDiffDestroy{
		Addr:         addr,
		ProviderAddr: n.ResolvedProvider,
		State:        &state,
		Output:       &change,
	}
	diags = diffDestroy.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	// Call pre-apply hook
	applyPre := &EvalApplyPre{
		Addr:   addr,
		State:  &state,
		Change: &change,
	}
	diags = applyPre.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	apply := &EvalApply{
		Addr:           addr,
		Config:         nil, // No configuration because we are destroying
		State:          &state,
		Change:         &change,
		Provider:       &provider,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		Output:         &state,
		Error:          &applyError,
	}
	diags = apply.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	// Always write the resource back to the state deposed. If it
	// was successfully destroyed it will be pruned. If it was not, it will
	// be caught on the next run.
	writeStateDeposed := &EvalWriteStateDeposed{
		Addr:           addr,
		Key:            n.DeposedKey,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		State:          &state,
	}
	_, err = writeStateDeposed.Eval(ctx)
	if err != nil {
		return err
	}

	applyPost := &EvalApplyPost{
		Addr:  addr,
		State: &state,
		Error: &applyError,
	}
	diags = applyPost.Eval(ctx)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}
	if applyError != nil {
		return applyError
	}
	UpdateStateHook(ctx)
	return nil
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
