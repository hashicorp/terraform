package dag

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/terraform/tfdiags"
)

// Walker is used to walk every vertex of a graph in parallel.
//
// A vertex will only be walked when the dependencies of that vertex have
// been walked. If two vertices can be walked at the same time, they will be.
//
// Update can be called to update the graph. This can be called even during
// a walk, cahnging vertices/edges mid-walk. This should be done carefully.
// If a vertex is removed but has already been executed, the result of that
// execution (any error) is still returned by Wait. Changing or re-adding
// a vertex that has already executed has no effect. Changing edges of
// a vertex that has already executed has no effect.
//
// Non-parallelism can be enforced by introducing a lock in your callback
// function. However, the goroutine overhead of a walk will remain.
// Walker will create V*2 goroutines (one for each vertex, and dependency
// waiter for each vertex). In general this should be of no concern unless
// there are a huge number of vertices.
//
// The walk is depth first by default. This can be changed with the Reverse
// option.
//
// A single walker is only valid for one graph walk. After the walk is complete
// you must construct a new walker to walk again. State for the walk is never
// deleted in case vertices or edges are changed.
type Walker struct {
	// Callback is what is called for each vertex
	Callback WalkFunc

	// Reverse, if true, causes the source of an edge to depend on a target.
	// When false (default), the target depends on the source.
	Reverse bool

	// changeLock must be held to modify any of the fields below. Only Update
	// should modify these fields. Modifying them outside of Update can cause
	// serious problems.
	changeLock sync.Mutex
	vertices   Set
	edges      Set
	vertexMap  map[Vertex]*walkerVertex

	// wait is done when all vertices have executed. It may become "undone"
	// if new vertices are added.
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
	DepsCh       chan bool
	DepsUpdateCh chan struct{}
	DepsLock     sync.Mutex

	// Below is not safe to read/write in parallel. This behavior is
	// enforced by changes only happening in Update. Nothing else should
	// ever modify these.
	deps         map[Vertex]chan struct{}
	depsCancelCh chan struct{}
}

// errWalkUpstream is used in the errMap of a walk to note that an upstream
// dependency failed so this vertex wasn't run. This is not shown in the final
// user-returned error.
var errWalkUpstream = errors.New("upstream dependency failed")

// Wait waits for the completion of the walk and returns diagnostics describing
// any problems that arose. Update should be called to populate the walk with
// vertices and edges prior to calling this.
//
// Wait will return as soon as all currently known vertices are complete.
// If you plan on calling Update with more vertices in the future, you
// should not call Wait until after this is done.
func (w *Walker) Wait() tfdiags.Diagnostics {
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

// Update updates the currently executing walk with the given graph.
// This will perform a diff of the vertices and edges and update the walker.
// Already completed vertices remain completed (including any errors during
// their execution).
//
// This returns immediately once the walker is updated; it does not wait
// for completion of the walk.
//
// Multiple Updates can be called in parallel. Update can be called at any
// time during a walk.
func (w *Walker) Update(g *AcyclicGraph) {
	log.Print("[TRACE] dag/walk: updating graph")
	var v, e *Set
	if g != nil {
		v, e = g.vertices, g.edges
	}

	// Grab the change lock so no more updates happen but also so that
	// no new vertices are executed during this time since we may be
	// removing them.
	w.changeLock.Lock()
	defer w.changeLock.Unlock()

	// Initialize fields
	if w.vertexMap == nil {
		w.vertexMap = make(map[Vertex]*walkerVertex)
	}

	// Calculate all our sets
	newEdges := e.Difference(&w.edges)
	oldEdges := w.edges.Difference(e)
	newVerts := v.Difference(&w.vertices)
	oldVerts := w.vertices.Difference(v)

	// Add the new vertices
	for _, raw := range newVerts.List() {
		v := raw.(Vertex)

		// Add to the waitgroup so our walk is not done until everything finishes
		w.wait.Add(1)

		// Add to our own set so we know about it already
		log.Printf("[TRACE] dag/walk: added new vertex: %q", VertexName(v))
		w.vertices.Add(raw)

		// Initialize the vertex info
		info := &walkerVertex{
			DoneCh:   make(chan struct{}),
			CancelCh: make(chan struct{}),
			deps:     make(map[Vertex]chan struct{}),
		}

		// Add it to the map and kick off the walk
		w.vertexMap[v] = info
	}

	// Remove the old vertices
	for _, raw := range oldVerts.List() {
		v := raw.(Vertex)

		// Get the vertex info so we can cancel it
		info, ok := w.vertexMap[v]
		if !ok {
			// This vertex for some reason was never in our map. This
			// shouldn't be possible.
			continue
		}

		// Cancel the vertex
		close(info.CancelCh)

		// Delete it out of the map
		delete(w.vertexMap, v)

		log.Printf("[TRACE] dag/walk: removed vertex: %q", VertexName(v))
		w.vertices.Delete(raw)
	}

	// Add the new edges
	var changedDeps Set
	for _, raw := range newEdges.List() {
		edge := raw.(Edge)
		waiter, dep := w.edgeParts(edge)

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

		// Record that the deps changed for this waiter
		changedDeps.Add(waiter)

		log.Printf(
			"[TRACE] dag/walk: added edge: %q waiting on %q",
			VertexName(waiter), VertexName(dep))
		w.edges.Add(raw)
	}

	// Process reoved edges
	for _, raw := range oldEdges.List() {
		edge := raw.(Edge)
		waiter, dep := w.edgeParts(edge)

		// Get the info for the waiter
		waiterInfo, ok := w.vertexMap[waiter]
		if !ok {
			// Vertex doesn't exist... shouldn't be possible but ignore.
			continue
		}

		// Delete the dependency from the waiter
		delete(waiterInfo.deps, dep)

		// Record that the deps changed for this waiter
		changedDeps.Add(waiter)

		log.Printf(
			"[TRACE] dag/walk: removed edge: %q waiting on %q",
			VertexName(waiter), VertexName(dep))
		w.edges.Delete(raw)
	}

	// For each vertex with changed dependencies, we need to kick off
	// a new waiter and notify the vertex of the changes.
	for _, raw := range changedDeps.List() {
		v := raw.(Vertex)
		info, ok := w.vertexMap[v]
		if !ok {
			// Vertex doesn't exist... shouldn't be possible but ignore.
			continue
		}

		// Create a new done channel
		doneCh := make(chan bool, 1)

		// Create the channel we close for cancellation
		cancelCh := make(chan struct{})

		// Build a new deps copy
		deps := make(map[Vertex]<-chan struct{})
		for k, v := range info.deps {
			deps[k] = v
		}

		// Update the update channel
		info.DepsLock.Lock()
		if info.DepsUpdateCh != nil {
			close(info.DepsUpdateCh)
		}
		info.DepsCh = doneCh
		info.DepsUpdateCh = make(chan struct{})
		info.DepsLock.Unlock()

		// Cancel the older waiter
		if info.depsCancelCh != nil {
			close(info.depsCancelCh)
		}
		info.depsCancelCh = cancelCh

		log.Printf(
			"[TRACE] dag/walk: dependencies changed for %q, sending new deps",
			VertexName(v))

		// Start the waiter
		go w.waitDeps(v, deps, doneCh, cancelCh)
	}

	// Start all the new vertices. We do this at the end so that all
	// the edge waiters and changes are setup above.
	for _, raw := range newVerts.List() {
		v := raw.(Vertex)
		go w.walkVertex(v, w.vertexMap[v])
	}
}

// edgeParts returns the waiter and the dependency, in that order.
// The waiter is waiting on the dependency.
func (w *Walker) edgeParts(e Edge) (Vertex, Vertex) {
	if w.Reverse {
		return e.Source(), e.Target()
	}

	return e.Target(), e.Source()
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
	var depsSuccess bool
	var depsUpdateCh chan struct{}
	depsCh := make(chan bool, 1)
	depsCh <- true
	close(depsCh)
	for {
		select {
		case <-info.CancelCh:
			// Cancel
			return

		case depsSuccess = <-depsCh:
			// Deps complete! Mark as nil to trigger completion handling.
			depsCh = nil

		case <-depsUpdateCh:
			// New deps, reloop
		}

		// Check if we have updated dependencies. This can happen if the
		// dependencies were satisfied exactly prior to an Update occurring.
		// In that case, we'd like to take into account new dependencies
		// if possible.
		info.DepsLock.Lock()
		if info.DepsCh != nil {
			depsCh = info.DepsCh
			info.DepsCh = nil
		}
		if info.DepsUpdateCh != nil {
			depsUpdateCh = info.DepsUpdateCh
		}
		info.DepsLock.Unlock()

		// If we still have no deps channel set, then we're done!
		if depsCh == nil {
			break
		}
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
	if depsSuccess {
		log.Printf("[TRACE] dag/walk: walking %q", VertexName(v))
		diags = w.Callback(v)
	} else {
		log.Printf("[TRACE] dag/walk: upstream errored, not walking %q", VertexName(v))
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
	doneCh chan<- bool,
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
				doneCh <- false
				return

			case <-time.After(time.Second * 5):
				log.Printf("[TRACE] dag/walk: vertex %q, waiting for: %q",
					VertexName(v), VertexName(dep))
			}
		}
	}

	// Dependencies satisfied! We need to check if any errored
	w.diagsLock.Lock()
	defer w.diagsLock.Unlock()
	for dep := range deps {
		if w.diagsMap[dep].HasErrors() {
			// One of our dependencies failed, so return false
			doneCh <- false
			return
		}
	}

	// All dependencies satisfied and successful
	doneCh <- true
}
