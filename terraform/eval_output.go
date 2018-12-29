package terraform

import (
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
)

// EvalReadOutputState is an EvalNode implementation that reads the state
// for a particular output value from the overall state.
type EvalReadOutputState struct {
	Addr addrs.OutputValue

	// Output, if non-nil, will have the current state for the requested
	// output assigned to its referent.
	Output **states.OutputValue
}

// TODO: test
func (n *EvalReadOutputState) Eval(ctx EvalContext) (interface{}, error) {
	addr := n.Addr.Absolute(ctx.Path())
	state := ctx.State()

	os := state.OutputValue(addr)
	if os != nil {
		if os.Sensitive {
			log.Printf("[TRACE] EvalReadOutputState: %s is sensitive", addr)
		} else {
			log.Printf("[TRACE] EvalReadOutputState: %s has stored value %#v", addr, os.Value)
		}
	} else {
		log.Printf("[TRACE] EvalReadOutputState: %s is not yet set", addr)
	}

	if n.Output != nil {
		*n.Output = os
	}

	return os, nil
}

// EvalPlanOutputChange is an EvalNode implementation that computes the
// required change (if any) for an output value.
type EvalPlanOutputChange struct {
	Addr       addrs.OutputValue
	PriorState **states.OutputValue
	Config     *configs.Output

	// Output, if non-nil, will have the planned change assigned to its
	// referent.
	Output **plans.OutputChange

	// OutputState, if non-nil, will have written to it a synthetic "planned
	// state" for the given output, which describes in as much detail as
	// possible the new state we expect for this output after apply.
	OutputState **states.OutputValue
}

// TODO: test
func (n *EvalPlanOutputChange) Eval(ctx EvalContext) (interface{}, error) {
	addr := n.Addr.Absolute(ctx.Path())
	config := n.Config
	state := *n.PriorState

	log.Printf("[TRACE] EvalPlanOutputChange: %s", addr)

	val, diags := ctx.EvaluateExpr(config.Expr, cty.DynamicPseudoType, nil)
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	change := &plans.OutputChange{
		Addr:      addr,
		Sensitive: config.Sensitive,
	}

	eqV := val.Equals(state.Value)
	if !eqV.IsKnown() || eqV.False() {
		change.After = val
		switch {
		case state == nil:
			change.Action = plans.Create
			change.Before = cty.NullVal(cty.DynamicPseudoType)
		default:
			change.Action = plans.Update
			change.Before = state.Value
			if state.Sensitive {
				change.Sensitive = true // change is sensitive if either Before or After are sensitive
			}
		}
	} else {
		change.Action = plans.NoOp
		change.After = state.Value
	}

	log.Printf("[TRACE] EvalPlanOutputChange: %s has change action %s", addr, change.Action)

	if n.Output != nil {
		*n.Output = change
	}

	if n.OutputState != nil {
		*n.OutputState = &states.OutputValue{
			Value:     change.After,
			Sensitive: config.Sensitive,
		}
	}

	return change, diags.ErrWithWarnings()
}

// EvalPlanOutputDestroy is an EvalNode implementation that produces a
// destroy change for an output value. This should be used only if the
// output has been entirely removed from configuration. Setting an output
// to null should be handled by EvalPlanOutputChange.
type EvalPlanOutputDestroy struct {
	Addr       addrs.OutputValue
	PriorState **states.OutputValue

	// Output, if non-nil, will have the planned change assigned to its
	// referent.
	Output **plans.OutputChange

	// OutputState, if non-nil, will have nil written to its referent, to
	// represent that there will be no state for this output after the
	// change is applied.
	OutputState **states.OutputValue
}

// TODO: test
func (n *EvalPlanOutputDestroy) Eval(ctx EvalContext) (interface{}, error) {
	addr := n.Addr.Absolute(ctx.Path())
	state := *n.PriorState

	change := &plans.OutputChange{
		Addr:      addr,
		Sensitive: state.Sensitive,
		Change: plans.Change{
			Action: plans.Delete,
			Before: state.Value,
			After:  cty.NullVal(cty.DynamicPseudoType),
		},
	}

	log.Printf("[TRACE] EvalPlanOutputDestroy: %s has change action %s", addr, change.Action)

	if n.Output != nil {
		*n.Output = change
	}

	if n.OutputState != nil {
		*n.OutputState = nil
	}

	return change, nil
}

// EvalWriteOutputChange is an EvalNode implementation that appends a given
// planned output change into the current changeset.
type EvalWriteOutputChange struct {
	Change **plans.OutputChange
}

// TODO: test
func (n *EvalWriteOutputChange) Eval(ctx EvalContext) (interface{}, error) {
	change := *n.Change

	src, err := change.Encode()
	if err != nil {
		// Should never happen, since our given change should always be valid
		// having been built previously by EvalPlanOutputChange
		return nil, fmt.Errorf("failed to encode %s change for plan: %s", change.Addr, err)
	}

	ctx.Changes().AppendOutputChange(src)
	return nil, nil
}

// EvalReadOutputChange is an EvalNode implementation that retrieves the
// planned change for the output of the given address from the current
// changeset.
type EvalReadOutputChange struct {
	Addr addrs.OutputValue

	Output **plans.OutputChange
}

// TODO: test
func (n *EvalReadOutputChange) Eval(ctx EvalContext) (interface{}, error) {
	addr := n.Addr.Absolute(ctx.Path())

	src := ctx.Changes().GetOutputChange(addr)
	oc, err := src.Decode()
	if err != nil {
		// Should happen only if someone's been tampering with the plan file
		// manually, so we don't bother with a "pretty" error.
		return nil, fmt.Errorf("failed to decode %s change from plan: %s", addr, err)
	}

	if n.Output != nil {
		*n.Output = oc
	}
	return oc, nil
}

// EvalOutput is an EvalNode implementation that produces an updated value
// for an output.
type EvalOutput struct {
	Addr   addrs.OutputValue
	Config *configs.Output

	// Output will be assigned the new state for the output value.
	Output **states.OutputValue
}

// TODO: test
func (n *EvalOutput) Eval(ctx EvalContext) (interface{}, error) {
	addr := n.Addr.Absolute(ctx.Path())
	config := *n.Config

	val, diags := ctx.EvaluateExpr(config.Expr, cty.DynamicPseudoType, nil)
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	var os *states.OutputValue
	os = &states.OutputValue{
		Value:     val,
		Sensitive: config.Sensitive,
	}
	if os.Sensitive {
		log.Printf("[TRACE] EvalOutput: %s is sensitive", addr)
	} else {
		log.Printf("[TRACE] EvalOutput: %s has new value %#v", addr, os.Value)
	}

	if n.Output != nil {
		*n.Output = os
	}

	return os, diags.ErrWithWarnings()
}

// EvalWriteOutputState is an EvalNode implementation that updates the state
// for a given output.
type EvalWriteOutputState struct {
	Addr  addrs.OutputValue
	State **states.OutputValue
}

// TODO: test
func (n *EvalWriteOutputState) Eval(ctx EvalContext) (interface{}, error) {
	addr := n.Addr.Absolute(ctx.Path())
	os := *n.State

	state := ctx.State()
	if state == nil {
		return nil, nil
	}

	state.SetOutputValue(addr, os.Value, os.Sensitive)
	return nil, nil
}

// EvalDeleteOutput is an EvalNode implementation that deletes an output
// from the state.
type EvalDeleteOutput struct {
	Addr addrs.OutputValue
}

// TODO: test
func (n *EvalDeleteOutput) Eval(ctx EvalContext) (interface{}, error) {
	state := ctx.State()
	if state == nil {
		return nil, nil
	}

	state.RemoveOutputValue(n.Addr.Absolute(ctx.Path()))
	return nil, nil
}
