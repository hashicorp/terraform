// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/providers"
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
	diags = diags.Append(validateSelfRef(n.RefParser(), addr.Resource, config.Config, providerSchema))
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
	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, n.Schema.ListBody, nil, keyData)
	diags = diags.Append(configDiags)
	if diags.HasErrors() {
		return diags
	}

	// Unmark before sending to provider, will re-mark before returning
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

	doneCh := make(chan struct{}, 1)
	resp := make([]cty.Value, 0)

	// If we get down here then our configuration is complete and we're ready
	// to actually call the provider to list the data.
	err = provider.ListResource(providers.ListResourceRequest{
		TypeName: n.Config.Type,
		Config:   unmarkedConfigVal,
		DiagEmitter: func(diag tfdiags.Diagnostics) {
			diags = diags.Append(diag.InConfigBody(config.Config, n.Addr.String()))
		},
		ResourceEmitter: func(resource providers.ListResult) {
			resp = append(resp, resource.ResourceObject)
		},
		DoneCh: doneCh,
	})
	if err != nil {
		return diags.Append(fmt.Errorf("failed to list %s: %s", n.Addr, err))
	}

	for {
		select {
		case <-doneCh:
			// We are done listing resources
			if len(resp) != 0 {
				ctx.NamedValues().SetResourceListInstance(n.Addr.ContainingResource(), n.Addr.Resource.Key, cty.ListVal(resp))
			}
			return diags
		default:
			// Maybe we want to set some limit on how long we wait or how much data can be sent?
			// do nothing
		}
	}
}
