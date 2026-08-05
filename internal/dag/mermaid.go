// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

func MermaidEscapeLabel(s string) string {
	// Escape brackets and newlines for Mermaid labels
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return fmt.Sprintf("\"%s\"", s)
}

// Mermaid produces a Mermaid flowchart representation of the graph.
// This is a minimal renderer that mirrors the DOT output semantics for
// nodes and edges
func (g *marshalGraph) Mermaid(opts *DotOpts) []byte {
	if opts == nil {
		opts = &DotOpts{
			DrawCycles: true,
			MaxDepth:   -1,
			Verbose:    true,
		}
	}

	var b bytes.Buffer

	// use left-to-right layout by default
	b.WriteString("flowchart LR\n")

	// Write nodes grouped into subgraphs (if present) to mirror DOT's clustering
	// behaviour.
	// Start with the top-level graph
	g.writeNodes(&b, opts)

	// detect cycle edges
	cycleEdges := map[string]bool{}
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
				key := src.ID + "->" + tgt.ID
				cycleEdges[key] = true
			}
		}
	}

	// emit edges in deterministic order
	edgeKeys := make([]string, 0, len(g.Edges))
	edgeMap := make(map[string]*marshalEdge)
	for _, e := range g.Edges {
		key := e.Source + "->" + e.Target
		edgeKeys = append(edgeKeys, key)
		edgeMap[key] = e
	}
	sort.Strings(edgeKeys)

	for _, key := range edgeKeys {
		e := edgeMap[key]
		src := g.vertexByID(e.Source)
		tgt := g.vertexByID(e.Target)
		if src == nil || tgt == nil {
			continue
		}

		b.WriteString(src.ID)
		b.WriteString(" --> ")
		b.WriteString(tgt.ID)
		if cycleEdges[key] {
			// mark with a CSS class we define below
			b.WriteString(":::cycle")
		}
		b.WriteString("\n")
	}

	// define cycle class if needed
	if len(cycleEdges) > 0 {
		b.WriteString("classDef cycle stroke:#ff0000,stroke-width:2px;\n")
	}

	return b.Bytes()
}

func (mg *marshalGraph) writeNodes(b *bytes.Buffer, opts *DotOpts) {
	// if this is a named subgraph, emit a Mermaid subgraph block.
	if strings.TrimSpace(mg.Name) != "" {
		b.WriteString("subgraph ")
		b.WriteString(mg.Name)
		b.WriteString("\n")
	}

	// collect and sort vertices for deterministic output
	ids := make([]string, 0, len(mg.Vertices))
	for _, v := range mg.Vertices {
		ids = append(ids, v.ID)
	}
	sort.Strings(ids)

	for _, id := range ids {
		v := mg.vertexByID(id)
		if v == nil {
			continue
		}

		label := v.Name

		// obtain DOT-style node info if available and merge attributes
		attrs := map[string]string{}
		for k, vv := range v.Attrs {
			attrs[k] = vv
		}

		// label preference: attrs.label then name
		if l, ok := attrs["label"]; ok && strings.TrimSpace(l) != "" {
			label = l
		}

		// choose Mermaid node shape syntax based on DOT 'shape' attribute
		shape := strings.ToLower(attrs["shape"]) // may be empty

		var nodeDef string
		switch shape {
		case "diamond":
			nodeDef = fmt.Sprintf("%s{%s}", v.ID, MermaidEscapeLabel(label))
		case "box", "rectangle", "rect":
			nodeDef = fmt.Sprintf("%s[%s]", v.ID, MermaidEscapeLabel(label))
		case "ellipse", "oval":
			nodeDef = fmt.Sprintf("%s(%s)", v.ID, MermaidEscapeLabel(label))
		case "circle":
			nodeDef = fmt.Sprintf("%s((%s))", v.ID, MermaidEscapeLabel(label))
		default:
			// default to rectangle
			nodeDef = fmt.Sprintf("%s[%s]", v.ID, MermaidEscapeLabel(label))
		}

		b.WriteString(nodeDef)
		b.WriteString("\n")
	}

	// recurse into subgraphs
	for _, sg := range mg.Subgraphs {
		sg.writeNodes(b, opts)
	}

	if strings.TrimSpace(mg.Name) != "" {
		b.WriteString("end\n")
	}
}
