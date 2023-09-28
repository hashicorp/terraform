// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
	wg sync.WaitGroup

	// handleDiags is called for each call to [walkState.AddDiags], with
	// a set of diagnostics produced from that call's arguments.
	//
	// Callback functions can choose between either accumulating diagnostics
	// into an overall set and finally returning it from getFinalDiags, or
	// immediately dispatching the diagnostics to some other location and
	// then returning nothing from the final call to getFinalDiags.
	handleDiags func(tfdiags.Diagnostics)

	// getFinalDiags should return any diagnostics that were previously
	// passed to handleDiags but not yet sent anywhere other than the
	// internal state of a particular walkState object.
	//
	// For handleDiags implementations that immediately send all diagnostics
	// somewhere out-of-hand, this should return nil to avoid those diagnostics
	// getting duplicated by being returned through multiple paths.
	getFinalDiags func() tfdiags.Diagnostics
}

// newWalkState creates a new walkState object that's ready to be passed down
// to child functions that will start asynchronous work.
//
// The second return value is a completion function which should be retained
// by the top-level function that is orchestrating the walk and called only
// once all downstream work has had a chance to start, so that it can block
// for all of those tasks to complete.
//
// This default variant of newWalkState maintains an internal set of
// accumulated diagnostics and eventually returns it from the completion
// callback. Callers that need to handle diagnostics differently -- for example,
// by streaming them to callers via an out-of-band mechanism as they arrive --
// can use newWalkStateCustomDiags to customize the diagnostics handling.
func newWalkState() (ws *walkState, complete func() tfdiags.Diagnostics) {
	var diags syncDiagnostics
	handleDiags := func(moreDiags tfdiags.Diagnostics) {
		diags.Append(moreDiags)
	}
	getFinalDiags := func() tfdiags.Diagnostics {
		return diags.Take()
	}
	return newWalkStateCustomDiags(
		handleDiags,
		getFinalDiags,
	)
}

// newWalkStateCustomDiags is like [newWalkState] except it allows for the
// caller to provide custom callbacks for handling diagnostics.
//
// See the documentation of the fields of the same name in [walkState]
// above for what each of these callbacks represents and how it ought to
// behave.
func newWalkStateCustomDiags(
	handleDiags func(tfdiags.Diagnostics), getFinalDiags func() tfdiags.Diagnostics,
) (ws *walkState, complete func() tfdiags.Diagnostics) {
	ret := &walkState{
		handleDiags:   handleDiags,
		getFinalDiags: getFinalDiags,
	}
	return ret, func() tfdiags.Diagnostics {
		ret.wg.Wait()
		diags := ret.getFinalDiags()
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
// It's safe to make nested calls to AsyncTask (inside another AsyncTask) as
// long as the child call returns, scheduling the child task, before the
// calling task completes. This constraint normally holds automatically when
// the child call is directly inside the parent's callback, but will require
// extra care if a task starts goroutines or non-walkState-supervised async
// tasks that might call this function.
func (ws *walkState) AsyncTask(ctx context.Context, impl func(ctx context.Context)) {
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
	var diags tfdiags.Diagnostics
	diags = diags.Append(new...)
	ws.handleDiags(diags)
}

type walkTaskContextKey struct{}

// walkWithOutput combines a [walkState] with some other object that allows
// emitting output events to a caller, so that walk codepaths can conveniently
// pass these both together as a single argument.
type walkWithOutput[Output any] struct {
	state *walkState
	out   Output
}

func (w *walkWithOutput[Output]) AsyncTask(ctx context.Context, impl func(ctx context.Context)) {
	w.state.AsyncTask(ctx, impl)
}
