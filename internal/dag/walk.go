// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"context"
	"errors"
	"log"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
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
	Callback walkFunc

	// Only Start should modify these fields. Modifying them after the walk has
	// started can cause serious problems.
	started   atomic.Bool
	vertices  VertexSet
	vertexMap map[Vertex]*walkerVertex

	// wait is done when all vertices have executed.
	wait sync.WaitGroup

	// diagsMap contains the diagnostics recorded so far for execution,
	// and upstreamFailed contains all the vertices whose problems were
	// caused by upstream failures, and thus whose diagnostics should be
	// excluded from the final set.
	//
	// Readers and writers of either map must hold diagsLock.
	diagsMap       map[Vertex]tfdiags.Diagnostics
	upstreamFailed map[Vertex]struct{}
	diagsLock      sync.Mutex
}

// NewWalker creates a new walker with the given callback function.
func NewWalker(cb walkFunc, opts ...func(*Walker)) *Walker {
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

	// Signals is the collection of all Signals that the walk has accumulated
	// from dependencies to this point.
	Signals []any

	// SignalCh returns the signal emitted by the vertex, or nil.
	SignalCh chan any

	// DepsCh is sent a single value that denotes whether the upstream deps
	// were successful (no errors). Any value sent means that the upstream
	// dependencies are complete. No other values will ever be sent again.
	DepsCh chan bool

	// deps maps each vertex to its walker structure
	deps map[Vertex]*walkerVertex
}

// Walk loads the graph and dispatches the concurrent walker. Walk can only be
// called once for a single Walker.
func (w *Walker) Walk(ctx context.Context, g *AcyclicGraph) tfdiags.Diagnostics {
	if g == nil {
		return nil
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
			V:        v,
			SignalCh: make(chan any),
			deps:     make(map[Vertex]*walkerVertex),
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

	var diags tfdiags.Diagnostics
	w.diagsLock.Lock()
	for v, vDiags := range w.diagsMap {
		if _, upstream := w.upstreamFailed[v]; upstream {
			// Ignore diagnostics for nodes that had failed upstreams, since the
			// downstream diagnostics are likely to be redundant.
			//
			// FIXME: we only need to ignore these because we add them here to
			// skip vertices. Make make it so we don't have these unused
			// diagnostics
			continue
		}
		diags = diags.Append(vDiags)
	}
	w.diagsLock.Unlock()

	return diags
}

// walkVertex walks a single vertex, waiting for any dependencies before
// executing the callback.
func (w *Walker) walkVertex(ctx context.Context, info *walkerVertex) {
	// When we're done executing, lower the waitgroup count
	defer w.wait.Done()

	// The happy paths with return signals, but this prevents any unexpected
	// blocking, and since a nil signal is ignored, a closed channel is fine.
	defer close(info.SignalCh)

	// if there are no deps we have a nil chan, so we need to initialize
	// something that won't block
	depsCh := info.DepsCh
	if depsCh == nil {
		depsCh = make(chan bool, 1)
		depsCh <- true
		close(depsCh)
	}

	// wait for all deps
	var depsSuccess bool
	select {
	case depsSuccess = <-depsCh:
	case <-ctx.Done():
		// context was canceled, stop processing callbacks
		return
	}

	// Run our callback or note that our upstream failed
	var diags tfdiags.Diagnostics
	var upstreamFailed bool

	if depsSuccess {
		signal, cbDiags := w.Callback(ctx, info.V)
		diags = diags.Append(cbDiags)
		// ignoring for now
		_ = signal

		// FIXME: temp work to still halt on errors
		if diags.HasErrors() {
			info.SignalCh <- errStopWalk
		}

	} else {
		log.Printf("[TRACE] dag/walk: upstream of %q errored, so skipping", info.V.Name())
		// This won't be displayed to the user because we'll set upstreamFailed,
		// but we need to ensure there's at least one error in here so that
		// the failures will cascade downstream.
		diags = diags.Append(errors.New("upstream dependencies failed"))
		upstreamFailed = true
	}

	// Record the result (we must do this after execution because we mustn't
	// hold diagsLock while visiting a vertex.)
	w.diagsLock.Lock()
	if w.diagsMap == nil {
		w.diagsMap = make(map[Vertex]tfdiags.Diagnostics)
	}
	w.diagsMap[info.V] = diags
	if w.upstreamFailed == nil {
		w.upstreamFailed = make(map[Vertex]struct{})
	}
	if upstreamFailed {
		w.upstreamFailed[info.V] = struct{}{}
	}
	w.diagsLock.Unlock()
}

func (w *Walker) waitDeps(v *walkerVertex) {
	signals := setMap[any]{make(map[any]bool)}

	// For each dependency given to us, wait for it to complete
	for depV, depInfo := range v.deps {
	DepSatisfied:
		for {
			select {
			case signal := <-depInfo.SignalCh:
				if signal != nil {
					signals.Add(signal)
				}
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

	// If the vertex implements [AlwaysRunVertex], then
	// return ignore the errored dependencies and return success.
	if _, ok := v.V.(AlwaysRunVertex); ok {
		// FIXME: this cancels out upstream errors for later nodes
		v.DepsCh <- true
		return
	}

	// FIXME: temp work to convert diags control to signal control
	for signal := range signals.All() {
		if signal == errStopWalk {
			v.DepsCh <- false
			return
		}
	}

	// All dependencies satisfied and successful
	v.DepsCh <- true
}
