package planner

import (
	"context"
	"sync"
)

// agglomerator is a helper for coordinating possibly many requests for the
// same information, with some requests then leading to further requests of
// their own.
//
// Specifically, agglomerator coalesces multiple requests for the same
// information into a single execution, allowing multiple callers to block
// on it and unblocking all of them once the requested information is
// available.
//
// Distinct requests are modelled as comparable types that implement
// dataRequest.
//
// Requests via agglomerator should typically be hidden inside well-named
// accessor methods that return a more specific result type, since agglomerator
// itself only deals in interface{}.
type agglomerator struct {
	planner *planner

	mu      sync.Mutex
	waiters map[interface{}][]chan<- interface{}
	results map[interface{}]interface{}
}

func (a *agglomerator) Request(ctx context.Context, req dataRequest) interface{} {
	a.mu.Lock()

	reqKey := req.requestKey()

	if result, ok := a.results[reqKey]; ok {
		// This request has already completed
		a.mu.Unlock()
		return result
	}

	if _, running := a.waiters[reqKey]; !running {
		// If we have no entry in "waiters" then this request isn't running
		// at all yet, and so we need to start it.
		go func(req dataRequest) {
			result := req.handleDataRequest(a.planner.coalescedCtx, a.planner)
			a.mu.Lock()
			if a.results == nil {
				a.results = make(map[interface{}]interface{})
			}
			a.results[reqKey] = result // future Request calls will now return immediately
			waiters := a.waiters[reqKey]
			delete(a.waiters, reqKey) // won't need these anymore
			a.mu.Unlock()

			// We intentionally unblock the waiters outside of our lock
			// because they are quite likely to go on to do more work that
			// involves blocking on the same agglomerator.
			for _, waitCh := range waiters {
				waitCh <- result
				close(waitCh)
			}
		}(req)
	}

	if a.waiters == nil {
		a.waiters = make(map[interface{}][]chan<- interface{})
	}

	// The above ensures that we'll only get here when the request is already
	// running, and so we can now block on its completion.
	waitCh := make(chan interface{})
	a.waiters[reqKey] = append(a.waiters[reqKey], waitCh)

	a.mu.Unlock()
	//span, _ := opentracing.StartSpanFromContext(ctx, fmt.Sprintf("await %T", req))
	//defer span.Finish()
	return <-waitCh
}

// dataRequest is the interface implemented by request types used with
// agglomerator.
type dataRequest interface {
	// requestKey returns a comparable value which will be equal to the
	// requestKey of another request if and only if the two requests should
	// coalesce together.
	requestKey() interface{}

	// handleDataRequest performs whatever operation this request represents,
	// and returns the resulting value.
	//
	// handleDataRequest may block, including blocking on other dataRequests
	// via the same agglomerator, but should respond to cancellation of the
	// given context by exiting early and returning a "do nothing" placeholder
	// value so that all of the related goroutines can unwind.
	handleDataRequest(ctx context.Context, p *planner) interface{}
}
