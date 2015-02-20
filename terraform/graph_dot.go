package terraform

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/dag"
)

// GraphNodeDotter can be implemented by a node to cause it to be included
// in the dot graph. The Dot method will be called which is expected to
// return a representation of this node.
type GraphNodeDotter interface {
	// Dot is called to return the dot formatting for the node.
	// The parameter must be the title of the node.
	Dot(string) string
}

// GraphDotOpts are the options for generating a dot formatted Graph.
type GraphDotOpts struct{}

// GraphDot returns the dot formatting of a visual representation of
// the given Terraform graph.
func GraphDot(g *Graph, opts *GraphDotOpts) string {
	buf := new(bytes.Buffer)

	// Start the graph
	buf.WriteString("digraph {\n")
	buf.WriteString("\tcompound = true;\n")

	// Go through all the vertices and draw it
	vertices := g.Vertices()
	dotVertices := make(map[dag.Vertex]struct{}, len(vertices))
	for _, v := range vertices {
		if dn, ok := v.(GraphNodeDotter); !ok {
			continue
		} else if dn.Dot("fake") == "" {
			continue
		}

		dotVertices[v] = struct{}{}
	}

	for v, _ := range dotVertices {
		dn := v.(GraphNodeDotter)
		scanner := bufio.NewScanner(strings.NewReader(
			dn.Dot(dag.VertexName(v))))
		for scanner.Scan() {
			buf.WriteString("\t" + scanner.Text() + "\n")
		}

		// Draw all the edges
		for _, t := range g.DownEdges(v).List() {
			target := t.(dag.Vertex)
			if _, ok := dotVertices[target]; !ok {
				continue
			}

			buf.WriteString(fmt.Sprintf(
				"\t\"%s\" -> \"%s\";\n",
				dag.VertexName(v),
				dag.VertexName(target)))
		}
	}

	// End the graph
	buf.WriteString("}\n")
	return buf.String()
}
