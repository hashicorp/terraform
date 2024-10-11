// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/stacks/stackstate/statekeys"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ApplyPlan internally instantiates a [Main] configured to apply the given
// raw plan, and then visits all of the relevant objects to collect up any
// diagnostics they emit while evaluating in terms of the change results.
//
// If the error result is non-nil then that means the apply process didn't
// even begin, because the given arguments were invalid. If the arguments
// are valid enough to start the apply process then the error will always
// be nil and any problems along the way will be reported as diagnostics
// through the [ApplyOutput] object.
//
// Returns the [Main] object that was used to track state during the process.
// Callers must call [Main.DoCleanup] on that object once they've finished
// with it to avoid leaking non-memory resources such as goroutines and
// provider plugin processes.
func ApplyPlan(ctx context.Context, config *stackconfig.Config, plan *stackplan.Plan, opts ApplyOpts, outp ApplyOutput) (*Main, error) {
	if !plan.Applyable {
		// We should not get here because a caller should not ask us to try
		// to apply a plan that wasn't marked as applyable, but we'll check
		// it anyway just to be robust in case there's a bug further up
		// the call stack.
		return nil, fmt.Errorf("plan is not applyable")
	}

	// We might need to discard some of the keys from the previous run state --
	// either in the raw state or in the state description -- if they are
	// unrecognized keys classified as needing to be discarded when unrecognized.
	discardRawKeys, discardDescKeys, err := stateKeysToDiscard(plan.PrevRunStateRaw, opts.PrevStateDescKeys)
	if err != nil {
		return nil, fmt.Errorf("invalid previous run state: %w", err)
	}

	// --------------------------------------------------------------------
	// NO ERROR RETURNS AFTER THIS POINT!
	// From here on we're actually executing the operation, so any problems
	// must be reported as diagnostics through outp.
	// --------------------------------------------------------------------

	hooks := hooksFromContext(ctx)
	hs, ctx := hookBegin(ctx, hooks.BeginApply, hooks.ContextAttach, struct{}{})
	defer hookMore(ctx, hs, hooks.EndApply, struct{}{})

	// Before doing anything else we'll emit zero or more events to deal
	// with discarding the previous run state data that's no longer needed.
	emitStateKeyDiscardEvents(ctx, discardRawKeys, discardDescKeys, outp)

	log.Printf("[TRACE] stackeval.ApplyPlan starting")
	withDiags, err := promising.MainTask(ctx, func(ctx context.Context) (withDiagnostics[*Main], error) {
		// We'll register all of the changes we intend to make up front, so we
		// can error rather than deadlock if something goes wrong and causes
		// us to try to depend on a result that isn't coming.
		results, begin := ChangeExec(ctx, func(ctx context.Context, reg *ChangeExecRegistry[*Main]) {
			for key, elem := range plan.Components.All() {
				addr := key
				componentInstPlan := elem
				action := componentInstPlan.PlannedAction
				dependencyAddrs := componentInstPlan.Dependencies
				dependentAddrs := componentInstPlan.Dependents

				reg.RegisterComponentInstanceChange(
					ctx, addr,
					func(ctx context.Context, main *Main) (*ComponentInstanceApplyResult, tfdiags.Diagnostics) {
						ctx, span := tracer.Start(ctx, addr.String()+" apply")
						defer span.End()
						log.Printf("[TRACE] stackeval: %s preparing to apply", addr)

						stack := main.Stack(ctx, addr.Stack, ApplyPhase)
						component, removed := stack.ApplyableComponents(ctx, addr.Item.Component)

						// A component change can be sourced from a removed
						// block or a component block. We'll try to find the
						// instance that we need to use to apply these changes.

						var inst ApplyableComponentInstance

						if removed != nil {
							if insts, unknown, _ := removed.Instances(ctx, ApplyPhase); unknown {
								// It might be that either the removed block
								// or component block was deferred but the
								// other one had proper changes. We'll note
								// this in the logs but just skip processing
								// it.
								log.Printf("[TRACE]: %s has planned changes, but was unknown. Check further messages to find out if this was an error.", addr)
							} else {
								i, ok := insts[addr.Item.Key]
								if !ok {
									// Again, this might be okay if the component
									// block was deferred but the removed block had
									// proper changes (or vice versa). We'll note
									// this in the logs but just skip processing it.
									log.Printf("[TRACE]: %s has planned changes, but does not seem to be declared. Check further messages to find out if this was an error.", addr)
								} else {
									inst = i
								}
							}
						}

						if component != nil {
							if insts, unknown := component.Instances(ctx, ApplyPhase); unknown {
								// It might be that either the removed block
								// or component block was deferred but the
								// other one had proper changes. We'll note
								// this in the logs but just skip processing
								// it.
								log.Printf("[TRACE]: %s has planned changes, but was unknown. Check further messages to find out if this was an error.", addr)
							} else {
								if i, ok := insts[addr.Item.Key]; !ok {
									// Again, this might be okay if the component
									// block was deferred but the removed block had
									// proper changes (or vice versa). We'll note
									// this in the logs but just skip processing it.
									log.Printf("[TRACE]: %s has planned changes, but does not seem to be declared. Check further messages to find out if this was an error.", addr)
								} else {
									if inst != nil {
										// Problem! We have both a removed block and
										// a component instance that point to the same
										// address. This should not happen. The plan
										// should have caught this and resulted in an
										// unapplyable plan.
										log.Printf("[ERROR] stackeval: %s has both a component and a removed block that point to the same address", addr)
										span.SetStatus(codes.Error, "both component and removed block present")
										return nil, nil
									}
									inst = i
								}
							}
						}

						if inst == nil {
							// Then we have a problem. We have a component
							// that has planned changes but no instance to
							// apply them to. This should not happen.
							log.Printf("[ERROR] stackeval: %s has planned changes, but no instance to apply them to", addr)
							span.SetStatus(codes.Error, "no instance to apply changes to")
							return nil, nil
						}

						modulesRuntimePlan, err := componentInstPlan.ForModulesRuntime()
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
							log.Printf("[ERROR] stackeval: %s has a plan inconsistent with its prior state: %s", addr, err)
							span.SetStatus(codes.Error, "plan is inconsistent with prior state")
							return nil, diags
						}

						var waitForComponents collections.Set[stackaddrs.AbsComponent]
						var waitForRemoveds collections.Set[stackaddrs.AbsComponent]
						if action == plans.Delete || action == plans.Forget {
							// If the effect of this apply will be to destroy this
							// component instance then we need to wait for all
							// of our dependents to be destroyed first, because
							// we're required to outlive them.
							//
							// (We can assume that all of the dependents are
							// also performing destroy plans, because we'd have
							// rejected the configuration as invalid if a
							// downstream component were referring to a
							// component that's been removed from the config.)
							waitForComponents = dependentAddrs

							// If we're being destroyed, then we're waiting for
							// everything that depended on us anyway.
							waitForRemoveds = dependencyAddrs
						} else {
							// For all other actions, we must wait for our
							// dependencies to finish applying their changes.
							waitForComponents = dependencyAddrs
						}
						if depCount := waitForComponents.Len(); depCount != 0 {
							log.Printf("[TRACE] stackeval: %s waiting for its predecessors (%d) to complete", addr, depCount)
						}
						for waitComponentAddr := range waitForComponents.All() {
							if stack := main.Stack(ctx, waitComponentAddr.Stack, ApplyPhase); stack != nil {
								if component := stack.Component(ctx, waitComponentAddr.Item); component != nil {
									span.AddEvent("awaiting predecessor", trace.WithAttributes(
										attribute.String("component_addr", waitComponentAddr.String()),
									))
									success := component.ApplySuccessful(ctx)
									if !success {
										// If anything we're waiting on does not succeed then we can't proceed without
										// violating the dependency invariants.
										log.Printf("[TRACE] stackeval: %s cannot start because %s changes did not apply completely", addr, waitComponentAddr)
										span.AddEvent("predecessor is incomplete", trace.WithAttributes(
											attribute.String("component_addr", waitComponentAddr.String()),
										))
										span.SetStatus(codes.Error, "predecessors did not completely apply")

										// We'll return a stub result that reports that nothing was changed, since
										// we're not going to run our apply phase at all.
										return inst.PlaceholderApplyResultForSkippedApply(ctx, modulesRuntimePlan), nil
										// Since we're not calling inst.ApplyModuleTreePlan at all in this
										// codepath, the stacks runtime will not emit any progress events for
										// this component instance or any of the objects inside it.
									}
								}
							}
						}
						for waitComponentAddr := range waitForRemoveds.All() {
							if stack := main.Stack(ctx, waitComponentAddr.Stack, ApplyPhase); stack != nil {
								if removed := stack.Removed(ctx, waitComponentAddr.Item); removed != nil {
									span.AddEvent("awaiting predecessor", trace.WithAttributes(
										attribute.String("component_addr", waitComponentAddr.String()),
									))
									success := removed.ApplySuccessful(ctx)
									if !success {
										// If anything we're waiting on does not succeed then we can't proceed without
										// violating the dependency invariants.
										log.Printf("[TRACE] stackeval: %s cannot start because %s changes did not apply completely", addr, waitComponentAddr)
										span.AddEvent("predecessor is incomplete", trace.WithAttributes(
											attribute.String("component_addr", waitComponentAddr.String()),
										))
										span.SetStatus(codes.Error, "predecessors did not completely apply")

										// We'll return a stub result that reports that nothing was changed, since
										// we're not going to run our apply phase at all.
										return inst.PlaceholderApplyResultForSkippedApply(ctx, modulesRuntimePlan), nil
										// Since we're not calling inst.ApplyModuleTreePlan at all in this
										// codepath, the stacks runtime will not emit any progress events for
										// this component instance or any of the objects inside it.
									}
								}
							}
						}
						log.Printf("[TRACE] stackeval: %s now applying", addr)

						ret, diags := inst.ApplyModuleTreePlan(ctx, modulesRuntimePlan)
						if !ret.Complete {
							span.SetStatus(codes.Error, "apply did not complete successfully")
						} else {
							span.SetStatus(codes.Ok, "apply complete")
						}
						return ret, diags
					},
				)
			}
		})

		main := NewForApplying(config, plan, results, opts)
		main.AllowLanguageExperiments(opts.ExperimentsAllowed)
		begin(ctx, main) // the change tasks registered above become runnable

		// With the planned changes now in progress, we'll visit everything and
		// each object to check itself (producing diagnostics) and announce any
		// changes that were applied to it.
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
			ApplyPhase,
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

		return withDiagnostics[*Main]{
			Result:      main,
			Diagnostics: diags,
		}, nil
	})
	diags := withDiags.Diagnostics
	main := withDiags.Result
	diags = diags.Append(diagnosticsForPromisingTaskError(err, main))
	if len(diags) > 0 {
		outp.AnnounceDiagnostics(ctx, diags)
	}
	log.Printf("[TRACE] stackeval.ApplyPlan complete")

	return main, nil
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
func (m *Main) walkApplyCheckObjectChanges(ctx context.Context, walk *applyWalk, obj Applyable) {
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

func stateKeysToDiscard(prevRunState map[string]*anypb.Any, prevDescKeys collections.Set[statekeys.Key]) (discardRaws, discardDescs collections.Set[statekeys.Key], err error) {
	discardRaws = statekeys.NewKeySet()
	discardDescs = statekeys.NewKeySet()

	for rawKey := range prevRunState {
		key, err := statekeys.Parse(rawKey)
		if err != nil {
			// We should not typically get here because if there was an invalid
			// key then we should've caught it during planning.
			return discardRaws, discardDescs, fmt.Errorf("invalid tracking key %q in previous run state: %w", rawKey, err)
		}
		if statekeys.RecognizedType(key) {
			// Nothing to do for a key of a recognized type.
			continue
		}
		if key.KeyType().UnrecognizedKeyHandling() == statekeys.DiscardIfUnrecognized {
			discardRaws.Add(key)
		}
	}

	return discardDescs, discardDescs, nil
}

func emitStateKeyDiscardEvents(ctx context.Context, discardRaws, discardDescs collections.Set[statekeys.Key], outp ApplyOutput) {
	if discardRaws.Len() == 0 && discardDescs.Len() == 0 {
		// Nothing to do, then!
		return
	}
	// If we have at least one key in either set then we can deal with all
	// of them at once in a single "applied change".
	outp.AnnounceAppliedChange(ctx, &stackstate.AppliedChangeDiscardKeys{
		DiscardRawKeys:  discardRaws,
		DiscardDescKeys: discardDescs,
	})
}
