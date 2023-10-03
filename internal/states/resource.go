// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package states

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Resource represents the state of a resource.
type Resource struct {
	// Addr is the absolute address for the resource this state object
	// belongs to.
	Addr addrs.AbsResource

	// Instances contains the potentially-multiple instances associated with
	// this resource. This map can contain a mixture of different key types,
	// but only the ones of InstanceKeyType are considered current.
	Instances map[addrs.InstanceKey]*ResourceInstance

	// ProviderConfig is the absolute address for the provider configuration that
	// most recently managed this resource. This is used to connect a resource
	// with a provider configuration when the resource configuration block is
	// not available, such as if it has been removed from configuration
	// altogether.
	ProviderConfig addrs.AbsProviderConfig
}

// Instance returns the state for the instance with the given key, or nil
// if no such instance is tracked within the state.
func (rs *Resource) Instance(key addrs.InstanceKey) *ResourceInstance {
	return rs.Instances[key]
}

// CreateInstance creates an instance and adds it to the resource
func (rs *Resource) CreateInstance(key addrs.InstanceKey) *ResourceInstance {
	is := NewResourceInstance()
	rs.Instances[key] = is
	return is
}

// EnsureInstance returns the state for the instance with the given key,
// creating a new empty state for it if one doesn't already exist.
//
// Because this may create and save a new state, it is considered to be
// a write operation.
func (rs *Resource) EnsureInstance(key addrs.InstanceKey) *ResourceInstance {
	ret := rs.Instance(key)
	if ret == nil {
		ret = NewResourceInstance()
		rs.Instances[key] = ret
	}
	return ret
}

// ResourceInstance represents the state of a particular instance of a resource.
type ResourceInstance struct {
	// Current, if non-nil, is the remote object that is currently represented
	// by the corresponding resource instance.
	Current *ResourceInstanceObjectSrc

	// Deposed, if len > 0, contains any remote objects that were previously
	// represented by the corresponding resource instance but have been
	// replaced and are pending destruction due to the create_before_destroy
	// lifecycle mode.
	Deposed map[DeposedKey]*ResourceInstanceObjectSrc
}

// NewResourceInstance constructs and returns a new ResourceInstance, ready to
// use.
func NewResourceInstance() *ResourceInstance {
	return &ResourceInstance{
		Deposed: map[DeposedKey]*ResourceInstanceObjectSrc{},
	}
}

// HasCurrent returns true if this resource instance has a "current"-generation
// object. Most instances do, but this can briefly be false during a
// create-before-destroy replace operation when the current has been deposed
// but its replacement has not yet been created.
func (i *ResourceInstance) HasCurrent() bool {
	return i != nil && i.Current != nil
}

// HasDeposed returns true if this resource instance has a deposed object
// with the given key.
func (i *ResourceInstance) HasDeposed(key DeposedKey) bool {
	return i != nil && i.Deposed[key] != nil
}

// HasAnyDeposed returns true if this resource instance has one or more
// deposed objects.
func (i *ResourceInstance) HasAnyDeposed() bool {
	return i != nil && len(i.Deposed) > 0
}

// HasObjects returns true if this resource has any objects at all, whether
// current or deposed.
func (i *ResourceInstance) HasObjects() bool {
	return i.Current != nil || len(i.Deposed) != 0
}

// deposeCurrentObject is part of the real implementation of
// SyncState.DeposeResourceInstanceObject. The exported method uses a lock
// to ensure that we can safely allocate an unused deposed key without
// collision.
func (i *ResourceInstance) deposeCurrentObject(forceKey DeposedKey) DeposedKey {
	if !i.HasCurrent() {
		return NotDeposed
	}

	key := forceKey
	if key == NotDeposed {
		key = i.findUnusedDeposedKey()
	} else {
		if _, exists := i.Deposed[key]; exists {
			panic(fmt.Sprintf("forced key %s is already in use", forceKey))
		}
	}
	i.Deposed[key] = i.Current
	i.Current = nil
	return key
}

// Object retrieves the object with the given deposed key from the
// ResourceInstance, or returns nil if there is no such object. Use
// [addrs.NotDeposed] to retrieve the "current" object, if any.
func (i *ResourceInstance) Object(dk DeposedKey) *ResourceInstanceObjectSrc {
	if dk == addrs.NotDeposed {
		return i.Current
	}
	return i.Deposed[dk]
}

// FindUnusedDeposedKey generates a unique DeposedKey that is guaranteed not to
// already be in use for this instance at the time of the call.
//
// Note that the validity of this result may change if new deposed keys are
// allocated before it is used. To avoid this risk, instead use the
// DeposeResourceInstanceObject method on the SyncState wrapper type, which
// allocates a key and uses it atomically.
func (i *ResourceInstance) FindUnusedDeposedKey() DeposedKey {
	return i.findUnusedDeposedKey()
}

// findUnusedDeposedKey generates a unique DeposedKey that is guaranteed not to
// already be in use for this instance.
func (i *ResourceInstance) findUnusedDeposedKey() DeposedKey {
	for {
		key := NewDeposedKey()
		if _, exists := i.Deposed[key]; !exists {
			return key
		}
		// Spin until we find a unique one. This shouldn't take long, because
		// we have a 32-bit keyspace and there's rarely more than one deposed
		// instance.
	}
}

// DeposedKey is an alias for [addrs.DeposedKey], representing keys assigned
// to deposed resource instance objects.
type DeposedKey = addrs.DeposedKey

// NotDeposed is an alias for the zero value of [addrs.DeposedKey], which
// represents the absense of a deposed key, i.e. that the associated object
// is the "current" object for some resource instance.
const NotDeposed = addrs.NotDeposed

// NewDeposedKey is an alias for [addrs.NewDeposedKey].
func NewDeposedKey() DeposedKey {
	return addrs.NewDeposedKey()
}

// ParseDeposedKey is an alias for [addrs.ParseDeposedKey].
func ParseDeposedKey(raw string) (DeposedKey, error) {
	return addrs.ParseDeposedKey(raw)
}
