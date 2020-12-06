package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

type phaseState int

const (
	workingState phaseState = iota
	refreshState
)

// UpdateStateHook calls the PostStateUpdate hook with the current state.
func UpdateStateHook(ctx EvalContext) error {
	// In principle we could grab the lock here just long enough to take a
	// deep copy and then pass that to our hooks below, but we'll instead
	// hold the hook for the duration to avoid the potential confusing
	// situation of us racing to call PostStateUpdate concurrently with
	// different state snapshots.
	stateSync := ctx.State()
	state := stateSync.Lock().DeepCopy()
	defer stateSync.Unlock()

	// Call the hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostStateUpdate(state)
	})
	return err
}

// EvalWriteStateDeposed is an EvalNode implementation that writes
// an InstanceState out to the Deposed list of a resource in the state.
type EvalWriteStateDeposed struct {
	// Addr is the address of the instance to read state for.
	Addr addrs.ResourceInstance

	// Key indicates which deposed object to write to.
	Key states.DeposedKey

	// State is the object state to save.
	State **states.ResourceInstanceObject

	// ProviderSchema is the schema for the provider given in ProviderAddr.
	ProviderSchema **ProviderSchema

	// ProviderAddr is the address of the provider configuration that
	// produced the given object.
	ProviderAddr addrs.AbsProviderConfig
}

func (n *EvalWriteStateDeposed) Eval(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if n.State == nil {
		// Note that a pointer _to_ nil is valid here, indicating the total
		// absense of an object as we'd see during destroy.
		panic("EvalWriteStateDeposed used with no ResourceInstanceObject")
	}

	absAddr := n.Addr.Absolute(ctx.Path())
	key := n.Key
	state := ctx.State()

	if key == states.NotDeposed {
		// should never happen
		diags = diags.Append(fmt.Errorf("can't save deposed object for %s without a deposed key; this is a bug in Terraform that should be reported", absAddr))
		return diags
	}

	obj := *n.State
	if obj == nil {
		// No need to encode anything: we'll just write it directly.
		state.SetResourceInstanceDeposed(absAddr, key, nil, n.ProviderAddr)
		log.Printf("[TRACE] EvalWriteStateDeposed: removing state object for %s deposed %s", absAddr, key)
		return diags
	}
	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		// Should never happen, unless our state object is nil
		panic("EvalWriteStateDeposed used with no ProviderSchema object")
	}

	schema, currentVersion := (*n.ProviderSchema).SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// It shouldn't be possible to get this far in any real scenario
		// without a schema, but we might end up here in contrived tests that
		// fail to set up their world properly.
		diags = diags.Append(fmt.Errorf("failed to encode %s in state: no resource type schema available", absAddr))
		return diags
	}
	src, err := obj.Encode(schema.ImpliedType(), currentVersion)
	if err != nil {
		diags = diags.Append(fmt.Errorf("failed to encode %s in state: %s", absAddr, err))
		return diags
	}

	log.Printf("[TRACE] EvalWriteStateDeposed: writing state object for %s deposed %s", absAddr, key)
	state.SetResourceInstanceDeposed(absAddr, key, src, n.ProviderAddr)
	return diags
}

// EvalDeposeState is an EvalNode implementation that moves the current object
// for the given instance to instead be a deposed object, leaving the instance
// with no current object.
// This is used at the beginning of a create-before-destroy replace action so
// that the create can create while preserving the old state of the
// to-be-destroyed object.
type EvalDeposeState struct {
	Addr addrs.ResourceInstance

	// ForceKey, if a value other than states.NotDeposed, will be used as the
	// key for the newly-created deposed object that results from this action.
	// If set to states.NotDeposed (the zero value), a new unique key will be
	// allocated.
	ForceKey states.DeposedKey

	// OutputKey, if non-nil, will be written with the deposed object key that
	// was generated for the object. This can then be passed to
	// EvalUndeposeState.Key so it knows which deposed instance to forget.
	OutputKey *states.DeposedKey
}
