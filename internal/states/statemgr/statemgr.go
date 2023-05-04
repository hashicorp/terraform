// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statemgr

// Storage is the union of Transient and Persistent, for state managers that
// have both transient and persistent storage.
//
// Types implementing this interface coordinate between their Transient
// and Persistent implementations so that the persistent operations read
// or write the transient store.
type Storage interface {
	Transient
	Persistent
}

// Full is the union of all of the more-specific state interfaces.
//
// This interface may grow over time, so state implementations aiming to
// implement it may need to be modified for future changes. To ensure that
// this need can be detected, always include a statement nearby the declaration
// of the implementing type that will fail at compile time if the interface
// isn't satisfied, such as:
//
//	var _ statemgr.Full = (*ImplementingType)(nil)
type Full interface {
	Storage
	Locker
}
