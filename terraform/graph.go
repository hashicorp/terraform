package terraform

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/dag"
)

// RootModuleName is the name given to the root module implicitly.
const RootModuleName = "root"

// RootModulePath is the path for the root module.
var RootModulePath = []string{RootModuleName}

// Graph represents the graph that Terraform uses to represent resources
// and their dependencies. Each graph represents only one module, but it
// can contain further modules, which themselves have their own graph.
type Graph struct {
	// Graph is the actual DAG. This is embedded so you can call the DAG
	// methods directly.
	dag.AcyclicGraph

	// Path is the path in the module tree that this Graph represents.
	// The root is represented by a single element list containing
	// RootModuleName
	Path []string

	// annotations are the annotations that are added to vertices. Annotations
	// are arbitrary metadata taht is used for various logic. Annotations
	// should have unique keys that are referenced via constants.
	annotations map[dag.Vertex]map[string]interface{}

	// dependableMap is a lookaside table for fast lookups for connecting
	// dependencies by their GraphNodeDependable value to avoid O(n^3)-like
	// situations and turn them into O(1) with respect to the number of new
	// edges.
	dependableMap map[string]dag.Vertex

	// debugName is a name for reference in the debug output. This is usually
	// to indicate what topmost builder was, and if this graph is a shadow or
	// not.
	debugName string

	once sync.Once
}

func (g *Graph) DirectedGraph() dag.Grapher {
	return &g.AcyclicGraph
}

// Annotations returns the annotations that are configured for the
// given vertex. The map is guaranteed to be non-nil but may be empty.
//
// The returned map may be modified to modify the annotations of the
// vertex.
func (g *Graph) Annotations(v dag.Vertex) map[string]interface{} {
	g.once.Do(g.init)

	// If this vertex isn't in the graph, then just return an empty map
	if !g.HasVertex(v) {
		return map[string]interface{}{}
	}

	// Get the map, if it doesn't exist yet then initialize it
	m, ok := g.annotations[v]
	if !ok {
		m = make(map[string]interface{})
		g.annotations[v] = m
	}

	return m
}

// Add is the same as dag.Graph.Add.
func (g *Graph) Add(v dag.Vertex) dag.Vertex {
	g.once.Do(g.init)

	// Call upwards to add it to the actual graph
	g.Graph.Add(v)

	// If this is a depend-able node, then store the lookaside info
	if dv, ok := v.(GraphNodeDependable); ok {
		for _, n := range dv.DependableName() {
			g.dependableMap[n] = v
		}
	}

	// If this initializes annotations, then do that
	if av, ok := v.(GraphNodeAnnotationInit); ok {
		as := g.Annotations(v)
		for k, v := range av.AnnotationInit() {
			as[k] = v
		}
	}

	return v
}

// Remove is the same as dag.Graph.Remove
func (g *Graph) Remove(v dag.Vertex) dag.Vertex {
	g.once.Do(g.init)

	// If this is a depend-able node, then remove the lookaside info
	if dv, ok := v.(GraphNodeDependable); ok {
		for _, n := range dv.DependableName() {
			delete(g.dependableMap, n)
		}
	}

	// Remove the annotations
	delete(g.annotations, v)

	// Call upwards to remove it from the actual graph
	return g.Graph.Remove(v)
}

// Replace is the same as dag.Graph.Replace
func (g *Graph) Replace(o, n dag.Vertex) bool {
	g.once.Do(g.init)

	// Go through and update our lookaside to point to the new vertex
	for k, v := range g.dependableMap {
		if v == o {
			if _, ok := n.(GraphNodeDependable); ok {
				g.dependableMap[k] = n
			} else {
				delete(g.dependableMap, k)
			}
		}
	}

	// Move the annotation if it exists
	if m, ok := g.annotations[o]; ok {
		g.annotations[n] = m
		delete(g.annotations, o)
	}

	return g.Graph.Replace(o, n)
}

// ConnectDependent connects a GraphNodeDependent to all of its
// GraphNodeDependables. It returns the list of dependents it was
// unable to connect to.
func (g *Graph) ConnectDependent(raw dag.Vertex) []string {
	v, ok := raw.(GraphNodeDependent)
	if !ok {
		return nil
	}

	return g.ConnectTo(v, v.DependentOn())
}

// ConnectDependents goes through the graph, connecting all the
// GraphNodeDependents to GraphNodeDependables. This is safe to call
// multiple times.
//
// To get details on whether dependencies could be found/made, the more
// specific ConnectDependent should be used.
func (g *Graph) ConnectDependents() {
	for _, v := range g.Vertices() {
		if dv, ok := v.(GraphNodeDependent); ok {
			g.ConnectDependent(dv)
		}
	}
}

// ConnectFrom creates an edge by finding the source from a DependableName
// and connecting it to the specific vertex.
func (g *Graph) ConnectFrom(source string, target dag.Vertex) {
	g.once.Do(g.init)

	if source := g.dependableMap[source]; source != nil {
		g.Connect(dag.BasicEdge(source, target))
	}
}

// ConnectTo connects a vertex to a raw string of targets that are the
// result of DependableName, and returns the list of targets that are missing.
func (g *Graph) ConnectTo(v dag.Vertex, targets []string) []string {
	g.once.Do(g.init)

	var missing []string
	for _, t := range targets {
		if dest := g.dependableMap[t]; dest != nil {
			g.Connect(dag.BasicEdge(v, dest))
		} else {
			missing = append(missing, t)
		}
	}

	return missing
}

// Dependable finds the vertices in the graph that have the given dependable
// names and returns them.
func (g *Graph) Dependable(n string) dag.Vertex {
	// TODO: do we need this?
	return nil
}

// Walk walks the graph with the given walker for callbacks. The graph
// will be walked with full parallelism, so the walker should expect
// to be called in concurrently.
func (g *Graph) Walk(walker GraphWalker) error {
	return g.walk(walker)
}

func (g *Graph) init() {
	if g.annotations == nil {
		g.annotations = make(map[dag.Vertex]map[string]interface{})
	}

	if g.dependableMap == nil {
		g.dependableMap = make(map[string]dag.Vertex)
	}
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
		log.Printf("[DEBUG] vertex '%s.%s': walking", path, dag.VertexName(v))
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
			log.Printf("[DEBUG] vertex '%s.%s': evaluating", path, dag.VertexName(v))

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
				"[DEBUG] vertex '%s.%s': expanding/walking dynamic subgraph",
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
				"[DEBUG] vertex '%s.%s': walking subgraph",
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

// GraphNodeAnnotationInit is an interface that allows a node to
// initialize it's annotations.
//
// AnnotationInit will be called _once_ when the node is added to a
// graph for the first time and is expected to return it's initial
// annotations.
type GraphNodeAnnotationInit interface {
	AnnotationInit() map[string]interface{}
}

// GraphNodeDependable is an interface which says that a node can be
// depended on (an edge can be placed between this node and another) according
// to the well-known name returned by DependableName.
//
// DependableName can return multiple names it is known by.
type GraphNodeDependable interface {
	DependableName() []string
}

// GraphNodeDependent is an interface which says that a node depends
// on another GraphNodeDependable by some name. By implementing this
// interface, Graph.ConnectDependents() can be called multiple times
// safely and efficiently.
type GraphNodeDependent interface {
	DependentOn() []string
}
