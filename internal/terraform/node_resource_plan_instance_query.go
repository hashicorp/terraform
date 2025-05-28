// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/genconfig"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
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

	log.Printf("[TRACE] NodePlannableResourceInstance: Re-validating config for %s", n.Addr)
	validateResp := provider.ValidateListResourceConfig(
		providers.ValidateListResourceConfigRequest{
			TypeName: n.Config.Type,
			Config:   unmarkedBlockVal,
		},
	)
	diags = diags.Append(validateResp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return diags
	}

	// If we get down here then our configuration is complete and we're ready
	// to actually call the provider to list the data.
	resp := provider.ListResource(providers.ListResourceRequest{
		TypeName: n.Config.Type,
		Config:   unmarkedBlockVal,
	})
	if resp.Diagnostics != nil {
		return diags.Append(resp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	}

	change := &plans.ResourceInstanceChange{
		Addr:         n.Addr,
		PrevRunAddr:  n.Addr,
		ProviderAddr: n.ResolvedProvider,
		Change: plans.Change{
			Action: plans.Read,
			Before: cty.DynamicVal,
			After:  resp.Result,
		},
		DeposedKey: states.NotDeposed,
	}

	ctx.Changes().AppendResourceInstanceChange(change)
	return diags
}

func (n *NodePlannableResourceInstance) generateListConfig(obj, identity cty.Value) (generated string, diags tfdiags.Diagnostics) {
	schema := n.Schema.Body
	filteredSchema := schema.Filter(
		configschema.FilterOr(
			configschema.FilterReadOnlyAttribute,
			configschema.FilterDeprecatedAttribute,

			// The legacy SDK adds an Optional+Computed "id" attribute to the
			// resource schema even if not defined in provider code.
			// During validation, however, the presence of an extraneous "id"
			// attribute in config will cause an error.
			// Remove this attribute so we do not generate an "id" attribute
			// where there is a risk that it is not in the real resource schema.
			//
			// TRADEOFF: Resources in which there actually is an
			// Optional+Computed "id" attribute in the schema will have that
			// attribute missing from generated config.
			configschema.FilterHelperSchemaIdAttribute,
		),
		configschema.FilterDeprecatedBlock,
	)

	providerAddr := addrs.LocalProviderConfig{
		LocalName: n.ResolvedProvider.Provider.Type,
		Alias:     n.ResolvedProvider.Alias,
	}

	return genconfig.GenerateListResourceContents(n.Addr, filteredSchema, n.Schema.Identity, providerAddr, obj, identity)
}
