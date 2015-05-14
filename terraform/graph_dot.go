package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/dot"
)

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

type GraphNodeDotOrigin interface {
	DotOrigin() bool
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
	dg := dot.NewGraph(map[string]string{
		"compound": "true",
		"newrank":  "true",
	})
	dg.Directed = true

	err := graphDotSubgraph(dg, "root", g, opts, 0)
	if err != nil {
		return "", err
	}

	return dg.String(), nil
}

func graphDotSubgraph(
	dg *dot.Graph, modName string, g *Graph, opts *GraphDotOpts, modDepth int) error {
	// Respect user-specified module depth
	if opts.MaxDepth >= 0 && modDepth > opts.MaxDepth {
		return nil
	}

	// Begin module subgraph
	var sg *dot.Subgraph
	if modDepth == 0 {
		sg = dg.AddSubgraph(modName)
	} else {
		sg = dg.AddSubgraph(modName)
		sg.Cluster = true
		sg.AddAttr("label", modName)
	}

	origins, err := graphDotFindOrigins(g)
	if err != nil {
		return err
	}

	drawableVertices := make(map[dag.Vertex]struct{})
	toDraw := make([]dag.Vertex, 0, len(g.Vertices()))
	subgraphVertices := make(map[dag.Vertex]*Graph)

	walk := func(v dag.Vertex, depth int) error {
		// We only care about nodes that yield non-empty Dot strings.
		if dn, ok := v.(GraphNodeDotter); !ok {
			return nil
		} else if dn.DotNode("fake", opts) == nil {
			return nil
		}

		drawableVertices[v] = struct{}{}
		toDraw = append(toDraw, v)

		if sn, ok := v.(GraphNodeSubgraph); ok {
			subgraphVertices[v] = sn.Subgraph()
		}
		return nil
	}

	if err := g.ReverseDepthFirstWalk(origins, walk); err != nil {
		return err
	}

	for _, v := range toDraw {
		dn := v.(GraphNodeDotter)
		nodeName := graphDotNodeName(modName, v)
		sg.AddNode(dn.DotNode(nodeName, opts))

		// Draw all the edges from this vertex to other nodes
		targets := dag.AsVertexList(g.DownEdges(v))
		for _, t := range targets {
			target := t.(dag.Vertex)
			// Only want edges where both sides are drawable.
			if _, ok := drawableVertices[target]; !ok {
				continue
			}

			if err := sg.AddEdgeBetween(
				graphDotNodeName(modName, v),
				graphDotNodeName(modName, target),
				map[string]string{}); err != nil {
				return err
			}
		}
	}

	// Recurse into any subgraphs
	for _, v := range toDraw {
		subgraph, ok := subgraphVertices[v]
		if !ok {
			continue
		}

		err := graphDotSubgraph(dg, dag.VertexName(v), subgraph, opts, modDepth+1)
		if err != nil {
			return err
		}
	}

	if opts.DrawCycles {
		colors := []string{"red", "green", "blue"}
		for ci, cycle := range g.Cycles() {
			for i, c := range cycle {
				// Catch the last wrapping edge of the cycle
				if i+1 >= len(cycle) {
					i = -1
				}
				edgeAttrs := map[string]string{
					"color":    colors[ci%len(colors)],
					"penwidth": "2.0",
				}

				if err := sg.AddEdgeBetween(
					graphDotNodeName(modName, c),
					graphDotNodeName(modName, cycle[i+1]),
					edgeAttrs); err != nil {
					return err
				}

			}
		}
	}

	return nil
}

func graphDotNodeName(modName, v dag.Vertex) string {
	return fmt.Sprintf("[%s] %s", modName, dag.VertexName(v))
}

func graphDotFindOrigins(g *Graph) ([]dag.Vertex, error) {
	var origin []dag.Vertex

	for _, v := range g.Vertices() {
		if dr, ok := v.(GraphNodeDotOrigin); ok {
			if dr.DotOrigin() {
				origin = append(origin, v)
			}
		}
	}

	if len(origin) == 0 {
		return nil, fmt.Errorf("No DOT origin nodes found.\nGraph: %s", g.String())
	}

	return origin, nil
}
