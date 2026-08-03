// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// AcyclicGraph is a specialization of Graph that cannot have cycles.
type AcyclicGraph struct {
	Graph
}

// walkFunc is the callback used for the primary concurrent walk of the graph.
type walkFunc func(Vertex) tfdiags.Diagnostics

// depthWalkFunc is a walk function that also receives the current depth of the
// walk as an argument. This is used for the various synchronous walks of the
// graph.
type depthWalkFunc func(Vertex, int) error

// Returns a VertexSet that includes every Vertex yielded by walking down from the
// provided Vertices vs.
func (g *AcyclicGraph) Ancestors(vs ...Vertex) VertexSet {
	s := NewVertexSet()
	memoFunc := func(v Vertex, d int) error {
		s.Add(v)
		return nil
	}

	start := NewVertexSet()
	for _, v := range vs {
		for dep := range g.edgesFrom[v].All() {
			start.Add(dep)
		}
	}

	if err := g.DepthFirstWalk(start, memoFunc); err != nil {
		return NewVertexSet()
	}

	return s
}

// FirstAncestorsWith returns a Set that includes every Vertex yielded by
// walking down from the provided starting Vertex v, and stopping each branch when
// match returns true. This will return the set of all first ancestors
// encountered which match some criteria.
func (g *AcyclicGraph) FirstAncestorsWith(v Vertex, match func(Vertex) bool) VertexSet {
	s := NewVertexSet()
	searchFunc := func(v Vertex, d int) error {
		if match(v) {
			s.Add(v)
			return errStopWalkBranch
		}

		return nil
	}

	start := NewVertexSet()
	for dep := range g.edgesFrom[v].All() {
		start.Add(dep)
	}

	// our memoFunc doesn't return an error
	g.DepthFirstWalk(start, searchFunc)

	return s
}

// MatchAncestor returns true if the given match function returns true for any
// descendants of the given Vertex.
func (g *AcyclicGraph) MatchAncestor(v Vertex, match func(Vertex) bool) bool {
	var ret bool
	matchFunc := func(v Vertex, d int) error {
		if match(v) {
			ret = true
			return errStopWalk
		}

		return nil
	}

	start := NewVertexSet()
	for dep := range g.edgesFrom[v].All() {
		start.Add(dep)
	}

	// our memoFunc doesn't return an error
	g.DepthFirstWalk(start, matchFunc)

	return ret
}

// Descendants returns a Set that includes every Vertex yielded by walking up
// from the provided starting Vertex v.
func (g *AcyclicGraph) Descendants(v Vertex) VertexSet {
	s := NewVertexSet()
	memoFunc := func(v Vertex, d int) error {
		s.Add(v)
		return nil
	}

	start := NewVertexSet()
	for dep := range g.edgesTo[v].All() {
		start.Add(dep)
	}

	// our memoFunc doesn't return an error
	g.ReverseDepthFirstWalk(start, memoFunc)

	return s
}

// FirstDescendantsWith returns a Set that includes every Vertex yielded by
// walking up from the provided starting Vertex v, and stopping each branch when
// match returns true. This will return the set of all first descendants
// encountered which match some criteria.
func (g *AcyclicGraph) FirstDescendantsWith(v Vertex, match func(Vertex) bool) VertexSet {
	s := NewVertexSet()
	searchFunc := func(v Vertex, d int) error {
		if match(v) {
			s.Add(v)
			return errStopWalkBranch
		}

		return nil
	}

	start := NewVertexSet()
	for dep := range g.edgesTo[v].All() {
		start.Add(dep)
	}

	// our memoFunc doesn't return an error
	g.ReverseDepthFirstWalk(start, searchFunc)

	return s
}

// MatchDescendant returns true if the given match function returns true for any
// descendants of the given Vertex.
func (g *AcyclicGraph) MatchDescendant(v Vertex, match func(Vertex) bool) bool {
	var ret bool
	matchFunc := func(v Vertex, d int) error {
		if match(v) {
			ret = true
			return errStopWalk
		}

		return nil
	}

	start := NewVertexSet()
	for dep := range g.edgesTo[v].All() {
		start.Add(dep)
	}

	// our memoFunc doesn't return an error
	g.ReverseDepthFirstWalk(start, matchFunc)

	return ret
}

// Root returns the root of the DAG, or an error.
//
// Complexity: O(V)
func (g *AcyclicGraph) Root() (Vertex, error) {
	roots := make([]Vertex, 0, 1)
	for v := range g.VerticesSeq() {
		if g.edgesTo[v].Len() == 0 {
			roots = append(roots, v)
		}
	}

	if len(roots) > 1 {
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
	for u := range g.VerticesSeq() {
		uTargets := g.edgesFrom[u]

		g.DepthFirstWalk(g.edgesFrom[u], func(v Vertex, d int) error {
			shared := uTargets.Intersection(g.edgesFrom[v])
			for vPrime := range shared.All() {
				g.RemoveEdge(u, vPrime)
			}

			return nil
		})
	}
}

// Validate validates the DAG. A DAG is valid if it has a single root with no
// cycles. A single root is not required for a graph to be walked, but it's an
// invariant we maintain for consistency.
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
				cycleStr[j] = vertex.Name()
			}

			// Reverse the cycle string so readers can interpret it
			// left-to-right or top-to-bottom.
			slices.Reverse(cycleStr)

			// Since the cycle can start anywhere, pick an arbitrary first node
			// for consistency. We're going to start with the shortest name, then
			// compare lexically.
			first := 0
			for i := 1; i < len(cycleStr); i++ {
				if len(cycleStr[i]) < len(cycleStr[first]) {
					first = i
					continue
				}

				if len(cycleStr[i]) == len(cycleStr[first]) && cycleStr[i] < cycleStr[first] {
					first = i
				}
			}

			// pivot the slice around our new first index
			if first > 0 {
				cycleStr = append(cycleStr[first:], cycleStr[:first]...)
			}

			err = errors.Join(err, fmt.Errorf(
				"Cycle:\n  %s", strings.Join(cycleStr, "\n  "),
			))
		}
	}

	// Look for cycles to self
	for _, e := range g.Edges() {
		if e.Source == e.Target {
			err = errors.Join(err, fmt.Errorf(
				"Self reference: %s", e.Source.Name()))
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

// Walk walks the graph, calling your callback as each node is visited. This
// will walk nodes in concurrently if it can. The resulting diagnostics contains
// problems from all graphs visited, in no particular order.
func (g *AcyclicGraph) Walk(cb walkFunc) tfdiags.Diagnostics {
	w := NewWalker(cb)
	w.Reverse = true
	w.Update(g)
	return w.Wait()
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
	sorted := make([]Vertex, 0, g.vertices.Len())

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
		var next VertexSet
		switch {
		case order&downOrder != 0:
			next = g.edgesFrom[v]
		case order&upOrder != 0:
			next = g.edgesTo[v]
		default:
			panic(fmt.Sprintln("invalid order", order))
		}

		for u := range next.All() {
			visit(u)
		}

		tmp[v] = false
		perm[v] = true
		sorted = append(sorted, v)
	}

	for v := range g.VerticesSeq() {
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

var (
	// stopWalkBranch halts the descent in the current branch of the walk
	// without adding any more edges from the current vertex, and continues with
	// the next vertex already added to the queue.
	errStopWalkBranch = errors.New("stop walk branch")

	// stopWalk halts the entire walk.
	errStopWalk = errors.New("stop walk")
)

// DepthFirstWalk does a depth-first walk of the graph starting from
// the vertices in start.
func (g *AcyclicGraph) DepthFirstWalk(start VertexSet, f depthWalkFunc) error {
	return g.walk(depthFirst|downOrder, false, start, f)
}

// ReverseDepthFirstWalk does a depth-first walk _up_ the graph starting from
// the vertices in start.
func (g *AcyclicGraph) ReverseDepthFirstWalk(start VertexSet, f depthWalkFunc) error {
	return g.walk(depthFirst|upOrder, false, start, f)
}

// BreadthFirstWalk does a breadth-first walk of the graph starting from
// the vertices in start.
func (g *AcyclicGraph) BreadthFirstWalk(start VertexSet, f depthWalkFunc) error {
	return g.walk(breadthFirst|downOrder, false, start, f)
}

// ReverseBreadthFirstWalk does a breadth-first walk _up_ the graph starting from
// the vertices in start.
func (g *AcyclicGraph) ReverseBreadthFirstWalk(start VertexSet, f depthWalkFunc) error {
	return g.walk(breadthFirst|upOrder, false, start, f)
}

type vertexAtDepth struct {
	Vertex Vertex
	Depth  int
}

// Setting test to true will walk sets of vertices in sorted order for
// deterministic testing.
func (g *AcyclicGraph) walk(order walkType, test bool, start VertexSet, f depthWalkFunc) error {
	seen := make(map[Vertex]struct{})
	var frontier []vertexAtDepth

	for v := range start.All() {
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
			switch err {
			case errStopWalk:
				return nil
			case errStopWalkBranch:
				continue
			}
			return err
		}

		var edges VertexSet
		switch {
		case order&downOrder != 0:
			edges = g.edgesFrom[current.Vertex]
		case order&upOrder != 0:
			edges = g.edgesTo[current.Vertex]
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

func appendNext(frontier []vertexAtDepth, next VertexSet, depth int) []vertexAtDepth {
	for v := range next.All() {
		frontier = append(frontier, vertexAtDepth{
			Vertex: v,
			Depth:  depth,
		})
	}
	return frontier
}

func testAppendNextSorted(frontier []vertexAtDepth, edges VertexSet, depth int) []vertexAtDepth {
	var newEdges []vertexAtDepth
	for v := range edges.All() {
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
		return f[i].Vertex.Name() < f[j].Vertex.Name()
	})
}
