package dag

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

// the marshal* structs are for serialization of the graph data.
type marshalGraph struct {
	ID        string             `json:",omitempty"`
	Name      string             `json:",omitempty"`
	Attrs     map[string]string  `json:",omitempty"`
	Vertices  []*marshalVertex   `json:",omitempty"`
	Edges     []*marshalEdge     `json:",omitempty"`
	Subgraphs []*marshalGraph    `json:",omitempty"`
	Cycles    [][]*marshalVertex `json:",omitempty"`
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
	ID    string
	Name  string            `json:",omitempty"`
	Attrs map[string]string `json:",omitempty"`
}

type vertices []*marshalVertex

func (v vertices) Less(i, j int) bool { return v[i].Name < v[j].Name }
func (v vertices) Len() int           { return len(v) }
func (v vertices) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }

type marshalEdge struct {
	Name   string
	Source string
	Target string
	Attrs  map[string]string `json:",omitempty"`
}

type edges []*marshalEdge

func (e edges) Less(i, j int) bool { return e[i].Name < e[j].Name }
func (e edges) Len() int           { return len(e) }
func (e edges) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

func newMarshalGraph(name string, g *Graph) *marshalGraph {
	dg := &marshalGraph{
		Name:  name,
		Attrs: make(map[string]string),
	}

	for _, v := range g.Vertices() {
		id := marshalVertexID(v)
		if sg, ok := marshalSubgrapher(v); ok {

			sdg := newMarshalGraph(VertexName(v), sg)
			sdg.ID = id
			dg.Subgraphs = append(dg.Subgraphs, sdg)
		}

		dv := &marshalVertex{
			ID:    id,
			Name:  VertexName(v),
			Attrs: make(map[string]string),
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

	// we could try harder by attempting to read the arbitrary value from the
	// interface, but we shouldn't get here from terraform right now.
	panic("unhashable value in graph")
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
