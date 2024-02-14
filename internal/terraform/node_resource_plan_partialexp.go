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
	addr             addrs.PartialExpandedResource
	config           *configs.Resource
	resolvedProvider addrs.AbsProviderConfig
	skipPlanChanges  bool
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

	var placeholderVal cty.Value
	var diags tfdiags.Diagnostics
	switch n.addr.Resource().Mode {
	case addrs.ManagedResourceMode:
		placeholderVal, diags = n.managedResourceExecute(ctx)
	case addrs.DataResourceMode:
		placeholderVal, diags = n.dataResourceExecute(ctx)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.config.Mode))
	}

	// Registering this allows downstream resources that depend on this one
	// to know that they need to defer themselves too, in order to preserve
	// correct dependency order.
	ctx.Deferrals().ReportResourceExpansionDeferred(n.addr, placeholderVal)
	return diags
}

func (n *nodePlannablePartialExpandedResource) managedResourceExecute(ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// We cannot fully plan partial-expanded resources because we don't know
	// what addresses they will have, but in this function we'll still go
	// through many of the familiar motions of planning so that we can give
	// feedback sooner if we can prove that the configuration is already
	// invalid even with the partial information we have here. This is just
	// to shorten the iterative journey, so nothing here actually contributes
	// new actions to the plan.

	provider, providerSchema, err := getProvider(ctx, n.resolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return cty.DynamicVal, diags
	}

	diags = diags.Append(validateSelfRef(n.addr.Resource(), n.config.Config, providerSchema))
	if diags.HasErrors() {
		return cty.DynamicVal, diags
	}

	schema, _ := providerSchema.SchemaForResourceAddr(n.addr.Resource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", n.addr.Resource().Type))
		return cty.DynamicVal, diags
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
		return cty.DynamicVal, diags
	}

	// Because we don't know the instance keys yet, we'll be evaluating using
	// suitable unknown values for count.index, each.key, and each.value
	// so that anything that varies between the instances will be unknown
	// but we can still check the arguments that they all have in common.
	var keyData instances.RepetitionData
	switch {
	case n.config.ForEach != nil:
		// We don't actually know the `for_each` type here, but we do at least
		// know it's for_each.
		keyData = instances.UnknownForEachRepetitionData(cty.DynamicPseudoType)
	case n.config.Count != nil:
		keyData = instances.UnknownCountRepetitionData
	default:
		// If we get here then we're presumably a single-instance resource
		// inside a multi-instance module whose instances aren't known yet,
		// and so we'll evaluate without any of the repetition symbols to
		// still generate the usual errors if someone tries to use them here.
		keyData = instances.RepetitionData{
			CountIndex: cty.NilVal,
			EachKey:    cty.NilVal,
			EachValue:  cty.NilVal,
		}
	}

	configVal, _, configDiags := ctx.EvaluateBlock(n.config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return cty.DynamicVal, diags
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
		return cty.DynamicVal, diags
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
		return cty.DynamicVal, diags
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
		return cty.DynamicVal, diags
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
			return cty.DynamicVal, diags
		}
	}

	// We need to combine the dynamic marks with the static marks implied by
	// the provider's schema.
	unmarkedPaths = dedupePathValueMarks(append(unmarkedPaths, schema.ValueMarks(plannedNewVal, nil)...))
	if len(unmarkedPaths) > 0 {
		plannedNewVal = plannedNewVal.MarkWithPaths(unmarkedPaths)
	}

	return plannedNewVal, diags
}

func (n *nodePlannablePartialExpandedResource) dataResourceExecute(ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// TODO: Ideally we should do an approximation of the normal data resource
	// planning process similar to what we're doing for managed resources in
	// managedResourceExecute, but we'll save that for a later phase of this
	// experiment since managed resources are enough to start getting real
	// experience with this new evaluation approach.
	return cty.DynamicVal, diags
}
