// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
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
	diags = diags.Append(validateSelfRef(addrs.ParseRefFromQueryScope, addr.Resource, config.Config, providerSchema))
	if diags.HasErrors() {
		return diags
	}

	// retrieve list schema? (already done in transformer)
	forEach, _, _ := evaluateForEachExpression(config.ForEach, ctx, false)
	keyData := EvalDataForInstanceKey(addr.Resource.Key, forEach)

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

	log.Printf("[TRACE] NodeQueryList: Re-validating config for %s", n.Addr)
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
	// retrieve resource schema
	resourceSchema := providerSchema.SchemaForResourceType(addrs.ManagedResourceMode, n.Config.Type)
	if resourceSchema.Body == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider %q does not support managed source %q", n.ResolvedProvider, n.Config.Type))
		return diags
	}

	set := make([]cty.Value, 0)

	// If we get down here then our configuration is complete and we're ready
	// to actually call the provider to list the data.
	err = provider.ListResource(providers.ListResourceRequest{
		TypeName:    n.Config.Type,
		Config:      unmarkedConfigVal,
		DiagEmitter: n.emitDiags,
		ResourceEmitter: func(resource providers.ListResult) {
			set = append(set, resource.ResourceObject)
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
			if len(set) != 0 {
				ctx.NamedValues().SetResourceListInstance(n.Addr.ContainingResource(), n.Addr.Resource.Key, cty.ListVal(set))
			}
			return diags
		default:
			// Maybe we want to set some limit on how long we wait or how much data can be sent?
			// do nothing
		}
	}
}

func (n *NodePlannableResourceInstance) emitDiags(diags tfdiags.Diagnostics) {
	if diags.HasErrors() {
		diags = diags.Append(diags.InConfigBody(n.Config.Config, n.Addr.String()))
		return
	}
}
