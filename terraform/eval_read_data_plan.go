package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs/configschema"
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

	// If we have a stored state we may not need to re-read the data source.
	// Check the config against the state to see if there are any difference.
	proposedVal, hasChanges := dataObjectHasChanges(schema, priorVal, configVal)

	if !hasChanges {
		log.Printf("[TRACE] evalReadDataPlan: %s no change detected, using existing state", absAddr)
		// state looks up to date, and must have been read during refresh
		return nil, diags.ErrWithWarnings()
	}

	log.Printf("[TRACE] evalReadDataPlan: %s configuration changed, planning data source", absAddr)

	newVal, readDiags := n.readDataSource(ctx, configVal)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return nil, diags.ErrWithWarnings()
	}

	// if we have a prior value, we can check for any irregularities in the response
	if !priorVal.IsNull() {
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

	action := plans.Read
	if priorVal.Equals(newVal).True() {
		action = plans.NoOp
	}

	// The returned value from ReadDataSource must be non-nil and known,
	// which we store in the change. Apply will use the fact that the After
	// value is wholly kown to save the state directly, rather than reading the
	// data source again.
	*n.OutputChange = &plans.ResourceInstanceChange{
		Addr:         absAddr,
		ProviderAddr: n.ProviderAddr,
		Change: plans.Change{
			Action: action,
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

// dataObjectHasChanges determines if the newly evaluated config would cause
// any changes in the stored value, indicating that we need to re-read this
// data source. The proposed value is returned for validation against the
// ReadDataSource response.
func dataObjectHasChanges(schema *configschema.Block, priorVal, configVal cty.Value) (proposedVal cty.Value, hasChanges bool) {
	if priorVal.IsNull() {
		return priorVal, true
	}

	// Applying the configuration to the stored state will allow us to detect any changes.
	proposedVal = objchange.ProposedNewObject(schema, priorVal, configVal)

	if !configVal.IsWhollyKnown() {
		// Config should have been known here, but handle it the same as ProposedNewObject
		return proposedVal, true
	}

	// Normalize the prior value so we can correctly compare the two even if
	// the prior value came through the legacy SDK.
	priorVal = createEmptyBlocks(schema, priorVal)

	return proposedVal, proposedVal.Equals(priorVal).False()
}

// createEmptyBlocks will fill in null TypeList or TypeSet blocks with Empty
// values.  Our decoder will always decode blocks as empty containers, but the
// legacy SDK may replace those will null values. Normalizing these values
// allows us to correctly compare the ProposedNewObject value in
// dataObjectyHasChanges.
func createEmptyBlocks(schema *configschema.Block, val cty.Value) cty.Value {
	if val.IsNull() || !val.IsKnown() {
		return val
	}
	if !val.Type().IsObjectType() {
		panic(fmt.Sprintf("unexpected type %#v\n", val.Type()))
	}

	// if there are no blocks, don't bother recreating the cty.Value
	if len(schema.BlockTypes) == 0 {
		return val
	}

	objMap := val.AsValueMap()

	for name, blockType := range schema.BlockTypes {
		block, ok := objMap[name]
		if !ok {
			continue
		}

		// helper to build the recursive block values
		nextBlocks := func() []cty.Value {
			// this is only called once we know this is a non-null List or Set
			// with a length > 0
			newVals := make([]cty.Value, 0, block.LengthInt())
			for it := block.ElementIterator(); it.Next(); {
				_, val := it.Element()
				newVals = append(newVals, createEmptyBlocks(&blockType.Block, val))
			}
			return newVals
		}

		// Blocks are always decoded as empty containers, but the legacy
		// SDK may return null when they are empty.
		switch blockType.Nesting {
		// We are only concerned with block types that can come from the legacy
		// sdk, which means TypeList or TypeSet.
		case configschema.NestingList:
			ety := block.Type().ElementType()
			switch {
			case block.IsNull():
				objMap[name] = cty.ListValEmpty(ety)
			case block.LengthInt() == 0:
				continue
			default:
				objMap[name] = cty.ListVal(nextBlocks())
			}

		case configschema.NestingSet:
			ety := block.Type().ElementType()
			switch {
			case block.IsNull():
				objMap[name] = cty.SetValEmpty(ety)
			case block.LengthInt() == 0:
				continue
			default:
				objMap[name] = cty.SetVal(nextBlocks())
			}
		}
	}

	return cty.ObjectVal(objMap)
}
