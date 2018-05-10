package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/dag"
)

// Graph represents the graph that Terraform uses to represent resources
// and their dependencies.
type Graph struct {
	// Graph is the actual DAG. This is embedded so you can call the DAG
	// methods directly.
	dag.AcyclicGraph

	// Path is the path in the module tree that this Graph represents.
	Path addrs.ModuleInstance

	// debugName is a name for reference in the debug output. This is usually
	// to indicate what topmost builder was, and if this graph is a shadow or
	// not.
	debugName string
}

func (g *Graph) DirectedGraph() dag.Grapher {
	return &g.AcyclicGraph
}

// Walk walks the graph with the given walker for callbacks. The graph
// will be walked with full parallelism, so the walker should expect
// to be called in concurrently.
func (g *Graph) Walk(walker GraphWalker) tfdiags.Diagnostics {
	return g.walk(walker)
}

func (g *Graph) walk(walker GraphWalker) tfdiags.Diagnostics {
	// The callbacks for enter/exiting a graph
	ctx := walker.EnterPath(g.Path)
	defer walker.ExitPath(g.Path)

	// Get the path for logs
	path := ctx.Path().String()

	debugName := "walk-graph.json"
	if g.debugName != "" {
		debugName = g.debugName + "-" + debugName
	}

	debugBuf := dbug.NewFileWriter(debugName)
	g.SetDebugWriter(debugBuf)
	defer debugBuf.Close()

	// Walk the graph.
	var walkFn dag.WalkFunc
	walkFn = func(v dag.Vertex) (diags tfdiags.Diagnostics) {
		log.Printf("[TRACE] vertex %q: starting visit (%T)", dag.VertexName(v), v)
		g.DebugVisitInfo(v, g.debugName)

		defer func() {
			log.Printf("[TRACE] vertex %q: visit complete", dag.VertexName(v))
		}()

		walker.EnterVertex(v)
		defer walker.ExitVertex(v, diags)

		// vertexCtx is the context that we use when evaluating. This
		// is normally the context of our graph but can be overridden
		// with a GraphNodeSubPath impl.
		vertexCtx := ctx
		if pn, ok := v.(GraphNodeSubPath); ok && len(pn.Path()) > 0 {
			vertexCtx = walker.EnterPath(pn.Path())
			defer walker.ExitPath(pn.Path())
		}

		// If the node is eval-able, then evaluate it.
		if ev, ok := v.(GraphNodeEvalable); ok {
			tree := ev.EvalTree()
			if tree == nil {
				panic(fmt.Sprintf("%q (%T): nil eval tree", dag.VertexName(v), v))
			}

			// Allow the walker to change our tree if needed. Eval,
			// then callback with the output.
			log.Printf("[TRACE] vertex %q: evaluating", dag.VertexName(v))

			g.DebugVertexInfo(v, fmt.Sprintf("evaluating %T(%s)", v, path))

			tree = walker.EnterEvalTree(v, tree)
			output, err := Eval(tree, vertexCtx)
			diags = diags.Append(walker.ExitEvalTree(v, output, err))
			if diags.HasErrors() {
				return
			}
		}

		// If the node is dynamically expanded, then expand it
		if ev, ok := v.(GraphNodeDynamicExpandable); ok {
			log.Printf("[TRACE] vertex %q: expanding dynamic subgraph", dag.VertexName(v))

			g.DebugVertexInfo(v, fmt.Sprintf("expanding %T(%s)", v, path))

			g, err := ev.DynamicExpand(vertexCtx)
			if err != nil {
				diags = diags.Append(err)
				return
			}
			if g != nil {
				// Walk the subgraph
				log.Printf("[TRACE] vertex %q: entering dynamic subgraph", dag.VertexName(v))
				subDiags := g.walk(walker)
				diags = diags.Append(subDiags)
				if subDiags.HasErrors() {
					log.Printf("[TRACE] vertex %q: dynamic subgraph encountered errors", dag.VertexName(v))
					return
				}
				log.Printf("[TRACE] vertex %q: dynamic subgraph completed successfully", dag.VertexName(v))
			} else {
				log.Printf("[TRACE] vertex %q: produced no dynamic subgraph", dag.VertexName(v))
			}
		}

		// If the node has a subgraph, then walk the subgraph
		if sn, ok := v.(GraphNodeSubgraph); ok {
			log.Printf("[TRACE] vertex %q: entering static subgraph", dag.VertexName(v))

			g.DebugVertexInfo(v, fmt.Sprintf("subgraph: %T(%s)", v, path))

			subDiags := sn.Subgraph().(*Graph).walk(walker)
			if subDiags.HasErrors() {
				log.Printf("[TRACE] vertex %q: static subgraph encountered errors", dag.VertexName(v))
				return
			}
			log.Printf("[TRACE] vertex %q: static subgraph completed successfully", dag.VertexName(v))
		}

		return
	}

	return g.AcyclicGraph.Walk(walkFn)
}
