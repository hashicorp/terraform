package terraform

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/objchange"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalReadDataDiff is an EvalNode implementation that executes a data
// resource's ReadDataDiff method to discover what attributes it exports.
type EvalReadDataDiff struct {
	Addr           addrs.ResourceInstance
	Config         *configs.Resource
	ProviderAddr   addrs.AbsProviderConfig
	ProviderSchema **ProviderSchema

	Output            **plans.ResourceInstanceChange
	OutputValue       *cty.Value
	OutputConfigValue *cty.Value
	OutputState       **states.ResourceInstanceObject

	// Set Previous when re-evaluating diff during apply, to ensure that
	// the "Destroy" flag is preserved.
	Previous **plans.ResourceInstanceChange
}

func (n *EvalReadDataDiff) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		return nil, fmt.Errorf("provider schema not available for %s", n.Addr)
	}

	var diags tfdiags.Diagnostics
	var change *plans.ResourceInstanceChange
	var configVal cty.Value

	if n.Previous != nil && *n.Previous != nil && (*n.Previous).Action == plans.Delete {
		// If we're re-diffing for a diff that was already planning to
		// destroy, then we'll just continue with that plan.

		nullVal := cty.NullVal(cty.DynamicPseudoType)
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreDiff(absAddr, states.CurrentGen, nullVal, nullVal)
		})
		if err != nil {
			return nil, err
		}

		change = &plans.ResourceInstanceChange{
			Addr:         absAddr,
			ProviderAddr: n.ProviderAddr,
			Change: plans.Change{
				Action: plans.Delete,
				Before: nullVal,
				After:  nullVal,
			},
		}
	} else {
		config := *n.Config
		providerSchema := *n.ProviderSchema
		schema := providerSchema.DataSources[n.Addr.Resource.Type]
		if schema == nil {
			// Should be caught during validation, so we don't bother with a pretty error here
			return nil, fmt.Errorf("provider does not support data source %q", n.Addr.Resource.Type)
		}

		objTy := schema.ImpliedType()
		priorVal := cty.NullVal(objTy) // for data resources, prior is always null because we start fresh every time

		keyData := EvalDataForInstanceKey(n.Addr.Key)

		var configDiags tfdiags.Diagnostics
		configVal, _, configDiags = ctx.EvaluateBlock(config.Config, schema, nil, keyData)
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return nil, diags.Err()
		}

		proposedNewVal := objchange.ProposedNewObject(schema, priorVal, configVal)

		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreDiff(absAddr, states.CurrentGen, priorVal, proposedNewVal)
		})
		if err != nil {
			return nil, err
		}

		change = &plans.ResourceInstanceChange{
			Addr:         absAddr,
			ProviderAddr: n.ProviderAddr,
			Change: plans.Change{
				Action: plans.Read,
				Before: priorVal,
				After:  proposedNewVal,
			},
		}
	}

	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(absAddr, states.CurrentGen, change.Action, change.Before, change.After)
	})
	if err != nil {
		return nil, err
	}

	if n.Output != nil {
		*n.Output = change
	}
	if n.OutputValue != nil {
		*n.OutputValue = change.After
	}
	if n.OutputConfigValue != nil {
		*n.OutputConfigValue = configVal
	}

	if n.OutputState != nil {
		state := &states.ResourceInstanceObject{
			Value:  change.After,
			Status: states.ObjectReady,
		}
		*n.OutputState = state
	}

	return nil, diags.ErrWithWarnings()
}

// EvalReadDataApply is an EvalNode implementation that executes a data
// resource's ReadDataApply method to read data from the data source.
type EvalReadDataApply struct {
	Addr            addrs.ResourceInstance
	Provider        *providers.Interface
	ProviderAddr    addrs.AbsProviderConfig
	ProviderSchema  **ProviderSchema
	Output          **states.ResourceInstanceObject
	Config          *configs.Resource
	Change          **plans.ResourceInstanceChange
	StateReferences []addrs.Referenceable
}

func (n *EvalReadDataApply) Eval(ctx EvalContext) (interface{}, error) {
	provider := *n.Provider
	change := *n.Change
	providerSchema := *n.ProviderSchema
	absAddr := n.Addr.Absolute(ctx.Path())

	var diags tfdiags.Diagnostics

	// If the diff is for *destroying* this resource then we'll
	// just drop its state and move on, since data resources don't
	// support an actual "destroy" action.
	if change != nil && change.Action == plans.Delete {
		if n.Output != nil {
			*n.Output = nil
		}
		return nil, nil
	}

	// For the purpose of external hooks we present a data apply as a
	// "Refresh" rather than an "Apply" because creating a data source
	// is presented to users/callers as a "read" operation.
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		// We don't have a state yet, so we'll just give the hook an
		// empty one to work with.
		return h.PreRefresh(absAddr, states.CurrentGen, cty.NullVal(cty.DynamicPseudoType))
	})
	if err != nil {
		return nil, err
	}

	resp := provider.ReadDataSource(providers.ReadDataSourceRequest{
		TypeName: n.Addr.Resource.Type,
		Config:   change.After,
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(n.Config.Config))
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	schema := providerSchema.DataSources[n.Addr.Resource.Type]
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider does not support data source %q", n.Addr.Resource.Type)
	}

	newVal := resp.State
	for _, err := range newVal.Type().TestConformance(schema.ImpliedType()) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q planned an invalid value for %s. The result could not be saved.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ProviderAddr.ProviderConfig.Type, tfdiags.FormatErrorPrefixed(err, absAddr.String()),
			),
		))
	}
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(absAddr, states.CurrentGen, change.Before, newVal)
	})
	if err != nil {
		return nil, err
	}

	if n.Output != nil {
		*n.Output = &states.ResourceInstanceObject{
			Value:        newVal,
			Status:       states.ObjectReady,
			Dependencies: n.StateReferences,
		}
	}

	return nil, diags.ErrWithWarnings()
}
