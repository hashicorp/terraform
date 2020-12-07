package terraform

import (
	"fmt"
	"log"
	"strings"

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

// readData implements shared methods and data for the individual  data
// source eval nodes.
type readData struct {
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
}

// readDataSource handles everything needed to call ReadDataSource on the provider.
// A previously evaluated configVal can be passed in, or a new one is generated
// from the resource configuration.
func (n *readData) readDataSource(ctx EvalContext, configVal cty.Value) (cty.Value, tfdiags.Diagnostics) {
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

	log.Printf("[TRACE] readDataSource: Re-validating config for %s", absAddr)
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
	log.Printf("[TRACE] readDataSource: %s configuration is complete, so reading from provider", absAddr)

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

func (n *readData) providerMetas(ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
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

// plan deals with the main part of the data resource lifecycle: either actually
// reading from the data source or generating a plan to do so.
func (n *readData) plan(ctx EvalContext) tfdiags.Diagnostics {
	absAddr := n.Addr.Absolute(ctx.Path())

	var diags tfdiags.Diagnostics
	var configVal cty.Value

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		diags = diags.Append(fmt.Errorf("provider schema not available for %s", n.Addr))
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
		return diags
	}

	configKnown := configVal.IsWhollyKnown()
	// If our configuration contains any unknown values, or we depend on any
	// unknown values then we must defer the read to the apply phase by
	// producing a "Read" change for this resource, and a placeholder value for
	// it in the state.
	if n.forcePlanRead(ctx) || !configKnown {
		if configKnown {
			log.Printf("[TRACE] readData.Plan: %s configuration is fully known, but we're forcing a read plan to be created", absAddr)
		} else {
			log.Printf("[TRACE] readData.Plan: %s configuration not fully known yet, so deferring to apply phase", absAddr)
		}

		proposedNewVal := objchange.PlannedDataResourceObject(schema, configVal)

		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreDiff(absAddr, states.CurrentGen, priorVal, proposedNewVal)
		}))
		if diags.HasErrors() {
			return diags
		}

		// Apply detects that the data source will need to be read by the After
		// value containing unknowns from PlanDataResourceObject.
		*n.OutputChange = &plans.ResourceInstanceChange{
			Addr:         absAddr,
			ProviderAddr: n.ProviderAddr,
			Change: plans.Change{
				Action: plans.Read,
				Before: priorVal,
				After:  proposedNewVal,
			},
		}

		*n.State = &states.ResourceInstanceObject{
			Value:  proposedNewVal,
			Status: states.ObjectPlanned,
		}

		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostDiff(absAddr, states.CurrentGen, plans.Read, priorVal, proposedNewVal)
		}))

		return diags
	}

	// We have a complete configuration with no dependencies to wait on, so we
	// can read the data source into the state.
	newVal, readDiags := n.readDataSource(ctx, configVal)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return diags
	}

	// if we have a prior value, we can check for any irregularities in the response
	if !priorVal.IsNull() {
		// While we don't propose planned changes for data sources, we can
		// generate a proposed value for comparison to ensure the data source
		// is returning a result following the rules of the provider contract.
		proposedVal := objchange.ProposedNewObject(schema, priorVal, configVal)
		if errs := objchange.AssertObjectCompatible(schema, proposedVal, newVal); len(errs) > 0 {
			// Resources have the LegacyTypeSystem field to signal when they are
			// using an SDK which may not produce precise values. While data
			// sources are read-only, they can still return a value which is not
			// compatible with the config+schema. Since we can't detect the legacy
			// type system, we can only warn about this for now.
			var buf strings.Builder
			fmt.Fprintf(&buf, "[WARN] Provider %q produced an unexpected new value for %s.",
				n.ProviderAddr.Provider.String(), absAddr)
			for _, err := range errs {
				fmt.Fprintf(&buf, "\n      - %s", tfdiags.FormatError(err))
			}
			log.Print(buf.String())
		}
	}

	*n.State = &states.ResourceInstanceObject{
		Value:  newVal,
		Status: states.ObjectReady,
	}

	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(absAddr, states.CurrentGen, plans.Update, priorVal, newVal)
	}))
	return diags
}

// forcePlanRead determines if we need to override the usual behavior of
// immediately reading from the data source where possible, instead forcing us
// to generate a plan.
func (n *readData) forcePlanRead(ctx EvalContext) bool {
	// Check and see if any depends_on dependencies have
	// changes, since they won't show up as changes in the
	// configuration.
	changes := ctx.Changes()
	for _, d := range n.dependsOn {
		if d.Resource.Mode == addrs.DataResourceMode {
			// Data sources have no external side effects, so they pose a need
			// to delay this read. If they do have a change planned, it must be
			// because of a dependency on a managed resource, in which case
			// we'll also encounter it in this list of dependencies.
			continue
		}

		for _, change := range changes.GetChangesForConfigResource(d) {
			if change != nil && change.Action != plans.NoOp {
				return true
			}
		}
	}
	return false
}

// apply deals with the main part of the data resource lifecycle: either
// actually reading from the data source or generating a plan to do so.
func (n *readData) apply(ctx EvalContext) tfdiags.Diagnostics {
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
