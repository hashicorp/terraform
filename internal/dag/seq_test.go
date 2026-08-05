// Copyright IBM Corp. 2014, 2026
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
	v2 := MockVertex2{id: 2}
	vertices := Set{v1: v1, v11: v11, v2: v2}

	graph := &Graph{vertices: vertices}
	seq := SelectSeq[MockVertex](graph.VerticesSeq())
	t.Run("Select objects of given type", func(t *testing.T) {
		count := len(seq.Collect())
		if count != 2 {
			t.Errorf("Expected 2, got %d", count)
		}
	})

	t.Run("Returns empty when looking for incompatible types", func(t *testing.T) {
		seq := SelectSeq[MockVertex2](seq.AsGeneric())
		count := len(seq.Collect())
		if count != 0 {
			t.Errorf("Expected empty, got %d", count)
		}
	})

	t.Run("Select objects of given interface", func(t *testing.T) {
		seq := SelectSeq[interface{ ZeroValue() any }](graph.VerticesSeq())
		count := len(seq.Collect())
		if count != 2 {
			t.Errorf("Expected 1, got %d", count)
		}
	})
}

func TestExcludeSeq(t *testing.T) {
	v1 := MockVertex{id: 1}
	v11 := MockVertex{id: 11}
	v2 := MockVertex2{id: 2}
	vertices := Set{v1: v1, v11: v11, v2: v2}

	graph := &Graph{vertices: vertices}
	seq := ExcludeSeq[MockVertex](graph.VerticesSeq())
	t.Run("Exclude objects of given type", func(t *testing.T) {
		count := len(seq.Collect())
		if count != 1 {
			t.Errorf("Expected 1, got %d", count)
		}
	})

	t.Run("Returns empty when looking for incompatible types", func(t *testing.T) {
		seq := ExcludeSeq[MockVertex2](seq)
		count := len(seq.Collect())
		if count != 0 {
			t.Errorf("Expected empty, got %d", count)
		}
	})

	t.Run("Exclude objects of given interface", func(t *testing.T) {
		seq := ExcludeSeq[interface{ ZeroValue() any }](graph.VerticesSeq())
		count := len(seq.Collect())
		if count != 1 {
			t.Errorf("Expected 1, got %d", count)
		}
	})
}
