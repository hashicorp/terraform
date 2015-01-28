package dag

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
)

// Graph is used to represent a dependency graph.
type Graph struct {
	vertices  []Vertex
	edges     []Edge
	downEdges map[Vertex]*set
	upEdges   map[Vertex]*set
	once      sync.Once
}

// Vertex of the graph.
type Vertex interface{}

// NamedVertex is an optional interface that can be implemented by Vertex
// to give it a human-friendly name that is used for outputting the graph.
type NamedVertex interface {
	Vertex
	Name() string
}

// Vertices returns the list of all the vertices in the graph.
func (g *Graph) Vertices() []Vertex {
	return g.vertices
}

// Edges returns the list of all the edges in the graph.
func (g *Graph) Edges() []Edge {
	return g.edges
}

// Add adds a vertex to the graph. This is safe to call multiple time with
// the same Vertex.
func (g *Graph) Add(v Vertex) Vertex {
	g.once.Do(g.init)
	g.vertices = append(g.vertices, v)
	return v
}

// Connect adds an edge with the given source and target. This is safe to
// call multiple times with the same value. Note that the same value is
// verified through pointer equality of the vertices, not through the
// value of the edge itself.
func (g *Graph) Connect(edge Edge) {
	g.once.Do(g.init)

	source := edge.Source()
	target := edge.Target()

	// Do we have this already? If so, don't add it again.
	if s, ok := g.downEdges[source]; ok && s.Include(target) {
		return
	}

	// TODO: add all edges
	g.edges = append(g.edges, edge)

	// Add the down edge
	s, ok := g.downEdges[source]
	if !ok {
		s = new(set)
		g.downEdges[source] = s
	}
	s.Add(target)

	// Add the up edge
	s, ok = g.upEdges[target]
	if !ok {
		s = new(set)
		g.upEdges[target] = s
	}
	s.Add(source)
}

// String outputs some human-friendly output for the graph structure.
func (g *Graph) String() string {
	var buf bytes.Buffer

	// Build the list of node names and a mapping so that we can more
	// easily alphabetize the output to remain deterministic.
	names := make([]string, 0, len(g.vertices))
	mapping := make(map[string]Vertex, len(g.vertices))
	for _, v := range g.vertices {
		name := VertexName(v)
		names = append(names, name)
		mapping[name] = v
	}
	sort.Strings(names)

	// Write each node in order...
	for _, name := range names {
		v := mapping[name]
		targets := g.downEdges[v]

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
	g.vertices = make([]Vertex, 0, 5)
	g.edges = make([]Edge, 0, 2)
	g.downEdges = make(map[Vertex]*set)
	g.upEdges = make(map[Vertex]*set)
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
