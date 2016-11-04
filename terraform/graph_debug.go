package terraform

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/dot"
)

// The NodeDebug method outputs debug information to annotate the graphs
// stored in the DebugInfo
type GraphNodeDebugger interface {
	NodeDebug() string
}

type GraphNodeDebugOrigin interface {
	DotOrigin() bool
}
type DebugGraph struct {
	// TODO: can we combine this and dot.Graph into a generalized graph representation?
	sync.Mutex
	Name string

	step int
	buf  bytes.Buffer

	Dot     *dot.Graph
	dotOpts *GraphDotOpts
}

// DebugGraph holds a dot representation of the Terraform graph, and can be
// written out to the DebugInfo log with DebugInfo.WriteGraph. A DebugGraph can
// log data to it's internal buffer via the Printf and Write methods, which
// will be also be written out to the DebugInfo archive.
func NewDebugGraph(name string, g *Graph, opts *GraphDotOpts) (*DebugGraph, error) {
	dg := &DebugGraph{
		Name:    name,
		dotOpts: opts,
	}

	err := dg.build(g)
	if err != nil {
		dbug.WriteFile(dg.Name, []byte(err.Error()))
		return nil, err
	}
	return dg, nil
}

// Printf to the internal buffer
func (dg *DebugGraph) Printf(f string, args ...interface{}) (int, error) {
	if dg == nil {
		return 0, nil
	}
	dg.Lock()
	defer dg.Unlock()
	return fmt.Fprintf(&dg.buf, f, args...)
}

// Write to the internal buffer
func (dg *DebugGraph) Write(b []byte) (int, error) {
	if dg == nil {
		return 0, nil
	}
	dg.Lock()
	defer dg.Unlock()
	return dg.buf.Write(b)
}

func (dg *DebugGraph) LogBytes() []byte {
	if dg == nil {
		return nil
	}
	dg.Lock()
	defer dg.Unlock()
	return dg.buf.Bytes()
}

func (dg *DebugGraph) DotBytes() []byte {
	if dg == nil {
		return nil
	}
	dg.Lock()
	defer dg.Unlock()
	return dg.Dot.Bytes()
}

func (dg *DebugGraph) DebugNode(v interface{}) {
	if dg == nil {
		return
	}
	dg.Lock()
	defer dg.Unlock()

	name := graphDotNodeName("root", v)

	var node *dot.Node
	// TODO: recursive
	for _, sg := range dg.Dot.Subgraphs {
		node, _ = sg.GetNode(name)
		if node != nil {
			break
		}
	}

	// record as much of the node data structure as we can
	spew.Fdump(&dg.buf, v)

	// for now, record the order of visits in the node label
	if node != nil {
		node.Attrs["label"] = fmt.Sprintf("%s %d", node.Attrs["label"], ord)
	}

	// if the node provides debug output, insert it into the graph, and log it
	if nd, ok := v.(GraphNodeDebugger); ok {
		out := nd.NodeDebug()
		if node != nil {
			node.Attrs["comment"] = out
			dg.buf.WriteString(fmt.Sprintf("NodeDebug (%s):'%s'\n", name, out))
		}
	}
}

//  takes a Terraform Graph and build the internal debug graph
func (dg *DebugGraph) build(g *Graph) error {
	if dg == nil {
		return nil
	}
	dg.Lock()
	defer dg.Unlock()

	dg.Dot = dot.NewGraph(map[string]string{
		"compound": "true",
		"newrank":  "true",
	})
	dg.Dot.Directed = true

	if dg.dotOpts == nil {
		dg.dotOpts = &GraphDotOpts{
			DrawCycles: true,
			MaxDepth:   -1,
			Verbose:    true,
		}
	}

	err := dg.buildSubgraph("root", g, 0)
	if err != nil {
		return err
	}

	return nil
}

func (dg *DebugGraph) buildSubgraph(modName string, g *Graph, modDepth int) error {
	// Respect user-specified module depth
	if dg.dotOpts.MaxDepth >= 0 && modDepth > dg.dotOpts.MaxDepth {
		return nil
	}

	// Begin module subgraph
	var sg *dot.Subgraph
	if modDepth == 0 {
		sg = dg.Dot.AddSubgraph(modName)
	} else {
		sg = dg.Dot.AddSubgraph(modName)
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
		} else if dn.DotNode("fake", dg.dotOpts) == nil {
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
		sg.AddNode(dn.DotNode(nodeName, dg.dotOpts))

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

		err := dg.buildSubgraph(dag.VertexName(v), subgraph, modDepth+1)
		if err != nil {
			return err
		}
	}

	if dg.dotOpts.DrawCycles {
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
		if dr, ok := v.(GraphNodeDebugOrigin); ok {
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
