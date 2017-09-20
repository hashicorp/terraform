package terraform

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"

	"github.com/hashicorp/terraform/dag"
)

// RootModuleName is the name given to the root module implicitly.
const RootModuleName = "root"

// RootModulePath is the path for the root module.
var RootModulePath = []string{RootModuleName}

// Graph represents the graph that Terraform uses to represent resources
// and their dependencies.
type Graph struct {
	// Graph is the actual DAG. This is embedded so you can call the DAG
	// methods directly.
	dag.AcyclicGraph

	// Path is the path in the module tree that this Graph represents.
	// The root is represented by a single element list containing
	// RootModuleName
	Path []string

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
func (g *Graph) Walk(walker GraphWalker) error {
	return g.walk(walker)
}

func (g *Graph) walk(walker GraphWalker) error {
	// The callbacks for enter/exiting a graph
	ctx := walker.EnterPath(g.Path)
	defer walker.ExitPath(g.Path)

	// Get the path for logs
	path := strings.Join(ctx.Path(), ".")

	// Determine if our walker is a panic wrapper
	panicwrap, ok := walker.(GraphWalkerPanicwrapper)
	if !ok {
		panicwrap = nil // just to be sure
	}

	debugName := "walk-graph.json"
	if g.debugName != "" {
		debugName = g.debugName + "-" + debugName
	}

	debugBuf := dbug.NewFileWriter(debugName)
	g.SetDebugWriter(debugBuf)
	defer debugBuf.Close()

	// Walk the graph.
	var walkFn dag.WalkFunc
	walkFn = func(v dag.Vertex) (rerr error) {
		log.Printf("[TRACE] vertex '%s.%s': walking", path, dag.VertexName(v))
		g.DebugVisitInfo(v, g.debugName)

		// If we have a panic wrap GraphWalker and a panic occurs, recover
		// and call that. We ensure the return value is an error, however,
		// so that future nodes are not called.
		defer func() {
			// If no panicwrap, do nothing
			if panicwrap == nil {
				return
			}

			// If no panic, do nothing
			err := recover()
			if err == nil {
				return
			}

			// Modify the return value to show the error
			rerr = fmt.Errorf("vertex %q captured panic: %s\n\n%s",
				dag.VertexName(v), err, debug.Stack())

			// Call the panic wrapper
			panicwrap.Panic(v, err)
		}()

		walker.EnterVertex(v)
		defer walker.ExitVertex(v, rerr)

		// vertexCtx is the context that we use when evaluating. This
		// is normally the context of our graph but can be overridden
		// with a GraphNodeSubPath impl.
		vertexCtx := ctx
		if pn, ok := v.(GraphNodeSubPath); ok && len(pn.Path()) > 0 {
			vertexCtx = walker.EnterPath(normalizeModulePath(pn.Path()))
			defer walker.ExitPath(pn.Path())
		}

		// If the node is eval-able, then evaluate it.
		if ev, ok := v.(GraphNodeEvalable); ok {
			tree := ev.EvalTree()
			if tree == nil {
				panic(fmt.Sprintf(
					"%s.%s (%T): nil eval tree", path, dag.VertexName(v), v))
			}

			// Allow the walker to change our tree if needed. Eval,
			// then callback with the output.
			log.Printf("[TRACE] vertex '%s.%s': evaluating", path, dag.VertexName(v))

			g.DebugVertexInfo(v, fmt.Sprintf("evaluating %T(%s)", v, path))

			tree = walker.EnterEvalTree(v, tree)
			output, err := Eval(tree, vertexCtx)
			if rerr = walker.ExitEvalTree(v, output, err); rerr != nil {
				return
			}
		}

		// If the node is dynamically expanded, then expand it
		if ev, ok := v.(GraphNodeDynamicExpandable); ok {
			log.Printf(
				"[TRACE] vertex '%s.%s': expanding/walking dynamic subgraph",
				path,
				dag.VertexName(v))

			g.DebugVertexInfo(v, fmt.Sprintf("expanding %T(%s)", v, path))

			g, err := ev.DynamicExpand(vertexCtx)
			if err != nil {
				rerr = err
				return
			}
			if g != nil {
				// Walk the subgraph
				if rerr = g.walk(walker); rerr != nil {
					return
				}
			}
		}

		// If the node has a subgraph, then walk the subgraph
		if sn, ok := v.(GraphNodeSubgraph); ok {
			log.Printf(
				"[TRACE] vertex '%s.%s': walking subgraph",
				path,
				dag.VertexName(v))

			g.DebugVertexInfo(v, fmt.Sprintf("subgraph: %T(%s)", v, path))

			if rerr = sn.Subgraph().(*Graph).walk(walker); rerr != nil {
				return
			}
		}

		return nil
	}

	return g.AcyclicGraph.Walk(walkFn)
}
