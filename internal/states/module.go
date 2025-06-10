// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package states

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

// Module is a container for the states of objects within a particular module.
type Module struct {
	Addr addrs.ModuleInstance

	// Resources contains the state for each resource. The keys in this map are
	// an implementation detail and must not be used by outside callers.
	Resources map[string]*Resource
}

// NewModule constructs an empty module state for the given module address.
func NewModule(addr addrs.ModuleInstance) *Module {
	return &Module{
		Addr:      addr,
		Resources: map[string]*Resource{},
	}
}

// Resource returns the state for the resource with the given address within
// the receiving module state, or nil if the requested resource is not tracked
// in the state.
func (ms *Module) Resource(addr addrs.Resource) *Resource {
	return ms.Resources[addr.String()]
}

// ResourceInstance returns the state for the resource instance with the given
// address within the receiving module state, or nil if the requested instance
// is not tracked in the state.
func (ms *Module) ResourceInstance(addr addrs.ResourceInstance) *ResourceInstance {
	rs := ms.Resource(addr.Resource)
	if rs == nil {
		return nil
	}
	return rs.Instance(addr.Key)
}

// SetResourceProvider updates the resource-level metadata for the resource
// with the given address, creating the resource state for it if it doesn't
// already exist.
func (ms *Module) SetResourceProvider(addr addrs.Resource, provider addrs.AbsProviderConfig) {
	rs := ms.Resource(addr)
	if rs == nil {
		rs = &Resource{
			Addr:      addr.Absolute(ms.Addr),
			Instances: map[addrs.InstanceKey]*ResourceInstance{},
		}
		ms.Resources[addr.String()] = rs
	}

	rs.ProviderConfig = provider
}

// RemoveResource removes the entire state for the given resource, taking with
// it any instances associated with the resource. This should generally be
// called only for resource objects whose instances have all been destroyed.
func (ms *Module) RemoveResource(addr addrs.Resource) {
	delete(ms.Resources, addr.String())
}

// SetResourceInstanceCurrent saves the given instance object as the current
// generation of the resource instance with the given address, simultaneously
// updating the recorded provider configuration address and dependencies.
//
// Any existing current instance object for the given resource is overwritten.
// Set obj to nil to remove the primary generation object altogether. If there
// are no deposed objects then the instance will be removed altogether.
//
// The provider address is a resource-wide setting and is updated for all other
// instances of the same resource as a side-effect of this call.
func (ms *Module) SetResourceInstanceCurrent(addr addrs.ResourceInstance, obj *ResourceInstanceObjectSrc, provider addrs.AbsProviderConfig) {
	rs := ms.Resource(addr.Resource)
	// if the resource is nil and the object is nil, don't do anything!
	// you'll probably just cause issues
	if obj == nil && rs == nil {
		return
	}
	if obj == nil && rs != nil {
		// does the resource have any other objects?
		// if not then delete the whole resource
		if len(rs.Instances) == 0 {
			delete(ms.Resources, addr.Resource.String())
			return
		}
		// check for an existing resource, now that we've ensured that rs.Instances is more than 0/not nil
		is := rs.Instance(addr.Key)
		if is == nil {
			// if there is no instance on the resource with this address and obj is nil, return and change nothing
			return
		}
		// if we have an instance, update the current
		is.Current = obj
		if !is.HasObjects() {
			// If we have no objects at all then we'll clean up.
			delete(rs.Instances, addr.Key)
			// Delete the resource if it has no instances, but only if NoEach
			if len(rs.Instances) == 0 {
				delete(ms.Resources, addr.Resource.String())
				return
			}
		}
		// Nothing more to do here, so return!
		return
	}
	if rs == nil && obj != nil {
		// We don't have have a resource so make one, which is a side effect of setResourceMeta
		ms.SetResourceProvider(addr.Resource, provider)
		// now we have a resource! so update the rs value to point to it
		rs = ms.Resource(addr.Resource)
	}
	// Get our instance from the resource; it could be there or not at this point
	is := rs.Instance(addr.Key)
	if is == nil {
		// if we don't have a resource, create one and add to the instances
		is = rs.CreateInstance(addr.Key)
		// update the resource meta because we have a new
		ms.SetResourceProvider(addr.Resource, provider)
	}
	// Update the resource's ProviderConfig, in case the provider has updated
	rs.ProviderConfig = provider
	is.Current = obj
}

// SetResourceInstanceDeposed saves the given instance object as a deposed
// generation of the resource instance with the given address and deposed key.
//
// Call this method only for pre-existing deposed objects that already have
// a known DeposedKey. For example, this method is useful if reloading objects
// that were persisted to a state file. To mark the current object as deposed,
// use DeposeResourceInstanceObject instead.
//
// The resource that contains the given instance must already exist in the
// state, or this method will panic. Use Resource to check first if its
// presence is not already guaranteed.
//
// Any existing current instance object for the given resource and deposed key
// is overwritten. Set obj to nil to remove the deposed object altogether. If
// the instance is left with no objects after this operation then it will
// be removed from its containing resource altogether.
func (ms *Module) SetResourceInstanceDeposed(addr addrs.ResourceInstance, key DeposedKey, obj *ResourceInstanceObjectSrc, provider addrs.AbsProviderConfig) {
	ms.SetResourceProvider(addr.Resource, provider)

	rs := ms.Resource(addr.Resource)
	is := rs.EnsureInstance(addr.Key)
	if obj != nil {
		is.Deposed[key] = obj
	} else {
		delete(is.Deposed, key)
	}

	if !is.HasObjects() {
		// If we have no objects at all then we'll clean up.
		delete(rs.Instances, addr.Key)
	}
	if len(rs.Instances) == 0 {
		// Also clean up if we only expect to have one instance anyway
		// and there are none. We leave the resource behind if an each mode
		// is active because an empty list or map of instances is a valid state.
		delete(ms.Resources, addr.Resource.String())
	}
}

// ForgetResourceInstanceAll removes the record of all objects associated with
// the specified resource instance, if present. If not present, this is a no-op.
func (ms *Module) ForgetResourceInstanceAll(addr addrs.ResourceInstance) {
	rs := ms.Resource(addr.Resource)
	if rs == nil {
		return
	}
	delete(rs.Instances, addr.Key)

	if len(rs.Instances) == 0 {
		// Also clean up if we only expect to have one instance anyway
		// and there are none. We leave the resource behind if an each mode
		// is active because an empty list or map of instances is a valid state.
		delete(ms.Resources, addr.Resource.String())
	}
}

// ForgetResourceInstanceCurrent removes the record of the current object with
// the given address, if present. If not present, this is a no-op.
func (ms *Module) ForgetResourceInstanceCurrent(addr addrs.ResourceInstance) {
	rs := ms.Resource(addr.Resource)
	if rs == nil {
		return
	}
	is := rs.Instance(addr.Key)
	if is == nil {
		return
	}

	is.Current = nil

	if !is.HasObjects() {
		// If we have no objects at all then we'll clean up.
		delete(rs.Instances, addr.Key)
	}
	if len(rs.Instances) == 0 {
		// Also clean up if we only expect to have one instance anyway
		// and there are none. We leave the resource behind if an each mode
		// is active because an empty list or map of instances is a valid state.
		delete(ms.Resources, addr.Resource.String())
	}
}

// ForgetResourceInstanceDeposed removes the record of the deposed object with
// the given address and key, if present. If not present, this is a no-op.
func (ms *Module) ForgetResourceInstanceDeposed(addr addrs.ResourceInstance, key DeposedKey) {
	rs := ms.Resource(addr.Resource)
	if rs == nil {
		return
	}
	is := rs.Instance(addr.Key)
	if is == nil {
		return
	}
	delete(is.Deposed, key)

	if !is.HasObjects() {
		// If we have no objects at all then we'll clean up.
		delete(rs.Instances, addr.Key)
	}
	if len(rs.Instances) == 0 {
		// Also clean up if we only expect to have one instance anyway
		// and there are none. We leave the resource behind if an each mode
		// is active because an empty list or map of instances is a valid state.
		delete(ms.Resources, addr.Resource.String())
	}
}

// deposeResourceInstanceObject is the real implementation of
// SyncState.DeposeResourceInstanceObject.
func (ms *Module) deposeResourceInstanceObject(addr addrs.ResourceInstance, forceKey DeposedKey) DeposedKey {
	is := ms.ResourceInstance(addr)
	if is == nil {
		return NotDeposed
	}
	return is.deposeCurrentObject(forceKey)
}

// maybeRestoreResourceInstanceDeposed is the real implementation of
// SyncState.MaybeRestoreResourceInstanceDeposed.
func (ms *Module) maybeRestoreResourceInstanceDeposed(addr addrs.ResourceInstance, key DeposedKey) bool {
	rs := ms.Resource(addr.Resource)
	if rs == nil {
		return false
	}
	is := rs.Instance(addr.Key)
	if is == nil {
		return false
	}
	if is.Current != nil {
		return false
	}
	if len(is.Deposed) == 0 {
		return false
	}
	is.Current = is.Deposed[key]
	delete(is.Deposed, key)
	return true
}

// PruneResourceHusks is a specialized method that will remove any Resource
// objects that do not contain any instances, even if they have an EachMode.
//
// You probably shouldn't call this! See the method of the same name on
// type State for more information on what this is for and the rare situations
// where it is safe to use.
func (ms *Module) PruneResourceHusks() {
	for _, rs := range ms.Resources {
		if len(rs.Instances) == 0 {
			ms.RemoveResource(rs.Addr.Resource)
		}
	}
}

// empty returns true if the receving module state is contributing nothing
// to the state. In other words, it returns true if the module could be
// removed from the state altogether without changing the meaning of the state.
//
// In practice a module containing no objects is the same as a non-existent
// module, and so we can opportunistically clean up once a module becomes
// empty on the assumption that it will be re-added if needed later.
func (ms *Module) empty() bool {
	if ms == nil {
		return true
	}

	// Resource instance objects -- each of which must belong to a resource --
	// are the only significant thing we track on a per-module basis.
	// (The presence of root module output values also causes a state to
	// be "not empty", but the main [State] object tracks those.)
	return len(ms.Resources) == 0
}
