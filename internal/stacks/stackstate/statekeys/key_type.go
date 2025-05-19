// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statekeys

import (
	"fmt"
)

// A KeyType represents a particular type of state key, which is typically
// associated with a particular kind of object that can be represented in
// stack state.
//
// Each KeyType consists of four ASCII letters which are intended to be
// somewhat mnemonic (at least for the more commonly-appearing ones)
// but are not intended for end-user consumption, because state storage
// keys are to be considered opaque by anything other than Terraform Core.
//
// There are some additional semantics encoded in the case of some of the
// letters, to help keep the encoding relatively compact:
//   - If the first letter is uppercase then that means the key type is
//     "mandatory", while if it's lowercase then the key type is "ignorable".
//     Terraform Core will raise an error during state decoding if it encounters
//     a mandatory key type that it isn't familiar with, but it will silently
//     allow unrecognized key types that are ignorable.
//   - For key types that are ignorable, if the _last_ letter is lowercase
//     then the key type is "discarded", while if it's uppercase then the
//     key type is "preserved". When Terraform Core encounters an unrecognized
//     key type that is both ignorable and "discarded" then it will proactively
//     emit an event to delete that unrecognized object from the state.
//     If the key type is "preserved" then Terraform Core will just ignore it
//     and let the existing object with that key continue to exist in the
//     state.
//
// These behaviors are intended as a lightweight way to achieve some
// forward-compatibility by allowing an older version of Terraform Core to,
// when it's safe to do so, silently discard or preserve objects that were
// presumably added by a later version of Terraform. When we add new key types
// in future we should consider which of the three unrecognized key handling
// methods is most appropriate, preferring one of the two "ignorable" modes
// if possible but using a "mandatory" key type if ignoring a particular
// object could cause an older version of Terraform Core to misinterpret
// the overall meaning of the prior state.
type KeyType string

const (
	ResourceInstanceObjectType KeyType = "RSRC"
	ComponentInstanceType      KeyType = "CMPT"
	OutputType                 KeyType = "OTPT"
	VariableType               KeyType = "VRBL"
)

// UnrecognizedKeyHandling returns an indication of which of the three possible
// actions should be taken if the receiver is an unrecognized key type.
//
// It only really makes sense to use this method for a [KeyType] included in
// an [UnrecognizedKey] value.
func (kt KeyType) UnrecognizedKeyHandling() UnrecognizedKeyHandling {
	first := kt[0]
	last := kt[3]
	switch {
	case first >= 'A' && first <= 'Z':
		return FailIfUnrecognized
	case last >= 'A' && last <= 'Z':
		return PreserveIfUnrecognized
	default:
		return DiscardIfUnrecognized
	}
}

func (kt KeyType) GoString() string {
	return fmt.Sprintf("statekeys.KeyType(%q)", kt)
}

func isPlausibleRawKeyType(s string) bool {
	if len(s) != 4 {
		return false
	}
	// All of the characters must be ASCII letters
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}

// UnrecognizedKeyHandling models the three different ways an unrecognized
// key type can be handled when decoding prior state.
//
// See the documentation for [KeyType] for more information.
type UnrecognizedKeyHandling rune

//go:generate go tool golang.org/x/tools/cmd/stringer -type UnrecognizedKeyHandling

const (
	FailIfUnrecognized     UnrecognizedKeyHandling = 'F'
	PreserveIfUnrecognized UnrecognizedKeyHandling = 'P'
	DiscardIfUnrecognized  UnrecognizedKeyHandling = 'D'
)
