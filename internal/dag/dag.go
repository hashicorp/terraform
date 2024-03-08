// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// AcyclicGraph is a specialization of Graph that cannot have cycles.
type AcyclicGraph struct {
	Graph
}

// WalkFunc is the callback used for walking the graph.
type WalkFunc func(Vertex) tfdiags.Diagnostics

// DepthWalkFunc is a walk function that also receives the current depth of the
// walk as an argument
type DepthWalkFunc func(Vertex, int) error

func (g *AcyclicGraph) DirectedGraph() Grapher {
	return g
}

// Returns a Set that includes every Vertex yielded by walking down from the
// provided starting Vertex v.
func (g *AcyclicGraph) Ancestors(v Vertex) (Set, error) {
	s := make(Set)
	memoFunc := func(v Vertex, d int) error {
		s.Add(v)
		return nil
	}

	if err := g.DepthFirstWalk(g.downEdgesNoCopy(v), memoFunc); err != nil {
		return nil, err
	}

	return s, nil
}

// Returns a Set that includes every Vertex yielded by walking up from the
// provided starting Vertex v.
func (g *AcyclicGraph) Descendents(v Vertex) (Set, error) {
	s := make(Set)
	memoFunc := func(v Vertex, d int) error {
		s.Add(v)
		return nil
	}

	if err := g.ReverseDepthFirstWalk(g.upEdgesNoCopy(v), memoFunc); err != nil {
		return nil, err
	}

	return s, nil
}

// Root returns the root of the DAG, or an error.
//
// Complexity: O(V)
func (g *AcyclicGraph) Root() (Vertex, error) {
	roots := make([]Vertex, 0, 1)
	for _, v := range g.Vertices() {
		if g.upEdgesNoCopy(v).Len() == 0 {
			roots = append(roots, v)
		}
	}

	if len(roots) > 1 {
		// TODO(mitchellh): make this error message a lot better
		return nil, fmt.Errorf("multiple roots: %#v", roots)
	}

	if len(roots) == 0 {
		return nil, fmt.Errorf("no roots found")
	}

	return roots[0], nil
}

// TransitiveReduction performs the transitive reduction of graph g in place.
// The transitive reduction of a graph is a graph with as few edges as
// possible with the same reachability as the original graph. This means
// that if there are three nodes A => B => C, and A connects to both
// B and C, and B connects to C, then the transitive reduction is the
// same graph with only a single edge between A and B, and a single edge
// between B and C.
//
// The graph must be free of cycles for this operation to behave properly.
//
// Complexity: O(V(V+E)), or asymptotically O(VE)
func (g *AcyclicGraph) TransitiveReduction() {
	// For each vertex u in graph g, do a DFS starting from each vertex
	// v such that the edge (u,v) exists (v is a direct descendant of u).
	//
	// For each v-prime reachable from v, remove the edge (u, v-prime).
	for _, u := range g.Vertices() {
		uTargets := g.downEdgesNoCopy(u)

		g.DepthFirstWalk(g.downEdgesNoCopy(u), func(v Vertex, d int) error {
			shared := uTargets.Intersection(g.downEdgesNoCopy(v))
			for _, vPrime := range shared {
				g.RemoveEdge(BasicEdge(u, vPrime))
			}

			return nil
		})
	}
}

// Validate validates the DAG. A DAG is valid if it has a single root
// with no cycles.
func (g *AcyclicGraph) Validate() error {
	if _, err := g.Root(); err != nil {
		return err
	}

	// Look for cycles of more than 1 component
	var err error
	cycles := g.Cycles()
	if len(cycles) > 0 {
		for _, cycle := range cycles {
			cycleStr := make([]string, len(cycle))
			for j, vertex := range cycle {
				cycleStr[j] = VertexName(vertex)
			}

			err = errors.Join(err, fmt.Errorf(
				"Cycle: %s", strings.Join(cycleStr, ", ")))
		}
	}

	// Look for cycles to self
	for _, e := range g.Edges() {
		if e.Source() == e.Target() {
			err = errors.Join(err, fmt.Errorf(
				"Self reference: %s", VertexName(e.Source())))
		}
	}

	return err
}

// Cycles reports any cycles between graph nodes.
// Self-referencing nodes are not reported, and must be detected separately.
func (g *AcyclicGraph) Cycles() [][]Vertex {
	var cycles [][]Vertex
	for _, cycle := range StronglyConnected(&g.Graph) {
		if len(cycle) > 1 {
			cycles = append(cycles, cycle)
		}
	}
	return cycles
}

// Walk walks the graph, calling your callback as each node is visited.
// This will walk nodes in parallel if it can. The resulting diagnostics
// contains problems from all graphs visited, in no particular order.
func (g *AcyclicGraph) Walk(cb WalkFunc) tfdiags.Diagnostics {
	w := &Walker{Callback: cb, Reverse: true}
	w.Update(g)
	return w.Wait()
}

// simple convenience helper for converting a dag.Set to a []Vertex
func AsVertexList(s Set) []Vertex {
	vertexList := make([]Vertex, 0, len(s))
	for _, raw := range s {
		vertexList = append(vertexList, raw.(Vertex))
	}
	return vertexList
}

type vertexAtDepth struct {
	Vertex Vertex
	Depth  int
}

// TopologicalOrder returns a topological sort of the given graph, with source
// vertices ordered before the targets of their edges. The nodes are not sorted,
// and any valid order may be returned. This function will panic if it
// encounters a cycle.
func (g *AcyclicGraph) TopologicalOrder() []Vertex {
	return g.topoOrder(upOrder)
}

// ReverseTopologicalOrder returns a topological sort of the given graph, with
// target vertices ordered before the sources of their edges. The nodes are not
// sorted, and any valid order may be returned. This function will panic if it
// encounters a cycle.
func (g *AcyclicGraph) ReverseTopologicalOrder() []Vertex {
	return g.topoOrder(downOrder)
}

func (g *AcyclicGraph) topoOrder(order walkType) []Vertex {
	// Use a dfs-based sorting algorithm, similar to that used in
	// TransitiveReduction.
	sorted := make([]Vertex, 0, len(g.vertices))

	// tmp track the current working node to check for cycles
	tmp := map[Vertex]bool{}

	// perm tracks completed nodes to end the recursion
	perm := map[Vertex]bool{}

	var visit func(v Vertex)

	visit = func(v Vertex) {
		if perm[v] {
			return
		}

		if tmp[v] {
			panic("cycle found in dag")
		}

		tmp[v] = true
		var next Set
		switch {
		case order&downOrder != 0:
			next = g.downEdgesNoCopy(v)
		case order&upOrder != 0:
			next = g.upEdgesNoCopy(v)
		default:
			panic(fmt.Sprintln("invalid order", order))
		}

		for _, u := range next {
			visit(u)
		}

		tmp[v] = false
		perm[v] = true
		sorted = append(sorted, v)
	}

	for _, v := range g.Vertices() {
		visit(v)
	}

	return sorted
}

type walkType uint64

const (
	depthFirst walkType = 1 << iota
	breadthFirst
	downOrder
	upOrder
)

// DepthFirstWalk does a depth-first walk of the graph starting from
// the vertices in start.
func (g *AcyclicGraph) DepthFirstWalk(start Set, f DepthWalkFunc) error {
	return g.walk(depthFirst|downOrder, false, start, f)
}

// ReverseDepthFirstWalk does a depth-first walk _up_ the graph starting from
// the vertices in start.
func (g *AcyclicGraph) ReverseDepthFirstWalk(start Set, f DepthWalkFunc) error {
	return g.walk(depthFirst|upOrder, false, start, f)
}

// BreadthFirstWalk does a breadth-first walk of the graph starting from
// the vertices in start.
func (g *AcyclicGraph) BreadthFirstWalk(start Set, f DepthWalkFunc) error {
	return g.walk(breadthFirst|downOrder, false, start, f)
}

// ReverseBreadthFirstWalk does a breadth-first walk _up_ the graph starting from
// the vertices in start.
func (g *AcyclicGraph) ReverseBreadthFirstWalk(start Set, f DepthWalkFunc) error {
	return g.walk(breadthFirst|upOrder, false, start, f)
}

// Setting test to true will walk sets of vertices in sorted order for
// deterministic testing.
func (g *AcyclicGraph) walk(order walkType, test bool, start Set, f DepthWalkFunc) error {
	seen := make(map[Vertex]struct{})
	frontier := make([]vertexAtDepth, 0, len(start))
	for _, v := range start {
		frontier = append(frontier, vertexAtDepth{
			Vertex: v,
			Depth:  0,
		})
	}

	if test {
		testSortFrontier(frontier)
	}

	for len(frontier) > 0 {
		// Pop the current vertex
		var current vertexAtDepth

		switch {
		case order&depthFirst != 0:
			// depth first, the frontier is used like a stack
			n := len(frontier)
			current = frontier[n-1]
			frontier = frontier[:n-1]
		case order&breadthFirst != 0:
			// breadth first, the frontier is used like a queue
			current = frontier[0]
			frontier = frontier[1:]
		default:
			panic(fmt.Sprint("invalid visit order", order))
		}

		// Check if we've seen this already and return...
		if _, ok := seen[current.Vertex]; ok {
			continue
		}
		seen[current.Vertex] = struct{}{}

		// Visit the current node
		if err := f(current.Vertex, current.Depth); err != nil {
			return err
		}

		var edges Set
		switch {
		case order&downOrder != 0:
			edges = g.downEdgesNoCopy(current.Vertex)
		case order&upOrder != 0:
			edges = g.upEdgesNoCopy(current.Vertex)
		default:
			panic(fmt.Sprint("invalid walk order", order))
		}

		if test {
			frontier = testAppendNextSorted(frontier, edges, current.Depth+1)
		} else {
			frontier = appendNext(frontier, edges, current.Depth+1)
		}
	}
	return nil
}

func appendNext(frontier []vertexAtDepth, next Set, depth int) []vertexAtDepth {
	for _, v := range next {
		frontier = append(frontier, vertexAtDepth{
			Vertex: v,
			Depth:  depth,
		})
	}
	return frontier
}

func testAppendNextSorted(frontier []vertexAtDepth, edges Set, depth int) []vertexAtDepth {
	var newEdges []vertexAtDepth
	for _, v := range edges {
		newEdges = append(newEdges, vertexAtDepth{
			Vertex: v,
			Depth:  depth,
		})
	}
	testSortFrontier(newEdges)
	return append(frontier, newEdges...)
}
func testSortFrontier(f []vertexAtDepth) {
	sort.Slice(f, func(i, j int) bool {
		return VertexName(f[i].Vertex) < VertexName(f[j].Vertex)
	})
}
