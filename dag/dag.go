package dag

import (
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
)

// AcyclicGraph is a specialization of Graph that cannot have cycles. With
// this property, we get the property of sane graph traversal.
type AcyclicGraph struct {
	Graph
}

// WalkFunc is the callback used for walking the graph.
type WalkFunc func(Vertex)

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

// Validate validates the DAG. A DAG is valid if it has a single root
// with no cycles.
func (g *AcyclicGraph) Validate() error {
	if _, err := g.Root(); err != nil {
		return err
	}

	// Look for cycles of more than 1 component
	var err error
	var cycles [][]Vertex
	for _, cycle := range StronglyConnected(&g.Graph) {
		if len(cycle) > 1 {
			cycles = append(cycles, cycle)
		}
	}
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

// Walk walks the graph, calling your callback as each node is visited.
// This will walk nodes in parallel if it can.
func (g *AcyclicGraph) Walk(cb WalkFunc) error {
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
	for _, v := range vertices {
		// Get the list of channels to wait on
		deps := g.DownEdges(v).List()
		depChs := make([]<-chan struct{}, len(deps))
		for i, dep := range deps {
			depChs[i] = vertMap[dep.(Vertex)]
		}

		// Get our channel
		ourCh := vertMap[v]

		// Start the goroutine
		go func(v Vertex, doneCh chan<- struct{}, chs []<-chan struct{}) {
			defer close(doneCh)
			defer wg.Done()

			// Wait on all our dependencies
			for _, ch := range chs {
				<-ch
			}

			// Call our callback
			cb(v)
		}(v, ourCh, depChs)
	}

	<-doneCh
	return nil
}
