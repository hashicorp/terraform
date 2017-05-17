package dag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// Graph is used to represent a dependency graph.
type Graph struct {
	vertices  *Set
	edges     *Set
	downEdges map[interface{}]*Set
	upEdges   map[interface{}]*Set

	// JSON encoder for recording debug information
	debug *encoder
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
	list := g.vertices.List()
	result := make([]Vertex, len(list))
	for i, v := range list {
		result[i] = v.(Vertex)
	}

	return result
}

// Edges returns the list of all the edges in the graph.
func (g *Graph) Edges() []Edge {
	list := g.edges.List()
	result := make([]Edge, len(list))
	for i, v := range list {
		result[i] = v.(Edge)
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
	g.debug.Add(v)
	return v
}

// Remove removes a vertex from the graph. This will also remove any
// edges with this vertex as a source or target.
func (g *Graph) Remove(v Vertex) Vertex {
	// Delete the vertex itself
	g.vertices.Delete(v)
	g.debug.Remove(v)

	// Delete the edges to non-existent things
	for _, target := range g.DownEdges(v).List() {
		g.RemoveEdge(BasicEdge(v, target))
	}
	for _, source := range g.UpEdges(v).List() {
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

	defer g.debug.BeginOperation("Replace", "").End("")

	// If they're the same, then don't do anything
	if original == replacement {
		return true
	}

	// Add our new vertex, then copy all the edges
	g.Add(replacement)
	for _, target := range g.DownEdges(original).List() {
		g.Connect(BasicEdge(replacement, target))
	}
	for _, source := range g.UpEdges(original).List() {
		g.Connect(BasicEdge(source, replacement))
	}

	// Remove our old vertex, which will also remove all the edges
	g.Remove(original)

	return true
}

// RemoveEdge removes an edge from the graph.
func (g *Graph) RemoveEdge(edge Edge) {
	g.init()
	g.debug.RemoveEdge(edge)

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

// DownEdges returns the outward edges from the source Vertex v.
func (g *Graph) DownEdges(v Vertex) *Set {
	g.init()
	return g.downEdges[hashcode(v)]
}

// UpEdges returns the inward edges to the destination Vertex v.
func (g *Graph) UpEdges(v Vertex) *Set {
	g.init()
	return g.upEdges[hashcode(v)]
}

// Connect adds an edge with the given source and target. This is safe to
// call multiple times with the same value. Note that the same value is
// verified through pointer equality of the vertices, not through the
// value of the edge itself.
func (g *Graph) Connect(edge Edge) {
	g.init()
	g.debug.Connect(edge)

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
		s = new(Set)
		g.downEdges[sourceCode] = s
	}
	s.Add(target)

	// Add the up edge
	s, ok = g.upEdges[targetCode]
	if !ok {
		s = new(Set)
		g.upEdges[targetCode] = s
	}
	s.Add(source)
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
		for _, target := range targets.List() {
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
		for _, target := range targets.List() {
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
		g.vertices = new(Set)
	}
	if g.edges == nil {
		g.edges = new(Set)
	}
	if g.downEdges == nil {
		g.downEdges = make(map[interface{}]*Set)
	}
	if g.upEdges == nil {
		g.upEdges = make(map[interface{}]*Set)
	}
}

// Dot returns a dot-formatted representation of the Graph.
func (g *Graph) Dot(opts *DotOpts) []byte {
	return newMarshalGraph("", g).Dot(opts)
}

// MarshalJSON returns a JSON representation of the entire Graph.
func (g *Graph) MarshalJSON() ([]byte, error) {
	dg := newMarshalGraph("root", g)
	return json.MarshalIndent(dg, "", "  ")
}

// SetDebugWriter sets the io.Writer where the Graph will record debug
// information. After this is set, the graph will immediately encode itself to
// the stream, and continue to record all subsequent operations.
func (g *Graph) SetDebugWriter(w io.Writer) {
	g.debug = &encoder{w: w}
	g.debug.Encode(newMarshalGraph("root", g))
}

// DebugVertexInfo encodes arbitrary information about a vertex in the graph
// debug logs.
func (g *Graph) DebugVertexInfo(v Vertex, info string) {
	va := newVertexInfo(typeVertexInfo, v, info)
	g.debug.Encode(va)
}

// DebugEdgeInfo encodes arbitrary information about an edge in the graph debug
// logs.
func (g *Graph) DebugEdgeInfo(e Edge, info string) {
	ea := newEdgeInfo(typeEdgeInfo, e, info)
	g.debug.Encode(ea)
}

// DebugVisitInfo records a visit to a Vertex during a walk operation.
func (g *Graph) DebugVisitInfo(v Vertex, info string) {
	vi := newVertexInfo(typeVisitInfo, v, info)
	g.debug.Encode(vi)
}

// DebugOperation marks the start of a set of graph transformations in
// the debug log, and returns a DebugOperationEnd func, which marks the end of
// the operation in the log. Additional information can be added to the log via
// the info parameter.
//
// The returned func's End method allows this method to be called from a single
// defer statement:
//     defer g.DebugOperationBegin("OpName", "operating").End("")
//
// The returned function must be called to properly close the logical operation
// in the logs.
func (g *Graph) DebugOperation(operation string, info string) DebugOperationEnd {
	return g.debug.BeginOperation(operation, info)
}

// VertexName returns the name of a vertex.
func VertexName(raw Vertex) string {
	switch v := raw.(type) {
	case NamedVertex:
		return v.Name()
	case fmt.Stringer:
		return fmt.Sprintf("%s", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
