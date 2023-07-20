package stackruntime

import (
	"context"

	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
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
	resp.Applyable = false // we'll reset this to true later if appropriate

	resp.Diagnostics <- tfdiags.Sourceless(
		tfdiags.Warning,
		"Fake planning implementation",
		"This plan contains no changes because this result was built from an early stub of the Terraform Core API for stack planning, which does not have any real logic for planning.",
	)

	close(resp.Diagnostics)
	close(resp.PlannedChanges) // MUST be the last channel to close
}

// PlanRequest represents the inputs to a [Plan] call.
type PlanRequest struct {
	Config *stackconfig.Config
	// TODO: Prior state

	// TODO: Provider factories and other similar such things
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
