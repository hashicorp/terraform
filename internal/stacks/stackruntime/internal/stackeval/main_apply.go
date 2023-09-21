package stackeval

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"google.golang.org/protobuf/types/known/anypb"
)

// ApplyPlan internally instantiates a [Main] configured to apply the given
// raw plan, and then visits all of the relevant objects to collect up any
// diagnostics they emit while evaluating in terms of the change results.
func ApplyPlan(ctx context.Context, config *stackconfig.Config, rawPlan []*anypb.Any, opts ApplyOpts, outp ApplyOutput) error {
	plan, err := stackplan.LoadFromProto(rawPlan)
	if err != nil {
		return fmt.Errorf("invalid raw plan: %w", err)
	}
	if !plan.Applyable {
		// We should not get here because a caller should not ask us to try
		// to apply a plan that wasn't marked as applyable, but we'll check
		// it anyway just to be robust in case there's a bug further up
		// the call stack.
		return fmt.Errorf("plan is not applyable")
	}

	// We'll register all of the changes we intend to make up front, so we
	// can error rather than deadlock if something goes wrong and causes
	// us to try to depend on a result that isn't coming.
	results, begin := ChangeExec(ctx, func(ctx context.Context, reg *ChangeExecRegistry[*Main]) {
		for _, elem := range plan.Components.Elems() {
			addr := elem.K
			componentInstPlan := elem.V
			reg.RegisterComponentInstanceChange(
				ctx, addr,
				func(ctx context.Context, main *Main) (*states.State, tfdiags.Diagnostics) {
					ctx, span := tracer.Start(ctx, addr.String()+" apply")
					defer span.End()

					stack := main.Stack(ctx, addr.Stack, ApplyPhase)
					component := stack.Component(ctx, addr.Item.Component)
					insts := component.Instances(ctx, ApplyPhase)
					inst, ok := insts[addr.Item.Key]
					if !ok {
						// If we managed to plan a change for this instance
						// during the plan phase but yet it doesn't exist
						// during the apply phase then that suggests that
						// something upstream has failed in a strange way
						// during apply and so this component's for_each or
						// count argument can't be properly evaluated anymore.
						// This is an unlikely case but we'll tolerate it by
						// returning a placeholder value and expect the cause
						// to be reported by some object when we do the apply
						// checking walk below.
						return nil, nil
					}

					// TODO: We should also turn the prior state into the form
					// the modules runtime expects and pass that in here,
					// instead of an empty prior state.
					modulesRuntimePlan, err := componentInstPlan.ForModulesRuntime(states.NewState())
					if err != nil {
						// Suggests that the state is inconsistent with the
						// plan, which is a bug in whatever provided us with
						// those two artifacts, but we don't know who that
						// caller is (it probably came from a client of the
						// Core RPC API) so we don't include our typical
						// "This is a bug in Terraform" language here.
						var diags tfdiags.Diagnostics
						diags = diags.Append(tfdiags.Sourceless(
							tfdiags.Error,
							"Inconsistent component instance plan",
							fmt.Sprintf("The plan for %s is inconsistent with its prior state: %s.", addr, err),
						))
						return nil, diags
					}

					return inst.ApplyModuleTreePlan(ctx, modulesRuntimePlan)
				},
			)
		}
	})

	main := NewForApplying(config, results, opts)
	begin(ctx, main) // the change tasks registered above become runnable

	// With the planned changes now in progress, we'll visit everything and
	// each object to check itself (producing diagnostics) and announce any
	// changes that were applied to it.
	diags, err := promising.MainTask(ctx, func(ctx context.Context) (tfdiags.Diagnostics, error) {
		ctx, span := tracer.Start(ctx, "apply-time checks")
		defer span.End()

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
		walk := &applyWalk{
			state: ws,
			out:   &outp,
		}

		walkDynamicObjects(
			ctx, walk, main,
			func(ctx context.Context, walk *walkWithOutput[*ApplyOutput], obj DynamicEvaler) {
				main.walkApplyCheckObjectChanges(ctx, walk, obj)
			},
		)

		// Note: in practice this "complete" cannot actually return any
		// diagnostics because our custom walkstate hooks above just announce
		// the diagnostics immediately. But "complete" still serves the purpose
		// of blocking until all of the async jobs are complete.
		diags := complete()

		// By the time we get here all of the scheduled changes should be
		// complete already anyway, since we should have visited them all
		// in walkCheckAppliedChanges, but just to make sure we don't leave
		// anything hanging in the background if walkCheckAppliedChanges is
		// buggy we'll also pause here until the ChangeExec scheduler thinks
		// everything it's supervising is complete.
		results.AwaitCompletion(ctx)

		return diags, nil
	})
	diags = diags.Append(diagnosticsForPromisingTaskError(err, main))
	if len(diags) > 0 {
		outp.AnnounceDiagnostics(ctx, diags)
	}

	return nil
}

type ApplyOutput struct {
	// Called each time we confirm that a planned change has now been applied.
	//
	// Each announced change can have a raw element, an external-facing
	// element, or both. The raw element is opaque to anything outside of
	// Terraform Core, while the external-facing element is never consumed
	// by Terraform Core and is instead for other uses such as presenting
	// changes in the UI.
	//
	// The callback should return relatively quickly to minimize the
	// backpressure applied to the planning process.
	AnnounceAppliedChange func(context.Context, stackstate.AppliedChange)

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

// applyWalk just bundles a [walkState] and an [ApplyOutput] together so we can
// concisely pass them both as a single argument between the all the apply walk
// driver functions below.
type applyWalk = walkWithOutput[*ApplyOutput]

// walkApplyCheckObjectChanges deals with the leaf objects that can directly
// contribute changes and/or diagnostics to the apply result, which should each
// implement [ApplyChecker].
//
// This function is not responsible for actually making the changes; they must
// be scheduled separately or this function will either block forever or
// return strange errors. (See [ApplyPlan] for more about how the apply phase
// deals with changes.)
func (m *Main) walkApplyCheckObjectChanges(ctx context.Context, walk *applyWalk, obj ApplyChecker) {
	walk.AsyncTask(ctx, func(ctx context.Context) {
		ctx, span := tracer.Start(ctx, obj.tracingName()+" apply-time checks")
		defer span.End()

		changes, diags := obj.CheckApply(ctx)
		for _, change := range changes {
			walk.out.AnnounceAppliedChange(ctx, change)
		}
		if len(diags) != 0 {
			walk.out.AnnounceDiagnostics(ctx, diags)
		}
	})
}
