// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

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

	// evaluate the list config block
	var configDiags tfdiags.Diagnostics
	blockVal, _, configDiags := ctx.EvaluateBlock(config.Config, n.Schema.Body, nil, keyData)
	diags = diags.Append(configDiags)
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

	includeRscCty, includeRsc, includeDiags := newIncludeRscEvaluator(false).EvaluateExpr(ctx, config.List.IncludeResource)
	diags = diags.Append(includeDiags)
	if includeDiags.HasErrors() {
		return diags
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
	results := plans.QueryResults{
		Value: resp.Result,
	}

	// If a path is specified, generate the config for the resource
	if n.generateConfigPath != "" {
		var gDiags tfdiags.Diagnostics
		results.Generated, gDiags = n.generateHCLResourceDef(addr, resp.Result.GetAttr("data"), providerSchema.ResourceTypes[n.Config.Type])
		diags = diags.Append(gDiags)
		if diags.HasErrors() {
			return diags
		}
	}

	ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostListQuery(rId, results)
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return diags
	}

	query := &plans.QueryInstance{
		Addr:         n.Addr,
		ProviderAddr: n.ResolvedProvider,
		Results:      results,
	}

	ctx.Changes().AppendQueryInstance(query)
	return diags
}
