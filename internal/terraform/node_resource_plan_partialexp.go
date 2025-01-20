// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodePlannablePartialExpandedResource is a graph node that stands in for
// an unbounded set of potential resource instances that we don't yet know.
//
// Its job is to check the configuration as much as we can with the information
// that's available (so we can raise an error early if something is clearly
// wrong across _all_ potential instances) and to record a placeholder value
// for use when evaluating other objects that refer to this resource.
//
// This is the partial-expanded equivalent of NodePlannableResourceInstance.
type nodePlannablePartialExpandedResource struct {
	addr              addrs.PartialExpandedResource
	config            *configs.Resource
	resolvedProvider  addrs.AbsProviderConfig
	skipPlanChanges   bool
	preDestroyRefresh bool
}

var (
	_ graphNodeEvalContextScope = (*nodePlannablePartialExpandedResource)(nil)
	_ GraphNodeConfigResource   = (*nodePlannablePartialExpandedResource)(nil)
	_ GraphNodeExecutable       = (*nodePlannablePartialExpandedResource)(nil)
)

// Name implements [dag.NamedVertex].
func (n *nodePlannablePartialExpandedResource) Name() string {
	return n.addr.String()
}

// Path implements graphNodeEvalContextScope.
func (n *nodePlannablePartialExpandedResource) Path() evalContextScope {
	if moduleAddr, ok := n.addr.ModuleInstance(); ok {
		return evalContextModuleInstance{Addr: moduleAddr}
	} else if moduleAddr, ok := n.addr.PartialExpandedModule(); ok {
		return evalContextPartialExpandedModule{Addr: moduleAddr}
	} else {
		// Should not get here: at least one of the two cases above
		// should always be true for any valid addrs.PartialExpandedResource
		panic("addrs.PartialExpandedResource has neither a partial-expanded or a fully-expanded module instance address")
	}
}

// ResourceAddr implements GraphNodeConfigResource.
func (n *nodePlannablePartialExpandedResource) ResourceAddr() addrs.ConfigResource {
	return n.addr.ConfigResource()
}

// Execute implements GraphNodeExecutable.
func (n *nodePlannablePartialExpandedResource) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	// Because this node type implements [graphNodeEvalContextScope], the
	// given EvalContext could either be for a fully-expanded module instance
	// or an unbounded set of potential module instances sharing a common
	// prefix. The logic here doesn't need to vary between the two because
	// the differences are encapsulated in the EvalContext abstraction,
	// but if you're unsure which of the two is being used then look for
	// the following line in the logs to see if there's a [*] marker on
	// any of the module instance steps, or if the [*] is applied only to
	// the resource itself.
	//
	// Fully-expanded module example:
	//
	//     module.foo["a"].type.name[*]
	//
	// Partial-expanded module example:
	//
	//     module.foo[*].type.name[*]
	//
	log.Printf("[TRACE] nodePlannablePartialExpandedResource: checking all of %s", n.addr.String())

	switch op {
	case walkPlanDestroy:
		// During destroy plans, we never include partial-expanded resources.
		// We're only interested in fully-expanded resources that we know we
		// need to destroy.
		return nil
	case walkPlan:
		if n.preDestroyRefresh || n.skipPlanChanges {
			// During any kind of refresh, we also don't really care about
			// partial resources. We only care about the fully-expanded resources
			// already in state, so we don't need to plan partial resources.
			return nil
		}

	default:
		// Continue with the normal planning process
	}

	var diags tfdiags.Diagnostics
	switch n.addr.Resource().Mode {
	case addrs.ManagedResourceMode:
		change, changeDiags := n.managedResourceExecute(ctx)
		diags = diags.Append(changeDiags)
		ctx.Deferrals().ReportResourceExpansionDeferred(n.addr, change)
	case addrs.DataResourceMode:
		change, changeDiags := n.dataResourceExecute(ctx)
		diags = diags.Append(changeDiags)
		ctx.Deferrals().ReportDataSourceExpansionDeferred(n.addr, change)
	case addrs.EphemeralResourceMode:
		ctx.Deferrals().ReportEphemeralResourceExpansionDeferred(n.addr)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.config.Mode))
	}

	// Registering this allows downstream resources that depend on this one
	// to know that they need to defer themselves too, in order to preserve
	// correct dependency order.
	return diags
}

// Logic here mirrors (*NodePlannableResourceInstance).managedResourceExecute.
func (n *nodePlannablePartialExpandedResource) managedResourceExecute(ctx EvalContext) (*plans.ResourceInstanceChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// We cannot fully plan partial-expanded resources because we don't know
	// what addresses they will have, but in this function we'll still go
	// through many of the familiar motions of planning so that we can give
	// feedback sooner if we can prove that the configuration is already
	// invalid even with the partial information we have here. This is just
	// to shorten the iterative journey, so nothing here actually contributes
	// new actions to the plan.

	// We'll make a basic change for us to use as a placeholder for the time
	// being, and we'll populate it as we get more info.
	change := plans.ResourceInstanceChange{
		Addr:         n.addr.UnknownResourceInstance(),
		ProviderAddr: n.resolvedProvider,
		Change: plans.Change{
			// We don't actually know the action, but we simulate the plan later
			// as a create action so we'll use that here too.
			Action: plans.Create,
			Before: cty.NullVal(cty.DynamicPseudoType),
			After:  cty.DynamicVal, // This will be populated later
		},
	}

	provider, providerSchema, err := getProvider(ctx, n.resolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return &change, diags
	}

	diags = diags.Append(validateSelfRef(n.addr.Resource(), n.config.Config, providerSchema))
	if diags.HasErrors() {
		return &change, diags
	}

	schema, _ := providerSchema.SchemaForResourceAddr(n.addr.Resource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", n.addr.Resource().Type))
		return &change, diags
	}

	// TODO: Normal managed resource planning
	// (in [NodePlannableResourceInstance.managedResourceExecute]) deals with
	// some additional things that we're just ignoring here for now. We should
	// confirm whether it's really okay to ignore them here or if we ought
	// to be partial-populating some results.
	//
	// Including but not necessarily limited to:
	// - Somehow reacting if one or more of the possible resource instances
	//   is affected by an import block
	// - Evaluating the preconditions/postconditions to see if they produce
	//   a definitive fail result even with the partial information.

	if n.skipPlanChanges {
		// If we're supposed to be making a refresh-only plan then there's
		// not really anything else to do here, since we can only refresh
		// specific known resource instances (which another graph node should
		// handle), so we'll just return early.
		return &change, diags
	}

	keyData := n.keyData()

	configVal, _, configDiags := ctx.EvaluateBlock(n.config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return &change, diags
	}

	unmarkedConfigVal, _ := configVal.UnmarkDeep()
	validateResp := provider.ValidateResourceConfig(
		providers.ValidateResourceConfigRequest{
			TypeName: n.addr.Resource().Type,
			Config:   unmarkedConfigVal,
		},
	)
	diags = diags.Append(validateResp.Diagnostics.InConfigBody(n.config.Config, n.addr.String()))
	if diags.HasErrors() {
		return &change, diags
	}

	unmarkedConfigVal, unmarkedPaths := configVal.UnmarkDeepWithPaths()
	priorVal := cty.NullVal(schema.ImpliedType()) // we don't have any specific prior value to use
	proposedNewVal := objchange.ProposedNew(schema, priorVal, unmarkedConfigVal)

	// The provider now gets to plan an imaginary substitute that represents
	// all of the possible resource instances together. Correctly-implemented
	// providers should handle the extra unknown values here just as if they
	// had been unknown an individual instance's configuration, but we can
	// still find out if any of the known values are somehow invalid and
	// learn a subset of the "computed" attribute values to save as part
	// of our placeholder value for downstream checks.
	resp := provider.PlanResourceChange(providers.PlanResourceChangeRequest{
		TypeName:         n.addr.Resource().Type,
		Config:           unmarkedConfigVal,
		PriorState:       priorVal,
		ProposedNewState: proposedNewVal,
		// TODO: Should we send "ProviderMeta" here? We don't have the
		// necessary data for that wired through here right now, but
		// we might need to do that before stabilizing support for unknown
		// resource instance expansion.
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(n.config.Config, n.addr.String()))
	if diags.HasErrors() {
		return &change, diags
	}

	plannedNewVal := resp.PlannedState
	if plannedNewVal == cty.NilVal {
		// Should never happen. Since real-world providers return via RPC a nil
		// is always a bug in the client-side stub. This is more likely caused
		// by an incompletely-configured mock provider in tests, though.
		panic(fmt.Sprintf("PlanResourceChange of %s produced nil value", n.addr.String()))
	}

	for _, err := range plannedNewVal.Type().TestConformance(schema.ImpliedType()) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid plan",
			fmt.Sprintf(
				"Provider %q planned an invalid value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.resolvedProvider.Provider, tfdiags.FormatErrorPrefixed(err, n.addr.String()),
			),
		))
	}
	if diags.HasErrors() {
		return &change, diags
	}

	if errs := objchange.AssertPlanValid(schema, priorVal, unmarkedConfigVal, plannedNewVal); len(errs) > 0 {
		if resp.LegacyTypeSystem {
			// The shimming of the old type system in the legacy SDK is not precise
			// enough to pass this consistency check, so we'll give it a pass here,
			// but we will generate a warning about it so that we are more likely
			// to notice in the logs if an inconsistency beyond the type system
			// leads to a downstream provider failure.
			var buf strings.Builder
			fmt.Fprintf(&buf,
				"[WARN] Provider %q produced an invalid plan for %s, but we are tolerating it because it is using the legacy plugin SDK.\n    The following problems may be the cause of any confusing errors from downstream operations:",
				n.resolvedProvider.Provider, n.addr.String(),
			)
			for _, err := range errs {
				fmt.Fprintf(&buf, "\n      - %s", tfdiags.FormatError(err))
			}
			log.Print(buf.String())
		} else {
			for _, err := range errs {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Provider produced invalid plan",
					fmt.Sprintf(
						"Provider %q planned an invalid value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
						n.resolvedProvider.Provider, tfdiags.FormatErrorPrefixed(err, n.addr.String()),
					),
				))
			}
			return &change, diags
		}
	}

	// We need to combine the dynamic marks with the static marks implied by
	// the provider's schema.
	plannedNewVal = plannedNewVal.MarkWithPaths(unmarkedPaths)
	if sensitivePaths := schema.SensitivePaths(plannedNewVal, nil); len(sensitivePaths) != 0 {
		plannedNewVal = marks.MarkPaths(plannedNewVal, marks.Sensitive, sensitivePaths)
	}

	change.After = plannedNewVal
	change.Private = resp.PlannedPrivate
	return &change, diags
}

// Logic here mirrors a combination of (*NodePlannableResourceInstance).dataResourceExecute
// and (*NodeAbstractResourceInstance).planDataSource.
func (n *nodePlannablePartialExpandedResource) dataResourceExecute(ctx EvalContext) (*plans.ResourceInstanceChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Start with a basic change, then attempt to fill in the After value.
	change := plans.ResourceInstanceChange{
		Addr:         n.addr.UnknownResourceInstance(),
		ProviderAddr: n.resolvedProvider,
		Change: plans.Change{
			// Data sources can only Read.
			Action: plans.Read,
			Before: cty.NullVal(cty.DynamicPseudoType),
			After:  cty.DynamicVal, // hoping to fill this in
		},
		// For now, this is the default reason for deferred data source reads.
		// It's _basically_ the truth!
		ActionReason: plans.ResourceInstanceReadBecauseConfigUnknown,
	}

	// Unlike with the managed path, we don't ask the provider to *do* anything.
	_, providerSchema, err := getProvider(ctx, n.resolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return &change, diags
	}

	diags = diags.Append(validateSelfRef(n.addr.Resource(), n.config.Config, providerSchema))
	if diags.HasErrors() {
		return &change, diags
	}

	// This is the point where we switch to mirroring logic from
	// NodeAbstractResourceInstance's planDataSource. If you were curious.

	schema, _ := providerSchema.SchemaForResourceAddr(n.addr.Resource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", n.addr.Resource().Type))
		return &change, diags
	}

	keyData := n.keyData()

	configVal, _, configDiags := ctx.EvaluateBlock(n.config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return &change, diags
	}

	// Note: We're deliberately not doing anything special for nested-in-a-check
	// data sources. (*NodeAbstractResourceInstance).planDataSource has some
	// special handling for these, but it's founded on the assumption that we're
	// going to be able to actually read the data source. (Specifically: it
	// blocks propagation of errors on read during plan, and ensures that we get
	// a planned Read to execute during apply even if the data source would have
	// been readable earlier.) But we're getting deferred anyway, so none of
	// that is relevant on this path. 👍🏼

	// Unlike the managed path, we don't call provider.ValidateResourceConfig;
	// Terraform handles planning for data sources without hands-on input from
	// the provider. BTW, this is about where we start mirroring planDataSource's
	// logic for a data source with unknown config, which is sort of what we
	// are, after all.
	unmarkedConfigVal, unmarkedPaths := configVal.UnmarkDeepWithPaths()
	proposedNewVal := objchange.PlannedDataResourceObject(schema, unmarkedConfigVal)
	proposedNewVal = proposedNewVal.MarkWithPaths(unmarkedPaths)
	if sensitivePaths := schema.SensitivePaths(proposedNewVal, nil); len(sensitivePaths) != 0 {
		proposedNewVal = marks.MarkPaths(proposedNewVal, marks.Sensitive, sensitivePaths)
	}
	// yay we made it
	change.After = proposedNewVal
	return &change, diags
}

// keyData returns suitable unknown values for count.index, each.key, and
// each.value, based on what we know of the resource config. When evaluating
// with this unknown key data, anything that varies between the instances will
// be unknown but we can still check the arguments that they all have in common.
func (n *nodePlannablePartialExpandedResource) keyData() instances.RepetitionData {
	switch {
	case n.config.ForEach != nil:
		// We don't actually know the `for_each` type here, but we do at least
		// know it's for_each.
		return instances.UnknownForEachRepetitionData(cty.DynamicPseudoType)
	case n.config.Count != nil:
		return instances.UnknownCountRepetitionData
	default:
		// If we get here then we're presumably a single-instance resource
		// inside a multi-instance module whose instances aren't known yet,
		// and so we'll evaluate without any of the repetition symbols to
		// still generate the usual errors if someone tries to use them here.
		return instances.RepetitionData{
			CountIndex: cty.NilVal,
			EachKey:    cty.NilVal,
			EachValue:  cty.NilVal,
		}
	}
}
