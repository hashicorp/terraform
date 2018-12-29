package plans

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
)

// ChangesSync is a wrapper around a Changes that provides a concurrency-safe
// interface to insert new changes and retrieve copies of existing changes.
//
// Each ChangesSync is independent of all others, so all concurrent writers
// to a particular Changes must share a single ChangesSync. Behavior is
// undefined if any other caller makes changes to the underlying Changes
// object or its nested objects concurrently with any of the methods of a
// particular ChangesSync.
type ChangesSync struct {
	lock    sync.Mutex
	changes *Changes
}

// AppendResourceInstanceChange records the given resource instance change in
// the set of planned resource changes.
//
// The caller must ensure that there are no concurrent writes to the given
// change while this method is running, but it is safe to resume mutating
// it after this method returns without affecting the saved change.
func (cs *ChangesSync) AppendResourceInstanceChange(changeSrc *ResourceInstanceChangeSrc) {
	if cs == nil {
		panic("AppendResourceInstanceChange on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()

	s := changeSrc.DeepCopy()
	cs.changes.Resources = append(cs.changes.Resources, s)
}

// GetResourceInstanceChange searches the set of resource instance changes for
// one matching the given address and generation, returning it if it exists.
//
// If no such change exists, nil is returned.
//
// The returned object is a deep copy of the change recorded in the plan, so
// callers may mutate it although it's generally better (less confusing) to
// treat planned changes as immutable after they've been initially constructed.
func (cs *ChangesSync) GetResourceInstanceChange(addr addrs.AbsResourceInstance, gen states.Generation) *ResourceInstanceChangeSrc {
	if cs == nil {
		panic("GetResourceInstanceChange on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()

	if gen == states.CurrentGen {
		return cs.changes.ResourceInstance(addr).DeepCopy()
	}
	if dk, ok := gen.(states.DeposedKey); ok {
		return cs.changes.ResourceInstanceDeposed(addr, dk).DeepCopy()
	}
	panic(fmt.Sprintf("unsupported generation value %#v", gen))
}

// RemoveResourceInstanceChange searches the set of resource instance changes
// for one matching the given address and generation, and removes it from the
// set if it exists.
func (cs *ChangesSync) RemoveResourceInstanceChange(addr addrs.AbsResourceInstance, gen states.Generation) {
	if cs == nil {
		panic("RemoveResourceInstanceChange on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()

	dk := states.NotDeposed
	if realDK, ok := gen.(states.DeposedKey); ok {
		dk = realDK
	}

	addrStr := addr.String()
	for i, r := range cs.changes.Resources {
		if r.Addr.String() != addrStr || r.DeposedKey != dk {
			continue
		}
		copy(cs.changes.Resources[i:], cs.changes.Resources[i+1:])
		cs.changes.Resources = cs.changes.Resources[:len(cs.changes.Resources)-1]
		return
	}
}

// AppendOutputChange records the given output value change in the set of
// planned value changes.
//
// The caller must ensure that there are no concurrent writes to the given
// change while this method is running, but it is safe to resume mutating
// it after this method returns without affecting the saved change.
func (cs *ChangesSync) AppendOutputChange(changeSrc *OutputChangeSrc) {
	if cs == nil {
		panic("AppendOutputChange on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()

	s := changeSrc.DeepCopy()
	cs.changes.Outputs = append(cs.changes.Outputs, s)
}

// GetOutputChange searches the set of output value changes for one matching
// the given address, returning it if it exists.
//
// If no such change exists, nil is returned.
//
// The returned object is a deep copy of the change recorded in the plan, so
// callers may mutate it although it's generally better (less confusing) to
// treat planned changes as immutable after they've been initially constructed.
func (cs *ChangesSync) GetOutputChange(addr addrs.AbsOutputValue) *OutputChangeSrc {
	if cs == nil {
		panic("GetOutputChange on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()

	return cs.changes.OutputValue(addr)
}

// RemoveOutputChange searches the set of output value changes for one matching
// the given address, and removes it from the set if it exists.
func (cs *ChangesSync) RemoveOutputChange(addr addrs.AbsOutputValue) {
	if cs == nil {
		panic("RemoveOutputChange on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()

	addrStr := addr.String()
	for i, o := range cs.changes.Outputs {
		if o.Addr.String() != addrStr {
			continue
		}
		copy(cs.changes.Outputs[i:], cs.changes.Outputs[i+1:])
		cs.changes.Outputs = cs.changes.Outputs[:len(cs.changes.Outputs)-1]
		return
	}
}
