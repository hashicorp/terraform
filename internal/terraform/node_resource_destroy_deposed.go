package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
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

	// skipRefresh indicates that we should skip refreshing individual instances
	skipRefresh bool

	// skipPlanChanges indicates we should skip trying to plan change actions
	// for any instances.
	skipPlanChanges bool
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
func (n *NodePlanDeposedResourceInstanceObject) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	log.Printf("[TRACE] NodePlanDeposedResourceInstanceObject: planning %s deposed object %s", n.Addr, n.DeposedKey)

	// Read the state for the deposed resource instance
	state, err := n.readResourceInstanceStateDeposed(ctx, n.Addr, n.DeposedKey)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// Note any upgrades that readResourceInstanceState might've done in the
	// prevRunState, so that it'll conform to current schema.
	diags = diags.Append(n.writeResourceInstanceStateDeposed(ctx, n.DeposedKey, state, prevRunState))
	if diags.HasErrors() {
		return diags
	}
	// Also the refreshState, because that should still reflect schema upgrades
	// even if not refreshing.
	diags = diags.Append(n.writeResourceInstanceStateDeposed(ctx, n.DeposedKey, state, refreshState))
	if diags.HasErrors() {
		return diags
	}

	// We don't refresh during the planDestroy walk, since that is only adding
	// the destroy changes to the plan and the provider will not be configured
	// at this point. The other nodes use separate types for plan and destroy,
	// while deposed instances are always a destroy operation, so the logic
	// here is a bit overloaded.
	if !n.skipRefresh && op != walkPlanDestroy {
		// Refresh this object even though it is going to be destroyed, in
		// case it's already been deleted outside of Terraform. If this is a
		// normal plan, providers expect a Read request to remove missing
		// resources from the plan before apply, and may not handle a missing
		// resource during Delete correctly. If this is a simple refresh,
		// Terraform is expected to remove the missing resource from the state
		// entirely
		refreshedState, refreshDiags := n.refresh(ctx, n.DeposedKey, state)
		diags = diags.Append(refreshDiags)
		if diags.HasErrors() {
			return diags
		}

		diags = diags.Append(n.writeResourceInstanceStateDeposed(ctx, n.DeposedKey, refreshedState, refreshState))
		if diags.HasErrors() {
			return diags
		}

		// If we refreshed then our subsequent planning should be in terms of
		// the new object, not the original object.
		state = refreshedState
	}

	if !n.skipPlanChanges {
		var change *plans.ResourceInstanceChange
		change, destroyPlanDiags := n.planDestroy(ctx, state, n.DeposedKey)
		diags = diags.Append(destroyPlanDiags)
		if diags.HasErrors() {
			return diags
		}

		// NOTE: We don't check prevent_destroy for deposed objects, even
		// though we would do so here for a "current" object, because
		// if we've reached a point where an object is already deposed then
		// we've already planned and partially-executed a create_before_destroy
		// replace and we would've checked prevent_destroy at that point. We're
		// now just need to get the deposed object destroyed, because there
		// should be a new object already serving as its replacement.

		diags = diags.Append(n.writeChange(ctx, change, n.DeposedKey))
		if diags.HasErrors() {
			return diags
		}

		diags = diags.Append(n.writeResourceInstanceStateDeposed(ctx, n.DeposedKey, nil, workingState))
	} else {
		// The working state should at least be updated with the result
		// of upgrading and refreshing from above.
		diags = diags.Append(n.writeResourceInstanceStateDeposed(ctx, n.DeposedKey, state, workingState))
	}

	return diags
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
func (n *NodeDestroyDeposedResourceInstanceObject) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	var change *plans.ResourceInstanceChange

	// Read the state for the deposed resource instance
	state, err := n.readResourceInstanceStateDeposed(ctx, n.Addr, n.DeposedKey)
	if err != nil {
		return diags.Append(err)
	}

	if state == nil {
		diags = diags.Append(fmt.Errorf("missing deposed state for %s (%s)", n.Addr, n.DeposedKey))
		return diags
	}

	change, destroyPlanDiags := n.planDestroy(ctx, state, n.DeposedKey)
	diags = diags.Append(destroyPlanDiags)
	if diags.HasErrors() {
		return diags
	}

	// Call pre-apply hook
	diags = diags.Append(n.preApplyHook(ctx, change))
	if diags.HasErrors() {
		return diags
	}

	// we pass a nil configuration to apply because we are destroying
	state, applyDiags := n.apply(ctx, state, change, nil, false)
	diags = diags.Append(applyDiags)
	// don't return immediately on errors, we need to handle the state

	// Always write the resource back to the state deposed. If it
	// was successfully destroyed it will be pruned. If it was not, it will
	// be caught on the next run.
	writeDiags := n.writeResourceInstanceState(ctx, state)
	diags.Append(writeDiags)
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.postApplyHook(ctx, state, diags.Err()))

	return diags.Append(updateStateHook(ctx))
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

func (n *NodeDestroyDeposedResourceInstanceObject) writeResourceInstanceState(ctx EvalContext, obj *states.ResourceInstanceObject) error {
	absAddr := n.Addr
	key := n.DeposedKey
	state := ctx.State()

	if key == states.NotDeposed {
		// should never happen
		return fmt.Errorf("can't save deposed object for %s without a deposed key; this is a bug in Terraform that should be reported", absAddr)
	}

	if obj == nil {
		// No need to encode anything: we'll just write it directly.
		state.SetResourceInstanceDeposed(absAddr, key, nil, n.ResolvedProvider)
		log.Printf("[TRACE] writeResourceInstanceStateDeposed: removing state object for %s deposed %s", absAddr, key)
		return nil
	}

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}
	if providerSchema == nil {
		// Should never happen, unless our state object is nil
		panic("writeResourceInstanceStateDeposed used with no ProviderSchema object")
	}

	schema, currentVersion := providerSchema.SchemaForResourceAddr(absAddr.ContainingResource().Resource)
	if schema == nil {
		// It shouldn't be possible to get this far in any real scenario
		// without a schema, but we might end up here in contrived tests that
		// fail to set up their world properly.
		return fmt.Errorf("failed to encode %s in state: no resource type schema available", absAddr)
	}
	src, err := obj.Encode(schema.ImpliedType(), currentVersion)
	if err != nil {
		return fmt.Errorf("failed to encode %s in state: %s", absAddr, err)
	}

	log.Printf("[TRACE] writeResourceInstanceStateDeposed: writing state object for %s deposed %s", absAddr, key)
	state.SetResourceInstanceDeposed(absAddr, key, src, n.ResolvedProvider)
	return nil
}
