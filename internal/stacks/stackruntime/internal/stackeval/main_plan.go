package stackeval

import (
	"context"
	"sync/atomic"

	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
)

// PlanAll visits all of the objects in the configuration and the prior state,
// performs all of the necessary internal preparation work, and emits a
// series of planned changes and diagnostics through the callbacks in the
// given [PlanOutput] value.
//
// Planning is a streaming operation and so this function does not directly
// return a value. Instead, callers must consume the data gradually passed into
// the provided callbacks and, if necessary, construct their own overall
// data structure by aggregating the results.
func (m *Main) PlanAll(ctx context.Context, outp PlanOutput) {
	hooks := hooksFromContext(ctx)
	hs, ctx := hookBegin(ctx, hooks.BeginPlan, hooks.ContextAttach, struct{}{})
	defer hookMore(ctx, hs, hooks.EndPlan, struct{}{})

	// An important design goal here is that only our main walk code in this
	// file interacts directly with the async PlanOutput API, with it calling
	// into "normal-shaped" functions elsewhere that just run to completion
	// and provide their results as return values.
	//
	// The purpose of the logic in this file is to provide that abstraction to
	// the rest of the code so that the async streaming behavior does not
	// dominate the overall design of package stackeval.

	outp.AnnouncePlannedChange(ctx, &stackplan.PlannedChangeHeader{
		TerraformVersion: version.SemVer,
	})

	diags, err := promising.MainTask(ctx, func(ctx context.Context) (tfdiags.Diagnostics, error) {
		// The idea here is just to iterate over everything in the configuration,
		// find its corresponding evaluation object, and then ask it to validate
		// itself. We make all of these calls asynchronously so that everything
		// can get started and then downstream calls will block on promise
		// resolution to achieve the correct evaluation order.

		var seenSelfDepDiag atomic.Bool
		ws, complete := newWalkStateCustomDiags(
			func(diags tfdiags.Diagnostics) {
				for _, diag := range diags {
					if diagIsPromiseSelfReference(diag) {
						// We'll discard all but the first promise-self-reference
						// diagnostic we see; these tend to get duplicated
						// because they emerge from all codepaths participating
						// in the self-reference at once.
						if !seenSelfDepDiag.CompareAndSwap(false, true) {
							continue
						}
					}
					outp.AnnounceDiagnostics(ctx, tfdiags.Diagnostics{diag})
				}
			},
			func() tfdiags.Diagnostics {
				// We emit all diagnostics immediately as they arrive, so
				// we never have any accumulated diagnostics to emit at the end.
				return nil
			},
		)
		walk := &planWalk{
			state: ws,
			out:   &outp,
		}

		// walkPlanStackChanges, and all of the downstream functions it calls,
		// must take care to ensure that there's always at least one
		// planWalk-tracked async task running until the entire process is
		// complete. If one task launches another then the child task call
		// must come before the caller's implementation function returns.
		m.walkPlanChanges(ctx, walk, m.MainStack(ctx))

		// Note: in practice this "complete" cannot actually return any
		// diagnostics because our custom walkstate hooks above just announce
		// the diagnostics immediately. But "complete" still serves the purpose
		// of blocking until all of the async jobs are complete.
		return complete(), nil
	})
	diags = diags.Append(diagnosticsForPromisingTaskError(err, m))
	if len(diags) > 0 {
		outp.AnnounceDiagnostics(ctx, diags)
	}

	// The caller (in stackruntime) is responsible for generating the final
	// stackplan.PlannedChangeApplyable message, just in case it detects
	// problems of its own before finally returning.
}

type PlanOutput struct {
	// Called each time we find a new change to announce as part of the
	// overall plan.
	//
	// Each announced change can have a raw element, an external-facing
	// element, or both. The raw element is opaque to anything outside of
	// Terraform Core, while the external-facing element is never consumed
	// by Terraform Core and is instead for other uses such as presenting
	// changes in the UI.
	//
	// The callback should return relatively quickly to minimize the
	// backpressure applied to the planning process.
	AnnouncePlannedChange func(context.Context, stackplan.PlannedChange)

	// Called each time we encounter some diagnostics. These are asynchronous
	// from planned changes because the evaluator will sometimes need to
	// aggregate together some diagnostics and post-process the set before
	// announcing them. Callers should not try to correlate diagnostics
	// with planned changes by announcement-time-proximity.
	//
	// The callback should return relatively quickly to minimize the
	// backpressure applied to the planning process.
	AnnounceDiagnostics func(context.Context, tfdiags.Diagnostics)
}

// planWalk just bundles a [walkState] and a [PlanOutput] together so we can
// concisely pass them both as a single argument between the all the plan walk
// driver functions below.
type planWalk struct {
	state *walkState
	out   *PlanOutput
}

func (w *planWalk) AsyncTask(ctx context.Context, impl func(ctx context.Context)) {
	w.state.AsyncTask(ctx, impl)
}

func (m *Main) walkPlanChanges(ctx context.Context, walk *planWalk, stack *Stack) {
	// We'll get the expansion of any child stack calls going first, so that
	// we can explore downstream stacks concurrently with this one. Each
	// stack call can represent zero or more child stacks that we'll analyze
	// by recursive calls to this function.
	for _, call := range stack.EmbeddedStackCalls(ctx) {
		// We need to perform the whole expansion in an overall async task
		// because it involves evaluating for_each expressions, and one
		// stack call's for_each might depend on the results of another.
		walk.AsyncTask(ctx, func(ctx context.Context) {
			insts := call.Instances(ctx, PlanPhase)
			for _, inst := range insts {
				// We'll visit both the call instance itself and the stack
				// instance it implies concurrently because output values
				// inside one stack can contribute to the per-instance
				// arguments of another stack.
				m.walkPlanObjectChanges(ctx, walk, inst)

				childStack := inst.CalledStack(ctx)
				m.walkPlanChanges(ctx, walk, childStack)
			}
		})
	}

	// We also need to plan all of the other declarations in the current stack.

	for _, obj := range stack.InputVariables(ctx) {
		m.walkPlanObjectChanges(ctx, walk, obj)
	}

	for _, obj := range stack.OutputValues(ctx) {
		m.walkPlanObjectChanges(ctx, walk, obj)
	}

	// We'll also finally plan the stack itself, which will deal with anything
	// that relates to the stack as a whole rather than to the objects declared
	// inside.
	m.walkPlanObjectChanges(ctx, walk, stack)
}

// walkPlanObjectChanges deals with the leaf objects that can directly
// contribute changes to the plan, which should each implement [Plannable].
func (m *Main) walkPlanObjectChanges(ctx context.Context, walk *planWalk, obj Plannable) {
	walk.AsyncTask(ctx, func(ctx context.Context) {
		changes, diags := obj.PlanChanges(ctx)
		for _, change := range changes {
			walk.out.AnnouncePlannedChange(ctx, change)
		}
		if len(diags) != 0 {
			walk.out.AnnounceDiagnostics(ctx, diags)
		}
	})
}
