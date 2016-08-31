package digraph

import (
	"fmt"
	"io"
)

// WriteDot is used to emit a GraphViz compatible definition
// for a directed graph. It can be used to dump a .dot file.
func WriteDot(w io.Writer, nodes []Node) (err error) {
	_, err = w.Write([]byte("digraph {\n"))
	if err != nil {
		return err
	}
	defer func() {
		_, err2 := w.Write([]byte("}\n"))
		if err == nil {
			err = err2
		}
	}()

	for _, n := range nodes {
		nodeLine := fmt.Sprintf("\t\"%s\";\n", n)

		_, err = w.Write([]byte(nodeLine))
		if err != nil {
			return err
		}

		for _, edge := range n.Edges() {
			target := edge.Tail()
			line := fmt.Sprintf("\t\"%s\" -> \"%s\" [label=\"%s\"];\n",
				n, target, edge)
			_, err = w.Write([]byte(line))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
