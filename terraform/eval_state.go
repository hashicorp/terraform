package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
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

// EvalWriteState is an EvalNode implementation that saves the given object
// as the current object for the selected resource instance.
type EvalWriteState struct {
	// Addr is the address of the instance to read state for.
	Addr addrs.ResourceInstance

	// State is the object state to save.
	State **states.ResourceInstanceObject

	// ProviderSchema is the schema for the provider given in ProviderAddr.
	ProviderSchema **ProviderSchema

	// ProviderAddr is the address of the provider configuration that
	// produced the given object.
	ProviderAddr addrs.AbsProviderConfig

	// Dependencies are the inter-resource dependencies to be stored in the
	// state.
	Dependencies *[]addrs.ConfigResource

	// targetState determines which context state we're writing to during plan.
	// The default is the global working state.
	targetState phaseState
}

func (n *EvalWriteState) Eval(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if n.State == nil {
		// Note that a pointer _to_ nil is valid here, indicating the total
		// absense of an object as we'd see during destroy.
		panic("EvalWriteState used with no ResourceInstanceObject")
	}

	absAddr := n.Addr.Absolute(ctx.Path())

	var state *states.SyncState
	switch n.targetState {
	case refreshState:
		log.Printf("[TRACE] EvalWriteState: using RefreshState for %s", absAddr)
		state = ctx.RefreshState()
	default:
		state = ctx.State()
	}

	if n.ProviderAddr.Provider.Type == "" {
		diags = diags.Append(fmt.Errorf("failed to write state for %s: missing provider type", absAddr))
		return diags
	}
	obj := *n.State
	if obj == nil || obj.Value.IsNull() {
		// No need to encode anything: we'll just write it directly.
		state.SetResourceInstanceCurrent(absAddr, nil, n.ProviderAddr)
		log.Printf("[TRACE] EvalWriteState: removing state object for %s", absAddr)
		return diags
	}

	// store the new deps in the state
	if n.Dependencies != nil {
		log.Printf("[TRACE] EvalWriteState: recording %d dependencies for %s", len(*n.Dependencies), absAddr)
		obj.Dependencies = *n.Dependencies
	}

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		// Should never happen, unless our state object is nil
		panic("EvalWriteState used with pointer to nil ProviderSchema object")
	}

	if obj != nil {
		log.Printf("[TRACE] EvalWriteState: writing current state object for %s", absAddr)
	} else {
		log.Printf("[TRACE] EvalWriteState: removing current state object for %s", absAddr)
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

	state.SetResourceInstanceCurrent(absAddr, src, n.ProviderAddr)
	return diags
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

// TODO: test
func (n *EvalDeposeState) Eval(ctx EvalContext) tfdiags.Diagnostics {
	absAddr := n.Addr.Absolute(ctx.Path())
	state := ctx.State()

	var key states.DeposedKey
	if n.ForceKey == states.NotDeposed {
		key = state.DeposeResourceInstanceObject(absAddr)
	} else {
		key = n.ForceKey
		state.DeposeResourceInstanceObjectForceKey(absAddr, key)
	}
	log.Printf("[TRACE] EvalDeposeState: prior object for %s now deposed with key %s", absAddr, key)

	if n.OutputKey != nil {
		*n.OutputKey = key
	}

	return nil
}

// EvalMaybeRestoreDeposedObject is an EvalNode implementation that will
// restore a particular deposed object of the specified resource instance
// to be the "current" object if and only if the instance doesn't currently
// have a current object.
//
// This is intended for use when the create leg of a create before destroy
// fails with no partial new object: if we didn't take any action, the user
// would be left in the unfortunate situation of having no current object
// and the previously-workign object now deposed. This EvalNode causes a
// better outcome by restoring things to how they were before the replace
// operation began.
//
// The create operation may have produced a partial result even though it
// failed and it's important that we don't "forget" that state, so in that
// situation the prior object remains deposed and the partial new object
// remains the current object, allowing the situation to hopefully be
// improved in a subsequent run.
type EvalMaybeRestoreDeposedObject struct {
	Addr addrs.ResourceInstance

	// PlannedChange might be the action we're performing that includes
	// the possiblity of restoring a deposed object. However, it might also
	// be nil. It's here only for use in error messages and must not be
	// used for business logic.
	PlannedChange **plans.ResourceInstanceChange

	// Key is a pointer to the deposed object key that should be forgotten
	// from the state, which must be non-nil.
	Key *states.DeposedKey
}

// TODO: test
func (n *EvalMaybeRestoreDeposedObject) Eval(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	absAddr := n.Addr.Absolute(ctx.Path())
	dk := *n.Key
	state := ctx.State()

	if dk == states.NotDeposed {
		// This should never happen, and so it always indicates a bug.
		// We should evaluate this node only if we've previously deposed
		// an object as part of the same operation.
		if n.PlannedChange != nil && *n.PlannedChange != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Attempt to restore non-existent deposed object",
				fmt.Sprintf(
					"Terraform has encountered a bug where it would need to restore a deposed object for %s without knowing a deposed object key for that object. This occurred during a %s action. This is a bug in Terraform; please report it!",
					absAddr, (*n.PlannedChange).Action,
				),
			))
		} else {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Attempt to restore non-existent deposed object",
				fmt.Sprintf(
					"Terraform has encountered a bug where it would need to restore a deposed object for %s without knowing a deposed object key for that object. This is a bug in Terraform; please report it!",
					absAddr,
				),
			))
		}
		return diags
	}

	restored := state.MaybeRestoreResourceInstanceDeposed(absAddr, dk)
	if restored {
		log.Printf("[TRACE] EvalMaybeRestoreDeposedObject: %s deposed object %s was restored as the current object", absAddr, dk)
	} else {
		log.Printf("[TRACE] EvalMaybeRestoreDeposedObject: %s deposed object %s remains deposed", absAddr, dk)
	}

	return diags
}

// EvalRefreshLifecycle is an EvalNode implementation that updates
// the status of the lifecycle options stored in the state.
// This currently only applies to create_before_destroy.
type EvalRefreshLifecycle struct {
	Addr addrs.AbsResourceInstance

	Config *configs.Resource
	// Prior State
	State **states.ResourceInstanceObject
	// ForceCreateBeforeDestroy indicates a create_before_destroy resource
	// depends on this resource.
	ForceCreateBeforeDestroy bool
}

func (n *EvalRefreshLifecycle) Eval(ctx EvalContext) tfdiags.Diagnostics {
	state := *n.State
	if state == nil {
		// no existing state
		return nil
	}

	// In 0.13 we could be refreshing a resource with no config.
	// We should be operating on managed resource, but check here to be certain
	if n.Config == nil || n.Config.Managed == nil {
		log.Printf("[WARN] EvalRefreshLifecycle: no Managed config value found in instance state for %q", n.Addr)
		return nil
	}

	state.CreateBeforeDestroy = n.Config.Managed.CreateBeforeDestroy || n.ForceCreateBeforeDestroy

	return nil
}
