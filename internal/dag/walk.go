// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"context"
	"log"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// Walker is used to walk every vertex of a graph in parallel.
//
// A vertex will only be walked when the dependencies of that vertex have
// been walked. If two vertices can be walked at the same time, they will be.
//
// Non-parallelism can be enforced by introducing a lock in your callback
// function. However, the goroutine overhead of a walk will remain.
// Walker will create V*2 goroutines (one for each vertex, and dependency
// waiter for each vertex). In general this should be of no concern unless
// there are a huge number of vertices.
//
// A single walker is only valid for one graph walk. After the walk is complete
// you must construct a new walker to walk again. State for the walk is never
// deleted in case vertices or edges are changed.
type Walker struct {
	// Callback is what is called for each vertex
	Callback WalkFunc

	// Only Start should modify these fields. Modifying them after the walk has
	// started can cause serious problems.
	started   atomic.Bool
	vertices  VertexSet
	vertexMap map[Vertex]*walkerVertex

	// wait is done when all vertices have executed.
	wait sync.WaitGroup
}

// WalkFunc is the callback used for the primary concurrent walk of the graph
// via Walker. WalkFunc returns an optional signal value along with diagnostics.
// The walker will execute the callback for every node in the graph unless the
// context is canceled. It is up to the callback implementation to coordinate
// any early returns via signals.
//
// Signals: The Walker will collect and deduplicate all upstream signals, and
// pass them into the callback within the Context. Signals are extracted from
// the context in the callback via the dag.Signals(ctx) function. The value and
// meaning of signals are completely up to the calling package.
type WalkFunc func(context.Context, Vertex) any

// NewWalker creates a new walker with the given callback function.
func NewWalker(cb WalkFunc, opts ...func(*Walker)) *Walker {
	w := &Walker{
		Callback: cb,
		vertices: NewVertexSet(),
	}
	for _, opt := range opts {
		opt(w)
	}

	return w
}

type ctxSignalKey string

var signalKey ctxSignalKey = "signals"

// Signals is used to extract any upstream signals from the context sent by
// prior vertex callbacks when using dag.Walker (or AcyclicGraph.Walk).
func Signals(ctx context.Context) []any {
	return ctx.Value(signalKey).([]any)
}

type walkerVertex struct {
	V Vertex

	// Signals is the collection of all Signals that the walk has accumulated to
	// this point. Signals will include this vertex's own callback after that
	// has been called as well. We collect all signals at a single point for
	// each vertex, because they need to be broadcast to all dependents, but
	// broadcast is done via a channel closure and cannot simultaneously
	// transmit the signals.
	Signals []any

	// DoneCh is closed to broadcast callback completion to all dependents.
	DoneCh chan bool

	// DepsCh is closed to denotes that all dependents have completed.
	DepsCh chan bool

	// deps maps each dependency's vertex to its walker structure.
	deps map[Vertex]*walkerVertex
}

// Walk loads the graph and dispatches the concurrent walker. Walk can only be
// called once for a single Walker.
func (w *Walker) Walk(ctx context.Context, g *AcyclicGraph) {
	if g == nil {
		return
	}

	if w.started.Load() {
		panic("graph walker already started")
	}
	w.started.Store(true)

	// Initialize fields
	if w.vertexMap == nil {
		w.vertexMap = make(map[Vertex]*walkerVertex)
	}

	// Add the new vertices
	for v := range g.vertices.All() {
		// Add to the waitgroup so our walk is not done until everything finishes
		w.wait.Add(1)

		// Add to our own set so we know about it already
		w.vertices.Add(v)

		// Initialize the vertex info
		info := &walkerVertex{
			V:      v,
			DoneCh: make(chan bool),
			deps:   make(map[Vertex]*walkerVertex),
		}

		// Add it to the map and kick off the walk
		w.vertexMap[v] = info
	}

	// Iterate over all edges so we can connect the waiters to their
	// dependencies channels, then kickoff a waiter for all the dependencies.
	for waiter, deps := range g.edgesFrom {
		// Get the info for the waiter
		waiterInfo, ok := w.vertexMap[waiter]
		if !ok {
			// Vertex doesn't exist... shouldn't be possible but ignore.
			continue
		}

		for dep := range deps.All() {
			// Get the info for the dep
			depInfo, ok := w.vertexMap[dep]
			if !ok {
				// Vertex doesn't exist... shouldn't be possible but ignore.
				continue
			}

			// Add the dependency to our waiter
			waiterInfo.deps[dep] = depInfo
		}

		// We have some deps, so make a chan to indicate when all deps are
		// complete
		waiterInfo.DepsCh = make(chan bool, 1)

		// Start the waiter
		go w.waitDeps(waiterInfo)
	}

	// Start all the new vertices. We do this at the end so that all
	// the edge waiters and changes are set up above.
	for _, info := range w.vertexMap {
		go w.walkVertex(ctx, info)
	}

	// Wait for completion
	w.wait.Wait()
}

// walkVertex walks a single vertex, waiting for any dependencies before
// executing the callback.
func (w *Walker) walkVertex(ctx context.Context, info *walkerVertex) {
	// When we're done executing, lower the waitgroup count
	defer w.wait.Done()

	// The happy paths with return signals, but this prevents any unexpected
	// blocking, and since a nil signal is ignored, a closed channel is fine.
	defer close(info.DoneCh)

	// if there are no deps we have a nil chan, so we need to initialize
	// something that won't block
	depsCh := info.DepsCh
	if depsCh == nil {
		depsCh = make(chan bool, 1)
		depsCh <- true
		close(depsCh)
	}

	// wait for all deps
	select {
	case <-depsCh:
	case <-ctx.Done():
		// context was canceled, stop processing callbacks
		return
	}

	ctx = context.WithValue(ctx, signalKey, info.Signals)

	signal := w.Callback(ctx, info.V)
	if signal != nil {
		info.Signals = append(info.Signals, signal)
	}
}

func (w *Walker) waitDeps(v *walkerVertex) {
	defer close(v.DepsCh)

	signals := setMap[any]{make(map[any]bool)}

	// For each dependency given to us, wait for it to complete
	for depV, depInfo := range v.deps {
	DepSatisfied:
		for {
			select {
			case <-depInfo.DoneCh:
				// collect all upstream signals
				for _, signal := range depInfo.Signals {
					signals.Add(signal)
				}
				// Dependency satisfied!
				break DepSatisfied

			case <-time.After(time.Second * 5):
				log.Printf("[TRACE] dag/walk: vertex %q is waiting for %q",
					v.V.Name(), depV.Name())
			}
		}
	}

	v.Signals = slices.Collect(signals.All())
}
