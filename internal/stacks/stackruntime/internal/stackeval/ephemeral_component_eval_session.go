// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ephemeralComponentEvalSession is a Terraform plan operation that is kept running
// after the modules runtime produces an evaluation scope. This is so the modules runtime
// keeps ownership of the ephemeral resource lifecycle during the operation (renewing them,
// and closing them) for as long as the ephemeral component's result is needed.
//
// The operation is started with [startEphemeralComponentEvalSession], the
// evaluation scope is retrieved once with [ephemeralComponentEvalSession.Wait],
// and [ephemeralComponentEvalSession.Release] lets the operation finish.
type ephemeralComponentEvalSession struct {
	// evalScope receives the evaluation scope from the modules runtime, which sends it at most once.
	evalScope chan *lang.Scope

	// done is closed when the plan operation returns, and diags holds its diagnostics from then on.
	done  chan struct{}
	diags tfdiags.Diagnostics

	// release is closed to let the plan operation complete.
	release     chan struct{}
	releaseOnce sync.Once
}

// startEphemeralComponentEvalSession runs a plan operation in the background using
// the given context and options.
func startEphemeralComponentEvalSession(ctx context.Context, tfCtx *terraform.Context, config *configs.Config, opts *terraform.PlanOpts) *ephemeralComponentEvalSession {
	h := &ephemeralComponentEvalSession{
		evalScope: make(chan *lang.Scope, 1),
		done:      make(chan struct{}),
		release:   make(chan struct{}),
	}
	opts.EphemeralHold = func(scope *lang.Scope) <-chan struct{} {
		h.evalScope <- scope
		return h.release
	}

	// When our given context is cancelled, we want to instruct the modules
	// runtime to stop the running operation. We use a nested context to ensure
	// that we don't leak a goroutine when the parent context isn't cancelled.
	operationCtx, operationCancel := context.WithCancel(ctx)
	go func() {
		<-operationCtx.Done()
		if ctx.Err() == context.Canceled {
			tfCtx.Stop()
		}
	}()

	go func() {
		defer operationCancel()
		_, diags := tfCtx.Plan(config, nil, opts)
		h.end(diags)
	}()
	return h
}

// Wait blocks until either the modules runtime provides the evaluation
// scope or the operation ended.
//
// If held is true then the scope can be used to read the result of the
// operation until [ephemeralComponentEvalSession.Release] is called.
//
// If held is false then the operation already ended without reaching the hold
// because it didn't complete successfully. The returned diagnostics are the
// complete diagnostics of the operation, the scope is nil, and there is
// nothing to release.
func (h *ephemeralComponentEvalSession) Wait() (scope *lang.Scope, held bool, diags tfdiags.Diagnostics) {
	select {
	case scope := <-h.evalScope:
		return scope, true, nil
	case <-h.done:
		// If the operation was stopped while it was held then the scope may
		// have been handed over just before it ended. The caller must still
		// get it, along with the obligation to release.
		select {
		case scope := <-h.evalScope:
			return scope, true, nil
		default:
			return nil, false, h.diags
		}
	}
}

// Release lets the plan operation complete, which closes any ephemeral resources,
// and waits for it to end. It returns the diagnostics of the entire operation,
// which can only be known at this point.
func (h *ephemeralComponentEvalSession) Release() tfdiags.Diagnostics {
	h.releaseOnce.Do(func() { close(h.release) })
	<-h.done
	return h.diags
}

// end records the diagnostics of the finished plan operation and releases
// everything waiting for it. It must be called exactly once.
func (h *ephemeralComponentEvalSession) end(diags tfdiags.Diagnostics) {
	h.diags = diags
	close(h.done)
}
