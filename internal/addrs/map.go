// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

// Map represents a mapping whose keys are address types that implement
// UniqueKeyer.
//
// Since not all address types are comparable in the Go language sense, this
// type cannot work with the typical Go map access syntax, and so instead has
// a method-based syntax. Use this type only for situations where the key
// type isn't guaranteed to always be a valid key for a standard Go map.
type Map[K UniqueKeyer, V any] struct {
	// Elems is the internal data structure of the map.
	//
	// This is exported to allow for comparisons during tests and other similar
	// careful read operations, but callers MUST NOT modify this map directly.
	// Use only the methods of Map to modify the contents of this structure,
	// to ensure that it remains correct and consistent.
	Elems map[UniqueKey]MapElem[K, V]
}

type MapElem[K UniqueKeyer, V any] struct {
	Key   K
	Value V
}

func MakeMap[K UniqueKeyer, V any](initialElems ...MapElem[K, V]) Map[K, V] {
	inner := make(map[UniqueKey]MapElem[K, V], len(initialElems))
	ret := Map[K, V]{inner}
	for _, elem := range initialElems {
		ret.Put(elem.Key, elem.Value)
	}
	return ret
}

func MakeMapElem[K UniqueKeyer, V any](key K, value V) MapElem[K, V] {
	return MapElem[K, V]{key, value}
}

// Put inserts a new element into the map, or replaces an existing element
// which has an equivalent key.
func (m Map[K, V]) Put(key K, value V) {
	realKey := key.UniqueKey()
	m.Elems[realKey] = MapElem[K, V]{key, value}
}

// PutElement is like Put but takes the key and value from the given MapElement
// structure instead of as individual arguments.
func (m Map[K, V]) PutElement(elem MapElem[K, V]) {
	m.Put(elem.Key, elem.Value)
}

// Remove deletes the element with the given key from the map, or does nothing
// if there is no such element.
func (m Map[K, V]) Remove(key K) {
	realKey := key.UniqueKey()
	delete(m.Elems, realKey)
}

// Get returns the value of the element with the given key, or the zero value
// of V if there is no such element.
func (m Map[K, V]) Get(key K) V {
	realKey := key.UniqueKey()
	return m.Elems[realKey].Value
}

// GetOk is like Get but additionally returns a flag for whether there was an
// element with the given key present in the map.
func (m Map[K, V]) GetOk(key K) (V, bool) {
	realKey := key.UniqueKey()
	elem, ok := m.Elems[realKey]
	return elem.Value, ok
}

// Has returns true if and only if there is an element in the map which has the
// given key.
func (m Map[K, V]) Has(key K) bool {
	realKey := key.UniqueKey()
	_, ok := m.Elems[realKey]
	return ok
}

// Len returns the number of elements in the map.
func (m Map[K, V]) Len() int {
	return len(m.Elems)
}

// Elements returns a slice containing a snapshot of the current elements of
// the map, in an unpredictable order.
func (m Map[K, V]) Elements() []MapElem[K, V] {
	if len(m.Elems) == 0 {
		return nil
	}
	ret := make([]MapElem[K, V], 0, len(m.Elems))
	for _, elem := range m.Elems {
		ret = append(ret, elem)
	}
	return ret
}

// Keys returns a Set[K] containing a snapshot of the current keys of elements
// of the map.
func (m Map[K, V]) Keys() Set[K] {
	if len(m.Elems) == 0 {
		return nil
	}
	ret := make(Set[K], len(m.Elems))

	// We mess with the internals of Set here, rather than going through its
	// public interface, because that means we can avoid re-calling UniqueKey
	// on all of the elements when we know that our own Put method would have
	// already done the same thing.
	for realKey, elem := range m.Elems {
		ret[realKey] = elem.Key
	}
	return ret
}

// Values returns a slice containing a snapshot of the current values of
// elements of the map, in an unpredictable order.
func (m Map[K, V]) Values() []V {
	if len(m.Elems) == 0 {
		return nil
	}
	ret := make([]V, 0, len(m.Elems))
	for _, elem := range m.Elems {
		ret = append(ret, elem.Value)
	}
	return ret
}
