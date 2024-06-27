// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"time"

	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type PlanOpts struct {
	PlanningMode plans.Mode

	InputVariableValues map[stackaddrs.InputVariable]ExternalInputValue

	ProviderFactories ProviderFactories

	PlanTimestamp time.Time

	DependencyLocks depsfile.Locks
}

// Plannable is implemented by objects that can participate in planning.
type Plannable interface {
	// PlanChanges produces zero or more [stackplan.PlannedChange] objects
	// representing changes needed to converge the current and desired states
	// for the reciever, and zero or more diagnostics that represent any
	// problems encountered while calcuating the changes.
	//
	// The diagnostics returned by PlanChanges must be shallow, which is to
	// say that in particular they _must not_ call the PlanChanges methods
	// of other objects that implement Plannable, and should also think
	// very hard about calling any planning-related methods of other objects,
	// to avoid generating duplicate diagnostics via two different return
	// paths.
	//
	// In general, assume that _all_ objects that implement Plannable will
	// have their Validate methods called at some point during planning, and
	// so it's unnecessary and harmful to for one object to try to handle
	// planning (or plan-time validation) on behalf of some other object.
	PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics)

	// Our general async planning helper relies on this to name its
	// tracing span.
	tracingNamer
}
