// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Apply performs the changes described in a previously-generated plan,
// aiming to make the real system converge with the desired state and
// then emit a series of patches that the caller must make to the
// current state to represent what has changed.
//
// Apply does not return a result directly because it emits results in a
// streaming fashion using channels provided in the given [ApplyResponse].
//
// Callers must not modify any values reachable directly or indirectly
// through resp after passing it to this function, aside from the implicit
// modifications to the internal state of channels caused by reading them.
func Apply(ctx context.Context, req *ApplyRequest, resp *ApplyResponse) {
	resp.Complete = false // We'll reset this to true only if we actually succeed

	var seenAnyErrors atomic.Bool
	outp := stackeval.ApplyOutput{
		AnnounceAppliedChange: func(ctx context.Context, change stackstate.AppliedChange) {
			resp.AppliedChanges <- change
		},
		AnnounceDiagnostics: func(ctx context.Context, diags tfdiags.Diagnostics) {
			for _, diag := range diags {
				if diag.Severity() == tfdiags.Error {
					seenAnyErrors.Store(true) // never becomes false again
				}
				resp.Diagnostics <- diag
			}
		},
	}

	// Whatever return path we take, we must close our channels to allow
	// a caller to see that the operation is complete.
	defer func() {
		close(resp.Diagnostics)
		close(resp.AppliedChanges) // MUST be the last channel to close
	}()

	main, err := stackeval.ApplyPlan(
		ctx,
		req.Config,
		req.Plan,
		stackeval.ApplyOpts{
			InputVariableValues: req.InputValues,
			ProviderFactories:   req.ProviderFactories,
			ExperimentsAllowed:  req.ExperimentsAllowed,
			DependencyLocks:     req.DependencyLocks,
		},
		outp,
	)
	if err != nil {
		// An error here means that the apply wasn't even able to _start_,
		// typically because the request itself was invalid. We'll announce
		// that as a diagnostic and then halt, though if we get here then
		// it's most likely a bug in the caller rather than end-user error.
		resp.Diagnostics <- tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid apply request",
			fmt.Sprintf("Cannot begin the apply phase: %s.", err),
		)
		return
	}

	if !seenAnyErrors.Load() {
		resp.Complete = true
	}

	cleanupDiags := main.DoCleanup(ctx)
	for _, diag := range cleanupDiags {
		// cleanup diagnostics don't stop the apply from being "complete",
		// since this should include only transient operational errors such
		// as failing to terminate a provider plugin.
		resp.Diagnostics <- diag
	}

}

// ApplyRequest represents the inputs to an [Apply] call.
type ApplyRequest struct {
	Config *stackconfig.Config
	Plan   *stackplan.Plan

	InputValues       map[stackaddrs.InputVariable]ExternalInputValue
	ProviderFactories map[addrs.Provider]providers.Factory

	ExperimentsAllowed bool
	DependencyLocks    depsfile.Locks
}

// ApplyResponse is used by [Apply] to describe the results of applying.
//
// [Apply] produces streaming results throughout its execution, and so it
// communicates with the caller by writing to provided channels during its work
// and then modifying other fields in this structure before returning. Callers
// MUST NOT access any non-channel fields of ApplyResponse until the
// AppliedChanges channel has been closed to signal the completion of the
// apply process.
type ApplyResponse struct {
	// [Apply] will set this field to true if the apply ran to completion
	// without encountering any errors, or set this to false if not.
	//
	// A caller might react to Complete: true by creating one follow-up plan
	// just to confirm that everything has converged and then, if so, consider
	// all of the configuration versions that contributed to this plan to now
	// be converged. If unsuccessful, none of the contributing configurations
	// are known to be converged and the operator will need to decide whether
	// to immediately try creating a new plan (if they think the error was
	// transient) or push a new configuration update to correct the problem.
	//
	// If this field is false after applying is complete then it's likely that
	// at least some of the planned side-effects already occurred, and so
	// it's important to still handle anything that was written to the
	// AppliedChanges channel to partially update the state with the subset
	// of changes that were completed.
	//
	// The initial value of this field is ignored; there's no reason to set
	// it to anything other than the zero value.
	Complete bool

	// AppliedChanges is the channel that will be sent each individual
	// applied change, in no predictable order, during the apply
	// operation.
	//
	// Callers MUST provide a non-nil channel and read from it from
	// another Goroutine throughout the apply operation, or apply
	// progress will be blocked. Callers that read slowly should provide
	// a buffered channel to reduce the backpressure they exert on the
	// apply process.
	//
	// The apply operation will close this channel before it returns.
	// AppliedChanges is guaranteed to be the last channel to close
	// (i.e. after Diagnostics is closed) so callers can use the close
	// signal of this channel alone to mark that the apply process is
	// over, but if Diagnostics is a buffered channel they must take
	// care to deplete its buffer afterwards to avoid losing diagnostics
	// delivered near the end of the apply process.
	AppliedChanges chan<- stackstate.AppliedChange

	// Diagnostics is the channel that will be sent any diagnostics
	// that arise during the apply process, in no particular order.
	//
	// In particular note that there's no guarantee that the diagnostics
	// for applying changes to a particular object will be emitted in close
	// proximity to an AppliedChanges write for that same object. Diagnostics
	// and applied changes are totally decoupled, since diagnostics might be
	// collected up and emitted later as a large batch if the runtime
	// needs to perform aggregate operations such as deduplication on
	// the diagnostics before exposing them.
	//
	// Callers MUST provide a non-nil channel and read from it from
	// another Goroutine throughout the plan operation, or apply
	// progress will be blocked. Callers that read slowly should provide
	// a buffered channel to reduce the backpressure they exert on the
	// apply process.
	//
	// The apply operation will close this channel before it returns, but
	// callers should use the close event of AppliedChanges as the definitive
	// signal that planning is complete.
	Diagnostics chan<- tfdiags.Diagnostic
}
