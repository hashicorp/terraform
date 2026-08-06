// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"errors"
	"log"
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

// DependencyResult indicates if a dependency check resulted in success, failure, or tolerance.
type DependencyResult string

const (
	// DependencyResultSuccess indicates that all dependencies were satisfied.
	DependencyResultSuccess DependencyResult = "success"

	// DependencyResultHardFailure indicates that one or more dependencies were not satisfied.
	DependencyResultHardFailure DependencyResult = "hard-failure"

	// DependencyResultSoftFailure indicates that there exists a dependency that could not be satisfied,
	// but the current vertex should still be evaluated.
	DependencyResultSoftFailure DependencyResult = "soft-failure"
)

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

type walkerVertex struct {
	// These should only be set once on initialization and never written again.
	// They are not protected by a lock since they don't need to be since
	// they are write-once.

	// DoneCh is closed when this vertex has completed execution, regardless
	// of success.
	//
	// CancelCh is closed when the vertex should cancel execution. If execution
	// is already complete (DoneCh is closed), this has no effect. Otherwise,
	// execution is cancelled as quickly as possible.
	DoneCh   chan struct{}
	CancelCh chan struct{}

	// Dependency information. Any changes to any of these fields requires
	// holding DepsLock.
	//
	// DepsCh is sent a single value that denotes whether the upstream deps
	// were successful (no errors). Any value sent means that the upstream
	// dependencies are complete. No other values will ever be sent again.
	//
	// DepsUpdateCh is closed when there is a new DepsCh set.
	DepsCh chan DependencyResult

	// Below is not safe to read/write in parallel. This behavior is
	// enforced by changes only happening in Update. Nothing else should
	// ever modify these.
	deps         map[Vertex]chan struct{}
	depsCancelCh chan struct{}
}

// Walk loads the graph and dispatches the concurrent walker. Walk can only be
// called once for a single Walker.
func (w *Walker) Walk(g *AcyclicGraph) tfdiags.Diagnostics {
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
			DoneCh:   make(chan struct{}),
			CancelCh: make(chan struct{}),
			deps:     make(map[Vertex]chan struct{}),
		}

		// Add it to the map and kick off the walk
		w.vertexMap[v] = info
	}

	// Iterate over all edges so we can connect the waiters to their
	// dependencies channels.
	for waiter, deps := range g.edgesFrom {
		for dep := range deps.All() {
			// Get the info for the waiter
			waiterInfo, ok := w.vertexMap[waiter]
			if !ok {
				// Vertex doesn't exist... shouldn't be possible but ignore.
				continue
			}

			// Get the info for the dep
			depInfo, ok := w.vertexMap[dep]
			if !ok {
				// Vertex doesn't exist... shouldn't be possible but ignore.
				continue
			}

			// Add the dependency to our waiter
			waiterInfo.deps[dep] = depInfo.DoneCh
		}
	}

	// kick off a new waiter and notify the vertex of the changes.
	for v := range g.edgesFrom {
		info, ok := w.vertexMap[v]
		if !ok {
			// Vertex doesn't exist... shouldn't be possible but ignore.
			continue
		}

		// Create a new done channel
		doneCh := make(chan DependencyResult, 1)

		// Create the channel we close for cancellation
		cancelCh := make(chan struct{})

		// Build a new deps copy
		deps := make(map[Vertex]<-chan struct{})
		for k, v := range info.deps {
			deps[k] = v
		}

		info.DepsCh = doneCh

		// Cancel the older waiter
		if info.depsCancelCh != nil {
			close(info.depsCancelCh)
		}
		info.depsCancelCh = cancelCh

		// Start the waiter
		go w.waitDeps(v, deps, doneCh, cancelCh)
	}

	// Start all the new vertices. We do this at the end so that all
	// the edge waiters and changes are set up above.
	for v, info := range w.vertexMap {
		go w.walkVertex(v, info)
	}

	// Wait for completion
	w.wait.Wait()

	var diags tfdiags.Diagnostics
	w.diagsLock.Lock()
	for v, vDiags := range w.diagsMap {
		if _, upstream := w.upstreamFailed[v]; upstream {
			// Ignore diagnostics for nodes that had failed upstreams, since
			// the downstream diagnostics are likely to be redundant.
			continue
		}
		diags = diags.Append(vDiags)
	}
	w.diagsLock.Unlock()

	return diags
}

// walkVertex walks a single vertex, waiting for any dependencies before
// executing the callback.
func (w *Walker) walkVertex(v Vertex, info *walkerVertex) {
	// When we're done executing, lower the waitgroup count
	defer w.wait.Done()

	// When we're done, always close our done channel
	defer close(info.DoneCh)

	// Wait for our dependencies. We create a [closed] deps channel so
	// that we can immediately fall through to load our actual DepsCh.
	var depsSuccess DependencyResult

	// if there are no deps we have a nil chan, so we need to initialize
	// something that won't block
	depsCh := make(chan DependencyResult, 1) //make(chan DependencyResult, 1)
	depsCh <- DependencyResultSuccess
	close(depsCh)

	if info.DepsCh != nil {
		depsCh = info.DepsCh
	}

	select {
	case <-info.CancelCh:
		// Cancel
		return

	case depsSuccess = <-depsCh:
		// New deps, reloop
	}

	// If we passed dependencies, we just want to check once more that
	// we're not cancelled, since this can happen just as dependencies pass.
	select {
	case <-info.CancelCh:
		// Cancelled during an update while dependencies completed.
		return
	default:
	}

	// Run our callback or note that our upstream failed
	var diags tfdiags.Diagnostics
	var upstreamFailed bool
	// We go through a three-way boolean logic here to handle the three possible
	// dependency results: success, soft failure, and hard failure.

	// We run the callback if the result is success or soft failure.
	if depsSuccess == DependencyResultSuccess || depsSuccess == DependencyResultSoftFailure {
		diags = w.Callback(v)
	}

	// We note that our upstream failed if the result is hard failure or soft failure.
	if depsSuccess == DependencyResultHardFailure || depsSuccess == DependencyResultSoftFailure {
		log.Printf("[TRACE] dag/walk: upstream of %q errored, so skipping", v.Name())
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
	w.diagsMap[v] = diags
	if w.upstreamFailed == nil {
		w.upstreamFailed = make(map[Vertex]struct{})
	}
	if upstreamFailed {
		w.upstreamFailed[v] = struct{}{}
	}
	w.diagsLock.Unlock()
}

func (w *Walker) waitDeps(
	v Vertex,
	deps map[Vertex]<-chan struct{},
	doneCh chan<- DependencyResult,
	cancelCh <-chan struct{}) {

	// For each dependency given to us, wait for it to complete
	for dep, depCh := range deps {
	DepSatisfied:
		for {
			select {
			case <-depCh:
				// Dependency satisfied!
				break DepSatisfied

			case <-cancelCh:
				// Wait cancelled. Note that we didn't satisfy dependencies
				// so that anything waiting on us also doesn't run.
				doneCh <- DependencyResultHardFailure
				return

			case <-time.After(time.Second * 5):
				log.Printf("[TRACE] dag/walk: vertex %q is waiting for %q",
					v.Name(), dep.Name())
			}
		}
	}

	// Dependencies satisfied! We need to check if any errored
	w.diagsLock.Lock()
	defer w.diagsLock.Unlock()

	var allowUpstreamFailure bool
	for dep := range deps {
		if w.diagsMap[dep].HasErrors() {

			// If the vertex allows upstream failures, we can tolerate this error
			if fv, ok := v.(TolerantVertex); ok && fv.AllowUpstreamFailure(dep) {
				allowUpstreamFailure = true
				continue
			}

			// One of our dependencies failed, so return a hard failure result
			doneCh <- DependencyResultHardFailure
			return
		}
	}

	// If we have an error from a dependency that we can tolerate, return a soft failure result
	// This allows us to treat such vertices specially, while still maintaining the flow
	// of errors to dependencies further down the DAG.
	if allowUpstreamFailure {
		doneCh <- DependencyResultSoftFailure
		return
	}

	// All dependencies satisfied and successful
	doneCh <- DependencyResultSuccess
}
