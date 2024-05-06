// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statemgr

import (
	"context"
	"time"

	version "github.com/hashicorp/go-version"

	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
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
	OutputReader
}

// OutputReader is the interface for managers that fetches output values from state
// or another source. This is a refinement of fetching the entire state and digging
// the output values from it because enhanced backends can apply special permissions
// to differentiate reading the state and reading the outputs within the state.
type OutputReader interface {
	// GetRootOutputValues fetches the root module output values from state or another source
	GetRootOutputValues(ctx context.Context) (map[string]*states.OutputValue, error)
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
//
// Some implementations may optionally utilize config schema to persist
// state. For example, when representing state in an external JSON
// representation.
type Persister interface {
	PersistState(*schemarepo.Schemas) error
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

// IntermediateStateConditionalPersister is an optional extension of
// [Persister] that allows an implementation to tailor the rules for
// whether to create intermediate state snapshots when Terraform Core emits
// events reporting that the state might have changed. This interface is used
// by the local backend when it's been configured to use another backend for
// state storage.
//
// For state managers that don't implement this interface, the local backend's
// StateHook uses a default set of rules that aim to be a good compromise
// between how long a state change can be active before it gets committed as a
// snapshot vs. how many intermediate snapshots will get created. That
// compromise is subject to change over time, but a state manager can implement
// this interface to exert full control over those rules.
type IntermediateStateConditionalPersister interface {
	// ShouldPersistIntermediateState will be called each time Terraform Core
	// emits an intermediate state event that is potentially eligible to be
	// persisted.
	//
	// The implemention should return true to signal that the state snapshot
	// most recently provided to the object's WriteState should be persisted,
	// or false if it should not be persisted. If this function returns true
	// then the receiver will see a subsequent call to
	// [statemgr.Persister.PersistState] to request persistence.
	//
	// The implementation must not modify anything reachable through the
	// arguments, and must not retain pointers to anything reachable through
	// them after the function returns. However, implementers can assume that
	// nothing will write to anything reachable through the arguments while
	// this function is active.
	ShouldPersistIntermediateState(info *IntermediateStatePersistInfo) bool
}

type IntermediateStatePersistInfo struct {
	// RequestedPersistInterval is the persist interval requested by whatever
	// instantiated the StateHook.
	//
	// Implementations of [IntermediateStateConditionalPersister] should ideally
	// respect this, but may ignore it if they use something other than the
	// passage of time to make their decision.
	RequestedPersistInterval time.Duration

	// LastPersist is the time when the last intermediate state snapshot was
	// persisted, or the time of the first report for Terraform Core if there
	// hasn't yet been a persisted snapshot.
	LastPersist time.Time

	// ForcePersist is true when Terraform CLI has receieved an interrupt
	// signal and is therefore trying to create snapshots more aggressively
	// in anticipation of possibly being terminated ungracefully.
	// [IntermediateStateConditionalPersister] implementations should ideally
	// persist every snapshot they get when this flag is set, unless they have
	// some external information that implies this shouldn't be necessary.
	ForcePersist bool
}

// DefaultIntermediateStatePersistRule is the default implementation of
// [IntermediateStateConditionalPersister.ShouldPersistIntermediateState] used
// when the selected state manager doesn't implement that interface.
//
// Implementers of that interface can optionally wrap a call to this function
// if they want to combine the default behavior with some logic of their own.
func DefaultIntermediateStatePersistRule(info *IntermediateStatePersistInfo) bool {
	return info.ForcePersist || time.Since(info.LastPersist) >= info.RequestedPersistInterval
}
