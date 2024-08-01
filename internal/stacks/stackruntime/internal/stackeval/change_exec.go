// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ChangeExec is a helper for making concurrent changes to a set of objects
// (known ahead of time, statically) during the apply phase.
//
// Most of the work stackeval does is assumed to be just harmless reads that
// can be scheduled at any time as long as implied data dependencies are
// preserved, but during the apply phase we want tighter supervision of the
// changes that actually affect external systems so we can ensure they will
// all run to completion (whether succeeding or failing) and provide the
// data needed for other evaluation work.
//
// The goal of ChangeExec is to encapsulate the promise-based machinery of
// scheduling actions during the apply phase, providing both automatic
// scheduling of the change code and an API for other parts of the system
// to consume the results from the changes, while also helping to deal with
// some slightly-awkward juggling we need to do to make sure that the change
// tasks can interact (in both directions) with the rest of the stackeval
// functionality.
//
// The expected usage pattern is:
//   - Call this function with a setup function that synchronously registers
//     execution tasks for any changes that appear in a plan that was
//     passed into this package's "apply" entry-point.
//   - Instantiate a [Main] object that represents the context for the apply
//     phase, including inside it the [ChangeExecResults] object returned
//     by this function.
//   - Call the function returned as this function's second return value with
//     the [Main] object just constructed, which will then allow all of the
//     registered tasks to begin execution using that object. This MUST be
//     done before the completion of whatever task called [ChangeExec].
//   - Evaluate all other relevant objects to collect up any errors they might
//     return. This process will typically cause indirect calls to the
//     methods of the [ChangeExecResults] object, which will therefore wait
//     until the action has completed and obtain its associated result.
func ChangeExec[Main any](
	ctx context.Context,
	setup func(ctx context.Context, reg *ChangeExecRegistry[Main])) (*ChangeExecResults, func(context.Context, Main),
) {
	// Internally here we're orchestrating a two-phase process: the "setup"
	// phase must synchronously register all of the change tasks that need
	// to be performed, and then the caller gets an opportunity to store the
	// now-frozen results object inside a Main object before starting the
	// "execution" phase, where the registered tasks will all become runnable
	// simultaneously.

	setupComplete, waitSetupComplete := promising.NewPromise[struct{}](ctx)
	beginExec, waitBeginExec := promising.NewPromise[Main](ctx)

	reg := &ChangeExecRegistry[Main]{
		waitBeginExec: waitBeginExec,

		results: ChangeExecResults{
			componentInstances: collections.NewMap[
				stackaddrs.AbsComponentInstance,
				func(ctx context.Context) (withDiagnostics[*ComponentInstanceApplyResult], error),
			](),
		},
	}

	// The asynchronous setup task is responsible for resolving setupComplete.
	promising.AsyncTask(ctx, setupComplete, func(ctx context.Context, setupComplete promising.PromiseResolver[struct{}]) {
		setup(ctx, reg)
		setupComplete.Resolve(ctx, struct{}{}, nil)
	})

	// We'll wait until the async setup callback has completed before we return,
	// so we can assume that all tasks are registered once we pass this point.
	waitSetupComplete(ctx)

	return &reg.results, func(ctx context.Context, m Main) {
		beginExec.Resolve(ctx, m, nil)
	}
}

// ChangeExecRegistry is used by [ChangeExec] setup functions to register
// change tasks that should run once the caller is ready to begin execution.
type ChangeExecRegistry[Main any] struct {
	waitBeginExec func(ctx context.Context) (Main, error)

	// Hold mu for changes to "results" during setup. After setup is
	// complete results becomes read-only and so it's no longer
	// necessary to hold mu when reading from it.
	mu      sync.Mutex
	results ChangeExecResults
}

// RegisterComponentInstanceChange registers a change task for a particular
// component instance, which will presumably apply any planned changes for
// that component instance and then return an object representing its
// finalized output values.
func (r *ChangeExecRegistry[Main]) RegisterComponentInstanceChange(
	ctx context.Context,
	addr stackaddrs.AbsComponentInstance,
	run func(ctx context.Context, main Main) (*ComponentInstanceApplyResult, tfdiags.Diagnostics),
) {
	resultProvider, waitResult := promising.NewPromise[withDiagnostics[*ComponentInstanceApplyResult]](ctx)
	r.mu.Lock()
	if r.results.componentInstances.HasKey(addr) {
		// This is always a bug in the caller.
		panic(fmt.Sprintf("duplicate change task registration for %s", addr))
	}
	r.results.componentInstances.Put(addr, waitResult)
	r.mu.Unlock()

	// The asynchronous execution task is responsible for resolving waitResult
	// through resultProvider.
	promising.AsyncTask(ctx, resultProvider, func(ctx context.Context, resultProvider promising.PromiseResolver[withDiagnostics[*ComponentInstanceApplyResult]]) {
		// We'll hold here until the ChangeExec caller signals that it's
		// time to begin, by providing a Main object to the begin-execution
		// callback that ChangeExec returned.
		main, err := r.waitBeginExec(ctx)
		if err != nil {
			// If we get here then that suggests that there was a self-reference
			// error or other promise-related inconsistency, so we'll just
			// bail out with a placeholder value and the error.
			resultProvider.Resolve(ctx, withDiagnostics[*ComponentInstanceApplyResult]{}, err)
			return
		}

		// Now the registered task can begin running, with access to the Main
		// object that is presumably by now configured to retrieve apply-phase
		// results from our corresponding [ChangeExecResults] object.
		applyResult, diags := run(ctx, main)
		resultProvider.Resolve(ctx, withDiagnostics[*ComponentInstanceApplyResult]{
			Result:      applyResult,
			Diagnostics: diags,
		}, nil)
	})
}

// ChangeExecResults is the API for callers of [ChangeExec] to access the
// results of any change tasks that were registered by the setup callback.
//
// The accessor methods of this type will block until the associated change
// action has completed, and so callers should first allow the ChangeExec
// tasks to begin executing by calling the activation function that was
// returned from [ChangeExec] alongside the [ChangeExecResults] object.
type ChangeExecResults struct {
	componentInstances collections.Map[
		stackaddrs.AbsComponentInstance,
		func(context.Context) (withDiagnostics[*ComponentInstanceApplyResult], error),
	]
}

func (r *ChangeExecResults) ComponentInstanceResult(ctx context.Context, addr stackaddrs.AbsComponentInstance) (*ComponentInstanceApplyResult, tfdiags.Diagnostics, error) {
	if r == nil {
		panic("no results for nil ChangeExecResults")
	}
	getter, ok := r.componentInstances.GetOk(addr)
	if !ok {
		return nil, nil, ErrChangeExecUnregistered{addr}
	}
	// This call will block until the corresponding execution function has
	// completed and resolved this promise.
	valWithDiags, err := getter(ctx)
	return valWithDiags.Result, valWithDiags.Diagnostics, err
}

// AwaitCompletion blocks until all of the scheduled changes have completed.
func (r *ChangeExecResults) AwaitCompletion(ctx context.Context) {
	// We don't have any single signal that everything is complete here,
	// but it's sufficient for us to just visit each of our saved promise
	// getters in turn and read from them.
	for _, cb := range r.componentInstances.All() {
		cb(ctx) // intentionally discards result; we only care that it's complete
	}
}

type ErrChangeExecUnregistered struct {
	Addr fmt.Stringer
}

func (err ErrChangeExecUnregistered) Error() string {
	return fmt.Sprintf("no result for unscheduled change to %s", err.Addr.String())
}
