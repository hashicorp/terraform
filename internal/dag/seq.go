// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"iter"
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

// SelectSeq filters a sequence to include only elements that can be type-asserted to type U.
// It returns a new sequence containing only the matching elements.
// The yield function can return false to stop iteration early.
func SelectSeq[T Vertex, U Vertex](seq VertexSeq[T], filter func(U)) VertexSeq[U] {
	return func(yield func(U) bool) {
		for v := range seq {
			// if the item is not of the type we're looking for, skip it
			u, ok := any(v).(U)
			if !ok {
				continue
			}
			if !yield(u) {
				return
			}
		}
	}
}

// ExcludeSeq filters a sequence to exclude elements that can be type-asserted to type U.
// It returns a new sequence containing only the non-matching elements.
// The yield function can return false to stop iteration early.
func ExcludeSeq[T Vertex, U Vertex](seq VertexSeq[T], filter func(U)) VertexSeq[T] {
	return func(yield func(T) bool) {
		for v := range seq {
			if _, ok := any(v).(U); ok {
				continue
			}
			if !yield(v) {
				return
			}
		}
	}
}
