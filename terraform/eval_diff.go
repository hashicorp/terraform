package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
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

// EvalWriteDiff is an EvalNode implementation that saves a planned change
// for an instance object into the set of global planned changes.
type EvalWriteDiff struct {
	Addr           addrs.ResourceInstance
	DeposedKey     states.DeposedKey
	ProviderSchema **ProviderSchema
	Change         **plans.ResourceInstanceChange
}

// TODO: test
func (n *EvalWriteDiff) Eval(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	changes := ctx.Changes()
	addr := n.Addr.Absolute(ctx.Path())
	if n.Change == nil || *n.Change == nil {
		// Caller sets nil to indicate that we need to remove a change from
		// the set of changes.
		gen := states.CurrentGen
		if n.DeposedKey != states.NotDeposed {
			gen = n.DeposedKey
		}
		changes.RemoveResourceInstanceChange(addr, gen)
		return nil
	}

	providerSchema := *n.ProviderSchema
	change := *n.Change

	if change.Addr.String() != addr.String() || change.DeposedKey != n.DeposedKey {
		// Should never happen, and indicates a bug in the caller.
		panic("inconsistent address and/or deposed key in EvalWriteDiff")
	}

	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Type))
		return diags
	}

	csrc, err := change.Encode(schema.ImpliedType())
	if err != nil {
		diags = diags.Append(fmt.Errorf("failed to encode planned changes for %s: %s", addr, err))
		return diags
	}

	changes.AppendResourceInstanceChange(csrc)
	if n.DeposedKey == states.NotDeposed {
		log.Printf("[TRACE] EvalWriteDiff: recorded %s change for %s", change.Action, addr)
	} else {
		log.Printf("[TRACE] EvalWriteDiff: recorded %s change for %s deposed object %s", change.Action, addr, n.DeposedKey)
	}

	return diags
}
