// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package collections

// UniqueKey represents a value that is comparable and uniquely identifies
// another value of type T.
//
// The Go type system offers no way to guarantee at compile time that
// implementations of this type are comparable, but if this interface is
// implemented by an uncomparable type then that will cause runtime panics
// when inserting elements into collection types that use unique keys.
//
// We use this to help with correctness of the unique-key-generator callbacks
// used with the collection types in this package, so help with type parameter
// inference and to raise compile-time errors if an inappropriate callback
// is used as the key generator for a particular collection.
type UniqueKey[T any] interface {
	// Implementations must include an IsUniqueKey method with an empty body
	// just as a compile-time assertion that they are intended to behave as
	// unique keys for a particular other type.
	//
	// This method is never actually called by the collection types. Other
	// callers could potentially call it, but it would be strange and pointless
	// to do so.
	IsUniqueKey(T)
}

// A UniqueKeyer is a type that knows how to calculate a unique key itself.
type UniqueKeyer[T any] interface {
	// UniqueKey returns the unique key of the reciever.
	//
	// A correct implementation of UniqueKey must return a distinct value
	// for each unique value of T, where the uniqueness of T values is decided
	// by the implementer. See [UniqueKey] for more information.
	//
	// Although not enforced directly by the Go type system, it doesn't make
	// sense for a type to implement [UniqueKeyer] for any type other than
	// itself. Such a nonsensical implementation will not be accepted by
	// functions like [NewSet] and [NewMap].
	UniqueKey() UniqueKey[T]
}

// cmpUniqueKey is an annoying little adapter used to make arbitrary
// comparable types usable with [Set] and [Map].
//
// It just wraps a single-element array of T around the value, so it
// remains exactly as comparable as T. However, it does unfortunately
// mean redundantly storing T twice -- both as the unique key and the
// value -- in our collections.
type cmpUniqueKey[T comparable] [1]T

func (cmpUniqueKey[T]) IsUniqueKey(T) {}

func cmpUniqueKeyFunc[T comparable](v T) UniqueKey[T] {
	return cmpUniqueKey[T]{v}
}
