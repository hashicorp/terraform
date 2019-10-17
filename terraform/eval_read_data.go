package terraform

import (
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/objchange"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalReadData is an EvalNode implementation that deals with the main part
// of the data resource lifecycle: either actually reading from the data source
// or generating a plan to do so.
type EvalReadData struct {
	Addr           addrs.ResourceInstance
	Config         *configs.Resource
	Provider       *providers.Interface
	ProviderAddr   addrs.AbsProviderConfig
	ProviderSchema **ProviderSchema

	// Planned is set when dealing with data resources that were deferred to
	// the apply walk, to let us see what was planned. If this is set, the
	// evaluation of the config is required to produce a wholly-known
	// configuration which is consistent with the partial object included
	// in this planned change.
	Planned **plans.ResourceInstanceChange

	// ForcePlanRead, if true, overrides the usual behavior of immediately
	// reading from the data source where possible, instead forcing us to
	// _always_ generate a plan. This is used during the plan walk, since we
	// mustn't actually apply anything there. (The resulting state doesn't
	// get persisted)
	ForcePlanRead bool

	// The result from this EvalNode has a few different possibilities
	// depending on the input:
	// - If Planned is nil then we assume we're aiming to _produce_ the plan,
	//   and so the following two outcomes are possible:
	//     - OutputChange.Action is plans.NoOp and OutputState is the complete
	//       result of reading from the data source. This is the easy path.
	//     - OutputChange.Action is plans.Read and OutputState is a planned
	//       object placeholder (states.ObjectPlanned). In this case, the
	//       returned change must be recorded in the overral changeset and
	//       eventually passed to another instance of this struct during the
	//       apply walk.
	// - If Planned is non-nil then we assume we're aiming to complete a
	//   planned read from an earlier plan walk. In this case the only possible
	//   non-error outcome is to set Output.Action (if non-nil) to a plans.NoOp
	//   change and put the complete resulting state in OutputState, ready to
	//   be saved in the overall state and used for expression evaluation.
	OutputChange      **plans.ResourceInstanceChange
	OutputValue       *cty.Value
	OutputConfigValue *cty.Value
	OutputState       **states.ResourceInstanceObject
}

func (n *EvalReadData) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())
	log.Printf("[TRACE] EvalReadData: working on %s", absAddr)

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		return nil, fmt.Errorf("provider schema not available for %s", n.Addr)
	}

	var diags tfdiags.Diagnostics
	var change *plans.ResourceInstanceChange
	var configVal cty.Value

	// TODO: Do we need to handle Delete changes here? EvalReadDataDiff and
	// EvalReadDataApply did, but it seems like we should handle that via a
	// separate mechanism since it boils down to just deleting the object from
	// the state... and we do that on every plan anyway, forcing the data
	// resource to re-read.

	config := *n.Config
	provider := *n.Provider
	providerSchema := *n.ProviderSchema
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider %q does not support data source %q", n.ProviderAddr.ProviderConfig.Type, n.Addr.Resource.Type)
	}

	// We'll always start by evaluating the configuration. What we do after
	// that will depend on the evaluation result along with what other inputs
	// we were given.
	objTy := schema.ImpliedType()
	priorVal := cty.NullVal(objTy) // for data resources, prior is always null because we start fresh every time

	forEach, _ := evaluateResourceForEachExpression(n.Config.ForEach, ctx)
	keyData := EvalDataForInstanceKey(n.Addr.Key, forEach)

	var configDiags tfdiags.Diagnostics
	configVal, _, configDiags = ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags.Err()
	}

	proposedNewVal := objchange.PlannedDataResourceObject(schema, configVal)

	// If our configuration contains any unknown values then we must defer the
	// read to the apply phase by producing a "Read" change for this resource,
	// and a placeholder value for it in the state.
	if n.ForcePlanRead || !configVal.IsWhollyKnown() {
		// If the configuration is still unknown when we're applying a planned
		// change then that indicates a bug in Terraform, since we should have
		// everything resolved by now.
		if n.Planned != nil && *n.Planned != nil {
			return nil, fmt.Errorf(
				"configuration for %s still contains unknown values during apply (this is a bug in Terraform; please report it!)",
				absAddr,
			)
		}
		if n.ForcePlanRead {
			log.Printf("[TRACE] EvalReadData: %s configuration is fully known, but we're forcing a read plan to be created", absAddr)
		} else {
			log.Printf("[TRACE] EvalReadData: %s configuration not fully known yet, so deferring to apply phase", absAddr)
		}

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

		err = ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostDiff(absAddr, states.CurrentGen, change.Action, priorVal, proposedNewVal)
		})
		if err != nil {
			return nil, err
		}

		if n.OutputChange != nil {
			*n.OutputChange = change
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
				Status: states.ObjectPlanned, // because the partial value in the plan must be used for now
			}
			*n.OutputState = state
		}

		return nil, diags.ErrWithWarnings()
	}

	if n.Planned != nil && *n.Planned != nil && (*n.Planned).Action != plans.Read {
		// If any other action gets in here then that's always a bug; this
		// EvalNode only deals with reading.
		return nil, fmt.Errorf(
			"invalid action %s for %s: only Read is supported (this is a bug in Terraform; please report it!)",
			(*n.Planned).Action, absAddr,
		)
	}

	log.Printf("[TRACE] Re-validating config for %s", absAddr)
	validateResp := provider.ValidateDataSourceConfig(
		providers.ValidateDataSourceConfigRequest{
			TypeName: n.Addr.Resource.Type,
			Config:   configVal,
		},
	)
	if validateResp.Diagnostics.HasErrors() {
		return nil, validateResp.Diagnostics.InConfigBody(n.Config.Config).Err()
	}

	// If we get down here then our configuration is complete and we're read
	// to actually call the provider to read the data.
	log.Printf("[TRACE] EvalReadData: %s configuration is complete, so reading from provider", absAddr)

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
		Config:   configVal,
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(n.Config.Config))
	if diags.HasErrors() {
		return nil, diags.Err()
	}
	newVal := resp.State
	if newVal == cty.NilVal {
		// This can happen with incompletely-configured mocks. We'll allow it
		// and treat it as an alias for a properly-typed null value.
		newVal = cty.NullVal(schema.ImpliedType())
	}

	for _, err := range newVal.Type().TestConformance(schema.ImpliedType()) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q produced an invalid value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ProviderAddr.ProviderConfig.Type, tfdiags.FormatErrorPrefixed(err, absAddr.String()),
			),
		))
	}
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	if newVal.IsNull() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced null object",
			fmt.Sprintf(
				"Provider %q produced a null value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ProviderAddr.ProviderConfig.Type, absAddr,
			),
		))
	}
	if !newVal.IsWhollyKnown() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q produced a value for %s that is not wholly known.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ProviderAddr.ProviderConfig.Type, absAddr,
			),
		))

		// We'll still save the object, but we need to eliminate any unknown
		// values first because we can't serialize them in the state file.
		// Note that this may cause set elements to be coalesced if they
		// differed only by having unknown values, but we don't worry about
		// that here because we're saving the value only for inspection
		// purposes; the error we added above will halt the graph walk.
		newVal = cty.UnknownAsNull(newVal)
	}

	// Since we've completed the read, we actually have no change to make, but
	// we'll produce a NoOp one anyway to preserve the usual flow of the
	// plan phase and allow it to produce a complete plan.
	change = &plans.ResourceInstanceChange{
		Addr:         absAddr,
		ProviderAddr: n.ProviderAddr,
		Change: plans.Change{
			Action: plans.NoOp,
			Before: newVal,
			After:  newVal,
		},
	}
	state := &states.ResourceInstanceObject{
		Value:  change.After,
		Status: states.ObjectReady, // because we completed the read from the provider
	}

	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(absAddr, states.CurrentGen, change.Before, newVal)
	})
	if err != nil {
		return nil, err
	}

	if n.OutputChange != nil {
		*n.OutputChange = change
	}
	if n.OutputValue != nil {
		*n.OutputValue = change.After
	}
	if n.OutputConfigValue != nil {
		*n.OutputConfigValue = configVal
	}
	if n.OutputState != nil {
		*n.OutputState = state
	}

	return nil, diags.ErrWithWarnings()
}

// EvalReadDataApply is an EvalNode implementation that executes a data
// resource's ReadDataApply method to read data from the data source.
type EvalReadDataApply struct {
	Addr           addrs.ResourceInstance
	Provider       *providers.Interface
	ProviderAddr   addrs.AbsProviderConfig
	ProviderSchema **ProviderSchema
	Output         **states.ResourceInstanceObject
	Config         *configs.Resource
	Change         **plans.ResourceInstanceChange
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

	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
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
			Value:  newVal,
			Status: states.ObjectReady,
		}
	}

	return nil, diags.ErrWithWarnings()
}
