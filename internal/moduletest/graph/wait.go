// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
)

// operationWaiter waits for an operation within
// a test run execution to complete.
type operationWaiter struct {
	ctx        *terraform.Context
	runningCtx context.Context
	run        *moduletest.Run
	file       *moduletest.File
	created    []*plans.ResourceInstanceChangeSrc
	progress   moduletest.Progress
	start      int64
	identifier string
	finished   bool
	evalCtx    *EvalContext
	renderer   views.Test
	lock       sync.Mutex
}

// NewOperationWaiter creates a new operation waiter.
func NewOperationWaiter(ctx *terraform.Context, evalCtx *EvalContext, n *NodeTestRun,
	progress moduletest.Progress, start int64) *operationWaiter {
	identifier := "validate"
	if n.file != nil {
		identifier = n.file.Name
		if n.run != nil {
			identifier = fmt.Sprintf("%s/%s", identifier, n.run.Name)
		}
	}

	return &operationWaiter{
		ctx:        ctx,
		run:        n.run,
		file:       n.file,
		progress:   progress,
		start:      start,
		identifier: identifier,
		evalCtx:    evalCtx,
		renderer:   evalCtx.Renderer(),
		lock:       sync.Mutex{},
	}
}

// Run executes the given function in a goroutine and waits for it to finish.
// If the function finishes, it returns false. If the function is cancelled or
// interrupted, it returns true.
func (w *operationWaiter) Run(fn func()) bool {
	runningCtx, doneRunning := context.WithCancel(context.Background())
	w.runningCtx = runningCtx

	go func() {
		fn()
		doneRunning()
	}()

	// either the function finishes or a cancel/stop signal is received
	return w.wait()
}

func (w *operationWaiter) wait() bool {
	log.Printf("[TRACE] TestFileRunner: waiting for execution during %s", w.identifier)

	for !w.finished {
		select {
		case <-time.After(2 * time.Second):
			w.updateProgress()
		case <-w.evalCtx.stopContext.Done():
			// Soft cancel - wait for completion or hard cancel
			for !w.finished {
				select {
				case <-time.After(2 * time.Second):
					w.updateProgress()
				case <-w.evalCtx.cancelContext.Done():
					return w.handleCancelled()
				case <-w.runningCtx.Done():
					w.finished = true
				}
			}
		case <-w.evalCtx.cancelContext.Done():
			return w.handleCancelled()
		case <-w.runningCtx.Done():
			w.finished = true
		}
	}

	return false
}

// update refreshes the operationWaiter with the latest terraform context, progress, and any newly created resources.
// This should be called before starting a new Terraform operation.
func (w *operationWaiter) update(ctx *terraform.Context, progress moduletest.Progress, created []*plans.ResourceInstanceChangeSrc) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.ctx = ctx
	w.progress = progress
	w.created = created
}

func (w *operationWaiter) updateProgress() {
	now := time.Now().UTC().UnixMilli()
	w.renderer.Run(w.run, w.file, w.progress, now-w.start)
}

// handleCancelled is called when the test execution is hard cancelled.
func (w *operationWaiter) handleCancelled() bool {
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

	return true
}
