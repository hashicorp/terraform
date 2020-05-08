package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// EvalReadDataApply is an EvalNode implementation that deals with the main part
// of the data resource lifecycle: either actually reading from the data source
// or generating a plan to do so.
type EvalReadDataApply struct {
	evalReadData
}

func (n *EvalReadDataApply) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())

	var diags tfdiags.Diagnostics

	var planned *plans.ResourceInstanceChange
	if n.Planned != nil {
		planned = *n.Planned
	}

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		return nil, fmt.Errorf("provider schema not available for %s", n.Addr)
	}

	if planned != nil && !(planned.Action == plans.Read || planned.Action == plans.Update) {
		// If any other action gets in here then that's always a bug; this
		// EvalNode only deals with reading.
		return nil, fmt.Errorf(
			"invalid action %s for %s: only Read or Update is supported (this is a bug in Terraform; please report it!)",
			planned.Action, absAddr,
		)
	}

	if err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreApply(absAddr, states.CurrentGen, planned.Action, planned.Before, planned.After)
	}); err != nil {
		return nil, err
	}

	// we have a change and it is complete, which means we read the data
	// source during plan and only need to store it in state.
	if planned.Action == plans.Update {
		outputState := &states.ResourceInstanceObject{
			Value:  planned.After,
			Status: states.ObjectReady,
		}

		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostApply(absAddr, states.CurrentGen, planned.After, nil)
		})
		if err != nil {
			return nil, err
		}

		if n.OutputChange != nil {
			*n.OutputChange = planned
		}
		if n.State != nil {
			*n.State = outputState
		}
		return nil, diags.ErrWithWarnings()
	}

	newVal, readDiags := n.readDataSource(ctx, cty.NilVal)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return nil, diags.ErrWithWarnings()
	}

	outputState := &states.ResourceInstanceObject{
		Value:  newVal,
		Status: states.ObjectReady,
	}

	if err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostApply(absAddr, states.CurrentGen, newVal, diags.Err())
	}); err != nil {
		return nil, err
	}

	if n.State != nil {
		*n.State = outputState
	}

	return nil, diags.ErrWithWarnings()
}
