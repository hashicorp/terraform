// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"strings"
	"testing"
)

func TestGraph_empty(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphEmptyStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraph_basic(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(1), testV(3))

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphBasicStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraph_remove(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(1), testV(3))
	g.Remove(testV(3))

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphRemoveStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraph_removeEdges(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(1), testV(3))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(3))

	g.RemoveEdge(testV(1), testV(2))
	g.RemoveEdge(testV(1), testV(3))

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphRemoveEdges)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraph_replace(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(3))
	g.Replace(testV(2), testV(42))

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphReplaceStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraph_replaceSelf(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(3))
	g.Replace(testV(2), testV(2))

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphReplaceSelfStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphEdgesFrom(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Add(testV(4))

	g.Connect(testV(1), testV(3))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(3))

	edges := g.EdgesFrom(testV(1))

	expected := NewVertexSet()
	expected.Add(testV(3))
	expected.Add(testV(2))

	if edges.Intersection(expected).Len() != expected.Len() {
		t.Fatalf("bad: %#v", edges)
	}
}

func TestGraphEdgesTo(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Add(testV(4))

	g.Connect(testV(1), testV(3))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(3))

	edges := g.EdgesTo(testV(3))

	expected := NewVertexSet()
	expected.Add(testV(1))
	expected.Add(testV(2))

	if edges.Intersection(expected).Len() != expected.Len() {
		t.Fatalf("bad: %#v", edges)
	}
}

func TestGraphUpdownEdges(t *testing.T) {
	// Verify that we can't inadvertently modify the internal graph sets
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(3))

	up := g.EdgesTo(testV(2))
	if up.Len() != 1 || !up.Contains(testV(1)) {
		t.Fatalf("expected only an up edge of '1', got %#v", up)
	}
	// modify the up set
	up.Add(testV(9))

	orig := g.EdgesTo(testV(2))
	diff := up.Difference(orig)
	if diff.Len() != 1 || !diff.Contains(testV(9)) {
		t.Fatalf("expected a diff of only '9', got %#v", diff)
	}

	down := g.EdgesFrom(testV(2))
	if down.Len() != 1 || !down.Contains(testV(3)) {
		t.Fatalf("expected only a down edge of '3', got %#v", down)
	}
	// modify the down set
	down.Add(testV(8))

	orig = g.EdgesFrom(testV(2))
	diff = down.Difference(orig)
	if diff.Len() != 1 || !diff.Contains(testV(8)) {
		t.Fatalf("expected a diff of only '8', got %#v", diff)
	}
}

const testGraphBasicStr = `
1
  3
2
3
`

const testGraphEmptyStr = `
1
2
3
`

const testGraphRemoveStr = `
1
2
`

const testGraphRemoveEdges = `
1
2
  3
3
`

const testGraphReplaceStr = `
1
  42
3
42
  3
`

const testGraphReplaceSelfStr = `
1
  2
2
  3
3
`
