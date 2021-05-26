package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
)

// reducePlan takes a planned resource instance change as might be produced by
// Plan or PlanDestroy and "simplifies" it to a single atomic action to be
// performed by a specific graph node.
//
// Callers must specify whether they are a destroy node or a regular apply node.
// If the result is NoOp then the given change requires no action for the
// specific graph node calling this and so evaluation of the that graph node
// should exit early and take no action.
//
// The returned object may either be identical to the input change or a new
// change object derived from the input. Because of the former case, the caller
// must not mutate the object returned in OutChange.
func reducePlan(addr addrs.ResourceInstance, in *plans.ResourceInstanceChange, destroy bool) *plans.ResourceInstanceChange {
	out := in.Simplify(destroy)
	if out.Action != in.Action {
		if destroy {
			log.Printf("[TRACE] reducePlan: %s change simplified from %s to %s for destroy node", addr, in.Action, out.Action)
		} else {
			log.Printf("[TRACE] reducePlan: %s change simplified from %s to %s for apply node", addr, in.Action, out.Action)
		}
	}
	return out
}
