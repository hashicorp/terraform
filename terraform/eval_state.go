package terraform

import (
	"fmt"

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
	src := ctx.State().ResourceInstanceObject(absAddr, states.CurrentGen)
	if src == nil {
		// Presumably we only have deposed objects, then.
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

	schema := (*n.ProviderSchema).ResourceTypes[absAddr.Resource.Resource.Type]
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
	src := ctx.State().ResourceInstanceObject(absAddr, key)
	if src == nil {
		// Presumably we only have deposed objects, then.
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

	schema := (*n.ProviderSchema).ResourceTypes[absAddr.Resource.Resource.Type]
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
	if obj == nil {
		// No need to encode anything: we'll just write it directly.
		state.SetResourceInstanceCurrent(absAddr, nil, n.ProviderAddr)
		return nil, nil
	}
	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		// Should never happen, unless our state object is nil
		panic("EvalWriteState used with pointer to nil ProviderSchema object")
	}

	// TODO: Update this to use providers.Schema and populate the real
	// schema version in the second argument to Encode below.
	schema := (*n.ProviderSchema).ResourceTypes[absAddr.Resource.Resource.Type]
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
		state.SetResourceInstanceCurrent(absAddr, nil, n.ProviderAddr)
		return nil, nil
	}
	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		// Should never happen, unless our state object is nil
		panic("EvalWriteStateDeposed used with no ProviderSchema object")
	}

	// TODO: Update this to use providers.Schema and populate the real
	// schema version in the second argument to Encode below.
	schema := (*n.ProviderSchema).ResourceTypes[absAddr.Resource.Resource.Type]
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

	// OutputKey, if non-nil, will be written with the deposed object key that
	// was generated for the object. This can then be passed to
	// EvalUndeposeState.Key so it knows which deposed instance to forget.
	OutputKey *states.DeposedKey
}

// TODO: test
func (n *EvalDeposeState) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())
	state := ctx.State()

	key := state.DeposeResourceInstanceObject(absAddr)

	if n.OutputKey != nil {
		*n.OutputKey = key
	}

	return nil, nil
}

// EvalUndeposeState is an EvalNode implementation that forgets a particular
// deposed object from the state, causing Terraform to no longer track it.
//
// Users of this must ensure that the upstream object that the object was
// tracking has been deleted in the remote system before this node is
// evaluated.
type EvalUndeposeState struct {
	Addr addrs.ResourceInstance

	// Key is a pointer to the deposed object key that should be forgotten
	// from the state, which must be non-nil.
	Key *states.DeposedKey
}

// TODO: test
func (n *EvalUndeposeState) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())
	state := ctx.State()

	state.ForgetResourceInstanceDeposed(absAddr, *n.Key)

	return nil, nil
}
