// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func (n *NodePlannableResourceInstance) listResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	log.Printf("[TRACE] NodePlannableResourceInstance: listing resources for %s", n.Addr)
	config := n.Config
	addr := n.ResourceInstanceAddr()
	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// validate self ref
	diags = diags.Append(validateSelfRef(addr.Resource, config.Config, providerSchema))
	if diags.HasErrors() {
		return diags
	}

	keyData := EvalDataForInstanceKey(addr.Resource.Key, nil)
	if config.ForEach != nil {
		forEach, _, _ := evaluateForEachExpression(config.ForEach, ctx, false)
		keyData = EvalDataForInstanceKey(addr.Resource.Key, forEach)
	}

	schema := providerSchema.SchemaForListResourceType(n.Config.Type)
	if schema.IsNil() { // Not possible, as the schema should have already been validated to exist
		diags = diags.Append(fmt.Errorf("no schema available for %s; this is a bug in Terraform and should be reported", addr))
		return diags
	}

	// evaluate the list config block
	var configDiags tfdiags.Diagnostics
	blockVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema.FullSchema, nil, keyData)
	diags = diags.Append(configDiags)
	if diags.HasErrors() {
		return diags
	}
	var deprecationDiags tfdiags.Diagnostics
	blockVal, deprecationDiags = ctx.Deprecations().ValidateConfig(blockVal, schema.FullSchema, n.ModulePath())
	diags = diags.Append(deprecationDiags.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return diags
	}

	// Unmark before sending to provider
	unmarkedBlockVal, _ := blockVal.UnmarkDeepWithPaths()
	configKnown := blockVal.IsWhollyKnown()
	if !configKnown {
		diags = diags.Append(fmt.Errorf("config is not known"))
		return diags
	}

	limitCty, limit, limitDiags := newLimitEvaluator(false).EvaluateExpr(ctx, config.List.Limit)
	diags = diags.Append(limitDiags)
	if limitDiags.HasErrors() {
		return diags
	}

	if config.List.Limit != nil {
		var limitDeprecationDiags tfdiags.Diagnostics
		limitCty, limitDeprecationDiags = ctx.Deprecations().Validate(limitCty, ctx.Path().Module(), config.List.Limit.Range().Ptr())
		diags = diags.Append(limitDeprecationDiags)
	}

	includeRscCty, includeRsc, includeDiags := newIncludeRscEvaluator(false).EvaluateExpr(ctx, config.List.IncludeResource)
	diags = diags.Append(includeDiags)
	if includeDiags.HasErrors() {
		return diags
	}

	if config.List.IncludeResource != nil {
		var includeDeprecationDiags tfdiags.Diagnostics
		includeRscCty, includeDeprecationDiags = ctx.Deprecations().Validate(includeRscCty, ctx.Path().Module(), config.List.IncludeResource.Range().Ptr())
		diags = diags.Append(includeDeprecationDiags)
	}

	rId := HookResourceIdentity{
		Addr:         addr,
		ProviderAddr: n.ResolvedProvider.Provider,
	}
	ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreListQuery(rId, unmarkedBlockVal.GetAttr("config"))
	})

	// if we are generating config, we implicitly set include_resource to true
	if n.generateConfigPath != "" {
		includeRscCty = cty.True
		includeRsc = true
	}

	log.Printf("[TRACE] NodePlannableResourceInstance: Re-validating config for %s", n.Addr)
	// if the config value is null, we still want to send a full object with all attributes being null
	if !unmarkedBlockVal.IsNull() && unmarkedBlockVal.GetAttr("config").IsNull() {
		mp := unmarkedBlockVal.AsValueMap()
		mp["config"] = schema.ConfigSchema.EmptyValue()
		unmarkedBlockVal = cty.ObjectVal(mp)
	}

	validateResp := provider.ValidateListResourceConfig(
		providers.ValidateListResourceConfigRequest{
			TypeName:              n.Config.Type,
			Config:                unmarkedBlockVal,
			IncludeResourceObject: includeRscCty,
			Limit:                 limitCty,
		},
	)
	diags = diags.Append(validateResp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return diags
	}

	// If we get down here then our configuration is complete and we're ready
	// to actually call the provider to list the data.
	resp := provider.ListResource(providers.ListResourceRequest{
		TypeName:              n.Config.Type,
		Config:                unmarkedBlockVal,
		Limit:                 limit,
		IncludeResourceObject: includeRsc,
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return diags
	}
	results := plans.QueryResults{
		Value: resp.Result,
	}

	// If a path is specified, generate the config for the resource
	if n.generateConfigPath != "" {
		var gDiags tfdiags.Diagnostics
		results.Generated, gDiags = n.generateHCLListResourceDef(ctx, addr, resp.Result.GetAttr("data"))
		diags = diags.Append(gDiags)
		if diags.HasErrors() {
			return diags
		}
	}

	identityVersion := providerSchema.SchemaForResourceType(addrs.ManagedResourceMode, addr.Resource.Resource.Type).IdentityVersion

	ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostListQuery(rId, results, identityVersion)
	})

	query := &plans.QueryInstance{
		Addr:         n.Addr,
		ProviderAddr: n.ResolvedProvider,
		Results:      results,
	}

	ctx.Changes().AppendQueryInstance(query)
	return diags
}
