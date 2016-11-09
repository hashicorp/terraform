package dag

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

// the marshal* structs are for serialization of the graph data.
type marshalGraph struct {
	// Each marshal structure require a unique ID so that it can be references
	// by other structures.
	ID string `json:",omitempty"`

	// Human readable name for this graph.
	Name string `json:",omitempty"`

	// Arbitrary attributes that can be added to the output.
	Attrs map[string]string `json:",omitempty"`

	// List of graph vertices, sorted by ID.
	Vertices []*marshalVertex `json:",omitempty"`

	// List of edges, sorted by Source ID.
	Edges []*marshalEdge `json:",omitempty"`

	// Any number of subgraphs. A subgraph itself is considered a vertex, and
	// may be referenced by either end of an edge.
	Subgraphs []*marshalGraph `json:",omitempty"`

	// Any lists of vertices that are included in cycles.
	Cycles [][]*marshalVertex `json:",omitempty"`
}

func (g *marshalGraph) vertexByID(id string) *marshalVertex {
	for _, v := range g.Vertices {
		if id == v.ID {
			return v
		}
	}
	return nil
}

type marshalVertex struct {
	// Unique ID, used to reference this vertex from other structures.
	ID string

	// Human readable name
	Name string `json:",omitempty"`

	Attrs map[string]string `json:",omitempty"`

	// This is to help transition from the old Dot interfaces. We record if the
	// node was a GraphNodeDotter here, so know if it should be included in the
	// dot output
	graphNodeDotter bool
}

// vertices is a sort.Interface implementation for sorting vertices by ID
type vertices []*marshalVertex

func (v vertices) Less(i, j int) bool { return v[i].Name < v[j].Name }
func (v vertices) Len() int           { return len(v) }
func (v vertices) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }

type marshalEdge struct {
	// Human readable name
	Name string

	// Source and Target Vertices by ID
	Source string
	Target string

	Attrs map[string]string `json:",omitempty"`
}

// edges is a sort.Interface implementation for sorting edges by Source ID
type edges []*marshalEdge

func (e edges) Less(i, j int) bool { return e[i].Name < e[j].Name }
func (e edges) Len() int           { return len(e) }
func (e edges) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

// build a marshalGraph structure from a *Graph
func newMarshalGraph(name string, g *Graph) *marshalGraph {
	dg := &marshalGraph{
		Name:  name,
		Attrs: make(map[string]string),
	}

	for _, v := range g.Vertices() {
		// We only care about nodes that yield non-empty Dot strings.
		dn, isDotter := v.(GraphNodeDotter)
		dotOpts := &DotOpts{
			Verbose:    true,
			DrawCycles: true,
		}
		if isDotter && dn.DotNode("fake", dotOpts) == nil {
			isDotter = false
		}

		id := marshalVertexID(v)
		if sg, ok := marshalSubgrapher(v); ok {

			sdg := newMarshalGraph(VertexName(v), sg)
			sdg.ID = id
			dg.Subgraphs = append(dg.Subgraphs, sdg)
		}

		dv := &marshalVertex{
			ID:              id,
			Name:            VertexName(v),
			Attrs:           make(map[string]string),
			graphNodeDotter: isDotter,
		}

		dg.Vertices = append(dg.Vertices, dv)
	}

	sort.Sort(vertices(dg.Vertices))

	for _, e := range g.Edges() {
		de := &marshalEdge{
			Name:   fmt.Sprintf("%s|%s", VertexName(e.Source()), VertexName(e.Target())),
			Source: marshalVertexID(e.Source()),
			Target: marshalVertexID(e.Target()),
			Attrs:  make(map[string]string),
		}
		dg.Edges = append(dg.Edges, de)
	}

	sort.Sort(edges(dg.Edges))

	for _, c := range (&AcyclicGraph{*g}).Cycles() {
		var cycle []*marshalVertex
		for _, v := range c {
			dv := &marshalVertex{
				ID:    marshalVertexID(v),
				Name:  VertexName(v),
				Attrs: make(map[string]string),
			}

			cycle = append(cycle, dv)
		}
		dg.Cycles = append(dg.Cycles, cycle)
	}

	return dg
}

// Attempt to return a unique ID for any vertex.
func marshalVertexID(v Vertex) string {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return strconv.Itoa(int(val.Pointer()))
	case reflect.Interface:
		return strconv.Itoa(int(val.InterfaceData()[1]))
	}

	if v, ok := v.(Hashable); ok {
		h := v.Hashcode()
		if h, ok := h.(string); ok {
			return h
		}
	}

	// fallback to a name, which we hope is unique.
	return VertexName(v)

	// we could try harder by attempting to read the arbitrary value from the
	// interface, but we shouldn't get here from terraform right now.
}

// check for a Subgrapher, and return the underlying *Graph.
func marshalSubgrapher(v Vertex) (*Graph, bool) {
	sg, ok := v.(Subgrapher)
	if !ok {
		return nil, false
	}

	switch g := sg.Subgraph().DirectedGraph().(type) {
	case *Graph:
		return g, true
	case *AcyclicGraph:
		return &g.Graph, true
	}

	return nil, false
}
