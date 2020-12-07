package terraform

import (
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalDiffDestroy is an EvalNode implementation that returns a plain
// destroy diff.
type EvalDiffDestroy struct {
	Addr         addrs.ResourceInstance
	DeposedKey   states.DeposedKey
	State        **states.ResourceInstanceObject
	ProviderAddr addrs.AbsProviderConfig

	Output      **plans.ResourceInstanceChange
	OutputState **states.ResourceInstanceObject
}

// TODO: test
func (n *EvalDiffDestroy) Eval(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	absAddr := n.Addr.Absolute(ctx.Path())
	state := *n.State

	if n.ProviderAddr.Provider.Type == "" {
		if n.DeposedKey == "" {
			panic(fmt.Sprintf("EvalDiffDestroy for %s does not have ProviderAddr set", absAddr))
		} else {
			panic(fmt.Sprintf("EvalDiffDestroy for %s (deposed %s) does not have ProviderAddr set", absAddr, n.DeposedKey))
		}
	}

	// If there is no state or our attributes object is null then we're already
	// destroyed.
	if state == nil || state.Value.IsNull() {
		return nil
	}

	// Call pre-diff hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreDiff(
			absAddr, n.DeposedKey.Generation(),
			state.Value,
			cty.NullVal(cty.DynamicPseudoType),
		)
	}))
	if diags.HasErrors() {
		return diags
	}

	// Change is always the same for a destroy. We don't need the provider's
	// help for this one.
	// TODO: Should we give the provider an opportunity to veto this?
	change := &plans.ResourceInstanceChange{
		Addr:       absAddr,
		DeposedKey: n.DeposedKey,
		Change: plans.Change{
			Action: plans.Delete,
			Before: state.Value,
			After:  cty.NullVal(cty.DynamicPseudoType),
		},
		Private:      state.Private,
		ProviderAddr: n.ProviderAddr,
	}

	// Call post-diff hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(
			absAddr,
			n.DeposedKey.Generation(),
			change.Action,
			change.Before,
			change.After,
		)
	}))
	if diags.HasErrors() {
		return diags
	}

	// Update our output
	*n.Output = change

	if n.OutputState != nil {
		// Record our proposed new state, which is nil because we're destroying.
		*n.OutputState = nil
	}

	return diags
}

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
