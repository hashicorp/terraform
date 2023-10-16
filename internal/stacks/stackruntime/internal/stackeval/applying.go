// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/stacks/stackstate/statekeys"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ApplyOpts struct {
	ProviderFactories ProviderFactories

	// PrevStateDescKeys is a set of all of the state description keys currently
	// known by the caller.
	//
	// The apply phase uses this to perform any broad "description maintenence"
	// that might need to happen to contend with changes to the state
	// description representation over time. For example, if any of the given
	// keys are unrecognized and classifed as needing to be discarded when
	// unrecognized then the apply phase will use this to emit the necessary
	// "discard" events to keep the state consistent.
	PrevStateDescKeys collections.Set[statekeys.Key]
}

// ApplyChecker is an interface implemented by types which represent objects
// that can potentially produce diagnostics and object change reports during
// the apply phase.
//
// Unlike [Plannable], ApplyChecker implementations do not actually apply
// changes themselves. Instead, the real changes get driven separately using
// the [ChangeExec] function (see [ApplyPlan]) and then we collect up any
// reports to send to the caller separately using this interface.
type ApplyChecker interface {
	// CheckApply checks the receiver's apply-time result and returns zero
	// or more applied change descriptions and zero or more diagnostics
	// describing any problems that occured for this specific object during
	// the apply phase.
	//
	// CheckApply must not report any diagnostics raised indirectly by
	// evaluating other objects. Those will be collected separately by calling
	// this same method on those other objects.
	CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics)

	// Our general async planning helper relies on this to name its
	// tracing span.
	tracingNamer
}
