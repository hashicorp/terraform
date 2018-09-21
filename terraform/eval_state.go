package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/configs"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
)

// EvalReadState is an EvalNode implementation that reads the
// current object for a specific instance in the state.
type EvalReadState struct {
	// Addr is the address of the instance to read state for.
	Addr addrs.ResourceInstance

	// ProviderSchema is the schema for the provider given in Provider.
	ProviderSchema **ProviderSchema

	// Provider is the provider that will subsequently perform actions on
	// the the state object. This is used to perform any schema upgrades
	// that might be required to prepare the stored data for use.
	Provider *providers.Interface

	// Output will be written with a pointer to the retrieved object.
	Output **states.ResourceInstanceObject
}

func (n *EvalReadState) Eval(ctx EvalContext) (interface{}, error) {
	if n.Provider == nil || *n.Provider == nil {
		panic("EvalReadState used with no Provider object")
	}
	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		panic("EvalReadState used with no ProviderSchema object")
	}

	absAddr := n.Addr.Absolute(ctx.Path())
	log.Printf("[TRACE] EvalReadState: reading state for %s", absAddr)

	src := ctx.State().ResourceInstanceObject(absAddr, states.CurrentGen)
	if src == nil {
		// Presumably we only have deposed objects, then.
		log.Printf("[TRACE] EvalReadState: no state present for %s", absAddr)
		return nil, nil
	}

	// TODO: Update n.ResourceTypeSchema to be a providers.Schema and then
	// check the version number here and upgrade if necessary.
	/*
		if src.SchemaVersion < n.ResourceTypeSchema.Version {
			// TODO: Implement schema upgrades
			return nil, fmt.Errorf("schema upgrading is not yet implemented to take state from version %d to version %d", src.SchemaVersion, n.ResourceTypeSchema.Version)
		}
	*/

	schema := (*n.ProviderSchema).SchemaForResourceAddr(n.Addr.ContainingResource())

	obj, err := src.Decode(schema.ImpliedType())
	if err != nil {
		return nil, err
	}

	if n.Output != nil {
		*n.Output = obj
	}
	return obj, nil
}

// EvalReadStateDeposed is an EvalNode implementation that reads the
// deposed InstanceState for a specific resource out of the state
type EvalReadStateDeposed struct {
	// Addr is the address of the instance to read state for.
	Addr addrs.ResourceInstance

	// Key identifies which deposed object we will read.
	Key states.DeposedKey

	// ProviderSchema is the schema for the provider given in Provider.
	ProviderSchema **ProviderSchema

	// Provider is the provider that will subsequently perform actions on
	// the the state object. This is used to perform any schema upgrades
	// that might be required to prepare the stored data for use.
	Provider *providers.Interface

	// Output will be written with a pointer to the retrieved object.
	Output **states.ResourceInstanceObject
}

func (n *EvalReadStateDeposed) Eval(ctx EvalContext) (interface{}, error) {
	key := n.Key
	if key == states.NotDeposed {
		return nil, fmt.Errorf("EvalReadStateDeposed used with no instance key; this is a bug in Terraform and should be reported")
	}
	absAddr := n.Addr.Absolute(ctx.Path())
	log.Printf("[TRACE] EvalReadStateDeposed: reading state for %s deposed object %s", absAddr, n.Key)

	src := ctx.State().ResourceInstanceObject(absAddr, key)
	if src == nil {
		// Presumably we only have deposed objects, then.
		log.Printf("[TRACE] EvalReadStateDeposed: no state present for %s deposed object %s", absAddr, n.Key)
		return nil, nil
	}

	// TODO: Update n.ResourceTypeSchema to be a providers.Schema and then
	// check the version number here and upgrade if necessary.
	/*
		if src.SchemaVersion < n.ResourceTypeSchema.Version {
			// TODO: Implement schema upgrades
			return nil, fmt.Errorf("schema upgrading is not yet implemented to take state from version %d to version %d", src.SchemaVersion, n.ResourceTypeSchema.Version)
		}
	*/

	schema := (*n.ProviderSchema).SchemaForResourceAddr(n.Addr.ContainingResource())
	obj, err := src.Decode(schema.ImpliedType())
	if err != nil {
		return nil, err
	}
	if n.Output != nil {
		*n.Output = obj
	}
	return obj, nil
}

// EvalRequireState is an EvalNode implementation that exits early if the given
// object is null.
type EvalRequireState struct {
	State **states.ResourceInstanceObject
}

func (n *EvalRequireState) Eval(ctx EvalContext) (interface{}, error) {
	if n.State == nil {
		return nil, EvalEarlyExitError{}
	}

	state := *n.State
	if state == nil || state.Value.IsNull() {
		return nil, EvalEarlyExitError{}
	}

	return nil, nil
}

// EvalUpdateStateHook is an EvalNode implementation that calls the
// PostStateUpdate hook with the current state.
type EvalUpdateStateHook struct{}

func (n *EvalUpdateStateHook) Eval(ctx EvalContext) (interface{}, error) {
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
	if err != nil {
		return nil, err
	}

	return nil, nil
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
}

func (n *EvalWriteState) Eval(ctx EvalContext) (interface{}, error) {
	if n.State == nil {
		// Note that a pointer _to_ nil is valid here, indicating the total
		// absense of an object as we'd see during destroy.
		panic("EvalWriteState used with no ResourceInstanceObject")
	}

	absAddr := n.Addr.Absolute(ctx.Path())
	state := ctx.State()

	obj := *n.State
	if obj == nil || obj.Value.IsNull() {
		// No need to encode anything: we'll just write it directly.
		state.SetResourceInstanceCurrent(absAddr, nil, n.ProviderAddr)
		log.Printf("[TRACE] EvalWriteState: removing state object for %s", absAddr)
		return nil, nil
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

	// TODO: Update this to use providers.Schema and populate the real
	// schema version in the second argument to Encode below.
	schema := (*n.ProviderSchema).SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// It shouldn't be possible to get this far in any real scenario
		// without a schema, but we might end up here in contrived tests that
		// fail to set up their world properly.
		return nil, fmt.Errorf("failed to encode %s in state: no resource type schema available", absAddr)
	}
	src, err := obj.Encode(schema.ImpliedType(), 0)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s in state: %s", absAddr, err)
	}

	state.SetResourceInstanceCurrent(absAddr, src, n.ProviderAddr)
	return nil, nil
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

func (n *EvalWriteStateDeposed) Eval(ctx EvalContext) (interface{}, error) {
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
		return nil, fmt.Errorf("can't save deposed object for %s without a deposed key; this is a bug in Terraform that should be reported", absAddr)
	}

	obj := *n.State
	if obj == nil {
		// No need to encode anything: we'll just write it directly.
		state.SetResourceInstanceDeposed(absAddr, key, nil, n.ProviderAddr)
		log.Printf("[TRACE] EvalWriteStateDeposed: removing state object for %s deposed %s", absAddr, key)
		return nil, nil
	}
	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		// Should never happen, unless our state object is nil
		panic("EvalWriteStateDeposed used with no ProviderSchema object")
	}

	// TODO: Update this to use providers.Schema and populate the real
	// schema version in the second argument to Encode below.
	schema := (*n.ProviderSchema).SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// It shouldn't be possible to get this far in any real scenario
		// without a schema, but we might end up here in contrived tests that
		// fail to set up their world properly.
		return nil, fmt.Errorf("failed to encode %s in state: no resource type schema available", absAddr)
	}
	src, err := obj.Encode(schema.ImpliedType(), 0)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s in state: %s", absAddr, err)
	}

	log.Printf("[TRACE] EvalWriteStateDeposed: writing state object for %s deposed %s", absAddr, key)
	state.SetResourceInstanceDeposed(absAddr, key, src, n.ProviderAddr)
	return nil, nil
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
func (n *EvalDeposeState) Eval(ctx EvalContext) (interface{}, error) {
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

	return nil, nil
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

	// Key is a pointer to the deposed object key that should be forgotten
	// from the state, which must be non-nil.
	Key *states.DeposedKey
}

// TODO: test
func (n *EvalMaybeRestoreDeposedObject) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())
	dk := *n.Key
	state := ctx.State()

	restored := state.MaybeRestoreResourceInstanceDeposed(absAddr, dk)
	if restored {
		log.Printf("[TRACE] EvalMaybeRestoreDeposedObject: %s deposed object %s was restored as the current object", absAddr, dk)
	} else {
		log.Printf("[TRACE] EvalMaybeRestoreDeposedObject: %s deposed object %s remains deposed", absAddr, dk)
	}

	return nil, nil
}

// EvalWriteResourceState is an EvalNode implementation that ensures that
// a suitable resource-level state record is present in the state, if that's
// required for the "each mode" of that resource.
//
// This is important primarily for the situation where count = 0, since this
// eval is the only change we get to set the resource "each mode" to list
// in that case, allowing expression evaluation to see it as a zero-element
// list rather than as not set at all.
type EvalWriteResourceState struct {
	Addr         addrs.Resource
	Config       *configs.Resource
	ProviderAddr addrs.AbsProviderConfig
}

// TODO: test
func (n *EvalWriteResourceState) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics
	absAddr := n.Addr.Absolute(ctx.Path())
	state := ctx.State()

	count, countDiags := evaluateResourceCountExpression(n.Config.Count, ctx)
	diags = diags.Append(countDiags)
	if countDiags.HasErrors() {
		return nil, diags.Err()
	}

	// Currently we ony support NoEach and EachList, because for_each support
	// is not fully wired up across Terraform. Once for_each support is added,
	// we'll need to handle that here too, setting states.EachMap if the
	// assigned expression is a map.
	eachMode := states.NoEach
	if count >= 0 { // -1 signals "count not set"
		eachMode = states.EachList
	}

	// This method takes care of all of the business logic of updating this
	// while ensuring that any existing instances are preserved, etc.
	state.SetResourceMeta(absAddr, eachMode, n.ProviderAddr)

	return nil, nil
}
