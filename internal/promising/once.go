// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package promising

import (
	"context"
	"sync"
)

// Once is a higher-level wrapper around promises that is similar to
// the Go standard library's sync.Once but executes the one-time function
// in an asynchronous task and makes all subsequent calls block until the
// asynchronous task has completed, after which all calls will return the
// result of the initial call.
type Once[T any] struct {
	get       PromiseGet[T]
	promiseID PromiseID
	mu        sync.Mutex
}

// Do makes an asynchronous call to function f only on the first call to
// this method.
//
// The first and all subsequent calls then block on the completion of that
// asynchronous call and all return its single result.
//
// The context used for the call must belong to an active task. The context
// passed to f will belong to a separate asynchronous task and so f can
// create its own promises and create further asynchronous tasks to deal
// with them, as normal.
//
// The typical way to use Do is to have only a single callsite where f is
// set to a function literal whose behavior and results would be equivalent
// regardless of which instance of its closure happens to be the chosen on
// to actually run. All subsequent calls will ignore f entirely, so it is
// incorrect (and useless) to try to vary the effect of f between calls.
//
// This function will return an error of type [ErrSelfDependent] if two
// different Once instances attempt to mutually depend on one another for
// completion. This means that, unlike standard library sync.Once,
// self-dependence cannot cause a deadlock. (Other non-promise-related
// synchronization between calls can still potentially deadlock, though.)
//
// If f panics then that prevents the internal promise from being resolved,
// and so all calls to Do will return [ErrUnresolved]. However, there is
// no built-in facility to catch and recover from such panics since they occur
// in a separate goroutine from all of the waiters.
func (o *Once[T]) Do(ctx context.Context, f func(ctx context.Context) (T, error)) (T, error) {
	AssertContextInTask(ctx)
	o.mu.Lock()
	if o.get == nil {
		// We seem to be the first call, so we'll get the asynchronous task
		// running and then block on its result.
		resolver, get := NewPromise[T](ctx)
		o.get = get
		o.promiseID = resolver.PromiseID()
		o.mu.Unlock()

		// The responsibility for resolving the promise transfers to the
		// asynchronous task, which makes it valid for this main task to
		// await it without a self-dependency error.
		AsyncTask(
			ctx, resolver,
			func(ctx context.Context, resolver PromiseResolver[T]) {
				v, err := f(ctx)
				resolver.Resolve(ctx, v, err)
			},
		)
	} else {
		o.mu.Unlock()
	}

	// Regardless of whether we launched the async task or not, we'll
	// wait for it to resolve the promise before we return.
	return o.get(ctx)
}

// PromiseID returns the unique identifier for the backing promise of the
// receiver, or [NoPromise] if the once hasn't been started yet.
//
// If PromiseID returns [NoPromise] then that result might be immediately
// invalidated by a concurrent or subsequent call to [Once.Do]. However,
// if PromiseID returns a nonzero promise ID then it's guaranteed to remain
// consistent for the remaining lifetime of the object.
func (o *Once[T]) PromiseID() PromiseID {
	o.mu.Lock()
	ret := o.promiseID
	o.mu.Unlock()
	return ret
}
