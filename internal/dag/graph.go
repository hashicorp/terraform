// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dag

import (
	"bytes"
	"fmt"
	"sort"
)

// Graph is used to represent a dependency graph.
type Graph struct {
	vertices  Set
	edges     Set
	downEdges map[interface{}]Set
	upEdges   map[interface{}]Set
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
type Vertex interface{}

// NamedVertex is an optional interface that can be implemented by Vertex
// to give it a human-friendly name that is used for outputting the graph.
type NamedVertex interface {
	Vertex
	Name() string
}

func (g *Graph) DirectedGraph() Grapher {
	return g
}

// Vertices returns the list of all the vertices in the graph.
func (g *Graph) Vertices() []Vertex {
	result := make([]Vertex, 0, len(g.vertices))
	for _, v := range g.vertices {
		result = append(result, v.(Vertex))
	}

	return result
}

// Edges returns the list of all the edges in the graph.
func (g *Graph) Edges() []Edge {
	result := make([]Edge, 0, len(g.edges))
	for _, v := range g.edges {
		result = append(result, v.(Edge))
	}

	return result
}

// EdgesFrom returns the list of edges from the given source.
func (g *Graph) EdgesFrom(v Vertex) []Edge {
	var result []Edge
	from := hashcode(v)
	for _, e := range g.Edges() {
		if hashcode(e.Source()) == from {
			result = append(result, e)
		}
	}

	return result
}

// EdgesTo returns the list of edges to the given target.
func (g *Graph) EdgesTo(v Vertex) []Edge {
	var result []Edge
	search := hashcode(v)
	for _, e := range g.Edges() {
		if hashcode(e.Target()) == search {
			result = append(result, e)
		}
	}

	return result
}

// HasVertex checks if the given Vertex is present in the graph.
func (g *Graph) HasVertex(v Vertex) bool {
	return g.vertices.Include(v)
}

// HasEdge checks if the given Edge is present in the graph.
func (g *Graph) HasEdge(e Edge) bool {
	return g.edges.Include(e)
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
	for _, target := range g.downEdgesNoCopy(v) {
		g.RemoveEdge(BasicEdge(v, target))
	}
	for _, source := range g.upEdgesNoCopy(v) {
		g.RemoveEdge(BasicEdge(source, v))
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
	for _, target := range g.downEdgesNoCopy(original) {
		g.Connect(BasicEdge(replacement, target))
	}
	for _, source := range g.upEdgesNoCopy(original) {
		g.Connect(BasicEdge(source, replacement))
	}

	// Remove our old vertex, which will also remove all the edges
	g.Remove(original)

	return true
}

// RemoveEdge removes an edge from the graph.
func (g *Graph) RemoveEdge(edge Edge) {
	g.init()

	// Delete the edge from the set
	g.edges.Delete(edge)

	// Delete the up/down edges
	if s, ok := g.downEdges[hashcode(edge.Source())]; ok {
		s.Delete(edge.Target())
	}
	if s, ok := g.upEdges[hashcode(edge.Target())]; ok {
		s.Delete(edge.Source())
	}
}

// UpEdges returns the vertices that are *sources* of edges that target the
// destination Vertex v.
func (g *Graph) UpEdges(v Vertex) Set {
	return g.upEdgesNoCopy(v).Copy()
}

// DownEdges returns the vertices that are *targets* of edges that originate
// from the source Vertex v.
func (g *Graph) DownEdges(v Vertex) Set {
	return g.downEdgesNoCopy(v).Copy()
}

// downEdgesNoCopy returns the vertices targeted by edges from the source Vertex
// v as a Set. This Set is the same as used internally by the Graph to prevent a
// copy, and must not be modified by the caller.
func (g *Graph) downEdgesNoCopy(v Vertex) Set {
	g.init()
	return g.downEdges[hashcode(v)]
}

// upEdgesNoCopy returns the vertices that are sources of edges targeting the
// destination Vertex v as a Set. This Set is the same as used internally by the
// Graph to prevent a copy, and must not be modified by the caller.
func (g *Graph) upEdgesNoCopy(v Vertex) Set {
	g.init()
	return g.upEdges[hashcode(v)]
}

// Connect adds an edge with the given source and target. This is safe to
// call multiple times with the same value. Note that the same value is
// verified through pointer equality of the vertices, not through the
// value of the edge itself.
func (g *Graph) Connect(edge Edge) {
	g.init()

	source := edge.Source()
	target := edge.Target()
	sourceCode := hashcode(source)
	targetCode := hashcode(target)

	// Do we have this already? If so, don't add it again.
	if s, ok := g.downEdges[sourceCode]; ok && s.Include(target) {
		return
	}

	// Add the edge to the set
	g.edges.Add(edge)

	// Add the down edge
	s, ok := g.downEdges[sourceCode]
	if !ok {
		s = make(Set)
		g.downEdges[sourceCode] = s
	}
	s.Add(target)

	// Add the up edge
	s, ok = g.upEdges[targetCode]
	if !ok {
		s = make(Set)
		g.upEdges[targetCode] = s
	}
	s.Add(source)
}

// Subsume imports all of the nodes and edges from the given graph into the
// reciever, leaving the given graph unchanged.
//
// If any of the nodes in the given graph are already present in the reciever
// then the existing node will be retained and any new edges from the given
// graph will be connected with it.
//
// If the given graph has edges in common with the reciever then they will be
// ignored, because each pair of nodes can only be connected once.
func (g *Graph) Subsume(other *Graph) {
	// We're using Set.Filter just as a "visit each element" here, so we're
	// not doing anything with the result (which will always be empty).
	other.vertices.Filter(func(i interface{}) bool {
		g.Add(i)
		return false
	})
	other.edges.Filter(func(i interface{}) bool {
		g.Connect(i.(Edge))
		return false
	})
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
		name := VertexName(v)
		names = append(names, name)
		mapping[name] = v
	}
	sort.Strings(names)

	// Write each node in order...
	for _, name := range names {
		v := mapping[name]
		targets := g.downEdges[hashcode(v)]

		buf.WriteString(fmt.Sprintf("%s - %T\n", name, v))

		// Alphabetize dependencies
		deps := make([]string, 0, targets.Len())
		targetNodes := make(map[string]Vertex)
		for _, target := range targets {
			dep := VertexName(target)
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
		name := VertexName(v)
		names = append(names, name)
		mapping[name] = v
	}
	sort.Strings(names)

	// Write each node in order...
	for _, name := range names {
		v := mapping[name]
		targets := g.downEdges[hashcode(v)]

		buf.WriteString(fmt.Sprintf("%s\n", name))

		// Alphabetize dependencies
		deps := make([]string, 0, targets.Len())
		for _, target := range targets {
			deps = append(deps, VertexName(target))
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
	if g.edges == nil {
		g.edges = make(Set)
	}
	if g.downEdges == nil {
		g.downEdges = make(map[interface{}]Set)
	}
	if g.upEdges == nil {
		g.upEdges = make(map[interface{}]Set)
	}
}

// Dot returns a dot-formatted representation of the Graph.
func (g *Graph) Dot(opts *DotOpts) []byte {
	return newMarshalGraph("", g).Dot(opts)
}

// VertexName returns the name of a vertex.
func VertexName(raw Vertex) string {
	switch v := raw.(type) {
	case NamedVertex:
		return v.Name()
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
