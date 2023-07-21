package stackeval

import (
	"context"

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

	outp.AnnounceDiagnostics(ctx, tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Warning,
			"Fake planning implementation",
			"This plan contains no changes because this result was built from an early stub of the Terraform Core API for stack planning, which does not have any real logic for planning.",
		),
	})
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
