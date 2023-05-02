// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package statemgr defines the interfaces and some supporting functionality
// for "state managers", which are components responsible for writing state
// to some persistent storage and then later retrieving it.
//
// State managers will usually (but not necessarily) use the state file formats
// implemented in the sibling directory "statefile" to serialize the persistent
// parts of state for storage.
//
// State managers are responsible for ensuring that stored state can be updated
// safely across multiple, possibly-concurrent Terraform runs (with reasonable
// constraints and limitations). The rest of Terraform considers state to be
// a mutable data structure, with state managers preserving that illusion
// by creating snapshots of the state and updating them over time.
//
// From the perspective of callers of the general state manager API, a state
// manager is able to return the latest snapshot and to replace that snapshot
// with a new one. Some state managers may also preserve historical snapshots
// using facilities offered by their storage backend, but this is always an
// implementation detail: the historical versions are not visible to a user
// of these interfaces.
package statemgr
