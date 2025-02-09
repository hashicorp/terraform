// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// RunAndWait runs the given function in a goroutine and waits for it to finish.
// The function is passed a function that can be called to signal that it should
// stop running.
func RunAndWait(fn func(), waiter *testWaiter) (tfdiags.Diagnostics, bool) {
	runningCtx, done := context.WithCancel(context.Background())
	waiter.runningCtx = runningCtx

	go func() {
		fn()
		done()
	}()

	// either the function finishes or a cancel/stop signal is received
	return waiter.wait()

}

type testWaiter struct {
	ctx          *terraform.Context
	runningCtx   context.Context
	run          *moduletest.Run
	file         *moduletest.File
	created      []*plans.ResourceInstanceChangeSrc
	progress     moduletest.Progress
	start        int64
	identifier   string
	finished     bool
	cancelledCtx context.Context
	stoppedCtx   context.Context
	evalCtx      *EvalContext
	renderer     views.Test
}

func NewTestWaiter(ctx *terraform.Context, cancelCtx, stopCtx context.Context, evalCtx *EvalContext, renderer views.Test,
	run *moduletest.Run, file *moduletest.File, created []*plans.ResourceInstanceChangeSrc,
	progress moduletest.Progress, start int64) *testWaiter {
	identifier := "validate"
	if file != nil {
		identifier = file.Name
		if run != nil {
			identifier = fmt.Sprintf("%s/%s", identifier, run.Name)
		}
	}

	return &testWaiter{
		ctx:          ctx,
		run:          run,
		file:         file,
		created:      created,
		progress:     progress,
		start:        start,
		identifier:   identifier,
		cancelledCtx: cancelCtx,
		stoppedCtx:   stopCtx,
		evalCtx:      evalCtx,
		renderer:     renderer,
	}
}

func (w *testWaiter) update(ctx *terraform.Context, progress moduletest.Progress, created []*plans.ResourceInstanceChangeSrc) {
	w.ctx = ctx
	w.progress = progress
	w.created = created
}

func (w *testWaiter) updateProgress() {
	now := time.Now().UTC().UnixMilli()
	w.renderer.Run(w.run, w.file, w.progress, now-w.start)
}

func (w *testWaiter) handleCancelled() (tfdiags.Diagnostics, bool) {
	log.Printf("[DEBUG] TestFileRunner: test execution cancelled during %s", w.identifier)

	states := make(map[*moduletest.Run]*states.State)
	mainKey := moduletest.MainStateIdentifier
	states[nil] = w.evalCtx.GetFileState(mainKey).State
	for key, module := range w.evalCtx.FileStates {
		if key == mainKey {
			continue
		}
		states[module.Run] = module.State
	}
	w.renderer.FatalInterruptSummary(w.run, w.file, states, w.created)

	go func() {
		if w.ctx != nil {
			w.ctx.Stop()
		}
	}()

	for !w.finished {
		select {
		case <-time.After(2 * time.Second):
			w.updateProgress()
		case <-w.runningCtx.Done():
			w.finished = true
		}
	}

	return nil, true
}

func (w *testWaiter) wait() (tfdiags.Diagnostics, bool) {
	log.Printf("[TRACE] TestFileRunner: waiting for execution during %s", w.identifier)

	for !w.finished {
		select {
		case <-time.After(2 * time.Second):
			w.updateProgress()
		case <-w.stoppedCtx.Done():
			w.evalCtx.Stop()
			// Soft cancel - wait for completion or hard cancel
			for !w.finished {
				select {
				case <-time.After(2 * time.Second):
					w.updateProgress()
				case <-w.cancelledCtx.Done():
					return w.handleCancelled()
				case <-w.runningCtx.Done():
					w.finished = true
				}
			}
		case <-w.cancelledCtx.Done():
			return w.handleCancelled()
		case <-w.runningCtx.Done():
			w.finished = true
		}
	}

	return nil, false
}
