// Copyright IBM Corp. 2014, 2026
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

func (seq VertexSeq[T]) AsGeneric() VertexSeq[Vertex] {
	return func(yield func(Vertex) bool) {
		for v := range seq {
			if !yield(v) {
				return
			}
		}
	}
}

// Vertices returns an iterator over all the vertices in the graph.
func (g *Graph) VerticesSeq() VertexSeq[Vertex] {
	return func(yield func(v Vertex) bool) {
		for _, v := range g.vertices {
			v, ok := v.(Vertex)
			if !ok {
				continue
			}
			if !yield(v) {
				return
			}
		}
	}
}

// SelectSeq filters a sequence to include only elements that can be type-asserted to type U.
// It returns a new sequence containing only the matching elements.
// The yield function can return false to stop iteration early.
func SelectSeq[U Vertex](seq VertexSeq[Vertex]) VertexSeq[U] {
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
func ExcludeSeq[U Vertex](seq VertexSeq[Vertex]) VertexSeq[Vertex] {
	return func(yield func(Vertex) bool) {
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
