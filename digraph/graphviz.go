package digraph

import (
	"fmt"
	"io"
)

// GenerateDot is used to emit a GraphViz compatible definition
// for a directed graph. It can be used to dump a .dot file.
func GenerateDot(nodes []Node, w io.Writer) {
	w.Write([]byte("digraph {\n"))
	defer w.Write([]byte("}\n"))
	for _, n := range nodes {
		w.Write([]byte(fmt.Sprintf("\t%s;\n", n)))
		for _, edge := range n.Edges() {
			target := edge.Tail()
			line := fmt.Sprintf("\t%s -> %s [label=\"%s\"];\n",
				n, target, edge)
			w.Write([]byte(line))
		}
	}
}
