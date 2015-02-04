package dag

import (
	"fmt"
	"sync"
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

// Walk walks the graph, calling your callback as each node is visited.
// This will walk nodes in parallel if it can.
func (g *AcyclicGraph) Walk(cb WalkFunc) error {
	// We require a root to walk.
	root, err := g.Root()
	if err != nil {
		return err
	}

	// Build the waitgroup that signals when we're done
	var wg sync.WaitGroup
	wg.Add(g.vertices.Len())
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		wg.Wait()
	}()

	// Start walking!
	visitCh := make(chan Vertex, g.vertices.Len())
	visitCh <- root
	for {
		select {
		case v := <-visitCh:
			go g.walkVertex(v, cb, visitCh, &wg)
		case <-doneCh:
			goto WALKDONE
		}
	}

WALKDONE:
	return nil
}

func (g *AcyclicGraph) walkVertex(
	v Vertex, cb WalkFunc, nextCh chan<- Vertex, wg *sync.WaitGroup) {
	defer wg.Done()

	// Call the callback on this vertex
	cb(v)

	// Walk all the children in parallel
	for _, v := range g.DownEdges(v).List() {
		nextCh <- v.(Vertex)
	}
}
