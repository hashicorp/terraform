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
	ctx := walker.EvalContext()

	// Walk the graph.
	var walkFn dag.WalkFunc
	walkFn = func(v dag.Vertex) (diags tfdiags.Diagnostics) {
		log.Printf("[TRACE] vertex %q: starting visit (%T)", dag.VertexName(v), v)

		defer func() {
			log.Printf("[TRACE] vertex %q: visit complete", dag.VertexName(v))
		}()

		walker.EnterVertex(v)
		defer walker.ExitVertex(v, diags)

		// vertexCtx is the context that we use when evaluating. This
		// is normally the context of our graph but can be overridden
		// with a GraphNodeModuleInstance impl.
		vertexCtx := ctx
		if pn, ok := v.(GraphNodeModuleInstance); ok {
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
		return
	}

	return g.AcyclicGraph.Walk(walkFn)
}
