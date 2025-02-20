// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"testing"
)

// Mock implementation of SeqVertex for testing
type MockVertex struct {
	id int
}

func (v MockVertex) ZeroValue() any {
	return MockVertex{}
}

type MockVertex2 struct {
	id int
}

func TestSelectSeq(t *testing.T) {
	v1 := MockVertex{id: 1}
	v11 := MockVertex{id: 11}
	v2 := MockVertex2{id: 1}
	vertices := Set{v1: v1, v11: v11, v2: v2}

	graph := &Graph{vertices: vertices}
	seq := SelectSeq(graph.VerticesSeq(), MockVertex{})
	t.Run("Select objects of given type", func(t *testing.T) {
		count := len(seq.Collect())
		if count != 2 {
			t.Errorf("Expected 2, got %d", count)
		}
	})

	t.Run("Returns empty when looking for incompatible types", func(t *testing.T) {
		seq := SelectSeq(seq, MockVertex2{})
		count := len(seq.Collect())
		if count != 0 {
			t.Errorf("Expected empty, got %d", count)
		}
	})

	t.Run("Select objects of given interface", func(t *testing.T) {
		seq := SelectSeq(graph.VerticesSeq(), interface{ ZeroValue() any }(nil))
		count := len(seq.Collect())
		if count != 2 {
			t.Errorf("Expected 1, got %d", count)
		}
	})
}

func TestExcludeSeq(t *testing.T) {
	v1 := MockVertex{id: 1}
	v11 := MockVertex{id: 11}
	v2 := MockVertex2{id: 1}
	vertices := Set{v1: v1, v11: v11, v2: v2}

	graph := &Graph{vertices: vertices}
	seq := ExcludeSeq(graph.VerticesSeq(), MockVertex{})
	t.Run("Exclude objects of given type", func(t *testing.T) {
		count := len(seq.Collect())
		if count != 1 {
			t.Errorf("Expected 1, got %d", count)
		}
	})

	t.Run("Returns empty when looking for incompatible types", func(t *testing.T) {
		seq := ExcludeSeq(seq, MockVertex2{})
		count := len(seq.Collect())
		if count != 0 {
			t.Errorf("Expected empty, got %d", count)
		}
	})

	t.Run("Exclude objects of given interface", func(t *testing.T) {
		seq := ExcludeSeq(graph.VerticesSeq(), interface{ ZeroValue() any }(nil))
		count := len(seq.Collect())
		if count != 1 {
			t.Errorf("Expected 1, got %d", count)
		}
	})
}
