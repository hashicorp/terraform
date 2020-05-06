package terraform

import (
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
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
	ProviderMetas  map[addrs.Provider]*configs.ProviderMeta
	ProviderSchema **ProviderSchema

	// Planned is set when dealing with data resources that were deferred to
	// the apply walk, to let us see what was planned. If this is set, the
	// evaluation of the config is required to produce a wholly-known
	// configuration which is consistent with the partial object included
	// in this planned change.
	Planned **plans.ResourceInstanceChange

	// State is the current state for the data source, and is updated once the
	// new state has need read.
	// While data source are read-only, we need to start with the prior state
	// to determine if we have a change or not.  If we needed to read a new
	// value, but it still matches the previous state, then we can record a
	// NoNop change. If the states don't match then we record a Read change so
	// that the new value is applied to the state.
	State **states.ResourceInstanceObject

	// The result from this EvalNode has a few different possibilities
	// depending on the input:
	// - If Planned is nil then we assume we're aiming to either read the
	//   resource or produce a plan, and so the following two outcomes are
	//   possible:
	//     - OutputChange.Action is plans.NoOp and the
	//       result of reading from the data source is stored in state. This is
	//       the easy path, and only happens during refresh.
	//     - OutputChange.Action is plans.Read and State is a planned
	//       object placeholder (states.ObjectPlanned). In this case, the
	//       returned change must be recorded in the overall changeset and this
	//       resource will be read during apply.
	//     - OutputChange.Action is plans.Update, in which case the change
	//       contains the complete state of this resource, and only needs to be
	//       stored into the final state during apply.
	// - If Planned is non-nil then we assume we're aiming to complete a
	//   planned read from an earlier plan walk, from one of the options above.
	OutputChange **plans.ResourceInstanceChange

	// dependsOn stores the list of transitive resource addresses that any
	// configuration depends_on references may resolve to. This is used to
	// determine if there are any changes that will force this data sources to
	// be deferred to apply.
	dependsOn []addrs.ConfigResource

	// refresh indicates this is being called from a refresh node, and we can't
	// resolve any depends_on dependencies.
	refresh bool
}

func (n *EvalReadData) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())

	var diags tfdiags.Diagnostics
	var configVal cty.Value

	var planned *plans.ResourceInstanceChange
	if n.Planned != nil {
		planned = *n.Planned
	}

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		return nil, fmt.Errorf("provider schema not available for %s", n.Addr)
	}

	config := *n.Config
	provider := *n.Provider
	providerSchema := *n.ProviderSchema
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider %q does not support data source %q", n.ProviderAddr.Provider.String(), n.Addr.Resource.Type)
	}

	objTy := schema.ImpliedType()
	priorVal := cty.NullVal(objTy)
	if n.State != nil && *n.State != nil {
		priorVal = (*n.State).Value
	}

	forEach, _ := evaluateForEachExpression(config.ForEach, ctx)
	keyData := EvalDataForInstanceKey(n.Addr.Key, forEach)

	var configDiags tfdiags.Diagnostics
	configVal, _, configDiags = ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags.Err()
	}

	metaConfigVal, metaDiags := n.providerMetas(ctx)
	diags = diags.Append(metaDiags)
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	configKnown := configVal.IsWhollyKnown()
	// If our configuration contains any unknown values, or we depend on any
	// unknown values then we must defer the read to the apply phase by
	// producing a "Read" change for this resource, and a placeholder value for
	// it in the state.
	if n.forcePlanRead(ctx) || !configKnown {
		if configKnown {
			log.Printf("[TRACE] EvalReadData: %s configuration is fully known, but we're forcing a read plan to be created", absAddr)
		} else {
			log.Printf("[TRACE] EvalReadData: %s configuration not fully known yet, so deferring to apply phase", absAddr)
		}

		proposedNewVal := objchange.PlannedDataResourceObject(schema, configVal)

		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreDiff(absAddr, states.CurrentGen, priorVal, proposedNewVal)
		})
		if err != nil {
			return nil, err
		}

		change := &plans.ResourceInstanceChange{
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
		if n.State != nil {
			*n.State = &states.ResourceInstanceObject{
				Value:  cty.NullVal(objTy),
				Status: states.ObjectPlanned,
			}
		}

		return nil, diags.ErrWithWarnings()
	}

	if planned != nil && !(planned.Action == plans.Read || planned.Action == plans.Update) {
		// If any other action gets in here then that's always a bug; this
		// EvalNode only deals with reading.
		return nil, fmt.Errorf(
			"invalid action %s for %s: only Read or Update is supported (this is a bug in Terraform; please report it!)",
			planned.Action, absAddr,
		)
	}

	// we have a change and it is complete, which means we read the data
	// source during plan and only need to store it in state.
	if planned != nil && planned.Action == plans.Update {
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

	var change *plans.ResourceInstanceChange

	log.Printf("[TRACE] Re-validating config for %s", absAddr)
	validateResp := provider.ValidateDataSourceConfig(
		providers.ValidateDataSourceConfigRequest{
			TypeName: n.Addr.Resource.Type,
			Config:   configVal,
		},
	)
	if validateResp.Diagnostics.HasErrors() {
		return nil, validateResp.Diagnostics.InConfigBody(config.Config).Err()
	}

	// If we get down here then our configuration is complete and we're read
	// to actually call the provider to read the data.
	log.Printf("[TRACE] EvalReadData: %s configuration is complete, so reading from provider", absAddr)

	err := ctx.Hook(func(h Hook) (HookAction, error) {
		// We don't have a state yet, so we'll just give the hook an
		// empty one to work with.
		return h.PreRefresh(absAddr, states.CurrentGen, priorVal)
	})
	if err != nil {
		return nil, err
	}

	resp := provider.ReadDataSource(providers.ReadDataSourceRequest{
		TypeName:     n.Addr.Resource.Type,
		Config:       configVal,
		ProviderMeta: metaConfigVal,
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config))
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
				n.ProviderAddr.Provider.String(), tfdiags.FormatErrorPrefixed(err, absAddr.String()),
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
				n.ProviderAddr.Provider.String(), absAddr,
			),
		))
	}

	if !newVal.IsNull() && !newVal.IsWhollyKnown() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q produced a value for %s that is not wholly known.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ProviderAddr.Provider.String(), absAddr,
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

	action := plans.NoOp
	if !newVal.IsNull() && newVal.IsKnown() && newVal.Equals(priorVal).False() {
		// since a data source is read-only, update here only means that we
		// need to update the state.
		action = plans.Update
	}

	// Produce a change regardless of the outcome.
	change = &plans.ResourceInstanceChange{
		Addr:         absAddr,
		ProviderAddr: n.ProviderAddr,
		Change: plans.Change{
			Action: action,
			Before: priorVal,
			After:  newVal,
		},
	}

	outputState := &states.ResourceInstanceObject{
		Value:  newVal,
		Status: states.ObjectReady, // because we completed the read from the provider
	}

	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(absAddr, states.CurrentGen, priorVal, newVal)
	})
	if err != nil {
		return nil, err
	}

	if n.OutputChange != nil {
		*n.OutputChange = change
	}
	if n.State != nil {
		*n.State = outputState
	}

	return nil, diags.ErrWithWarnings()
}

func (n *EvalReadData) providerMetas(ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	metaConfigVal := cty.NullVal(cty.DynamicPseudoType)
	if n.ProviderMetas != nil {
		if m, ok := n.ProviderMetas[n.ProviderAddr.Provider]; ok && m != nil {
			// if the provider doesn't support this feature, throw an error
			if (*n.ProviderSchema).ProviderMeta == nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Provider %s doesn't support provider_meta", n.ProviderAddr.Provider.String()),
					Detail:   fmt.Sprintf("The resource %s belongs to a provider that doesn't support provider_meta blocks", n.Addr),
					Subject:  &m.ProviderRange,
				})
			} else {
				var configDiags tfdiags.Diagnostics
				metaConfigVal, _, configDiags = ctx.EvaluateBlock(m.Config, (*n.ProviderSchema).ProviderMeta, nil, EvalDataForNoInstanceKey)
				diags = diags.Append(configDiags)
			}
		}
	}
	return metaConfigVal, diags
}

// ForcePlanRead, if true, overrides the usual behavior of immediately
// reading from the data source where possible, instead forcing us to
// _always_ generate a plan. This is used during the plan walk, since we
// mustn't actually apply anything there. (The resulting state doesn't
// get persisted)
func (n *EvalReadData) forcePlanRead(ctx EvalContext) bool {
	if n.refresh && len(n.Config.DependsOn) > 0 {
		return true
	}

	// Check and see if any depends_on dependencies have
	// changes, since they won't show up as changes in the
	// configuration.
	changes := ctx.Changes()
	for _, d := range n.dependsOn {
		for _, change := range changes.GetConfigResourceChanges(d) {
			if change != nil && change.Action != plans.NoOp {
				return true
			}
		}
	}
	return false
}
