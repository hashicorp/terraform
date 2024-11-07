// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package collections

import "iter"

// Set represents an unordered set of values of a particular type.
//
// A caller-provided "key function" defines how to produce a comparable unique
// key for each distinct value of type T.
//
// Set operations are not concurrency-safe. Use external locking if multiple
// goroutines might modify the set concurrently or if one goroutine might
// read a set while another is modifying it.
type Set[T any] struct {
	members map[UniqueKey[T]]T
	key     func(T) UniqueKey[T]
}

// NewSet constructs a new set whose element type knows how to calculate its own
// unique keys, by implementing [UniqueKeyer] of itself.
func NewSet[T UniqueKeyer[T]](elems ...T) Set[T] {
	return NewSetFunc(T.UniqueKey, elems...)
}

// NewSetFunc constructs a new set with the given "key function".
//
// A valid key function must produce only values of types that can be compared
// for equality using the Go == operator, and must guarantee that each unique
// value of T has a corresponding key that uniquely identifies it. The
// implementer of the key function can decide what constitutes a
// "unique value of T", based on the meaning of type T.
func NewSetFunc[T any](keyFunc func(T) UniqueKey[T], elems ...T) Set[T] {
	set := Set[T]{
		members: make(map[UniqueKey[T]]T),
		key:     keyFunc,
	}
	for _, elem := range elems {
		set.Add(elem)
	}
	return set
}

// NewSetCmp constructs a new set for any comparable type, using the built-in
// == operator as the definition of element equivalence.
func NewSetCmp[T comparable](elems ...T) Set[T] {
	return NewSetFunc(cmpUniqueKeyFunc[T], elems...)
}

// Has returns true if the given value is present in the set, or false
// otherwise.
func (s Set[T]) Has(v T) bool {
	if len(s.members) == 0 {
		// We'll skip calling "s.key" in this case, so that we don't panic
		// if called on an uninitialized Set.
		return false
	}
	k := s.key(v)
	_, ok := s.members[k]
	return ok
}

// Add inserts new members into the set.
//
// If any existing member of the set is considered to be equivalent to a
// given value per the rules in the set's "key function", the old value will
// be discarded and replaced by the new value.
//
// If multiple of the given arguments is considered to be equivalent then
// only the later one is retained.
func (s Set[T]) Add(vs ...T) {
	for _, v := range vs {
		k := s.key(v)
		s.members[k] = v
	}
}

// Remove removes the given member from the set, or does nothing if no
// equivalent value was present.
func (s Set[T]) Remove(v T) {
	k := s.key(v)
	delete(s.members, k)
}

// All returns an iterator over the elements of the set, in an unspecified
// order.
//
// The result of this function is part of the internal state of the set
// and so callers MUST NOT modify it. If a caller is using locks to ensure
// safe concurrent access then any reads of the resulting map must be
// guarded by the same lock as would be used for other methods that read
// data from the set.
//
// All returns an iterator over the elements of the set, in an unspecified
// order.
//
//	for elem := range set.All() {
//		// do something with elem
//	}
//
// Modifying the set during iteration causes unspecified results. Modifying
// the set concurrently with advancing the iterator causes undefined behavior
// including possible memory unsafety.
func (s Set[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s.members {
			if !yield(v) {
				return
			}
		}
	}
}

// Len returns the number of unique elements in the set.
func (s Set[T]) Len() int {
	return len(s.members)
}
