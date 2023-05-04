// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dag

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

// the marshal* structs are for serialization of the graph data.
type marshalGraph struct {
	// Type is always "Graph", for identification as a top level object in the
	// JSON stream.
	Type string

	// Each marshal structure requires a unique ID so that it can be referenced
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
	// node was a GraphNodeDotter here, so we can call it to get attributes.
	graphNodeDotter GraphNodeDotter
}

func newMarshalVertex(v Vertex) *marshalVertex {
	dn, ok := v.(GraphNodeDotter)
	if !ok {
		dn = nil
	}

	// the name will be quoted again later, so we need to ensure it's properly
	// escaped without quotes.
	name := strconv.Quote(VertexName(v))
	name = name[1 : len(name)-1]

	return &marshalVertex{
		ID:              marshalVertexID(v),
		Name:            name,
		Attrs:           make(map[string]string),
		graphNodeDotter: dn,
	}
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

func newMarshalEdge(e Edge) *marshalEdge {
	return &marshalEdge{
		Name:   fmt.Sprintf("%s|%s", VertexName(e.Source()), VertexName(e.Target())),
		Source: marshalVertexID(e.Source()),
		Target: marshalVertexID(e.Target()),
		Attrs:  make(map[string]string),
	}
}

// edges is a sort.Interface implementation for sorting edges by Source ID
type edges []*marshalEdge

func (e edges) Less(i, j int) bool { return e[i].Name < e[j].Name }
func (e edges) Len() int           { return len(e) }
func (e edges) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

// build a marshalGraph structure from a *Graph
func newMarshalGraph(name string, g *Graph) *marshalGraph {
	mg := &marshalGraph{
		Type:  "Graph",
		Name:  name,
		Attrs: make(map[string]string),
	}

	for _, v := range g.Vertices() {
		id := marshalVertexID(v)
		if sg, ok := marshalSubgrapher(v); ok {
			smg := newMarshalGraph(VertexName(v), sg)
			smg.ID = id
			mg.Subgraphs = append(mg.Subgraphs, smg)
		}

		mv := newMarshalVertex(v)
		mg.Vertices = append(mg.Vertices, mv)
	}

	sort.Sort(vertices(mg.Vertices))

	for _, e := range g.Edges() {
		mg.Edges = append(mg.Edges, newMarshalEdge(e))
	}

	sort.Sort(edges(mg.Edges))

	for _, c := range (&AcyclicGraph{*g}).Cycles() {
		var cycle []*marshalVertex
		for _, v := range c {
			mv := newMarshalVertex(v)
			cycle = append(cycle, mv)
		}
		mg.Cycles = append(mg.Cycles, cycle)
	}

	return mg
}

// Attempt to return a unique ID for any vertex.
func marshalVertexID(v Vertex) string {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return strconv.Itoa(int(val.Pointer()))
	case reflect.Interface:
		// A vertex shouldn't contain another layer of interface, but handle
		// this just in case.
		return fmt.Sprintf("%#v", val.Interface())
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
