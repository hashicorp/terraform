// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package collections

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
func NewSetCmp[T comparable]() Set[T] {
	return NewSetFunc(cmpUniqueKeyFunc[T])
}

// Has returns true if the given value is present in the set, or false
// otherwise.
func (s Set[T]) Has(v T) bool {
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

// Elems exposes the internal underlying map representation of the set
// directly, as a pragmatic compromise for efficient iteration.
//
// The result of this function is part of the internal state of the set
// and so callers MUST NOT modify it. If a caller is using locks to ensure
// safe concurrent access then any reads of the resulting map must be
// guarded by the same lock as would be used for other methods that read
// data from the set.
//
// The only correct use of this function is as part of a "for ... range"
// statement using only the values of the resulting map:
//
//	for _, elem := range set.Elems() {
//	    // ...
//	}
//
// Do not access or make any assumptions about the keys of the resulting
// map. Their exact values are an implementation detail of the set.
func (s Set[T]) Elems() map[UniqueKey[T]]T {
	// This is regrettable but the only viable way to support efficient
	// iteration over set members until Go gains support for range
	// loops over custom iterator functions.
	return s.members
}

// Len returns the number of unique elements in the set.
func (s Set[T]) Len() int {
	return len(s.members)
}
