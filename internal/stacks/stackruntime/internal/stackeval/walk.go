package stackeval

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// walkState is a helper for codepaths that intend to visit various different
// interdependent objects and evaluate them all concurrently, waiting for any
// dependencies to be resolved and accumulating diagnostics along the way.
//
// Unlike in traditional Terraform Core, there isn't any special inversion of
// control technique to sequence the work, and instead callers are expected
// to use normal control flow in conjunction with the "promising" package's
// async tasks and promises to drive the evaluation forward. Therefore this
// only carries some minimal state that all of the asynchronous tasks need
// to share, with everything else dealt with using -- as much as possible --
// "normal code".
type walkState struct {
	wg    sync.WaitGroup
	diags syncDiagnostics
}

// newWalkState creates a new walkState object that's ready to be passed down
// to child functions that will start asynchronous work.
//
// The second return value is a completion function which should be retained
// by the top-level function that is orchestrating the walk and called only
// once all downstream work has had a chance to start, so that it can block
// for all of those tasks to complete.
func newWalkState() (ws *walkState, complete func() tfdiags.Diagnostics) {
	ret := &walkState{}
	return ret, func() tfdiags.Diagnostics {
		ret.wg.Wait()
		diags := ret.diags.Take()
		diags.Sort()
		return diags
	}
}

// AsyncTask runs the given callback function as an asynchronous task and
// ensures that a future call to the [walkState]'s completion function will
// block until it has returned.
//
// The given callback runs under a [promising.AsyncTask] call and so
// is allowed to interact with promises, but if it passes responsibility for
// a promise to another async task it must block until that promise has
// been resolved, so that the promise resolution cannot outlive the
// supervising walkState.
//
// This AsyncTask method is only for top-level AsyncTasks started while
// preparing for the walk. If the given context descends from some earlier
// call to AsyncTask then this method will panic. Use [promising.AsyncTask]
// directly if you need to create indirect asynchronous tasks (but be sure to
// wait for them to complete before returning from the top-level task!)
func (ws *walkState) AsyncTask(ctx context.Context, impl func(ctx context.Context)) {
	ctx = contextInWalkTask(ctx) // panics if given ctx is descended from another contextInWalkTask
	ws.wg.Add(1)
	promising.AsyncTask(ctx, promising.NoPromises, func(ctx context.Context, none promising.PromiseContainer) {
		impl(ctx)
		ws.wg.Done()
	})
}

// AddDiags converts each of the arguments to zero or more diagnostics and
// appends them to the internal log of diagnostics for the walk.
//
// This is safe to call from multiple concurrent tasks. The full set of
// diagnostics will be returned from the [walkState]'s completion function.
func (ws *walkState) AddDiags(new ...any) {
	ws.diags.Append(new...)
}

func contextInWalkTask(parent context.Context) context.Context {
	if parent.Value(walkTaskContextKey{}) != nil {
		panic("call to walkState.AsyncTask from inside another async task")
	}
	return context.WithValue(parent, walkTaskContextKey{}, walkTaskContextKey{})
}

type walkTaskContextKey struct{}
