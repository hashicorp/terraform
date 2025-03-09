// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/states"
)

type ComponentInstanceApplyResult struct {
	// FinalState is the final state snapshot returned by the modules runtime
	// after the apply phase completed.
	FinalState *states.State

	// AffectedResourceInstanceObjects is a set of the addresses of all
	// resource instance objects that might've been affected by any part
	// of this apply process.
	//
	// This includes both objects that had real planned changes and also
	// objects that might have had their state updated by the refresh actions
	// during plan, even though no external actions were taken during the
	// apply phase.
	AffectedResourceInstanceObjects addrs.Set[addrs.AbsResourceInstanceObject]

	// Complete is set to true if the apply ran to completion successfully
	// such that it should be safe to try to apply other component instances
	// that are awaiting the completion of this one.
	//
	// If set to false, some changes may have been made but there may be
	// some changes still pending and so other waiting component instances
	// should not try to apply themselves at all.
	Complete bool
}

// resourceInstanceObjectsAffectedByStackPlan finds an exhaustive set of
// addresses for all resource instance objects that could potentially have their
// state changed while applying the given plan.
//
// Along with the objects targeted by explicit planned changes, this also
// includes objects whose state might just get updated to capture changes
// made outside of Terraform that were detected during the planning phase.
func resourceInstanceObjectsAffectedByStackPlan(plan *stackplan.Component) addrs.Set[addrs.AbsResourceInstanceObject] {
	// For now we conservatively just enumerate everything that exists
	// either before or after the change. This is technically more than
	// we strictly need to return -- it will include objects that have
	// no planned change and whose refresh step changed nothing -- but
	// it's better to over-report than to under-report because under-reporting
	// will cause stale objects to get left in the state.

	ret := addrs.MakeSet[addrs.AbsResourceInstanceObject]()
	if plan.ResourceInstancePlanned.Len() > 0 {
		for _, ch := range plan.ResourceInstancePlanned.Elems {
			ret.Add(ch.Key)
		}
	}
	if plan.ResourceInstancePriorState.Len() > 0 {
		for _, addr := range plan.ResourceInstancePriorState.Elems {
			ret.Add(addr.Key)
		}
	}
	return ret
}
