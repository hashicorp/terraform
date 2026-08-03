// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"iter"
	"maps"
)

func NewVertexSet() VertexSet {
	return setMap[Vertex]{make(map[Vertex]bool)}
}

type VertexSet = setMap[Vertex]

func newEdgeSet() edgeSet {
	return setMap[Edge]{make(map[Edge]bool)}
}

type edgeSet = setMap[Edge]

// setMap is a set data structure, used to hold
type setMap[T comparable] struct {
	m map[T]bool
}

// Add adds an item to the set
func (s setMap[T]) Add(v T) {
	s.m[v] = true
}

// Delete removes an item from the set.
func (s setMap[T]) Delete(v T) {
	delete(s.m, v)
}

// Contains returns true/false of whether a value is in the set.
func (s setMap[T]) Contains(v T) bool {
	return s.m[v]
}

// Intersection computes the set intersection with other.
func (s setMap[T]) Intersection(other setMap[T]) setMap[T] {
	result := setMap[T]{make(map[T]bool)}
	if s.m == nil || other.m == nil {
		return result
	}
	// Iteration over a smaller set has better performance.
	if other.Len() < s.Len() {
		s, other = other, s
	}
	for v := range s.m {
		if other.Contains(v) {
			result.Add(v)
		}
	}
	return result
}

// Union computes the union of the two sets
func (s setMap[T]) Union(other setMap[T]) setMap[T] {
	result := setMap[T]{maps.Clone(s.m)}
	for v := range other.All() {
		result.Add(v)
	}
	return result
}

// Difference returns a set with the elements that s has but
// other doesn't.
func (s setMap[T]) Difference(other setMap[T]) setMap[T] {
	if other.m == nil || other.Len() == 0 {
		return s.Clone()
	}

	result := setMap[T]{make(map[T]bool)}
	for v := range s.m {
		if _, ok := other.m[v]; !ok {
			result.Add(v)
		}
	}

	return result
}

// Filter returns a set that contains the elements from the receiver
// where the given callback returns true.
func (s setMap[T]) Filter(cb func(T) bool) setMap[T] {
	result := setMap[T]{make(map[T]bool)}

	for v := range s.m {
		if cb(v) {
			result.Add(v)
		}
	}

	return result
}

// Len is the number of items in the set.
func (s setMap[T]) Len() int {
	return len(s.m)
}

// All returns the sequence of set elements.
func (s setMap[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range s.m {
			if !yield(v) {
				return
			}
		}
	}
}

// Clone returns a shallow copy of the set.
func (s setMap[T]) Clone() setMap[T] {
	return setMap[T]{maps.Clone(s.m)}
}
