// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dag

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// DotOpts are the options for generating a dot formatted Graph.
type DotOpts struct {
	// Allows some nodes to decide to only show themselves when the user has
	// requested the "verbose" graph.
	Verbose bool

	// Highlight Cycles
	DrawCycles bool

	// How many levels to expand modules as we draw
	MaxDepth int

	// use this to keep the cluster_ naming convention from the previous dot writer
	cluster bool
}

// GraphNodeDotter can be implemented by a node to cause it to be included
// in the dot graph. The Dot method will be called which is expected to
// return a representation of this node.
type GraphNodeDotter interface {
	// Dot is called to return the dot formatting for the node.
	// The first parameter is the title of the node.
	// The second parameter includes user-specified options that affect the dot
	// graph. See GraphDotOpts below for details.
	DotNode(string, *DotOpts) *DotNode
}

// DotNode provides a structure for Vertices to return in order to specify their
// dot format.
type DotNode struct {
	Name  string
	Attrs map[string]string
}

// Returns the DOT representation of this Graph.
func (g *marshalGraph) Dot(opts *DotOpts) []byte {
	if opts == nil {
		opts = &DotOpts{
			DrawCycles: true,
			MaxDepth:   -1,
			Verbose:    true,
		}
	}

	var w indentWriter
	w.WriteString("digraph {\n")
	w.Indent()

	// some dot defaults
	w.WriteString(`compound = "true"` + "\n")
	w.WriteString(`newrank = "true"` + "\n")

	// the top level graph is written as the first subgraph
	w.WriteString(`subgraph "root" {` + "\n")
	g.writeBody(opts, &w)

	// cluster isn't really used other than for naming purposes in some graphs
	opts.cluster = opts.MaxDepth != 0
	maxDepth := opts.MaxDepth
	if maxDepth == 0 {
		maxDepth = -1
	}

	for _, s := range g.Subgraphs {
		g.writeSubgraph(s, opts, maxDepth, &w)
	}

	w.Unindent()
	w.WriteString("}\n")
	return w.Bytes()
}

func (v *marshalVertex) dot(g *marshalGraph, opts *DotOpts) []byte {
	var buf bytes.Buffer
	graphName := g.Name
	if graphName == "" {
		graphName = "root"
	}

	name := v.Name
	attrs := v.Attrs
	if v.graphNodeDotter != nil {
		node := v.graphNodeDotter.DotNode(name, opts)
		if node == nil {
			return []byte{}
		}

		newAttrs := make(map[string]string)
		for k, v := range attrs {
			newAttrs[k] = v
		}
		for k, v := range node.Attrs {
			newAttrs[k] = v
		}

		name = node.Name
		attrs = newAttrs
	}

	buf.WriteString(fmt.Sprintf(`"[%s] %s"`, graphName, name))
	writeAttrs(&buf, attrs)
	buf.WriteByte('\n')

	return buf.Bytes()
}

func (e *marshalEdge) dot(g *marshalGraph) string {
	var buf bytes.Buffer
	graphName := g.Name
	if graphName == "" {
		graphName = "root"
	}

	sourceName := g.vertexByID(e.Source).Name
	targetName := g.vertexByID(e.Target).Name
	s := fmt.Sprintf(`"[%s] %s" -> "[%s] %s"`, graphName, sourceName, graphName, targetName)
	buf.WriteString(s)
	writeAttrs(&buf, e.Attrs)

	return buf.String()
}

func cycleDot(e *marshalEdge, g *marshalGraph) string {
	return e.dot(g) + ` [color = "red", penwidth = "2.0"]`
}

// Write the subgraph body. The is recursive, and the depth argument is used to
// record the current depth of iteration.
func (g *marshalGraph) writeSubgraph(sg *marshalGraph, opts *DotOpts, depth int, w *indentWriter) {
	if depth == 0 {
		return
	}
	depth--

	name := sg.Name
	if opts.cluster {
		// we prefix with cluster_ to match the old dot output
		name = "cluster_" + name
		sg.Attrs["label"] = sg.Name
	}
	w.WriteString(fmt.Sprintf("subgraph %q {\n", name))
	sg.writeBody(opts, w)

	for _, sg := range sg.Subgraphs {
		g.writeSubgraph(sg, opts, depth, w)
	}
}

func (g *marshalGraph) writeBody(opts *DotOpts, w *indentWriter) {
	w.Indent()

	for _, as := range attrStrings(g.Attrs) {
		w.WriteString(as + "\n")
	}

	// list of Vertices that aren't to be included in the dot output
	skip := map[string]bool{}

	for _, v := range g.Vertices {
		if v.graphNodeDotter == nil {
			skip[v.ID] = true
			continue
		}

		w.Write(v.dot(g, opts))
	}

	var dotEdges []string

	if opts.DrawCycles {
		for _, c := range g.Cycles {
			if len(c) < 2 {
				continue
			}

			for i, j := 0, 1; i < len(c); i, j = i+1, j+1 {
				if j >= len(c) {
					j = 0
				}
				src := c[i]
				tgt := c[j]

				if skip[src.ID] || skip[tgt.ID] {
					continue
				}

				e := &marshalEdge{
					Name:   fmt.Sprintf("%s|%s", src.Name, tgt.Name),
					Source: src.ID,
					Target: tgt.ID,
					Attrs:  make(map[string]string),
				}

				dotEdges = append(dotEdges, cycleDot(e, g))
				src = tgt
			}
		}
	}

	for _, e := range g.Edges {
		dotEdges = append(dotEdges, e.dot(g))
	}

	// srot these again to match the old output
	sort.Strings(dotEdges)

	for _, e := range dotEdges {
		w.WriteString(e + "\n")
	}

	w.Unindent()
	w.WriteString("}\n")
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

// Provide a bytes.Buffer like structure, which will indent when starting a
// newline.
type indentWriter struct {
	bytes.Buffer
	level int
}

func (w *indentWriter) indent() {
	newline := []byte("\n")
	if !bytes.HasSuffix(w.Bytes(), newline) {
		return
	}
	for i := 0; i < w.level; i++ {
		w.Buffer.WriteString("\t")
	}
}

// Indent increases indentation by 1
func (w *indentWriter) Indent() { w.level++ }

// Unindent decreases indentation by 1
func (w *indentWriter) Unindent() { w.level-- }

// the following methods intercecpt the byte.Buffer writes and insert the
// indentation when starting a new line.
func (w *indentWriter) Write(b []byte) (int, error) {
	w.indent()
	return w.Buffer.Write(b)
}

func (w *indentWriter) WriteString(s string) (int, error) {
	w.indent()
	return w.Buffer.WriteString(s)
}
func (w *indentWriter) WriteByte(b byte) error {
	w.indent()
	return w.Buffer.WriteByte(b)
}
func (w *indentWriter) WriteRune(r rune) (int, error) {
	w.indent()
	return w.Buffer.WriteRune(r)
}
