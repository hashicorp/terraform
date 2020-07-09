package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// evalReadDataApply is an EvalNode implementation that deals with the main part
// of the data resource lifecycle: either actually reading from the data source
// or generating a plan to do so.
type evalReadDataApply struct {
	evalReadData
}

func (n *evalReadDataApply) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())

	var diags tfdiags.Diagnostics

	var planned *plans.ResourceInstanceChange
	if n.Planned != nil {
		planned = *n.Planned
	}

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		return nil, fmt.Errorf("provider schema not available for %s", n.Addr)
	}

	if planned != nil && planned.Action != plans.Read {
		// If any other action gets in here then that's always a bug; this
		// EvalNode only deals with reading.
		return nil, fmt.Errorf(
			"invalid action %s for %s: only Read is supported (this is a bug in Terraform; please report it!)",
			planned.Action, absAddr,
		)
	}

	if err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreApply(absAddr, states.CurrentGen, planned.Action, planned.Before, planned.After)
	}); err != nil {
		return nil, err
	}

	// We have a change and it is complete, which means we read the data
	// source during plan and only need to store it in state.
	if planned.After.IsWhollyKnown() {
		if err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostApply(absAddr, states.CurrentGen, planned.After, nil)
		}); err != nil {
			diags = diags.Append(err)
		}

		*n.State = &states.ResourceInstanceObject{
			Value:  planned.After,
			Status: states.ObjectReady,
		}
		return nil, diags.ErrWithWarnings()
	}

	config := *n.Config
	providerSchema := *n.ProviderSchema
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider %q does not support data source %q", n.ProviderAddr.Provider.String(), n.Addr.Resource.Type)
	}

	forEach, _ := evaluateForEachExpression(config.ForEach, ctx)
	keyData := EvalDataForInstanceKey(n.Addr.Key, forEach)

	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags.ErrWithWarnings()
	}

	newVal, readDiags := n.readDataSource(ctx, configVal)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return nil, diags.ErrWithWarnings()
	}

	*n.State = &states.ResourceInstanceObject{
		Value:  newVal,
		Status: states.ObjectReady,
	}

	if err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostApply(absAddr, states.CurrentGen, newVal, diags.Err())
	}); err != nil {
		diags = diags.Append(err)
	}

	return nil, diags.ErrWithWarnings()
}
