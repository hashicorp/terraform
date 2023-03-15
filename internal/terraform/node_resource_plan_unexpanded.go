package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// nodePartialExpandedResource represents an undetermined set of resource
// instances that all share the same [addrs.ConfigResource] address but
// may have certain module instance keys known.
//
// Terraform uses nodes of this type to produce "placeholder plans" that
// are not applyable but approximate the final state of all of the instances
// by using unknown values to stand in for any values that might vary between
// the instances. The goal here is just to help the operator get a sense of
// how their resource configurations will be interpreted even when we don't
// have enough information to plan individual instances fully.
type nodePartialExpandedResource struct {
	Addr addrs.PartialExpandedResource

	Config        *configs.Resource
	Schema        *configschema.Block
	SchemaVersion uint64 // Schema version of "Schema", as decided by the provider

	ProvisionerSchemas map[string]*configschema.Block

	// Set from GraphNodeTargetable
	Targets []addrs.Targetable

	// The address of the provider this resource will use
	ResolvedProvider addrs.AbsProviderConfig

	// skipRefresh indicates that we should skip refreshing individual instances
	skipRefresh bool

	// skipPlanChanges indicates we should skip trying to plan change actions
	// for any instances.
	skipPlanChanges bool
}

func (n *nodePartialExpandedResource) Name() string {
	return n.Addr.String()
}

var (
	_ GraphNodeModuleEvalScope = (*nodePartialExpandedResource)(nil)
	_ GraphNodeExecutable      = (*nodePartialExpandedResource)(nil)
	_ GraphNodeReferenceable   = (*nodePartialExpandedResource)(nil)
	_ GraphNodeReferencer      = (*nodePartialExpandedResource)(nil)
)

// ModuleEvalScope implements GraphNodeModuleEvalScope
func (n *nodePartialExpandedResource) ModuleEvalScope() addrs.ModuleEvalScope {
	// This could either be an addrs.ModuleInstance or an
	// addrs.PartialExpandedModule depending on whether it's just the
	// resource's own expansion that isn't known, or if some or all of the
	// module address is also unknown.
	return n.Addr.ModuleEvalScope()
}

// Execute implements GraphNodeExecutable
func (n *nodePartialExpandedResource) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	// The ctx we get here can perform expression evaluation but might do so
	// in the "partial evaluation" mode if this node is representing instances
	// of a resource in a module whose expansion isn't known yet either.
	log.Printf("[TRACE] nodePartialExpandedResource: Generate placeholder object for all instances matching %s", n.Addr)

	configVal, schema, diags := n.evaluateConfig(ctx, op)
	if diags.HasErrors() {
		return diags
	}

	switch mode := n.Addr.Resource().Mode; mode {
	case addrs.ManagedResourceMode:
		return n.executeManagedResource(ctx, op, configVal, schema)
	case addrs.DataResourceMode:
		return n.executeDataResource(ctx, op, configVal, schema)
	default:
		panic(fmt.Sprintf("unsupported resource mode %s", mode))
	}
}

func (n *nodePartialExpandedResource) evaluateConfig(ctx EvalContext, op walkOperation) (cty.Value, *configschema.Block, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// TODO: Preconditions

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return cty.DynamicVal, nil, diags
	}

	config := n.Config
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.Resource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", n.Addr.Resource().Type))
		return cty.DynamicVal, nil, diags
	}

	var keyData instances.RepetitionData
	switch {
	case config.ForEach != nil:
		// TODO: Somehow determine the for_each type here, ideally without
		// re-evaluating it, so that the keyData can have a more specific
		// type constraint for its each.value value.
		keyData = instances.UnknownForEachRepetitionData(cty.DynamicPseudoType)
	case config.Count != nil:
		keyData = instances.UnknownCountRepetitionData
	}

	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	return configVal, schema, diags
}

func (n *nodePartialExpandedResource) executeManagedResource(ctx EvalContext, op walkOperation, configVal cty.Value, schema *configschema.Block) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// TODO: Ask the provider to validate the configuration.

	proposedNewVal := objchange.ProposedNew(schema, cty.NullVal(schema.ImpliedType()), configVal)

	// TODO: Ask the provider to PlanResourceChange with the proposed new
	// value so we can find out about any plan-time-checked problems early
	// and to potentially generate a more complete placeholder value.

	log.Printf("[TRACE] nodePartialExpandedResource: all %s are %#v", n.Addr, proposedNewVal)

	// TODO: Postconditions

	return diags
}

func (n *nodePartialExpandedResource) executeDataResource(ctx EvalContext, op walkOperation, configVal cty.Value, schema *configschema.Block) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// TODO: Ask the provider to validate the configuration.

	// TODO: If the configuration is fully known and the dependencies have no
	// pending changes then we should proactively read the data source now
	// and use its result as the placeholder for all instances of this
	// resource.

	// TODO: If the configuration isn't fully known then we should put a plan
	// to read all instances of this data resource into the bucket of deferred
	// actions.
	proposedNewVal := objchange.PlannedDataResourceObject(schema, configVal)
	log.Printf("[TRACE] nodePartialExpandedResource: all %s are %#v", n.Addr, proposedNewVal)

	// TODO: Postconditions

	return diags
}

// GraphNodeReferenceable
func (n *nodePartialExpandedResource) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{
		n.Addr.ConfigResource().Resource,
	}
}

// GraphNodeReferencer
func (n *nodePartialExpandedResource) References() []*addrs.Reference {
	return referencesForResource(n.Addr.ConfigResource(), n.Config, n.Schema, n.ProvisionerSchemas)
}
