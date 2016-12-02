package dag

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
)

// AcyclicGraph is a specialization of Graph that cannot have cycles. With
// this property, we get the property of sane graph traversal.
type AcyclicGraph struct {
	Graph
}

// WalkFunc is the callback used for walking the graph.
type WalkFunc func(Vertex) error

// DepthWalkFunc is a walk function that also receives the current depth of the
// walk as an argument
type DepthWalkFunc func(Vertex, int) error

func (g *AcyclicGraph) DirectedGraph() Grapher {
	return g
}

// Returns a Set that includes every Vertex yielded by walking down from the
// provided starting Vertex v.
func (g *AcyclicGraph) Ancestors(v Vertex) (*Set, error) {
	s := new(Set)
	start := AsVertexList(g.DownEdges(v))
	memoFunc := func(v Vertex, d int) error {
		s.Add(v)
		return nil
	}

	if err := g.DepthFirstWalk(start, memoFunc); err != nil {
		return nil, err
	}

	return s, nil
}

// Returns a Set that includes every Vertex yielded by walking up from the
// provided starting Vertex v.
func (g *AcyclicGraph) Descendents(v Vertex) (*Set, error) {
	s := new(Set)
	start := AsVertexList(g.UpEdges(v))
	memoFunc := func(v Vertex, d int) error {
		s.Add(v)
		return nil
	}

	if err := g.ReverseDepthFirstWalk(start, memoFunc); err != nil {
		return nil, err
	}

	return s, nil
}

// Root returns the root of the DAG, or an error.
//
// Complexity: O(V)
func (g *AcyclicGraph) Root() (Vertex, error) {
	roots := make([]Vertex, 0, 1)
	for _, v := range g.Vertices() {
		if g.UpEdges(v).Len() == 0 {
			roots = append(roots, v)
		}
	}

	if len(roots) > 1 {
		// TODO(mitchellh): make this error message a lot better
		return nil, fmt.Errorf("multiple roots: %#v", roots)
	}

	if len(roots) == 0 {
		return nil, fmt.Errorf("no roots found")
	}

	return roots[0], nil
}

// TransitiveReduction performs the transitive reduction of graph g in place.
// The transitive reduction of a graph is a graph with as few edges as
// possible with the same reachability as the original graph. This means
// that if there are three nodes A => B => C, and A connects to both
// B and C, and B connects to C, then the transitive reduction is the
// same graph with only a single edge between A and B, and a single edge
// between B and C.
//
// The graph must be valid for this operation to behave properly. If
// Validate() returns an error, the behavior is undefined and the results
// will likely be unexpected.
//
// Complexity: O(V(V+E)), or asymptotically O(VE)
func (g *AcyclicGraph) TransitiveReduction() {
	// For each vertex u in graph g, do a DFS starting from each vertex
	// v such that the edge (u,v) exists (v is a direct descendant of u).
	//
	// For each v-prime reachable from v, remove the edge (u, v-prime).
	defer g.debug.BeginOperation("TransitiveReduction", "").End("")

	for _, u := range g.Vertices() {
		uTargets := g.DownEdges(u)
		vs := AsVertexList(g.DownEdges(u))

		g.DepthFirstWalk(vs, func(v Vertex, d int) error {
			shared := uTargets.Intersection(g.DownEdges(v))
			for _, vPrime := range AsVertexList(shared) {
				g.RemoveEdge(BasicEdge(u, vPrime))
			}

			return nil
		})
	}
}

// Validate validates the DAG. A DAG is valid if it has a single root
// with no cycles.
func (g *AcyclicGraph) Validate() error {
	if _, err := g.Root(); err != nil {
		return err
	}

	// Look for cycles of more than 1 component
	var err error
	cycles := g.Cycles()
	if len(cycles) > 0 {
		for _, cycle := range cycles {
			cycleStr := make([]string, len(cycle))
			for j, vertex := range cycle {
				cycleStr[j] = VertexName(vertex)
			}

			err = multierror.Append(err, fmt.Errorf(
				"Cycle: %s", strings.Join(cycleStr, ", ")))
		}
	}

	// Look for cycles to self
	for _, e := range g.Edges() {
		if e.Source() == e.Target() {
			err = multierror.Append(err, fmt.Errorf(
				"Self reference: %s", VertexName(e.Source())))
		}
	}

	return err
}

func (g *AcyclicGraph) Cycles() [][]Vertex {
	var cycles [][]Vertex
	for _, cycle := range StronglyConnected(&g.Graph) {
		if len(cycle) > 1 {
			cycles = append(cycles, cycle)
		}
	}
	return cycles
}

// Walk walks the graph, calling your callback as each node is visited.
// This will walk nodes in parallel if it can. Because the walk is done
// in parallel, the error returned will be a multierror.
func (g *AcyclicGraph) Walk(cb WalkFunc) error {
	defer g.debug.BeginOperation(typeWalk, "").End("")

	// Cache the vertices since we use it multiple times
	vertices := g.Vertices()

	// Build the waitgroup that signals when we're done
	var wg sync.WaitGroup
	wg.Add(len(vertices))
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		wg.Wait()
	}()

	// The map of channels to watch to wait for vertices to finish
	vertMap := make(map[Vertex]chan struct{})
	for _, v := range vertices {
		vertMap[v] = make(chan struct{})
	}

	// The map of whether a vertex errored or not during the walk
	var errLock sync.Mutex
	var errs error
	errMap := make(map[Vertex]bool)
	for _, v := range vertices {
		// Build our list of dependencies and the list of channels to
		// wait on until we start executing for this vertex.
		deps := AsVertexList(g.DownEdges(v))
		depChs := make([]<-chan struct{}, len(deps))
		for i, dep := range deps {
			depChs[i] = vertMap[dep]
		}

		// Get our channel so that we can close it when we're done
		ourCh := vertMap[v]

		// Start the goroutine to wait for our dependencies
		readyCh := make(chan bool)
		go func(v Vertex, deps []Vertex, chs []<-chan struct{}, readyCh chan<- bool) {
			// First wait for all the dependencies
			for i, ch := range chs {
			DepSatisfied:
				for {
					select {
					case <-ch:
						break DepSatisfied
					case <-time.After(time.Second * 5):
						log.Printf("[DEBUG] vertex %q, waiting for: %q",
							VertexName(v), VertexName(deps[i]))
					}
				}
				log.Printf("[DEBUG] vertex %q, got dep: %q",
					VertexName(v), VertexName(deps[i]))
			}

			// Then, check the map to see if any of our dependencies failed
			errLock.Lock()
			defer errLock.Unlock()
			for _, dep := range deps {
				if errMap[dep] {
					errMap[v] = true
					readyCh <- false
					return
				}
			}

			readyCh <- true
		}(v, deps, depChs, readyCh)

		// Start the goroutine that executes
		go func(v Vertex, doneCh chan<- struct{}, readyCh <-chan bool) {
			defer close(doneCh)
			defer wg.Done()

			var err error
			if ready := <-readyCh; ready {
				err = cb(v)
			}

			errLock.Lock()
			defer errLock.Unlock()
			if err != nil {
				errMap[v] = true
				errs = multierror.Append(errs, err)
			}
		}(v, ourCh, readyCh)
	}

	<-doneCh
	return errs
}

// simple convenience helper for converting a dag.Set to a []Vertex
func AsVertexList(s *Set) []Vertex {
	rawList := s.List()
	vertexList := make([]Vertex, len(rawList))
	for i, raw := range rawList {
		vertexList[i] = raw.(Vertex)
	}
	return vertexList
}

type vertexAtDepth struct {
	Vertex Vertex
	Depth  int
}

// depthFirstWalk does a depth-first walk of the graph starting from
// the vertices in start. This is not exported now but it would make sense
// to export this publicly at some point.
func (g *AcyclicGraph) DepthFirstWalk(start []Vertex, f DepthWalkFunc) error {
	defer g.debug.BeginOperation(typeDepthFirstWalk, "").End("")

	seen := make(map[Vertex]struct{})
	frontier := make([]*vertexAtDepth, len(start))
	for i, v := range start {
		frontier[i] = &vertexAtDepth{
			Vertex: v,
			Depth:  0,
		}
	}
	for len(frontier) > 0 {
		// Pop the current vertex
		n := len(frontier)
		current := frontier[n-1]
		frontier = frontier[:n-1]

		// Check if we've seen this already and return...
		if _, ok := seen[current.Vertex]; ok {
			continue
		}
		seen[current.Vertex] = struct{}{}

		// Visit the current node
		if err := f(current.Vertex, current.Depth); err != nil {
			return err
		}

		// Visit targets of this in a consistent order.
		targets := AsVertexList(g.DownEdges(current.Vertex))
		sort.Sort(byVertexName(targets))
		for _, t := range targets {
			frontier = append(frontier, &vertexAtDepth{
				Vertex: t,
				Depth:  current.Depth + 1,
			})
		}
	}

	return nil
}

// reverseDepthFirstWalk does a depth-first walk _up_ the graph starting from
// the vertices in start.
func (g *AcyclicGraph) ReverseDepthFirstWalk(start []Vertex, f DepthWalkFunc) error {
	defer g.debug.BeginOperation(typeReverseDepthFirstWalk, "").End("")

	seen := make(map[Vertex]struct{})
	frontier := make([]*vertexAtDepth, len(start))
	for i, v := range start {
		frontier[i] = &vertexAtDepth{
			Vertex: v,
			Depth:  0,
		}
	}
	for len(frontier) > 0 {
		// Pop the current vertex
		n := len(frontier)
		current := frontier[n-1]
		frontier = frontier[:n-1]

		// Check if we've seen this already and return...
		if _, ok := seen[current.Vertex]; ok {
			continue
		}
		seen[current.Vertex] = struct{}{}

		// Add next set of targets in a consistent order.
		targets := AsVertexList(g.UpEdges(current.Vertex))
		sort.Sort(byVertexName(targets))
		for _, t := range targets {
			frontier = append(frontier, &vertexAtDepth{
				Vertex: t,
				Depth:  current.Depth + 1,
			})
		}

		// Visit the current node
		if err := f(current.Vertex, current.Depth); err != nil {
			return err
		}
	}

	return nil
}

// byVertexName implements sort.Interface so a list of Vertices can be sorted
// consistently by their VertexName
type byVertexName []Vertex

func (b byVertexName) Len() int      { return len(b) }
func (b byVertexName) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byVertexName) Less(i, j int) bool {
	return VertexName(b[i]) < VertexName(b[j])
}
