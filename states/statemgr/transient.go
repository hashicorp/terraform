package statemgr

import "github.com/hashicorp/terraform/states"

// Transient is a union of the Reader and Writer interfaces, for types that
// deal with transient snapshots.
//
// Transient snapshots are ones that are generally retained only locally and
// to not create any historical version record when updated. Transient
// snapshots are not expected to outlive a particular Terraform process,
// and are not shared with any other process.
//
// A state manager type that is primarily concerned with persistent storage
// may embed type Transient and then call State from its PersistState and
// WriteState from its RefreshState in order to build on any existing
// Transient implementation, such as the one returned by NewTransientInMemory.
type Transient interface {
	Reader
	Writer
}

// Reader is the interface for managers that can return transient snapshots
// of state.
//
// Retrieving the snapshot must not fail, so retrieving a snapshot from remote
// storage (for example) should be dealt with elsewhere, often in an
// implementation of Refresher. For a type that implements both Reader
// and Refresher, it is okay for State to return nil if called before
// a RefreshState call has completed.
//
// For a type that implements both Reader and Writer, State must return the
// result of the most recently completed call to WriteState, and the state
// manager must accept concurrent calls to both State and WriteState.
//
// Each caller of this function must get a distinct copy of the state, and
// it must also be distinct from any instance cached inside the reader, to
// ensure that mutations of the returned state will not affect the values
// returned to other callers.
type Reader interface {
	// State returns the latest state.
	//
	// Each call to State returns an entirely-distinct copy of the state, with
	// no storage shared with any other call, so the caller may freely mutate
	// the returned object via the state APIs.
	State() *states.State
}

// Writer is the interface for managers that can create transient snapshots
// from state.
//
// Writer is the opposite of Reader, and so it must update whatever the State
// method reads from. It does not write the state to any persistent
// storage, and (for managers that support historical versions) must not
// be recorded as a persistent new version of state.
//
// Implementations that cache the state in memory must take a deep copy of it,
// since the caller may continue to modify the given state object after
// WriteState returns.
type Writer interface {
	// Write state saves a transient snapshot of the given state.
	//
	// The caller must ensure that the given state object is not concurrently
	// modified while a WriteState call is in progress. WriteState itself
	// will never modify the given state.
	WriteState(*states.State) error
}
