package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/objchange"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// evalReadDataPlan is an EvalNode implementation that deals with the main part
// of the data resource lifecycle: either actually reading from the data source
// or generating a plan to do so.
type evalReadDataPlan struct {
	evalReadData
}

func (n *evalReadDataPlan) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())

	var diags tfdiags.Diagnostics
	var configVal cty.Value

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		return nil, fmt.Errorf("provider schema not available for %s", n.Addr)
	}

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

	var configDiags tfdiags.Diagnostics
	configVal, _, configDiags = ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags.ErrWithWarnings()
	}

	configKnown := configVal.IsWhollyKnown()
	// If our configuration contains any unknown values, or we depend on any
	// unknown values then we must defer the read to the apply phase by
	// producing a "Read" change for this resource, and a placeholder value for
	// it in the state.
	if n.forcePlanRead(ctx) || !configKnown {
		if configKnown {
			log.Printf("[TRACE] evalReadDataPlan: %s configuration is fully known, but we're forcing a read plan to be created", absAddr)
		} else {
			log.Printf("[TRACE] evalReadDataPlan: %s configuration not fully known yet, so deferring to apply phase", absAddr)
		}

		proposedNewVal := objchange.PlannedDataResourceObject(schema, configVal)

		if err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreDiff(absAddr, states.CurrentGen, priorVal, proposedNewVal)
		}); err != nil {
			diags = diags.Append(err)
			return nil, diags.ErrWithWarnings()
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

		if err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostDiff(absAddr, states.CurrentGen, plans.Read, priorVal, proposedNewVal)
		}); err != nil {
			diags = diags.Append(err)
		}

		return nil, diags.ErrWithWarnings()
	}

	// We have a complete configuration with no dependencies to wait on, so we
	// can read the data source into the state.
	newVal, readDiags := n.readDataSource(ctx, configVal)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return nil, diags.ErrWithWarnings()
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

	if err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(absAddr, states.CurrentGen, plans.Update, priorVal, newVal)
	}); err != nil {
		return nil, err
	}

	return nil, diags.ErrWithWarnings()
}

// forcePlanRead determines if we need to override the usual behavior of
// immediately reading from the data source where possible, instead forcing us
// to generate a plan.
func (n *evalReadDataPlan) forcePlanRead(ctx EvalContext) bool {
	// Check and see if any depends_on dependencies have
	// changes, since they won't show up as changes in the
	// configuration.
	changes := ctx.Changes()
	for _, d := range n.dependsOn {
		for _, change := range changes.GetChangesForConfigResource(d) {
			if change != nil && change.Action != plans.NoOp {
				return true
			}
		}
	}
	return false
}
