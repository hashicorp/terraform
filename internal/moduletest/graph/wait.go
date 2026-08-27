// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
)

// operationWaiter waits for a Terraform operation within
// a test run execution to complete.
type operationWaiter struct {
	ctx        *terraform.Context
	run        *moduletest.Run
	file       *moduletest.File
	created    []*plans.ResourceInstanceChangeSrc
	progress   atomic.Pointer[moduletest.Progress]
	start      int64
	identifier string
	evalCtx    *EvalContext
	renderer   views.Test
}

// NewOperationWaiter creates a new operation waiter.
func NewOperationWaiter(ctx *terraform.Context, evalCtx *EvalContext, file *moduletest.File, run *moduletest.Run,
	progress moduletest.Progress, start int64) *operationWaiter {
	identifier := "validate"
	if file != nil {
		identifier = file.Name
		if run != nil {
			identifier = fmt.Sprintf("%s/%s", identifier, run.Name)
		}
	}

	p := atomic.Pointer[moduletest.Progress]{}
	p.Store(&progress)

	return &operationWaiter{
		ctx:        ctx,
		run:        run,
		file:       file,
		progress:   p,
		start:      start,
		identifier: identifier,
		evalCtx:    evalCtx,
		renderer:   evalCtx.Renderer(),
	}
}

// Run executes the given function in a goroutine and waits for it to finish.
// If the function finishes successfully, it returns false. If the function is cancelled or
// interrupted, it returns true.
func (w *operationWaiter) Run(fn func()) bool {
	runningCtx, doneRunning := context.WithCancel(context.Background())
	go func() {
		fn()
		doneRunning()
	}()

	// either the function finishes or a cancel/stop signal is received
	return w.wait(runningCtx)
}

// wait waits for the operation to finish or be cancelled. It returns true if the operation is cancelled.
func (w *operationWaiter) wait(runningCtx context.Context) (cancelled bool) {
	log.Printf("[TRACE] TestFileRunner: waiting for execution during %s", w.identifier)

	// We wait for the operation to finish or be cancelled.
	for {
		select {
		case <-time.After(2 * time.Second):
			// Update progress every 2 seconds
			w.updateProgress()
		case <-w.evalCtx.stopContext.Done():
			// Soft cancel - wait for completion or hard cancel
			for {
				select {
				case <-time.After(2 * time.Second):
					w.updateProgress()
				case <-w.evalCtx.cancelContext.Done():
					// hard cancel. We can stop now
					w.handleCancelled()
					return true
				case <-runningCtx.Done():
					return false
				}
			}
		case <-w.evalCtx.cancelContext.Done():
			// hard cancel. We can stop now
			w.handleCancelled()
			return true
		case <-runningCtx.Done():
			return false
		}
	}
}

// update refreshes the operationWaiter with the latest terraform context, progress, and any newly created resources.
// This should be called before starting a new Terraform operation.
func (w *operationWaiter) update(ctx *terraform.Context, progress moduletest.Progress, created []*plans.ResourceInstanceChangeSrc) {
	w.ctx = ctx
	w.progress.Store(&progress)
	w.created = created
}

func (w *operationWaiter) updateProgress() {
	now := time.Now().UTC().UnixMilli()
	progress := w.progress.Load()
	w.renderer.Run(w.run, w.file, *progress, now-w.start)
}

// handleCancelled is called when the test execution is hard cancelled.
func (w *operationWaiter) handleCancelled() {
	log.Printf("[DEBUG] TestFileRunner: test execution cancelled during %s", w.identifier)
	states := make(map[string]*states.State)
	for key, module := range w.evalCtx.FileStates {
		states[key] = module.State
	}
	w.renderer.FatalInterruptSummary(w.run, w.file, states, w.created)

	// inform the terraform context to stop
	if w.ctx != nil {
		w.ctx.Stop()
	}
}
