// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"bytes"
	"fmt"
	"sort"
)

// Graph is used to represent a dependency graph.
type Graph struct {
	vertices VertexSet

	// The information here is redundant, but because we traverse the graph so
	// frequently the duplication pays off in much better performance
	edgesFrom map[Vertex]VertexSet
	edgesTo   map[Vertex]VertexSet
}

// Vertex of the graph.
type Vertex interface {
	Name() string
}

// TolerantVertex is an optional interface that can be implemented by Vertex
// to allow it to tolerate upstream failures.
type TolerantVertex interface {
	Vertex

	// AllowUpstreamFailure returns true if the receiver vertex can tolerate a
	// failure in the given vertex.
	AllowUpstreamFailure(Vertex) bool
}

func (g *Graph) VertexCount() int {
	return g.vertices.Len()
}

// edgeSet is used to collect all dependencies for the concurrent graph walk.
// Because the Walker allows dynamically updating the graph being walked, a set
// data structure is useful to diff the two sets of dependencies.
func (g *Graph) edgeSet() edgeSet {
	edges := edgeSet{make(map[Edge]Edge)}
	for from, tos := range g.edgesFrom {
		for to := range tos.All() {
			edges.Add(Edge{from, to})
		}
	}
	return edges
}

// Edges returns the list of all the edges in the graph.
func (g *Graph) Edges() []Edge {
	result := make([]Edge, 0, len(g.edgesFrom))

	for from, tos := range g.edgesFrom {
		for to := range tos.All() {
			result = append(result, Edge{from, to})
		}
	}

	return result
}

// Add adds a vertex to the graph. This is safe to call multiple time with
// the same Vertex.
func (g *Graph) Add(v Vertex) Vertex {
	g.init()
	g.vertices.Add(v)
	return v
}

// Remove removes a vertex from the graph. This will also remove any
// edges with this vertex as a source or target.
func (g *Graph) Remove(v Vertex) {
	// Delete the vertex itself
	g.vertices.Delete(v)

	// Delete the edges to non-existent things
	for target := range g.edgesFrom[v].All() {
		g.RemoveEdge(v, target)
	}
	for source := range g.edgesTo[v].All() {
		g.RemoveEdge(source, v)
	}
}

// Replace replaces the original Vertex with replacement. If the original
// does not exist within the graph, then false is returned. Otherwise, true
// is returned.
func (g *Graph) Replace(original, replacement Vertex) bool {
	// If we don't have the original, we can't do anything
	if !g.vertices.Contains(original) {
		return false
	}

	// If they're the same, then don't do anything
	if original == replacement {
		return true
	}

	// Add our new vertex, then copy all the edges
	g.Add(replacement)
	for target := range g.edgesFrom[original].All() {
		g.Connect(replacement, target)
	}
	for source := range g.edgesTo[original].All() {
		g.Connect(source, replacement)
	}

	// Remove our old vertex, which will also remove all the edges
	g.Remove(original)

	return true
}

// RemoveEdge removes an edge from the graph.
func (g *Graph) RemoveEdge(from, to Vertex) {
	g.init()

	// Delete the up/down edges
	if s, ok := g.edgesFrom[from]; ok {
		s.Delete(to)
	}
	if s, ok := g.edgesTo[from]; ok {
		s.Delete(from)
	}
}

// EdgesTo returns the vertices that are *sources* of edges that target the
// destination Vertex v.
func (g *Graph) EdgesTo(v Vertex) VertexSet {
	return g.edgesTo[v].Clone()
}

// EdgesFrom returns the vertices that are *targets* of edges that originate
// from the source Vertex v.
func (g *Graph) EdgesFrom(v Vertex) VertexSet {
	return g.edgesFrom[v].Clone()
}

// Connect adds an edge with the given source and target. This is safe to
// call multiple times with the same value.
func (g *Graph) Connect(source, target Vertex) {
	g.init()

	// Do we have this already? If so, don't add it again.
	if s, ok := g.edgesFrom[source]; ok && s.Contains(target) {
		return
	}

	// Add the down edge
	s, ok := g.edgesFrom[source]
	if !ok {
		s = NewVertexSet()
		g.edgesFrom[source] = s
	}
	s.Add(target)

	// Add the up edge
	s, ok = g.edgesTo[target]
	if !ok {
		s = NewVertexSet()
		g.edgesTo[target] = s
	}
	s.Add(source)
}

// Subsume imports all of the nodes and edges from the given graph into the
// receiver, leaving the given graph unchanged.
//
// If any of the nodes in the given graph are already present in the receiver
// then the existing node will be retained and any new edges from the given
// graph will be connected with it.
func (g *Graph) Subsume(other *Graph) {
	g.init()
	g.vertices = g.vertices.Union(other.vertices)

	for v := range other.edgesFrom {
		g.edgesFrom[v] = other.edgesFrom[v].Clone()
	}
	for v := range other.edgesTo {
		g.edgesTo[v] = other.edgesTo[v].Clone()
	}
}

// String outputs some human-friendly output for the graph structure.
func (g *Graph) StringWithNodeTypes() string {
	var buf bytes.Buffer

	// Build the list of node names and a mapping so that we can more
	// easily alphabetize the output to remain deterministic.
	names := make([]string, 0, g.vertices.Len())
	mapping := make(map[string]Vertex, g.vertices.Len())
	for v := range g.VerticesSeq() {
		name := v.Name()
		names = append(names, name)
		mapping[name] = v
	}
	sort.Strings(names)

	// Write each node in order...
	for _, name := range names {
		v := mapping[name]
		targets := g.edgesFrom[v]

		buf.WriteString(fmt.Sprintf("%s - %T\n", name, v))

		// Alphabetize dependencies
		deps := make([]string, 0, targets.Len())
		targetNodes := make(map[string]Vertex)
		for target := range targets.All() {
			dep := target.Name()
			deps = append(deps, dep)
			targetNodes[dep] = target
		}
		sort.Strings(deps)

		// Write dependencies
		for _, d := range deps {
			buf.WriteString(fmt.Sprintf("  %s - %T\n", d, targetNodes[d]))
		}
	}

	return buf.String()
}

// String outputs some human-friendly output for the graph structure.
func (g *Graph) String() string {
	var buf bytes.Buffer

	// Build the list of node names and a mapping so that we can more
	// easily alphabetize the output to remain deterministic.
	names := make([]string, 0, g.vertices.Len())
	mapping := make(map[string]Vertex, g.vertices.Len())
	for v := range g.VerticesSeq() {
		name := v.Name()
		names = append(names, name)
		mapping[name] = v
	}
	sort.Strings(names)

	// Write each node in order...
	for _, name := range names {
		v := mapping[name]
		targets := g.edgesFrom[v]

		buf.WriteString(fmt.Sprintf("%s\n", name))

		// Alphabetize dependencies
		deps := make([]string, 0, targets.Len())
		for target := range targets.All() {
			deps = append(deps, target.Name())
		}
		sort.Strings(deps)

		// Write dependencies
		for _, d := range deps {
			buf.WriteString(fmt.Sprintf("  %s\n", d))
		}
	}

	return buf.String()
}

func (g *Graph) init() {
	if g.vertices.m == nil {
		g.vertices = NewVertexSet()
	}
	if g.edgesFrom == nil {
		g.edgesFrom = make(map[Vertex]VertexSet)
	}
	if g.edgesTo == nil {
		g.edgesTo = make(map[Vertex]VertexSet)
	}
}

// Dot returns a dot-formatted representation of the Graph.
func (g *Graph) Dot(opts *DotOpts) []byte {
	return newMarshalGraph("", g).Dot(opts)
}

// Mermaid returns a Mermaid flowchart formatted representation of the Graph.
func (g *Graph) Mermaid(opts *DotOpts) []byte {
	return newMarshalGraph("", g).Mermaid(opts)
}
