package terraform

import "github.com/hashicorp/terraform/dot"

// GraphNodeDotter can be implemented by a node to cause it to be included
// in the dot graph. The Dot method will be called which is expected to
// return a representation of this node.
type GraphNodeDotter interface {
	// Dot is called to return the dot formatting for the node.
	// The first parameter is the title of the node.
	// The second parameter includes user-specified options that affect the dot
	// graph. See GraphDotOpts below for details.
	DotNode(string, *GraphDotOpts) *dot.Node
}

// GraphDotOpts are the options for generating a dot formatted Graph.
type GraphDotOpts struct {
	// Allows some nodes to decide to only show themselves when the user has
	// requested the "verbose" graph.
	Verbose bool

	// Highlight Cycles
	DrawCycles bool

	// How many levels to expand modules as we draw
	MaxDepth int
}

// GraphDot returns the dot formatting of a visual representation of
// the given Terraform graph.
func GraphDot(g *Graph, opts *GraphDotOpts) (string, error) {
	dg, err := NewDebugGraph("root", g, opts)
	if err != nil {
		return "", err
	}
	return dg.Dot.String(), nil
}
