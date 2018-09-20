package states

import (
	"log"
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
)

// SyncState is a wrapper around State that provides concurrency-safe access to
// various common operations that occur during a Terraform graph walk, or other
// similar concurrent contexts.
//
// When a SyncState wrapper is in use, no concurrent direct access to the
// underlying objects is permitted unless the caller first acquires an explicit
// lock, using the Lock and Unlock methods. Most callers should _not_
// explicitly lock, and should instead use the other methods of this type that
// handle locking automatically.
//
// Since SyncState is able to safely consolidate multiple updates into a single
// atomic operation, many of its methods are at a higher level than those
// of the underlying types, and operate on the state as a whole rather than
// on individual sub-structures of the state.
//
// SyncState can only protect against races within its own methods. It cannot
// provide any guarantees about the order in which concurrent operations will
// be processed, so callers may still need to employ higher-level techniques
// for ensuring correct operation sequencing, such as building and walking
// a dependency graph.
type SyncState struct {
	state *State
	lock  sync.RWMutex
}

// Module returns a snapshot of the state of the module instance with the given
// address, or nil if no such module is tracked.
//
// The return value is a pointer to a copy of the module state, which the
// caller may then freely access and mutate. However, since the module state
// tends to be a large data structure with many child objects, where possible
// callers should prefer to use a more granular accessor to access a child
// module directly, and thus reduce the amount of copying required.
func (s *SyncState) Module(addr addrs.ModuleInstance) *Module {
	s.lock.RLock()
	ret := s.state.Module(addr).DeepCopy()
	s.lock.RUnlock()
	return ret
}

// OutputValue returns a snapshot of the state of the output value with the
// given address, or nil if no such output value is tracked.
//
// The return value is a pointer to a copy of the output value state, which the
// caller may then freely access and mutate.
func (s *SyncState) OutputValue(addr addrs.AbsOutputValue) *OutputValue {
	s.lock.RLock()
	ret := s.state.OutputValue(addr).DeepCopy()
	s.lock.RUnlock()
	return ret
}

// SetOutputValue writes a given output value into the state, overwriting
// any existing value of the same name.
//
// If the module containing the output is not yet tracked in state then it
// be added as a side-effect.
func (s *SyncState) SetOutputValue(addr addrs.AbsOutputValue, value cty.Value, sensitive bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ms := s.state.EnsureModule(addr.Module)
	ms.SetOutputValue(addr.OutputValue.Name, value, sensitive)
}

// RemoveOutputValue removes the stored value for the output value with the
// given address.
//
// If this results in its containing module being empty, the module will be
// pruned from the state as a side-effect.
func (s *SyncState) RemoveOutputValue(addr addrs.AbsOutputValue) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ms := s.state.Module(addr.Module)
	if ms == nil {
		return
	}
	ms.RemoveOutputValue(addr.OutputValue.Name)
	s.maybePruneModule(addr.Module)
}

// LocalValue returns the current value associated with the given local value
// address.
func (s *SyncState) LocalValue(addr addrs.AbsLocalValue) cty.Value {
	s.lock.RLock()
	// cty.Value is immutable, so we don't need any extra copying here.
	ret := s.state.LocalValue(addr)
	s.lock.RUnlock()
	return ret
}

// SetLocalValue writes a given output value into the state, overwriting
// any existing value of the same name.
//
// If the module containing the local value is not yet tracked in state then it
// will be added as a side-effect.
func (s *SyncState) SetLocalValue(addr addrs.AbsLocalValue, value cty.Value) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ms := s.state.EnsureModule(addr.Module)
	ms.SetLocalValue(addr.LocalValue.Name, value)
}

// RemoveLocalValue removes the stored value for the local value with the
// given address.
//
// If this results in its containing module being empty, the module will be
// pruned from the state as a side-effect.
func (s *SyncState) RemoveLocalValue(addr addrs.AbsLocalValue) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ms := s.state.Module(addr.Module)
	if ms == nil {
		return
	}
	ms.RemoveLocalValue(addr.LocalValue.Name)
	s.maybePruneModule(addr.Module)
}

// Resource returns a snapshot of the state of the resource with the given
// address, or nil if no such resource is tracked.
//
// The return value is a pointer to a copy of the resource state, which the
// caller may then freely access and mutate.
func (s *SyncState) Resource(addr addrs.AbsResource) *Resource {
	s.lock.RLock()
	ret := s.state.Resource(addr).DeepCopy()
	s.lock.RUnlock()
	return ret
}

// ResourceInstance returns a snapshot of the state the resource instance with
// the given address, or nil if no such instance is tracked.
//
// The return value is a pointer to a copy of the instance state, which the
// caller may then freely access and mutate.
func (s *SyncState) ResourceInstance(addr addrs.AbsResourceInstance) *ResourceInstance {
	s.lock.RLock()
	ret := s.state.ResourceInstance(addr).DeepCopy()
	s.lock.RUnlock()
	return ret
}

// ResourceInstanceObject returns a snapshot of the current instance object
// of the given generation belonging to the instance with the given address,
// or nil if no such object is tracked..
//
// The return value is a pointer to a copy of the object, which the caller may
// then freely access and mutate.
func (s *SyncState) ResourceInstanceObject(addr addrs.AbsResourceInstance, gen Generation) *ResourceInstanceObjectSrc {
	s.lock.RLock()
	defer s.lock.RUnlock()

	inst := s.state.ResourceInstance(addr)
	if inst == nil {
		return nil
	}
	return inst.GetGeneration(gen)
}

// SetResourceMeta updates the resource-level metadata for the resource at
// the given address, creating the containing module state and resource state
// as a side-effect if not already present.
func (s *SyncState) SetResourceMeta(addr addrs.AbsResource, eachMode EachMode, provider addrs.AbsProviderConfig) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ms := s.state.EnsureModule(addr.Module)
	ms.SetResourceMeta(addr.Resource, eachMode, provider)
}

// RemoveResource removes the entire state for the given resource, taking with
// it any instances associated with the resource. This should generally be
// called only for resource objects whose instances have all been destroyed,
// but that is not enforced by this method.
func (s *SyncState) RemoveResource(addr addrs.AbsResource) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ms := s.state.EnsureModule(addr.Module)
	ms.RemoveResource(addr.Resource)
	s.maybePruneModule(addr.Module)
}

// MaybeFixUpResourceInstanceAddressForCount deals with the situation where a
// resource has changed from having "count" set to not set, or vice-versa, and
// so we need to rename the zeroth instance key to no key at all, or vice-versa.
//
// Set countEnabled to true if the resource has count set in its new
// configuration, or false if it does not.
//
// The state is modified in-place if necessary, moving a resource instance
// between the two addresses. The return value is true if a change was made,
// and false otherwise.
func (s *SyncState) MaybeFixUpResourceInstanceAddressForCount(addr addrs.AbsResource, countEnabled bool) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	ms := s.state.Module(addr.Module)
	if ms == nil {
		return false
	}

	relAddr := addr.Resource
	rs := ms.Resource(relAddr)
	if rs == nil {
		return false
	}
	huntKey := addrs.NoKey
	replaceKey := addrs.InstanceKey(addrs.IntKey(0))
	if !countEnabled {
		huntKey, replaceKey = replaceKey, huntKey
	}

	is, exists := rs.Instances[huntKey]
	if !exists {
		return false
	}

	if _, exists := rs.Instances[replaceKey]; exists {
		// If the replacement key also exists then we'll do nothing and keep both.
		return false
	}

	// If we get here then we need to "rename" from hunt to replace
	rs.Instances[replaceKey] = is
	delete(rs.Instances, huntKey)
	return true
}

// SetResourceInstanceCurrent saves the given instance object as the current
// generation of the resource instance with the given address, simulataneously
// updating the recorded provider configuration address, dependencies, and
// resource EachMode.
//
// Any existing current instance object for the given resource is overwritten.
// Set obj to nil to remove the primary generation object altogether. If there
// are no deposed objects then the instance as a whole will be removed, which
// may in turn also remove the containing module if it becomes empty.
//
// The caller must ensure that the given ResourceInstanceObject is not
// concurrently mutated during this call, but may be freely used again once
// this function returns.
//
// The provider address and "each mode" are resource-wide settings and so they
// are updated for all other instances of the same resource as a side-effect of
// this call.
//
// If the containing module for this resource or the resource itself are not
// already tracked in state then they will be added as a side-effect.
func (s *SyncState) SetResourceInstanceCurrent(addr addrs.AbsResourceInstance, obj *ResourceInstanceObjectSrc, provider addrs.AbsProviderConfig) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ms := s.state.EnsureModule(addr.Module)
	ms.SetResourceInstanceCurrent(addr.Resource, obj.DeepCopy(), provider)
	s.maybePruneModule(addr.Module)
}

// SetResourceInstanceDeposed saves the given instance object as a deposed
// generation of the resource instance with the given address and deposed key.
//
// Call this method only for pre-existing deposed objects that already have
// a known DeposedKey. For example, this method is useful if reloading objects
// that were persisted to a state file. To mark the current object as deposed,
// use DeposeResourceInstanceObject instead.
//
// The caller must ensure that the given ResourceInstanceObject is not
// concurrently mutated during this call, but may be freely used again once
// this function returns.
//
// The resource that contains the given instance must already exist in the
// state, or this method will panic. Use Resource to check first if its
// presence is not already guaranteed.
//
// Any existing current instance object for the given resource and deposed key
// is overwritten. Set obj to nil to remove the deposed object altogether. If
// the instance is left with no objects after this operation then it will
// be removed from its containing resource altogether.
//
// If the containing module for this resource or the resource itself are not
// already tracked in state then they will be added as a side-effect.
func (s *SyncState) SetResourceInstanceDeposed(addr addrs.AbsResourceInstance, key DeposedKey, obj *ResourceInstanceObjectSrc, provider addrs.AbsProviderConfig) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ms := s.state.EnsureModule(addr.Module)
	ms.SetResourceInstanceDeposed(addr.Resource, key, obj.DeepCopy(), provider)
	s.maybePruneModule(addr.Module)
}

// DeposeResourceInstanceObject moves the current instance object for the
// given resource instance address into the deposed set, leaving the instance
// without a current object.
//
// The return value is the newly-allocated deposed key, or NotDeposed if the
// given instance is already lacking a current object.
//
// If the containing module for this resource or the resource itself are not
// already tracked in state then there cannot be a current object for the
// given instance, and so NotDeposed will be returned without modifying the
// state at all.
func (s *SyncState) DeposeResourceInstanceObject(addr addrs.AbsResourceInstance) DeposedKey {
	s.lock.Lock()
	defer s.lock.Unlock()

	ms := s.state.Module(addr.Module)
	if ms == nil {
		return NotDeposed
	}

	return ms.deposeResourceInstanceObject(addr.Resource, NotDeposed)
}

// DeposeResourceInstanceObjectForceKey is like DeposeResourceInstanceObject
// but uses a pre-allocated key. It's the caller's responsibility to ensure
// that there aren't any races to use a particular key; this method will panic
// if the given key is already in use.
func (s *SyncState) DeposeResourceInstanceObjectForceKey(addr addrs.AbsResourceInstance, forcedKey DeposedKey) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if forcedKey == NotDeposed {
		// Usage error: should use DeposeResourceInstanceObject in this case
		panic("DeposeResourceInstanceObjectForceKey called without forced key")
	}

	ms := s.state.Module(addr.Module)
	if ms == nil {
		return // Nothing to do, since there can't be any current object either.
	}

	ms.deposeResourceInstanceObject(addr.Resource, forcedKey)
}

// ForgetResourceInstanceDeposed removes the record of the deposed object with
// the given address and key, if present. If not present, this is a no-op.
func (s *SyncState) ForgetResourceInstanceDeposed(addr addrs.AbsResourceInstance, key DeposedKey) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ms := s.state.Module(addr.Module)
	if ms == nil {
		return
	}
	ms.ForgetResourceInstanceDeposed(addr.Resource, key)
	s.maybePruneModule(addr.Module)
}

// RemovePlannedResourceInstanceObjects removes from the state any resource
// instance objects that have the status ObjectPlanned, indiciating that they
// are just transient placeholders created during planning.
//
// Note that this does not restore any "ready" or "tainted" object that might
// have been present before the planned object was written. The only real use
// for this method is in preparing the state created during a refresh walk,
// where we run the planning step for certain instances just to create enough
// information to allow correct expression evaluation within provider and
// data resource blocks. Discarding planned instances in that case is okay
// because the refresh phase only creates planned objects to stand in for
// objects that don't exist yet, and thus the planned object must have been
// absent before by definition.
func (s *SyncState) RemovePlannedResourceInstanceObjects() {
	// TODO: Merge together the refresh and plan phases into a single walk,
	// so we can remove the need to create this "partial plan" during refresh
	// that we then need to clean up before proceeding.

	s.lock.Lock()
	defer s.lock.Unlock()

	for _, ms := range s.state.Modules {
		moduleAddr := ms.Addr

		for _, rs := range ms.Resources {
			resAddr := rs.Addr

			for ik, is := range rs.Instances {
				instAddr := resAddr.Instance(ik)

				if is.Current != nil && is.Current.Status == ObjectPlanned {
					// Setting the current instance to nil removes it from the
					// state altogether if there are not also deposed instances.
					ms.SetResourceInstanceCurrent(instAddr, nil, rs.ProviderConfig)
				}

				for dk, obj := range is.Deposed {
					// Deposed objects should never be "planned", but we'll
					// do this anyway for the sake of completeness.
					if obj.Status == ObjectPlanned {
						ms.ForgetResourceInstanceDeposed(instAddr, dk)
					}
				}
			}
		}

		// We may have deleted some objects, which means that we may have
		// left a module empty, and so we must prune to preserve the invariant
		// that only the root module is allowed to be empty.
		s.maybePruneModule(moduleAddr)
	}
}

// Lock acquires an explicit lock on the state, allowing direct read and write
// access to the returned state object. The caller must call Unlock once
// access is no longer needed, and then immediately discard the state pointer
// pointer.
//
// Most callers should not use this. Instead, use the concurrency-safe
// accessors and mutators provided directly on SyncState.
func (s *SyncState) Lock() *State {
	s.lock.Lock()
	return s.state
}

// Unlock releases a lock previously acquired by Lock, at which point the
// caller must cease all use of the state pointer that was returned.
//
// Do not call this method except to end an explicit lock acquired by
// Lock. If a caller calls Unlock without first holding the lock, behavior
// is undefined.
func (s *SyncState) Unlock() {
	s.lock.Unlock()
}

// maybePruneModule will remove a module from the state altogether if it is
// empty, unless it's the root module which must always be present.
//
// This helper method is not concurrency-safe on its own, so must only be
// called while the caller is already holding the lock for writing.
func (s *SyncState) maybePruneModule(addr addrs.ModuleInstance) {
	if addr.IsRoot() {
		// We never prune the root.
		return
	}

	ms := s.state.Module(addr)
	if ms == nil {
		return
	}

	if ms.empty() {
		log.Printf("[TRACE] states.SyncState: pruning %s because it is empty", addr)
		s.state.RemoveModule(addr)
	}
}
