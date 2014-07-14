package digraph

import (
	"fmt"
	"io"
)

// WriteDot is used to emit a GraphViz compatible definition
// for a directed graph. It can be used to dump a .dot file.
func WriteDot(w io.Writer, nodes []Node) error {
	w.Write([]byte("digraph {\n"))
	defer w.Write([]byte("}\n"))

	for _, n := range nodes {
		nodeLine := fmt.Sprintf("\t\"%s\";\n", n)

		w.Write([]byte(nodeLine))

		for _, edge := range n.Edges() {
			target := edge.Tail()
			line := fmt.Sprintf("\t\"%s\" -> \"%s\" [label=\"%s\"];\n",
				n, target, edge)
			w.Write([]byte(line))
		}
	}

	return nil
}
