// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statekeys

// Key is implemented by types that can be used as state keys.
type Key interface {
	// KeyType returns the [KeyType] used for keys belonging to a particular
	// implementation of [Key].
	KeyType() KeyType

	// rawSuffix returns additional characters that should appear after the
	// key type portion of the final raw key.
	//
	// This is unexported both to help prevent accidental misuse (external
	// callers MUST use [String] to obtain the correct string representation],
	// and to prevent implementations of this interface from other packages.
	// This package is the sole authority on state keys.
	rawSuffix() string
}

// String returns the string representation of the given key, ready to be used
// in the RPC API representation of a [stackstate.AppliedChange] object.
func String(k Key) string {
	if k == nil {
		panic("called statekeys.String with nil Key")
	}
	return string(k.KeyType()) + k.rawSuffix()
}

// RecognizedType returns true if the given key has a [KeyType] that's known
// to the current version of this package, or false otherwise.
//
// If RecognizedType returns false, use the key's KeyType method to obtain
// the unrecognized type and then use its UnrecognizedKeyHandling method
// to determine the appropriate handling for the unrecognized key type.
func RecognizedType(k Key) bool {
	if k == nil {
		panic("called statekeys.RecognizedType with nil Key")
	}
	_, unrecognized := k.(Unrecognized)
	return !unrecognized
}

// Unrecognized is a fallback [Key] implementation used when a given
// key has an unrecognized type.
//
// Unrecognized keys are round-trippable in that the RawKey method will return
// the same string that was originally parsed. Use
// KeyType.UnrecognizedKeyHandling to determine how Terraform Core should
// respond to the key having an unrecognized type.
type Unrecognized struct {
	// ApparentKeyType is a [KeyType] representation of the type portion of the
	// unrecognized key. Unlike most other [KeyType] values, this one
	// will presumably not match any of the [KeyType] constants defined
	// elsewhere in this package.
	ApparentKeyType KeyType

	// Remainder is a verbatim copy of whatever appeared after the type
	// in the given key string. This is preserved only for round-tripping
	// purposes and so should be treated as opaque.
	remainder string
}

// KeyType returns the value from the ApparentKeyType field, which will
// presumably not match any of the [KeyType] constants in this package
// (because otherwise we would've used a different implementation of [Key]).
func (k Unrecognized) KeyType() KeyType {
	return k.ApparentKeyType
}

func (k Unrecognized) rawSuffix() string {
	return k.remainder
}
