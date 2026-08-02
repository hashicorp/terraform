// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"bytes"
	"fmt"
	"maps"
	"sort"
)

// Graph is used to represent a dependency graph.
type Graph struct {
	vertices  Set
	edgesFrom map[Vertex]Set
	edgesTo   map[Vertex]Set
}

// Subgrapher allows a Vertex to be a Graph itself, by returning a Grapher.
type Subgrapher interface {
	Subgraph() Grapher
}

// A Grapher is any type that returns a Grapher, mainly used to identify
// dag.Graph and dag.AcyclicGraph.  In the case of Graph and AcyclicGraph, they
// return themselves.
type Grapher interface {
	DirectedGraph() Grapher
}

// Vertex of the graph.
type Vertex = interface {
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

func (g *Graph) DirectedGraph() Grapher {
	return g
}

// Vertices returns the list of all the vertices in the graph.
func (g *Graph) Vertices() []Vertex {
	result := make([]Vertex, 0, len(g.vertices))
	for _, v := range g.vertices {
		result = append(result, v)
	}

	return result
}

func (g *Graph) edgeSet() edgeSet {
	edges := make(edgeSet)
	for from, tos := range g.edgesFrom {
		for _, to := range tos {
			edges.Add(Edge{from, to})
		}
	}
	return edges
}

// Edges returns the list of all the edges in the graph.
func (g *Graph) Edges() []Edge {
	result := make([]Edge, 0, len(g.edgesTo)+len(g.edgesFrom))

	for from, tos := range g.edgesFrom {
		for _, to := range tos {
			result = append(result, Edge{from, to})
		}
	}

	return result
}

// HasVertex checks if the given Vertex is present in the graph.
func (g *Graph) HasVertex(v Vertex) bool {
	return g.vertices.Include(v)
}

// HasEdge checks if the given Edge is present in the graph.
func (g *Graph) HasEdge(from, to Vertex) bool {
	tos, hasFrom := g.edgesFrom[from]
	if !hasFrom {
		return false
	}
	return tos.Include(to)
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
func (g *Graph) Remove(v Vertex) Vertex {
	// Delete the vertex itself
	g.vertices.Delete(v)

	// Delete the edges to non-existent things
	for _, target := range g.edgesFromNoCopy(v) {
		g.RemoveEdge(v, target)
	}
	for _, source := range g.edgesToNoCopy(v) {
		g.RemoveEdge(source, v)
	}

	return nil
}

// Replace replaces the original Vertex with replacement. If the original
// does not exist within the graph, then false is returned. Otherwise, true
// is returned.
func (g *Graph) Replace(original, replacement Vertex) bool {
	// If we don't have the original, we can't do anything
	if !g.vertices.Include(original) {
		return false
	}

	// If they're the same, then don't do anything
	if original == replacement {
		return true
	}

	// Add our new vertex, then copy all the edges
	g.Add(replacement)
	for _, target := range g.edgesFromNoCopy(original) {
		g.Connect(replacement, target)
	}
	for _, source := range g.edgesToNoCopy(original) {
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
		// FIXME: is the correct and is there a test?
		s.Delete(from)
	}
}

// UpEdges returns the vertices that are *sources* of edges that target the
// destination Vertex v.
func (g *Graph) UpEdges(v Vertex) Set {
	return g.edgesToNoCopy(v).Copy()
}

// DownEdges returns the vertices that are *targets* of edges that originate
// from the source Vertex v.
func (g *Graph) DownEdges(v Vertex) Set {
	return g.edgesFromNoCopy(v).Copy()
}

// edgesFromNoCopy returns the vertices targeted by edges from the source Vertex
// v as a Set. This Set is the same as used internally by the Graph to prevent a
// copy, and must not be modified by the caller.
func (g *Graph) edgesFromNoCopy(v Vertex) Set {
	g.init()
	return g.edgesFrom[v]
}

// edgesToNoCopy returns the vertices that are sources of edges targeting the
// destination Vertex v as a Set. This Set is the same as used internally by the
// Graph to prevent a copy, and must not be modified by the caller.
func (g *Graph) edgesToNoCopy(v Vertex) Set {
	g.init()
	return g.edgesTo[v]
}

// Connect adds an edge with the given source and target. This is safe to
// call multiple times with the same value.
func (g *Graph) Connect(source, target Vertex) {
	g.init()

	// Do we have this already? If so, don't add it again.
	if s, ok := g.edgesFrom[source]; ok && s.Include(target) {
		return
	}

	// Add the down edge
	s, ok := g.edgesFrom[source]
	if !ok {
		s = make(Set)
		g.edgesFrom[source] = s
	}
	s.Add(target)

	// Add the up edge
	s, ok = g.edgesTo[target]
	if !ok {
		s = make(Set)
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
	maps.Insert(g.vertices, maps.All(other.vertices))

	for v := range other.edgesFrom {
		g.edgesFrom[v] = other.DownEdges(v)
	}
	for v := range other.edgesTo {
		g.edgesTo[v] = other.UpEdges(v)
	}
}

// String outputs some human-friendly output for the graph structure.
func (g *Graph) StringWithNodeTypes() string {
	var buf bytes.Buffer

	// Build the list of node names and a mapping so that we can more
	// easily alphabetize the output to remain deterministic.
	vertices := g.Vertices()
	names := make([]string, 0, len(vertices))
	mapping := make(map[string]Vertex, len(vertices))
	for _, v := range vertices {
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
		for _, target := range targets {
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
	vertices := g.Vertices()
	names := make([]string, 0, len(vertices))
	mapping := make(map[string]Vertex, len(vertices))
	for _, v := range vertices {
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
		for _, target := range targets {
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
	if g.vertices == nil {
		g.vertices = make(Set)
	}
	if g.edgesFrom == nil {
		g.edgesFrom = make(map[Vertex]Set)
	}
	if g.edgesTo == nil {
		g.edgesTo = make(map[Vertex]Set)
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
