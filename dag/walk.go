package dag

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
)

// walker performs a graph walk and supports walk-time changing of vertices
// and edges.
//
// A single walker is only valid for one graph walk. After the walk is complete
// you must construct a new walker to walk again. State for the walk is never
// deleted in case vertices or edges are changed.
type walker struct {
	// Callback is what is called for each vertex
	Callback WalkFunc

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

	// errMap contains the errors recorded so far for execution. Reading
	// and writing should hold errLock.
	errMap  map[Vertex]error
	errLock sync.Mutex
}

type walkerVertex struct {
	// These should only be set once on initialization and never written again
	DoneCh   chan struct{}
	CancelCh chan struct{}

	// Dependency information. Any changes to any of these fields requires
	// holding DepsLock.
	DepsCh       chan struct{}
	DepsUpdateCh chan struct{}
	DepsLock     sync.Mutex

	// Below is not safe to read/write in parallel. This behavior is
	// enforced by changes only happening in Update.
	deps         map[Vertex]chan struct{}
	depsCancelCh chan struct{}
}

// Wait waits for the completion of the walk and returns any errors (
// in the form of a multierror) that occurred. Update should be called
// to populate the walk with vertices and edges prior to calling this.
//
// Wait will return as soon as all currently known vertices are complete.
// If you plan on calling Update with more vertices in the future, you
// should not call Wait until after this is done.
func (w *walker) Wait() error {
	// Wait for completion
	w.wait.Wait()

	// Grab the error lock
	w.errLock.Lock()
	defer w.errLock.Unlock()

	// Build the error
	var result error
	for v, err := range w.errMap {
		result = multierror.Append(result, fmt.Errorf(
			"%s: %s", VertexName(v), err))
	}

	return result
}

// Update updates the currently executing walk with the given vertices
// and edges. It does not block until completion.
//
// Update can be called in parallel to Walk.
func (w *walker) Update(v, e *Set) {
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
		log.Printf("[DEBUG] dag/walk: added new vertex: %q", VertexName(v))
		w.vertices.Add(raw)

		// Initialize the vertex info
		info := &walkerVertex{
			DoneCh:   make(chan struct{}),
			CancelCh: make(chan struct{}),
			DepsCh:   make(chan struct{}),
			deps:     make(map[Vertex]chan struct{}),
		}

		// Close the deps channel immediately so it passes
		close(info.DepsCh)

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

		log.Printf("[DEBUG] dag/walk: removed vertex: %q", VertexName(v))
		w.vertices.Delete(raw)
	}

	// Add the new edges
	var changedDeps Set
	for _, raw := range newEdges.List() {
		edge := raw.(Edge)

		// waiter is the vertex that is "waiting" on this edge
		waiter := edge.Target()

		// dep is the dependency we're waiting on
		dep := edge.Source()

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
			"[DEBUG] dag/walk: added edge: %q waiting on %q",
			VertexName(waiter), VertexName(dep))
		w.edges.Add(raw)
	}

	// Process reoved edges
	for _, raw := range oldEdges.List() {
		edge := raw.(Edge)

		// waiter is the vertex that is "waiting" on this edge
		waiter := edge.Target()

		// dep is the dependency we're waiting on
		dep := edge.Source()

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
			"[DEBUG] dag/walk: removed edge: %q waiting on %q",
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
		doneCh := make(chan struct{})

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

// walkVertex walks a single vertex, waiting for any dependencies before
// executing the callback.
func (w *walker) walkVertex(v Vertex, info *walkerVertex) {
	// When we're done executing, lower the waitgroup count
	defer w.wait.Done()

	// When we're done, always close our done channel
	defer close(info.DoneCh)

	// Wait for our dependencies
	depsCh := info.DepsCh
	for {
		select {
		case <-info.CancelCh:
			// Cancel
			return

		case <-depsCh:
			// Deps complete!
			depsCh = nil

		case <-info.DepsUpdateCh:
			// New deps, reloop
		}

		// Check if we have updated dependencies. This can happen if the
		// dependencies were satisfied exactly prior to an Update occuring.
		// In that case, we'd like to take into account new dependencies
		// if possible.
		info.DepsLock.Lock()
		if info.DepsCh != nil {
			depsCh = info.DepsCh
			info.DepsCh = nil
		}
		info.DepsLock.Unlock()

		// If we still have no deps channel set, then we're done!
		if depsCh == nil {
			break
		}
	}

	// Call our callback
	log.Printf("[DEBUG] dag/walk: walking %q", VertexName(v))
	if err := w.Callback(v); err != nil {
		w.errLock.Lock()
		defer w.errLock.Unlock()

		if w.errMap == nil {
			w.errMap = make(map[Vertex]error)
		}
		w.errMap[v] = err
	}
}

func (w *walker) waitDeps(
	v Vertex,
	deps map[Vertex]<-chan struct{},
	doneCh chan<- struct{},
	cancelCh <-chan struct{}) {
	// Whenever we return, mark ourselves as complete
	defer close(doneCh)

	// For each dependency given to us, wait for it to complete
	for dep, depCh := range deps {
	DepSatisfied:
		for {
			select {
			case <-depCh:
				// Dependency satisfied!
				break DepSatisfied

			case <-cancelCh:
				// Wait cancelled
				return

			case <-time.After(time.Second * 5):
				log.Printf("[DEBUG] vertex %q, waiting for: %q",
					VertexName(v), VertexName(dep))
			}
		}
	}
}
