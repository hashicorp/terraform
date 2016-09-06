// The dot package contains utilities for working with DOT graphs.
package dot

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// Graph is a representation of a drawable DOT graph.
type Graph struct {
	// Whether this is a "digraph" or just a "graph"
	Directed bool

	// Used for K/V settings in the DOT
	Attrs map[string]string

	Nodes     []*Node
	Edges     []*Edge
	Subgraphs []*Subgraph

	nodesByName map[string]*Node
}

// Subgraph is a Graph that lives inside a Parent graph, and contains some
// additional parameters to control how it is drawn.
type Subgraph struct {
	Graph
	Name    string
	Parent  *Graph
	Cluster bool
}

// An Edge in a DOT graph, as expressed by recording the Name of the Node at
// each end.
type Edge struct {
	// Name of source node.
	Source string

	// Name of dest node.
	Dest string

	// List of K/V attributes for this edge.
	Attrs map[string]string
}

// A Node in a DOT graph.
type Node struct {
	Name  string
	Attrs map[string]string
}

// Creates a properly initialized DOT Graph.
func NewGraph(attrs map[string]string) *Graph {
	return &Graph{
		Attrs:       attrs,
		nodesByName: make(map[string]*Node),
	}
}

func NewEdge(src, dst string, attrs map[string]string) *Edge {
	return &Edge{
		Source: src,
		Dest:   dst,
		Attrs:  attrs,
	}
}

func NewNode(n string, attrs map[string]string) *Node {
	return &Node{
		Name:  n,
		Attrs: attrs,
	}
}

// Initializes a Subgraph with the provided name, attaches is to this Graph,
// and returns it.
func (g *Graph) AddSubgraph(name string) *Subgraph {
	subgraph := &Subgraph{
		Graph:  *NewGraph(map[string]string{}),
		Parent: g,
		Name:   name,
	}
	g.Subgraphs = append(g.Subgraphs, subgraph)
	return subgraph
}

func (g *Graph) AddAttr(k, v string) {
	g.Attrs[k] = v
}

func (g *Graph) AddNode(n *Node) {
	g.Nodes = append(g.Nodes, n)
	g.nodesByName[n.Name] = n
}

func (g *Graph) AddEdge(e *Edge) {
	g.Edges = append(g.Edges, e)
}

// Adds an edge between two Nodes.
//
// Note this does not do any verification of the existence of these nodes,
// which means that any strings you provide that are not existing nodes will
// result in extra auto-defined nodes in your resulting DOT.
func (g *Graph) AddEdgeBetween(src, dst string, attrs map[string]string) error {
	g.AddEdge(NewEdge(src, dst, attrs))

	return nil
}

// Look up a node by name
func (g *Graph) GetNode(name string) (*Node, error) {
	node, ok := g.nodesByName[name]
	if !ok {
		return nil, fmt.Errorf("Could not find node: %s", name)
	}
	return node, nil
}

// Returns the DOT representation of this Graph.
func (g *Graph) String() string {
	w := newGraphWriter()

	g.drawHeader(w)
	w.Indent()
	g.drawBody(w)
	w.Unindent()
	g.drawFooter(w)

	return w.String()
}

func (g *Graph) drawHeader(w *graphWriter) {
	if g.Directed {
		w.Printf("digraph {\n")
	} else {
		w.Printf("graph {\n")
	}
}

func (g *Graph) drawBody(w *graphWriter) {
	for _, as := range attrStrings(g.Attrs) {
		w.Printf("%s\n", as)
	}

	nodeStrings := make([]string, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeStrings = append(nodeStrings, n.String())
	}
	sort.Strings(nodeStrings)
	for _, ns := range nodeStrings {
		w.Printf(ns)
	}

	edgeStrings := make([]string, 0, len(g.Edges))
	for _, e := range g.Edges {
		edgeStrings = append(edgeStrings, e.String())
	}
	sort.Strings(edgeStrings)
	for _, es := range edgeStrings {
		w.Printf(es)
	}

	for _, s := range g.Subgraphs {
		s.drawHeader(w)
		w.Indent()
		s.drawBody(w)
		w.Unindent()
		s.drawFooter(w)
	}
}

func (g *Graph) drawFooter(w *graphWriter) {
	w.Printf("}\n")
}

// Returns the DOT representation of this Edge.
func (e *Edge) String() string {
	var buf bytes.Buffer
	buf.WriteString(
		fmt.Sprintf(
			"%q -> %q", e.Source, e.Dest))
	writeAttrs(&buf, e.Attrs)
	buf.WriteString("\n")

	return buf.String()
}

func (s *Subgraph) drawHeader(w *graphWriter) {
	name := s.Name
	if s.Cluster {
		name = fmt.Sprintf("cluster_%s", name)
	}
	w.Printf("subgraph %q {\n", name)
}

// Returns the DOT representation of this Node.
func (n *Node) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%q", n.Name))
	writeAttrs(&buf, n.Attrs)
	buf.WriteString("\n")

	return buf.String()
}

func writeAttrs(buf *bytes.Buffer, attrs map[string]string) {
	if len(attrs) > 0 {
		buf.WriteString(" [")
		buf.WriteString(strings.Join(attrStrings(attrs), ", "))
		buf.WriteString("]")
	}
}

func attrStrings(attrs map[string]string) []string {
	strings := make([]string, 0, len(attrs))
	for k, v := range attrs {
		strings = append(strings, fmt.Sprintf("%s = %q", k, v))
	}
	sort.Strings(strings)
	return strings
}
