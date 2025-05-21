// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func (n *NodePlannableResourceInstance) listResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
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

	// retrieve list schema? (already done in transformer)
	// evaluate the list config block
	var configDiags tfdiags.Diagnostics
	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, n.Schema.Body, nil, keyData)
	diags = diags.Append(configDiags)
	if diags.HasErrors() {
		return diags
	}

	// Unmark before sending to provider
	unmarkedConfigVal, _ := configVal.UnmarkDeepWithPaths()
	configKnown := configVal.IsWhollyKnown()
	if !configKnown {
		diags = diags.Append(fmt.Errorf("config is not known"))
		return diags
	}

	log.Printf("[TRACE] NodePlannableResourceInstance: Re-validating config for %s", n.Addr)
	validateResp := provider.ValidateListResourceConfig(
		providers.ValidateListResourceConfigRequest{
			TypeName: n.Config.Type,
			Config:   unmarkedConfigVal,
		},
	)
	diags = diags.Append(validateResp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return diags
	}

	// If we get down here then our configuration is complete and we're ready
	// to actually call the provider to list the data.
	resp, err := provider.ListResource(providers.ListResourceRequest{
		TypeName: n.Config.Type,
		Config:   unmarkedConfigVal,
	})
	if err != nil {
		return diags.Append(fmt.Errorf("failed to list %s: %s", n.Addr, err))
	}

	resources := make([]cty.Value, 0)
	identities := make([]cty.Value, 0)

	for evt := range resp {
		if evt.Diagnostics.HasErrors() {
			return diags.Append(evt.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
		}
		resources = append(resources, evt.ResourceObject)
		identities = append(identities, evt.Identity)
	}

	var vals, ids cty.Value
	if len(resources) > 0 {
		vals = cty.ListVal(resources)
		ids = cty.ListVal(identities)
	} else {
		vals = cty.ListValEmpty(cty.Object(map[string]cty.Type{}))
		ids = cty.ListValEmpty(cty.Object(map[string]cty.Type{}))
	}

	ctx.State().SetListResourceInstance(n.Addr, &states.ResourceInstanceObject{
		Value:    vals,
		Identity: ids,
	})
	return diags
}
