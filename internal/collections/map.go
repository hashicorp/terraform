// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package collections

import "iter"

// Set represents an associative array from keys of type K to values of type V.
//
// A caller-provided "key function" defines how to produce a comparable unique
// key for each distinct value of type K.
//
// Map operations are not concurrency-safe. Use external locking if multiple
// goroutines might modify the map concurrently or if one goroutine might
// read a map while another is modifying it.
type Map[K, V any] struct {
	elems map[UniqueKey[K]]MapElem[K, V]
	key   func(K) UniqueKey[K]
}

// MapElem represents a single element of a map.
type MapElem[K, V any] struct {
	K K
	V V
}

// NewMap constructs a new map whose key type knows how to calculate its own
// unique keys, by implementing [UniqueKeyer] of itself.
func NewMap[K UniqueKeyer[K], V any](elems ...MapElem[K, V]) Map[K, V] {
	m := NewMapFunc[K, V](K.UniqueKey)
	for _, elems := range elems {
		m.Put(elems.K, elems.V)
	}
	return m
}

// NewMapFunc constructs a new map with the given "map function".
//
// A valid key function must produce only values of types that can be compared
// for equality using the Go == operator, and must guarantee that each unique
// value of K has a corresponding key that uniquely identifies it. The
// implementer of the key function can decide what constitutes a
// "unique value of K", based on the meaning of type K.
//
// Type V is unconstrained by the arguments, so callers must explicitly provide
// the key and value type arguments when calling this function.
func NewMapFunc[K, V any](keyFunc func(K) UniqueKey[K]) Map[K, V] {
	return Map[K, V]{
		elems: make(map[UniqueKey[K]]MapElem[K, V]),
		key:   keyFunc,
	}
}

// NewMapCmp constructs a new set for any comparable key type, using the
// built-in == operator as the definition of key equivalence.
//
// This is here for completeness in case it's useful when talking to a
// generalized API that operates on maps of any key type, but if your
// key type is fixed and known to be comparable then it's pointless to
// use [Map]; use Go's built-in map types instead, which will then avoid
// redundantly storing the keys twice.
func NewMapCmp[K comparable, V any]() Map[K, V] {
	return NewMapFunc[K, V](cmpUniqueKeyFunc[K])
}

// HasKey returns true if the map has an element with the given key, or
// false otherwise.
func (m Map[K, V]) HasKey(k K) bool {
	if m.key == nil {
		return false // an uninitialized map has no keys
	}
	uniq := m.key(k)
	_, exists := m.elems[uniq]
	return exists
}

// Get retrieves the value associated with the given key, or the zero value
// of V if no matching element exists in the map.
func (m Map[K, V]) Get(k K) V {
	ret, _ := m.GetOk(k)
	return ret
}

// GetOk is like [Map.Get] but also returns a second boolean result reporting
// whether a matching element was present in the map.
func (m Map[K, V]) GetOk(k K) (V, bool) {
	if m.key == nil {
		var zero V
		return zero, false // an uninitialized map has no keys
	}
	uniq := m.key(k)
	ret, ok := m.elems[uniq]
	return ret.V, ok
}

// Put writes a new element into the map, with the given key and value.
//
// If there is already an element with an equivalent key (as determined by the
// set's "key function") then the new element replaces that existing element.
func (m Map[K, V]) Put(k K, v V) {
	if m.key == nil {
		panic("Put into uninitialized collections.Map")
	}
	uniq := m.key(k)
	m.elems[uniq] = MapElem[K, V]{
		K: k,
		V: v,
	}
}

// Delete removes from the map the element with the given key, or does nothing
// if there is no such element.
func (m Map[K, V]) Delete(k K) {
	if m.key == nil {
		panic("Delete from uninitialized collections.Map")
	}
	uniq := m.key(k)
	delete(m.elems, uniq)
}

// All returns an iterator over the elements of the map, in an unspecified
// order.
//
// This is intended for use in a range-over-func statement, like this:
//
//	for k, v := range map.All() {
//		// do something with k and/or v
//	}
//
// Modifying the map during iteration causes unspecified results. Modifying
// the map concurrently with advancing the iterator causes undefined behavior
// including possible memory unsafety.
func (m Map[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, elem := range m.elems {
			if !yield(elem.K, elem.V) {
				return
			}
		}
	}
}

// Len returns the number of elements in the map.
func (m Map[K, V]) Len() int {
	return len(m.elems)
}
