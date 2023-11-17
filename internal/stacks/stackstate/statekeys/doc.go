// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package statekeys contains the definitions for the various different kinds
// of tracking key we use (or have historically used) for objects in a stack
// state.
//
// Stack state is a mutable data structure whose storage strategy is delegated
// to whatever is calling into Terraform Core. To allow Terraform Core to
// emit updates to that data structure piecemeal, rather than having to return
// the whole dataset over and over, we use tracking keys for each
// separately-updatable element of the state that are opaque to the caller but
// meaningful to Terraform Core.
//
// Callers are expected to use simple character-for-character string matching
// to compare these to recognize whether an update is describing an entirely
// new object or a replacement for ane existing object, and so the main
// requirement is that the content of these keys remains consistent across
// Terraform Core releases. However, from Terraform Core's perspective we
// also use these keys to carry some metadata about what is being tracked
// so we can avoid redundantly storing the same information in both the key
// and in the associated stored object.
//
// The keys defined in this package are in principle valid for use both as
// raw state keys and as external description keys, but some of them are used
// only for one or the other since the raw and external description forms
// don't necessarily have the same level of detail.
package statekeys
