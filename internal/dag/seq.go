// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"iter"
	"reflect"
	"slices"
)

type VertexSeq[T Vertex] iter.Seq[T]

func (seq VertexSeq[T]) Collect() []T {
	return slices.Collect(iter.Seq[T](seq))
}

// Vertices returns an iterator over all the vertices in the graph.
func (g *Graph) VerticesSeq() VertexSeq[Vertex] {
	return func(yield func(v Vertex) bool) {
		for _, v := range g.vertices {
			if _, ok := v.(Vertex); !ok {
				continue
			}
			if !yield(v.(Vertex)) {
				return
			}
		}
	}
}

// SelectSeq filters a sequence based on whether elements implement a given interface,
// or are the zero value of a given type. It returns a new sequence containing only
// the elements that match the criteria.
func SelectSeq[T Vertex, U Vertex](seq VertexSeq[T], ty U) VertexSeq[U] {
	return func(yield func(U) bool) {
		targetType := reflect.TypeFor[U]()
		for v := range seq {
			// Check if this item is of the type interface we're looking for
			if targetType.Kind() == reflect.Interface {
				if !reflect.TypeOf(v).Implements(targetType) {
					continue
				}
			} else if reflect.TypeOf(v) != targetType {
				// Check if this item is the zero value of the type we're looking for
				continue
			}
			if !yield(any(v).(U)) {
				return
			}
		}
	}
}

// ExcludeSeq filters a sequence based on whether elements implement a given interface,
// or are the zero value of a given type. It returns a new sequence containing only
// the elements that do not match the criteria.
func ExcludeSeq[T Vertex, U Vertex](seq VertexSeq[T], ty U) VertexSeq[T] {
	return func(yield func(T) bool) {
		targetType := reflect.TypeFor[U]()
		for v := range seq {
			// Skip if this item is of the type interface we're looking for
			if targetType.Kind() == reflect.Interface {
				if reflect.TypeOf(v).Implements(targetType) {
					continue
				}
			} else if reflect.TypeOf(v) == targetType {
				// Skip if this item is the zero value of the type we're looking for
				continue
			}
			if !yield(v) {
				return
			}
		}
	}
}
