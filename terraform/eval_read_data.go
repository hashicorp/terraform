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

// evalReadData implements shared methods and data for the individual  data
// source eval nodes.
type evalReadData struct {
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
	// new state has been read.
	// While data sources are read-only, we need to start with the prior state
	// to determine if we have a change or not.  If we needed to read a new
	// value, but it still matches the previous state, then we can record a
	// NoNop change. If the states don't match then we record a Read change so
	// that the new value is applied to the state.
	State **states.ResourceInstanceObject

	// Output change records any change for this data source, which is
	// interpreted differently than changes for managed resources.
	// - During Refresh, this change is only used to correctly evaluate
	// references to the data source, but it is not saved.
	// - If a planned change has the action of plans.Read, it indicates that the
	// data source could not be evaluated yet, and reading is being deferred to
	// apply.
	// - If planned action is plans.Update, it indicates that the data source
	// was read, and the result needs to be stored in state during apply.
	OutputChange **plans.ResourceInstanceChange

	// dependsOn stores the list of transitive resource addresses that any
	// configuration depends_on references may resolve to. This is used to
	// determine if there are any changes that will force this data sources to
	// be deferred to apply.
	dependsOn []addrs.ConfigResource
	// forceDependsOn indicates that resources may be missing from dependsOn,
	// but the parent module may have depends_on configured.
	forceDependsOn bool
}

// readDataSource handles everything needed to call ReadDataSource on the provider.
// A previously evaluated configVal can be passed in, or a new one is generated
// from the resource configuration.
func (n *evalReadData) readDataSource(ctx EvalContext, configVal cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var newVal cty.Value

	config := *n.Config
	absAddr := n.Addr.Absolute(ctx.Path())

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		diags = diags.Append(fmt.Errorf("provider schema not available for %s", n.Addr))
		return newVal, diags
	}

	provider := *n.Provider

	providerSchema := *n.ProviderSchema
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider %q does not support data source %q", n.ProviderAddr.Provider.String(), n.Addr.Resource.Type))
		return newVal, diags
	}

	metaConfigVal, metaDiags := n.providerMetas(ctx)
	diags = diags.Append(metaDiags)
	if diags.HasErrors() {
		return newVal, diags
	}

	log.Printf("[TRACE] EvalReadData: Re-validating config for %s", absAddr)
	validateResp := provider.ValidateDataSourceConfig(
		providers.ValidateDataSourceConfigRequest{
			TypeName: n.Addr.Resource.Type,
			Config:   configVal,
		},
	)
	if validateResp.Diagnostics.HasErrors() {
		return newVal, validateResp.Diagnostics.InConfigBody(config.Config)
	}

	// If we get down here then our configuration is complete and we're read
	// to actually call the provider to read the data.
	log.Printf("[TRACE] EvalReadData: %s configuration is complete, so reading from provider", absAddr)

	resp := provider.ReadDataSource(providers.ReadDataSourceRequest{
		TypeName:     n.Addr.Resource.Type,
		Config:       configVal,
		ProviderMeta: metaConfigVal,
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config))
	if diags.HasErrors() {
		return newVal, diags
	}
	newVal = resp.State
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
		return newVal, diags
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

	return newVal, diags
}

func (n *evalReadData) providerMetas(ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
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

// evalReadDataRefresh is an EvalNode implementation that handled the data
// resource lifecycle during refresh
type evalReadDataRefresh struct {
	evalReadData
}

func (n *evalReadDataRefresh) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		return nil, fmt.Errorf("provider schema not available for %s", n.Addr)
	}

	absAddr := n.Addr.Absolute(ctx.Path())
	config := *n.Config
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

	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags.ErrWithWarnings()
	}

	configKnown := configVal.IsWhollyKnown()
	// If our configuration contains any unknown values, then we must defer the
	// read until plan or apply. If we've never read this data source and we
	// have any depends_on, we will have to defer reading until plan to resolve
	// the dependency changes.
	// Assuming we can read the data source with depends_on if we have
	// existing state is a compromise to prevent data sources from continually
	// showing a diff. We have to make the assumption that if we have a prior
	// state, since there are no prior dependency changes happening during
	// refresh, that we can read this resource. If there are dependency updates
	// in the config, they we be discovered in plan and the data source will be
	// read again.
	if !configKnown || (priorVal.IsNull() && n.forcePlanRead()) {
		if configKnown {
			log.Printf("[TRACE] evalReadDataRefresh: %s configuration is fully known, but we're forcing a read plan to be created", absAddr)
		} else {
			log.Printf("[TRACE] evalReadDataRefresh: %s configuration not fully known yet, so deferring to apply phase", absAddr)
		}

		// We need to store a change so tat other references to this data
		// source can resolve correctly, since the state is not going to be up
		// to date.
		*n.OutputChange = &plans.ResourceInstanceChange{
			Addr:         absAddr,
			ProviderAddr: n.ProviderAddr,
			Change: plans.Change{
				Action: plans.Read,
				Before: priorVal,
				After:  objchange.PlannedDataResourceObject(schema, configVal),
			},
		}

		*n.State = &states.ResourceInstanceObject{
			Value:  cty.NullVal(objTy),
			Status: states.ObjectPlanned,
		}

		return nil, diags.ErrWithWarnings()
	}

	if err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreRefresh(absAddr, states.CurrentGen, priorVal)
	}); err != nil {
		diags = diags.Append(err)
		return nil, diags.ErrWithWarnings()
	}

	newVal, readDiags := n.readDataSource(ctx, configVal)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return nil, diags.ErrWithWarnings()
	}

	// This may still have been refreshed with references to resources that
	// will be updated, but that will be caught as a change during plan.
	*n.State = &states.ResourceInstanceObject{
		Value:  newVal,
		Status: states.ObjectReady,
	}

	if err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(absAddr, states.CurrentGen, priorVal, newVal)
	}); err != nil {
		diags = diags.Append(err)
	}

	return nil, diags.ErrWithWarnings()
}

// forcePlanRead determines if we need to override the usual behavior of
// immediately reading from the data source where possible, instead forcing us
// to generate a plan.
func (n *evalReadDataRefresh) forcePlanRead() bool {
	return len(n.Config.DependsOn) > 0 || len(n.dependsOn) > 0 || n.forceDependsOn
}
