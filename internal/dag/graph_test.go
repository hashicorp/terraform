// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"fmt"
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
	g.Connect(BasicEdge(testV(1), testV(3)))

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
	g.Connect(BasicEdge(testV(1), testV(3)))
	g.Remove(testV(3))

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphRemoveStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraph_replace(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(BasicEdge(testV(1), testV(2)))
	g.Connect(BasicEdge(testV(2), testV(3)))
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
	g.Connect(BasicEdge(testV(1), testV(2)))
	g.Connect(BasicEdge(testV(2), testV(3)))
	g.Replace(testV(2), testV(2))

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphReplaceSelfStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphHasVertex(t *testing.T) {
	var g Graph
	g.Add(testV(1))

	if !g.HasVertex(testV(1)) {
		t.Fatal("should have 1")
	}
	if g.HasVertex(testV(2)) {
		t.Fatal("should not have 2")
	}
}

func TestGraphHasEdge(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Connect(BasicEdge(testV(1), testV(2)))

	if !g.HasEdge(BasicEdge(testV(1), testV(2))) {
		t.Fatal("should have 1,2")
	}
	if g.HasEdge(BasicEdge(testV(2), testV(3))) {
		t.Fatal("should not have 2,3")
	}
}

/* TODO: move to dag_test with up and down edges
func TestGraphEdgesFrom(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(1, 3))
	g.Connect(BasicEdge(2, 3))

	edges := g.EdgesFrom(1)

	expected := make(Set)
	expected.Add(BasicEdge(1, 3))

	s := make(Set)
	for _, e := range edges {
		s.Add(e)
	}

	if s.Intersection(expected).Len() != expected.Len() {
		t.Fatalf("bad: %#v", edges)
	}
}

func TestGraphEdgesTo(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(1, 3))
	g.Connect(BasicEdge(1, 2))

	edges := g.EdgesTo(3)

	expected := make(Set)
	expected.Add(BasicEdge(1, 3))

	s := make(Set)
	for _, e := range edges {
		s.Add(e)
	}

	if s.Intersection(expected).Len() != expected.Len() {
		fmt.Printf("%#v\n", s)
		t.Fatalf("bad: %#v", edges)
	}
}
*/

func TestGraphUpdownEdges(t *testing.T) {
	// Verify that we can't inadvertently modify the internal graph sets
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(BasicEdge(testV(1), testV(2)))
	g.Connect(BasicEdge(testV(2), testV(3)))

	up := g.UpEdges(testV(2))
	if up.Len() != 1 || !up.Include(testV(1)) {
		t.Fatalf("expected only an up edge of '1', got %#v", up)
	}
	// modify the up set
	up.Add(testV(9))

	orig := g.UpEdges(testV(2))
	diff := up.Difference(orig)
	if diff.Len() != 1 || !diff.Include(testV(9)) {
		t.Fatalf("expected a diff of only '9', got %#v", diff)
	}

	down := g.DownEdges(testV(2))
	if down.Len() != 1 || !down.Include(testV(3)) {
		t.Fatalf("expected only a down edge of '3', got %#v", down)
	}
	// modify the down set
	down.Add(testV(8))

	orig = g.DownEdges(testV(2))
	diff = down.Difference(orig)
	if diff.Len() != 1 || !diff.Include(testV(8)) {
		t.Fatalf("expected a diff of only '8', got %#v", diff)
	}
}

type hashVertex struct {
	code any
}

func (v *hashVertex) Name() string {
	return fmt.Sprintf("%#v", v.code)
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
