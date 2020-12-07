package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// Apply deals with the main part of the data resource lifecycle: either
// actually reading from the data source or generating a plan to do so.
func (n *evalReadData) Apply(ctx EvalContext) tfdiags.Diagnostics {
	absAddr := n.Addr.Absolute(ctx.Path())

	var diags tfdiags.Diagnostics

	var planned *plans.ResourceInstanceChange
	if n.Planned != nil {
		planned = *n.Planned
	}

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		diags = diags.Append(fmt.Errorf("provider schema not available for %s", n.Addr))
		return diags
	}

	if planned != nil && planned.Action != plans.Read {
		// If any other action gets in here then that's always a bug; this
		// EvalNode only deals with reading.
		diags = diags.Append(fmt.Errorf(
			"invalid action %s for %s: only Read is supported (this is a bug in Terraform; please report it!)",
			planned.Action, absAddr,
		))
		return diags
	}

	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreApply(absAddr, states.CurrentGen, planned.Action, planned.Before, planned.After)
	}))
	if diags.HasErrors() {
		return diags
	}

	config := *n.Config
	providerSchema := *n.ProviderSchema
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider %q does not support data source %q", n.ProviderAddr.Provider.String(), n.Addr.Resource.Type))
		return diags
	}

	forEach, _ := evaluateForEachExpression(config.ForEach, ctx)
	keyData := EvalDataForInstanceKey(n.Addr.Key, forEach)

	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return diags
	}

	newVal, readDiags := n.readDataSource(ctx, configVal)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return diags
	}

	*n.State = &states.ResourceInstanceObject{
		Value:  newVal,
		Status: states.ObjectReady,
	}

	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostApply(absAddr, states.CurrentGen, newVal, diags.Err())
	}))

	return diags
}
