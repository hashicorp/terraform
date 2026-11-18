// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"iter"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/terraform/internal/addrs"
)

// ChangesSync is a wrapper around a Changes that provides a concurrency-safe
// interface to insert new changes and retrieve copies of existing changes.
//
// Each ChangesSync is independent of all others, so all concurrent writers
// to a particular Changes must share a single ChangesSync. Behavior is
// undefined if any other caller makes changes to the underlying Changes
// object or its nested objects while a particular ChangesSync is in use.
// Once the object is closed, it is no longer writable and any further
// modifications will panic.
//
// Output changes removed through a ChangesSync leave nil entries in the
// Outputs slice of the underlying Changes, which are compacted periodically
// and when the ChangesSync is closed.
type ChangesSync struct {
	lock    sync.Mutex
	changes *Changes
	closed  atomic.Bool

	// outputIndex maps the address of each output change in changes.Outputs
	// to its position, so that output changes can be found without scanning
	// all of them. It is built on first use, because the wrapped Changes may
	// already contain output changes, and is reset to nil whenever it needs
	// to be rebuilt.
	outputIndex map[string]int

	// outputHoles counts the nil entries left in changes.Outputs by removals.
	outputHoles int

	// outputDuplicates records that changes.Outputs contains more than one
	// change for the same address, in which case outputIndex refers to the
	// first of them.
	outputDuplicates bool
}

// AppendResourceInstanceChange records the given resource instance change in
// the set of planned resource changes.
//
// The caller must ensure that there are no concurrent writes to the given
// change while this method is running, but it is safe to resume mutating
// it after this method returns without affecting the saved change.
func (cs *ChangesSync) AppendResourceInstanceChange(change *ResourceInstanceChange) {
	if cs == nil {
		panic("AppendResourceInstanceChange on nil ChangesSync")
	}
	defer cs.beginWrite()()

	s := change.DeepCopy()
	cs.changes.Resources = append(cs.changes.Resources, s)
}

func (cs *ChangesSync) AppendQueryInstance(query *QueryInstance) {
	if cs == nil {
		panic("AppendQueryInstance on nil ChangesSync")
	}
	defer cs.beginWrite()()

	s := query.DeepCopy() // TODO do we need to deep copy here?
	cs.changes.Queries = append(cs.changes.Queries, s)
}

// GetResourceInstanceChange searches the set of resource instance changes for
// one matching the given address and deposed key, returning it if it exists.
// Use [addrs.NotDeposed] as the deposed key to represent the "current"
// object for the given resource instance.
//
// If no such change exists, nil is returned.
//
// The returned object is a deep copy of the change recorded in the plan, so
// callers may mutate it although it's generally better (less confusing) to
// treat planned changes as immutable after they've been initially constructed.
func (cs *ChangesSync) GetResourceInstanceChange(addr addrs.AbsResourceInstance, dk addrs.DeposedKey) *ResourceInstanceChange {
	if cs == nil {
		panic("GetResourceInstanceChange on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()

	if dk == addrs.NotDeposed {
		return cs.changes.ResourceInstance(addr).DeepCopy()
	}
	return cs.changes.ResourceInstanceDeposed(addr, dk).DeepCopy()
}

// GetChangesForConfigResource searches the set of resource instance
// changes and returns all changes related to a given configuration address.
// This is be used to find possible changes related to a configuration
// reference.
//
// If no such changes exist, nil is returned.
//
// The returned objects are a deep copy of the change recorded in the plan, so
// callers may mutate them although it's generally better (less confusing) to
// treat planned changes as immutable after they've been initially constructed.
func (cs *ChangesSync) GetChangesForConfigResource(addr addrs.ConfigResource) []*ResourceInstanceChange {
	if cs == nil {
		panic("GetChangesForConfigResource on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()
	var changes []*ResourceInstanceChange
	for c := range cs.changes.InstancesForConfigResource(addr) {
		changes = append(changes, c.DeepCopy())
	}
	return changes
}

// ReadInstancesForConfigResource returns an iterator over the changes for a given
// configuration resource, applying the given selector function to each change.
// The changes must no longer be writable otherwise a panic will occur.
func ReadInstancesForConfigResource(changesSync *ChangesSync, addr addrs.ConfigResource) iter.Seq[*ResourceInstanceChange] {
	if changesSync == nil {
		panic("ReadInstancesForConfigResource on nil ChangesSync")
	}
	if !changesSync.closed.Load() {
		panic("ReadEachConfigResourceChange requires closed ChangesSync")
	}

	return changesSync.changes.InstancesForConfigResource(addr)
}

// GetChangesForAbsResource searches the set of resource instance
// changes and returns all changes related to a given configuration address.
//
// If no such changes exist, nil is returned.
//
// The returned objects are a deep copy of the change recorded in the plan, so
// callers may mutate them although it's generally better (less confusing) to
// treat planned changes as immutable after they've been initially constructed.
func (cs *ChangesSync) GetChangesForAbsResource(addr addrs.AbsResource) []*ResourceInstanceChange {
	if cs == nil {
		panic("GetChangesForAbsResource on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()
	var changes []*ResourceInstanceChange
	for _, c := range cs.changes.InstancesForAbsResource(addr) {
		changes = append(changes, c.DeepCopy())
	}
	return changes
}

func (cs *ChangesSync) GetQueryInstancesForAbsResource(addr addrs.AbsResource) []*QueryInstance {
	if cs == nil {
		panic("GetQueryInstancesForAbsResource on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()
	var queries []*QueryInstance
	for _, q := range cs.changes.QueriesForAbsResource(addr) {
		queries = append(queries, q.DeepCopy())
	}
	return queries
}

// RemoveResourceInstanceChange searches the set of resource instance changes
// for one matching the given address and deposed key, and removes it from the
// set if it exists.
func (cs *ChangesSync) RemoveResourceInstanceChange(addr addrs.AbsResourceInstance, dk addrs.DeposedKey) {
	if cs == nil {
		panic("RemoveResourceInstanceChange on nil ChangesSync")
	}
	defer cs.beginWrite()()

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
func (cs *ChangesSync) AppendOutputChange(changeSrc *OutputChange) {
	if cs == nil {
		panic("AppendOutputChange on nil ChangesSync")
	}
	defer cs.beginWrite()()

	index := cs.outputChangeIndex()
	key := changeSrc.Addr.String()
	if _, exists := index[key]; exists {
		cs.outputDuplicates = true
	} else {
		index[key] = len(cs.changes.Outputs)
	}
	cs.changes.Outputs = append(cs.changes.Outputs, changeSrc)
}

// GetOutputChange searches the set of output value changes for one matching
// the given address, returning it if it exists.
//
// If no such change exists, nil is returned.
//
// The returned object is a deep copy of the change recorded in the plan, so
// callers may mutate it although it's generally better (less confusing) to
// treat planned changes as immutable after they've been initially constructed.
func (cs *ChangesSync) GetOutputChange(addr addrs.AbsOutputValue) *OutputChange {
	if cs == nil {
		panic("GetOutputChange on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()

	i, ok := cs.outputChangeIndex()[addr.String()]
	if !ok {
		return nil
	}
	return cs.changes.Outputs[i]
}

// RemoveOutputChange searches the set of output value changes for one matching
// the given address, and removes it from the set if it exists.
func (cs *ChangesSync) RemoveOutputChange(addr addrs.AbsOutputValue) {
	if cs == nil {
		panic("RemoveOutputChange on nil ChangesSync")
	}
	defer cs.beginWrite()()

	index := cs.outputChangeIndex()
	key := addr.String()
	i, ok := index[key]
	if !ok {
		return
	}

	// Rather than shifting all of the following changes, we leave a hole
	// which preserves the positions of the others.
	cs.changes.Outputs[i] = nil
	delete(index, key)
	cs.outputHoles++

	switch {
	case cs.outputDuplicates:
		// Another change for the same address may now be the first one, so
		// the index must be rebuilt to find it.
		cs.outputIndex = nil
	case cs.outputHoles > len(cs.changes.Outputs)/2:
		cs.compactOutputChanges()
	}
}

// outputChangeIndex returns the index of output changes by address, building
// it first if necessary. The caller must hold the lock.
func (cs *ChangesSync) outputChangeIndex() map[string]int {
	if cs.outputIndex != nil {
		return cs.outputIndex
	}

	cs.outputIndex = make(map[string]int, len(cs.changes.Outputs))
	cs.outputHoles = 0
	cs.outputDuplicates = false
	for i, oc := range cs.changes.Outputs {
		if oc == nil {
			cs.outputHoles++
			continue
		}
		key := oc.Addr.String()
		if _, exists := cs.outputIndex[key]; exists {
			cs.outputDuplicates = true
			continue
		}
		cs.outputIndex[key] = i
	}
	return cs.outputIndex
}

// compactOutputChanges removes the holes left in the output changes by
// removals, preserving the order of the remaining changes. The caller must
// hold the lock.
func (cs *ChangesSync) compactOutputChanges() {
	if cs.outputHoles == 0 {
		return
	}
	cs.changes.Outputs = slices.DeleteFunc(cs.changes.Outputs, func(oc *OutputChange) bool {
		return oc == nil
	})
	// The positions have changed, so the index must be rebuilt.
	cs.outputIndex = nil
	cs.outputHoles = 0
}

// GetActionInvocation gets an action invocation based on the action address, the triggering
// resource address, the action trigger block index, and the action list index.
func (cs *ChangesSync) GetActionInvocation(addr addrs.AbsActionInstance, actionTrigger ActionTrigger) *ActionInvocationInstance {
	if cs == nil {
		panic("GetActionInvocation on nil ChangesSync")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()

	for _, a := range cs.changes.ActionInvocations {
		if a.Addr.Equal(addr) {
			if a.ActionTrigger.Equals(actionTrigger) {
				return a
			}
		}
	}
	return nil
}

// AppendActionInvocation
func (cs *ChangesSync) AppendActionInvocation(action *ActionInvocationInstance) {
	if cs == nil {
		panic("AppendActionInvocation on nil ChangesSync")
	}
	defer cs.beginWrite()()

	cs.changes.ActionInvocations = append(cs.changes.ActionInvocations, action)
}

// RemoveActionInvocation searches the set of action invocations for one
// matching the given address, and removes it from the set if it exists.
func (cs *ChangesSync) RemoveActionInvocation(addr addrs.AbsActionInstance) {
	if cs == nil {
		panic("RemoveActionInvocation on nil ChangesSync")
	}
	defer cs.beginWrite()()

	addrStr := addr.String()
	for i, a := range cs.changes.ActionInvocations {
		if a.Addr.String() != addrStr {
			continue
		}
		copy(cs.changes.ActionInvocations[i:], cs.changes.ActionInvocations[i+1:])
		cs.changes.ActionInvocations = cs.changes.ActionInvocations[:len(cs.changes.ActionInvocations)-1]
		return
	}
}

// Close marks the wrapper as closed, so any further modifications will panic.
func (cs *ChangesSync) Close() {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	cs.compactOutputChanges()
	cs.closed.Store(true) // only reading is allowed after close
}

func (cs *ChangesSync) beginWrite() func() {
	cs.lock.Lock()
	if cs.closed.Load() {
		cs.lock.Unlock()
		panic("write on closed ChangesSync")
	}
	return func() {
		cs.lock.Unlock()
	}
}
