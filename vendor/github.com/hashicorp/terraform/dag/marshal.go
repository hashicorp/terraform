package dag

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"sort"
	"strconv"
	"sync"
)

const (
	typeOperation             = "Operation"
	typeTransform             = "Transform"
	typeWalk                  = "Walk"
	typeDepthFirstWalk        = "DepthFirstWalk"
	typeReverseDepthFirstWalk = "ReverseDepthFirstWalk"
	typeTransitiveReduction   = "TransitiveReduction"
	typeEdgeInfo              = "EdgeInfo"
	typeVertexInfo            = "VertexInfo"
	typeVisitInfo             = "VisitInfo"
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

// The add, remove, connect, removeEdge methods mirror the basic Graph
// manipulations to reconstruct a marshalGraph from a debug log.
func (g *marshalGraph) add(v *marshalVertex) {
	g.Vertices = append(g.Vertices, v)
	sort.Sort(vertices(g.Vertices))
}

func (g *marshalGraph) remove(v *marshalVertex) {
	for i, existing := range g.Vertices {
		if v.ID == existing.ID {
			g.Vertices = append(g.Vertices[:i], g.Vertices[i+1:]...)
			return
		}
	}
}

func (g *marshalGraph) connect(e *marshalEdge) {
	g.Edges = append(g.Edges, e)
	sort.Sort(edges(g.Edges))
}

func (g *marshalGraph) removeEdge(e *marshalEdge) {
	for i, existing := range g.Edges {
		if e.Source == existing.Source && e.Target == existing.Target {
			g.Edges = append(g.Edges[:i], g.Edges[i+1:]...)
			return
		}
	}
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

	return &marshalVertex{
		ID:              marshalVertexID(v),
		Name:            VertexName(v),
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

// The DebugOperationEnd func type provides a way to call an End function via a
// method call, allowing for the chaining of methods in a defer statement.
type DebugOperationEnd func(string)

// End calls function e with the info parameter, marking the end of this
// operation in the logs.
func (e DebugOperationEnd) End(info string) { e(info) }

// encoder provides methods to write debug data to an io.Writer, and is a noop
// when no writer is present
type encoder struct {
	sync.Mutex
	w io.Writer
}

// Encode is analogous to json.Encoder.Encode
func (e *encoder) Encode(i interface{}) {
	if e == nil || e.w == nil {
		return
	}
	e.Lock()
	defer e.Unlock()

	js, err := json.Marshal(i)
	if err != nil {
		log.Println("[ERROR] dag:", err)
		return
	}
	js = append(js, '\n')

	_, err = e.w.Write(js)
	if err != nil {
		log.Println("[ERROR] dag:", err)
		return
	}
}

func (e *encoder) Add(v Vertex) {
	e.Encode(marshalTransform{
		Type:      typeTransform,
		AddVertex: newMarshalVertex(v),
	})
}

// Remove records the removal of Vertex v.
func (e *encoder) Remove(v Vertex) {
	e.Encode(marshalTransform{
		Type:         typeTransform,
		RemoveVertex: newMarshalVertex(v),
	})
}

func (e *encoder) Connect(edge Edge) {
	e.Encode(marshalTransform{
		Type:    typeTransform,
		AddEdge: newMarshalEdge(edge),
	})
}

func (e *encoder) RemoveEdge(edge Edge) {
	e.Encode(marshalTransform{
		Type:       typeTransform,
		RemoveEdge: newMarshalEdge(edge),
	})
}

// BeginOperation marks the start of set of graph transformations, and returns
// an EndDebugOperation func to be called once the opration is complete.
func (e *encoder) BeginOperation(op string, info string) DebugOperationEnd {
	if e == nil {
		return func(string) {}
	}

	e.Encode(marshalOperation{
		Type:  typeOperation,
		Begin: op,
		Info:  info,
	})

	return func(info string) {
		e.Encode(marshalOperation{
			Type: typeOperation,
			End:  op,
			Info: info,
		})
	}
}

// structure for recording graph transformations
type marshalTransform struct {
	// Type: "Transform"
	Type         string
	AddEdge      *marshalEdge   `json:",omitempty"`
	RemoveEdge   *marshalEdge   `json:",omitempty"`
	AddVertex    *marshalVertex `json:",omitempty"`
	RemoveVertex *marshalVertex `json:",omitempty"`
}

func (t marshalTransform) Transform(g *marshalGraph) {
	switch {
	case t.AddEdge != nil:
		g.connect(t.AddEdge)
	case t.RemoveEdge != nil:
		g.removeEdge(t.RemoveEdge)
	case t.AddVertex != nil:
		g.add(t.AddVertex)
	case t.RemoveVertex != nil:
		g.remove(t.RemoveVertex)
	}
}

// this structure allows us to decode any object in the json stream for
// inspection, then re-decode it into a proper struct if needed.
type streamDecode struct {
	Type string
	Map  map[string]interface{}
	JSON []byte
}

func (s *streamDecode) UnmarshalJSON(d []byte) error {
	s.JSON = d
	err := json.Unmarshal(d, &s.Map)
	if err != nil {
		return err
	}

	if t, ok := s.Map["Type"]; ok {
		s.Type, _ = t.(string)
	}
	return nil
}

// structure for recording the beginning and end of any multi-step
// transformations. These are informational, and not required to reproduce the
// graph state.
type marshalOperation struct {
	Type  string
	Begin string `json:",omitempty"`
	End   string `json:",omitempty"`
	Info  string `json:",omitempty"`
}

// decodeGraph decodes a marshalGraph from an encoded graph stream.
func decodeGraph(r io.Reader) (*marshalGraph, error) {
	dec := json.NewDecoder(r)

	// a stream should always start with a graph
	g := &marshalGraph{}

	err := dec.Decode(g)
	if err != nil {
		return nil, err
	}

	// now replay any operations that occurred on the original graph
	for dec.More() {
		s := &streamDecode{}
		err := dec.Decode(s)
		if err != nil {
			return g, err
		}

		// the only Type we're concerned with here is Transform to complete the
		// Graph
		if s.Type != typeTransform {
			continue
		}

		t := &marshalTransform{}
		err = json.Unmarshal(s.JSON, t)
		if err != nil {
			return g, err
		}
		t.Transform(g)
	}
	return g, nil
}

// marshalVertexInfo allows encoding arbitrary information about the a single
// Vertex in the logs. These are accumulated for informational display while
// rebuilding the graph.
type marshalVertexInfo struct {
	Type   string
	Vertex *marshalVertex
	Info   string
}

func newVertexInfo(infoType string, v Vertex, info string) *marshalVertexInfo {
	return &marshalVertexInfo{
		Type:   infoType,
		Vertex: newMarshalVertex(v),
		Info:   info,
	}
}

// marshalEdgeInfo allows encoding arbitrary information about the a single
// Edge in the logs. These are accumulated for informational display while
// rebuilding the graph.
type marshalEdgeInfo struct {
	Type string
	Edge *marshalEdge
	Info string
}

func newEdgeInfo(infoType string, e Edge, info string) *marshalEdgeInfo {
	return &marshalEdgeInfo{
		Type: infoType,
		Edge: newMarshalEdge(e),
		Info: info,
	}
}

// JSON2Dot reads a Graph debug log from and io.Reader, and converts the final
// graph dot format.
//
// TODO: Allow returning the output at a certain point during decode.
//       Encode extra information from the json log into the Dot.
func JSON2Dot(r io.Reader) ([]byte, error) {
	g, err := decodeGraph(r)
	if err != nil {
		return nil, err
	}

	return g.Dot(nil), nil
}
