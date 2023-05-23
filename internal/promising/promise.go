package promising

import (
	"context"
	"sync"
	"sync/atomic"
)

// promise represents a result that will become available at some point
// in the future, delivered by an asynchronous [Task].
type promise struct {
	responsible atomic.Pointer[task]
	result      atomic.Pointer[promiseResult]

	waiting   []chan<- struct{}
	waitingMu sync.Mutex
}

func (p *promise) promiseID() PromiseID {
	return PromiseID{p}
}

type promiseResult struct {
	val any
	err error
}

func getResolvedPromiseResult[T any](result *promiseResult) (T, error) {
	// v might fail this type assertion if it's been set to nil
	// due to its responsible task exiting without resolving it,
	// in which case we'll just return the zero value of T along
	// with the error.
	v, _ := result.val.(T)
	err := result.err
	return v, err
}

// PromiseID is an opaque, comparable unique identifier for a promise, which
// can therefore be used by callers to produce a lookup table of metadata for
// each active promise they are interested in.
//
// The identifier for a promise follows it as the responsibility to resolve it
// transfers beween tasks.
//
// For example, this can be useful for retaining contextual information that
// can help explain which work was implicated in a dependency cycle between
// tasks.
type PromiseID struct {
	promise *promise
}

// NoPromise is the zero value of [PromiseID] and used to represent the absense
// of a promise.
var NoPromise PromiseID

// NewPromise creates a new promise that the calling task is initially
// responsible for and returns both its resolver and its getter.
//
// The given context must be a task context or this function will panic.
//
// The caller should retain the resolver for its own use and pass the getter
// to any other tasks that will consume the result of the promise.
func NewPromise[T any](ctx context.Context) (PromiseResolver[T], PromiseGet[T]) {
	initialResponsible := mustTaskFromContext(ctx)
	p := &promise{}
	p.responsible.Store(initialResponsible)
	initialResponsible.responsible[p] = struct{}{}

	resolver := PromiseResolver[T]{p}
	getter := PromiseGet[T](func(ctx context.Context) (T, error) {
		reqT := mustTaskFromContext(ctx)

		ok := reqT.awaiting.CompareAndSwap(nil, p)
		if !ok {
			// If we get here then the task seems to have forked into two
			// goroutines that are trying to await promises concurrently,
			// which is illegal per the contract for tasks.
			panic("racing promise get")
		}
		defer func() {
			ok := reqT.awaiting.CompareAndSwap(p, nil)
			if !ok {
				panic("racing promise get")
			}
		}()

		// We'll first test whether waiting for this promise is possible
		// without creating a deadlock, by following the awaiting->responsible
		// chain.
		checkP := p
		checkT := p.responsible.Load()
		steps := 1
		for checkT != reqT {
			steps++
			if checkT == nil {
				break
			}
			nextCheckP := checkT.awaiting.Load()
			if nextCheckP == nil {
				break
			}
			if checkP.responsible.Load() != checkT {
				break
			}
			checkP = nextCheckP
			checkT = checkP.responsible.Load()
		}
		if checkT == reqT {
			// We've found a self-dependency, but to report it in a useful
			// way we need to collect up all of the promises, so we'll
			// repeat the above and collect up all of the promises we find
			// along the way this time, instead of just counting them.
			err := make(ErrSelfDependent, 0, steps)
			checkP := p
			checkT := p.responsible.Load()
			err = append(err, checkP.promiseID())
			for checkT != reqT {
				if checkT == nil {
					break
				}
				nextCheckP := checkT.awaiting.Load()
				if nextCheckP == nil {
					break
				}
				if checkP.responsible.Load() != checkT {
					break
				}
				checkP = nextCheckP
				checkT = checkP.responsible.Load()
				err = append(err, checkP.promiseID())
			}
			var zero T
			return zero, err
		}

		// If we get here then it's safe to actually await.
		p.waitingMu.Lock()
		if result := p.result.Load(); result != nil {
			// No need to wait because the result is already available.
			p.waitingMu.Unlock()
			return getResolvedPromiseResult[T](result)
		}

		ch := make(chan struct{})
		p.waiting = append(p.waiting, ch)
		p.waitingMu.Unlock()

		<-ch // channel will be closed once promise is resolved
		if result := p.result.Load(); result != nil {
			return getResolvedPromiseResult[T](result)
		} else {
			// If we get here then there's a bug in resolvePromise below
			panic("promise signaled resolved but has no result")
		}
	})

	return resolver, getter
}

func resolvePromise(p *promise, v any, err error) {
	p.waitingMu.Lock()
	defer p.waitingMu.Unlock()

	respT := p.responsible.Load()
	p.responsible.Store(nil)
	respT.responsible.Remove(p)

	ok := p.result.CompareAndSwap(nil, &promiseResult{
		val: v,
		err: err,
	})
	if !ok {
		panic("promise resolved more than once")
	}

	for _, waitingCh := range p.waiting {
		close(waitingCh)
	}
	p.waiting = nil
}

// PromiseGet is the signature of a promise "getter" function, which blocks
// until a promise is resolved and then returns its result values.
//
// A PromiseGet function may be called only within a task, using a context
// value that descends from that task's context.
//
// If the given context is cancelled or reaches its deadline then the function
// will return the relevant context-related error to describe that situation.
type PromiseGet[T any] func(ctx context.Context) (T, error)
