package statemgr

import (
	version "github.com/hashicorp/go-version"
)

// Persistent is a union of the Refresher and Persistent interfaces, for types
// that deal with persistent snapshots.
//
// Persistent snapshots are ones that are retained in storage that will
// outlive a particular Terraform process, and are shared with other Terraform
// processes that have a similarly-configured state manager.
//
// A manager may also choose to retain historical persistent snapshots, but
// that is an implementation detail and not visible via this API.
type Persistent interface {
	Refresher
	Persister
}

// Refresher is the interface for managers that can read snapshots from
// persistent storage.
//
// Refresher is usually implemented in conjunction with Reader, with
// RefreshState copying the latest persistent snapshot into the latest
// transient snapshot.
//
// For a type that implements both Refresher and Persister, RefreshState must
// return the result of the most recently completed successful call to
// PersistState, unless another concurrently-running process has persisted
// another snapshot in the mean time.
//
// The Refresher implementation must guarantee that the snapshot is read
// from persistent storage in a way that is safe under concurrent calls to
// PersistState that may be happening in other processes.
type Refresher interface {
	// RefreshState retrieves a snapshot of state from persistent storage,
	// returning an error if this is not possible.
	//
	// Types that implement RefreshState generally also implement a State
	// method that returns the result of the latest successful refresh.
	//
	// Since only a subset of the data in a state is included when persisting,
	// a round-trip through PersistState and then RefreshState will often
	// return only a subset of what was written. Callers must assume that
	// ephemeral portions of the state may be unpopulated after calling
	// RefreshState.
	RefreshState() error
}

// Persister is the interface for managers that can write snapshots to
// persistent storage.
//
// Persister is usually implemented in conjunction with Writer, with
// PersistState copying the latest transient snapshot to be the new latest
// persistent snapshot.
//
// A Persister implementation must detect updates made by other processes
// that may be running concurrently and avoid destroying those changes. This
// is most commonly achieved by making use of atomic write capabilities on
// the remote storage backend in conjunction with book-keeping with the
// Serial and Lineage fields in the standard state file formats.
type Persister interface {
	PersistState() error
}

// PersistentMeta is an optional extension to Persistent that allows inspecting
// the metadata associated with the snapshot that was most recently either
// read by RefreshState or written by PersistState.
type PersistentMeta interface {
	// StateSnapshotMeta returns metadata about the state snapshot most
	// recently created either by a call to PersistState or read by a call
	// to RefreshState.
	//
	// If no persistent snapshot is yet available in the manager then
	// the return value is meaningless. This method is primarily available
	// for testing and logging purposes, and is of little use otherwise.
	StateSnapshotMeta() SnapshotMeta
}

// SnapshotMeta contains metadata about a persisted state snapshot.
//
// This metadata is usually (but not necessarily) included as part of the
// "header" of a state file, which is then written to a raw blob storage medium
// by a persistent state manager.
//
// Not all state managers will have useful values for all fields in this
// struct, so SnapshotMeta values are of little use beyond testing and logging
// use-cases.
type SnapshotMeta struct {
	// Lineage and Serial can be used to understand the relationships between
	// snapshots.
	//
	// If two snapshots both have an identical, non-empty Lineage
	// then the one with the higher Serial is newer than the other.
	// If the Lineage values are different or empty then the two snapshots
	// are unrelated and cannot be compared for relative age.
	Lineage string
	Serial  uint64

	// TerraformVersion is the number of the version of Terraform that created
	// the snapshot.
	TerraformVersion *version.Version
}
