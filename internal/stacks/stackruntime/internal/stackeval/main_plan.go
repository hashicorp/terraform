// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"sync/atomic"

	"google.golang.org/protobuf/types/known/anypb"

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

	var prevRunStateRaw map[string]*anypb.Any
	if prevRunState := m.PlanPrevState(); prevRunState != nil {
		prevRunStateRaw = prevRunState.InputRaw()
	}
	outp.AnnouncePlannedChange(ctx, &stackplan.PlannedChangeHeader{
		TerraformVersion: version.SemVer,
	})
	for k, raw := range prevRunStateRaw {
		outp.AnnouncePlannedChange(ctx, &stackplan.PlannedChangePriorStateElement{
			Key: k,
			Raw: raw,
		})
	}

	outp.AnnouncePlannedChange(ctx, &stackplan.PlannedChangePlannedTimestamp{
		PlannedTimestamp: m.PlanTimestamp(),
	})

	// TODO: Announce an extra planned change here if we have any unrecognized
	// raw state or state description keys that we'll need to delete during the
	// apply phase.

	diags, err := promising.MainTask(ctx, func(ctx context.Context) (tfdiags.Diagnostics, error) {
		// The idea here is just to iterate over everything in the configuration,
		// find its corresponding evaluation object, and then ask it to validate
		// itself. We make all of these calls asynchronously so that everything
		// can get started and then downstream calls will block on promise
		// resolution to achieve the correct evaluation order.

		var seenSelfDepDiag atomic.Bool
		var seenAnyErrors atomic.Bool
		reportDiags := func(diags tfdiags.Diagnostics) {
			for _, diag := range diags {
				if diag.Severity() == tfdiags.Error {
					seenAnyErrors.Store(true)
				}
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
		}
		noopComplete := func() tfdiags.Diagnostics {
			// We emit all diagnostics immediately as they arrive, so
			// we never have any accumulated diagnostics to emit at the end.
			return nil
		}

		// First we walk the static objects to give them a chance to check
		// whether they are configured appropriately for planning. This
		// allows us to report static problems only once for an entire
		// configuration object, rather than redundantly reporting for every
		// instance of the object.
		ws, complete := newWalkStateCustomDiags(reportDiags, noopComplete)
		walk := &planWalk{
			state: ws,
			out:   &outp,
		}
		walkStaticObjects(
			ctx, walk, m,
			func(ctx context.Context, walk *walkWithOutput[*PlanOutput], obj StaticEvaler) {
				m.walkPlanObjectChanges(ctx, walk, obj)
			},
		)
		// Note: in practice this "complete" cannot actually return any
		// diagnostics because our custom walkstate hooks above just announce
		// the diagnostics immediately. But "complete" still serves the purpose
		// of blocking until all of the async jobs are complete.
		diags := complete()
		if seenAnyErrors.Load() {
			// If we already found static errors then we'll halt here to have
			// the user correct those first.
			return diags, nil
		}

		// If the static walk completed then we'll now perform a dynamic walk
		// which is where we'll actually produce the plan and where we'll
		// learn about any dynamic errors which affect only specific instances
		// of objects.
		// We'll use a fresh walkState here because we already completed
		// the previous one after the static walk.
		ws, complete = newWalkStateCustomDiags(reportDiags, noopComplete)
		walk.state = ws
		walkDynamicObjects(
			ctx, walk, m,
			PlanPhase,
			func(ctx context.Context, walk *planWalk, obj DynamicEvaler) {
				m.walkPlanObjectChanges(ctx, walk, obj)
			},
		)

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

	// Now, that we've finished walking the graph. We'll announce the
	// provider function results so that they can be used during the apply
	// phase.
	hashes := m.providerFunctionResults.GetHashes()
	if len(hashes) > 0 {
		// Only add this planned change if we actually have any results.
		outp.AnnouncePlannedChange(ctx, &stackplan.PlannedChangeProviderFunctionResults{
			Results: m.providerFunctionResults.GetHashes(),
		})
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
type planWalk = walkWithOutput[*PlanOutput]

// walkPlanObjectChanges deals with the leaf objects that can directly
// contribute changes to the plan, which should each implement [Plannable].
func (m *Main) walkPlanObjectChanges(ctx context.Context, walk *planWalk, obj Plannable) {
	walk.AsyncTask(ctx, func(ctx context.Context) {
		ctx, span := tracer.Start(ctx, obj.tracingName()+" planning")
		defer span.End()

		changes, diags := obj.PlanChanges(ctx)
		for _, change := range changes {
			walk.out.AnnouncePlannedChange(ctx, change)
		}
		if len(diags) != 0 {
			walk.state.AddDiags(diags)
		}
	})
}
