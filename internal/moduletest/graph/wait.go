// Copyright (c) HashiCorp, Inc.
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

// operationWaiter waits for an operation within
// a test run execution to complete.
type operationWaiter struct {
	ctx        *terraform.Context
	runningCtx context.Context
	run        *moduletest.Run
	file       *moduletest.File
	created    []*plans.ResourceInstanceChangeSrc
	progress   atomicProgress[moduletest.Progress]
	start      int64
	identifier string
	finished   bool
	evalCtx    *EvalContext
	renderer   views.Test
}

type atomicProgress[T moduletest.Progress] struct {
	internal atomic.Value
}

func (a *atomicProgress[T]) Load() T {
	return a.internal.Load().(T)
}

func (a *atomicProgress[T]) Store(progress T) {
	a.internal.Store(progress)
}

// NewOperationWaiter creates a new operation waiter.
func NewOperationWaiter(ctx *terraform.Context, evalCtx *EvalContext, n *NodeTestRun,
	progress moduletest.Progress, start int64) *operationWaiter {
	identifier := "validate"
	if n.File() != nil {
		identifier = n.File().Name
		if n.run != nil {
			identifier = fmt.Sprintf("%s/%s", identifier, n.run.Name)
		}
	}

	p := atomicProgress[moduletest.Progress]{}
	p.Store(progress)

	return &operationWaiter{
		ctx:        ctx,
		run:        n.run,
		file:       n.File(),
		progress:   p,
		start:      start,
		identifier: identifier,
		evalCtx:    evalCtx,
		renderer:   evalCtx.Renderer(),
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
	w.ctx = ctx
	w.progress.Store(progress)
	w.created = created
}

func (w *operationWaiter) updateProgress() {
	now := time.Now().UTC().UnixMilli()
	progress := w.progress.Load()
	w.renderer.Run(w.run, w.file, progress, now-w.start)
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
