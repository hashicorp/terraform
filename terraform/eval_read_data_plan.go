package terraform

import (
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/objchange"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalReadDataPlan is an EvalNode implementation that deals with the main part
// of the data resource lifecycle: either actually reading from the data source
// or generating a plan to do so.
type EvalReadDataPlan struct {
	evalReadData

	// dependsOn stores the list of transitive resource addresses that any
	// configuration depends_on references may resolve to. This is used to
	// determine if there are any changes that will force this data sources to
	// be deferred to apply.
	dependsOn []addrs.ConfigResource
}

func (n *EvalReadDataPlan) Eval(ctx EvalContext) (interface{}, error) {
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
			log.Printf("[TRACE] EvalReadDataPlan: %s configuration is fully known, but we're forcing a read plan to be created", absAddr)
		} else {
			log.Printf("[TRACE] EvalReadDataPlan: %s configuration not fully known yet, so deferring to apply phase", absAddr)
		}

		proposedNewVal := objchange.PlannedDataResourceObject(schema, configVal)

		if err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreDiff(absAddr, states.CurrentGen, priorVal, proposedNewVal)
		}); err != nil {
			diags = diags.Append(err)
			return nil, diags.ErrWithWarnings()
		}

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
			Value:  cty.NullVal(objTy),
			Status: states.ObjectPlanned,
		}

		if err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostDiff(absAddr, states.CurrentGen, plans.Read, priorVal, proposedNewVal)
		}); err != nil {
			diags = diags.Append(err)
		}

		return nil, diags.ErrWithWarnings()
	}

	// If we have a stored state we may not need to re-read the data source.
	// Check the config against the state to see if there are any difference.
	if !priorVal.IsNull() {
		// Applying the configuration to the prior state lets us see if there
		// are any differences.
		proposed := objchange.ProposedNewObject(schema, priorVal, configVal)
		if proposed.Equals(priorVal).True() {
			log.Printf("[TRACE] EvalReadDataPlan: %s no change detected, using existing state", absAddr)
			// state looks up to date, and must have been read during refresh
			return nil, diags.ErrWithWarnings()
		}
	}

	newVal, readDiags := n.readDataSource(ctx, configVal)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return nil, diags.ErrWithWarnings()
	}

	// Produce a change regardless of the outcome.
	*n.OutputChange = &plans.ResourceInstanceChange{
		Addr:         absAddr,
		ProviderAddr: n.ProviderAddr,
		Change: plans.Change{
			Action: plans.Update,
			Before: priorVal,
			After:  newVal,
		},
	}

	*n.State = &states.ResourceInstanceObject{
		Value:  newVal,
		Status: states.ObjectPlanned,
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
func (n *EvalReadDataPlan) forcePlanRead(ctx EvalContext) bool {
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
