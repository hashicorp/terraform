package plans

import (
	"sync"
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
