// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"sort"
)

// Set represents a set of addresses of types that implement UniqueKeyer.
//
// Modify the set only by the methods on this type. This type exposes its
// internals for convenience during reading, such as iterating over set elements
// by ranging over the map values, but making direct modifications could
// potentially make the set data invalid or inconsistent, leading to undefined
// behavior elsewhere.
//
// This implementation of Set is specific to our [UniqueKey] and [UniqueKeyer]
// convention here in package addrs, which predated Go supporting type
// parameters. For types outside of addrs consider using the generalized version
// in sibling package "collections". Perhaps one day we'll rework this
// addrs-specific implementation to use [collections.Set] instead.
type Set[T UniqueKeyer] map[UniqueKey]T

func MakeSet[T UniqueKeyer](elems ...T) Set[T] {
	ret := Set[T](make(map[UniqueKey]T, len(elems)))
	for _, elem := range elems {
		ret.Add(elem)
	}
	return ret
}

// Has returns true if and only if the set includes the given address.
func (s Set[T]) Has(addr T) bool {
	_, exists := s[addr.UniqueKey()]
	return exists
}

// Add inserts the given address into the set, if not already present. If
// an equivalent address is already in the set, this replaces that address
// with the new value.
func (s Set[T]) Add(addr T) {
	s[addr.UniqueKey()] = addr
}

// Remove deletes the given address from the set, if present. If not present,
// this is a no-op.
func (s Set[T]) Remove(addr T) {
	delete(s, addr.UniqueKey())
}

// Union returns a new set which contains the union of all of the elements
// of both the reciever and the given other set.
func (s Set[T]) Union(other Set[T]) Set[T] {
	ret := make(Set[T])
	for k, addr := range s {
		ret[k] = addr
	}
	for k, addr := range other {
		ret[k] = addr
	}
	return ret
}

// Intersection returns a new set which contains the intersection of all of the
// elements of both the reciever and the given other set.
func (s Set[T]) Intersection(other Set[T]) Set[T] {
	ret := make(Set[T])
	for k, addr := range s {
		if _, exists := other[k]; exists {
			ret[k] = addr
		}
	}
	for k, addr := range other {
		if _, exists := s[k]; exists {
			ret[k] = addr
		}
	}
	return ret
}

// Sorted returns a slice of all of the elements of the receiving set, sorted
// into an order defined by the given callback function.
//
// The callback should return true if the first element should sort before
// the second, or false otherwise.
func (s Set[T]) Sorted(less func(i, j T) bool) []T {
	if len(s) == 0 {
		return nil
	}
	ret := make([]T, 0, len(s))
	for _, elem := range s {
		ret = append(ret, elem)
	}
	sort.Slice(ret, func(i, j int) bool {
		return less(ret[i], ret[j])
	})
	return ret
}

// SetSortedNatural returns a slice containing the elements of the given set
// sorted into their "natural" order, as defined by the type's method "Less".
//
// For element types that don't have a natural order, or to sort by something
// other than the natural order, use [Set.Sorted] instead.
func SetSortedNatural[T interface {
	UniqueKeyer
	Less(T) bool
}](set Set[T]) []T {
	return set.Sorted(func(i, j T) bool {
		return i.Less(j)
	})
}
