// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Plan evaluates the given configuration to calculate a desired state,
// updates the given prior state to match the current state of real
// infrastructure, and then compares the desired state with the updated prior
// state to produce a proposed set of changes that should reduce the number
// of differences between the two.
//
// Plan does not return a result directly because it emits results in a
// streaming fashion using channels provided in the given [PlanResponse].
//
// Callers must not modify any values reachable directly or indirectly
// through resp after passing it to this function, aside from the implicit
// modifications to the internal state of channels caused by reading them.
func Plan(ctx context.Context, req *PlanRequest, resp *PlanResponse) {
	var respMu sync.Mutex // must hold this when accessing fields of resp, aside from channel sends
	resp.Applyable = true // we'll reset this to false later if appropriate

	// Whatever return path we take, we must close our channels to allow
	// a caller to see that the operation is complete.
	defer func() {
		close(resp.Diagnostics)
		close(resp.PlannedChanges) // MUST be the last channel to close
	}()

	main := stackeval.NewForPlanning(req.Config, req.PrevState, stackeval.PlanOpts{
		PlanningMode:        req.PlanMode,
		InputVariableValues: req.InputValues,
		ProviderFactories:   req.ProviderFactories,

		ForcePlanTimestamp: req.ForcePlanTimestamp,
	})
	main.AllowLanguageExperiments(req.ExperimentsAllowed)
	main.PlanAll(ctx, stackeval.PlanOutput{
		AnnouncePlannedChange: func(ctx context.Context, change stackplan.PlannedChange) {
			resp.PlannedChanges <- change
		},
		AnnounceDiagnostics: func(ctx context.Context, diags tfdiags.Diagnostics) {
			for _, diag := range diags {
				if diag.Severity() == tfdiags.Error {
					respMu.Lock()
					// NOTE: Applyable can never become true again after this point.
					resp.Applyable = false
					respMu.Unlock()
				}
				resp.Diagnostics <- diag
			}
		},
	})
	cleanupDiags := main.DoCleanup(ctx)
	for _, diag := range cleanupDiags {
		// cleanup diagnostics don't stop a plan from being applyable, because
		// the cleanup process should not affect the content of and validity
		// of the plan. This should only include transient operational errors
		// such as failing to terminate a provider plugin.
		resp.Diagnostics <- diag
	}

	// Before we return we'll emit one more special planned change just to
	// remember in the raw plan sequence whether we considered this plan to be
	// applyable, so we don't need to rely on the caller to remember
	// resp.Applyable separately.
	resp.PlannedChanges <- &stackplan.PlannedChangeApplyable{
		Applyable: resp.Applyable,
	}
}

// PlanRequest represents the inputs to a [Plan] call.
type PlanRequest struct {
	PlanMode plans.Mode

	Config    *stackconfig.Config
	PrevState *stackstate.State

	InputValues       map[stackaddrs.InputVariable]ExternalInputValue
	ProviderFactories map[addrs.Provider]providers.Factory

	// ForcePlanTimestamp, if not nil, will force the plantimestamp function
	// to return the given value instead of whatever real time the plan
	// operation started. This is for testing purposes only.
	ForcePlanTimestamp *time.Time

	ExperimentsAllowed bool
}

// PlanResponse is used by [Plan] to describe the results of planning.
//
// [Plan] produces streaming results throughout its execution, and so it
// communicates with the caller by writing to provided channels during its work
// and then modifying other fields in this structure before returning. Callers
// MUST NOT access any fields of PlanResponse until the PlannedChanges
// channel has been closed to signal the completion of the planning process.
type PlanResponse struct {
	// [Plan] will set this field to true if the plan ran to completion and
	// is valid enough to be applied, or set this to false if not.
	//
	// The initial value of this field is ignored; there's no reason to set
	// it to anything other than the zero value.
	Applyable bool

	// PlannedChanges is the channel that will be sent each individual
	// planned change, in no predictable order, during the planning
	// operation.
	//
	// Callers MUST provide a non-nil channel and read from it from
	// another Goroutine throughout the plan operation, or planning
	// progress will be blocked. Callers that read slowly should provide
	// a buffered channel to reduce the backpressure they exert on the
	// planning process.
	//
	// The plan operation will close this channel before it returns
	// PlannedChanges is guaranteed to be the last channel to close
	// (i.e. after Diagnostics is closed) so callers can use the close
	// signal of this channel alone to mark that the plan process is
	// over, but if Diagnostics is a buffered channel they must take
	// care to deplete its buffer afterwards to avoid losing diagnostics
	// delivered near the end of the planning process.
	PlannedChanges chan<- stackplan.PlannedChange

	// Diagnostics is the channel that will be sent any diagnostics
	// that arise during the planning process, in no particular order.
	//
	// In particular note that there's no guarantee that the diagnostics
	// for planning a particular object will be emitted in close proximity
	// to a PlannedChanges write for that same object. Diagnostics and
	// planned changes are totally decoupled, since diagnostics might be
	// collected up and emitted later as a large batch if the runtime
	// needs to perform aggregate operations such as deduplication on
	// the diagnostics before exposing them.
	//
	// Callers MUST provide a non-nil channel and read from it from
	// another Goroutine throughout the plan operation, or planning
	// progress will be blocked. Callers that read slowly should provide
	// a buffered channel to reduce the backpressure they exert on the
	// planning process.
	//
	// The plan operation will close this channel before it returns, but
	// callers should use the close event of PlannedChanges as the definitive
	// signal that planning is complete.
	Diagnostics chan<- tfdiags.Diagnostic
}

type ExternalInputValue = stackeval.ExternalInputValue
