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
	err = provider.ListResource(providers.ListResourceRequest{
		TypeName: n.Config.Type,
		Config:   unmarkedConfigVal,
	})
	if err != nil {
		return diags.Append(fmt.Errorf("failed to list %s: %s", n.Addr, err))
	}

	// TODO: Store the result of the list call in the context
	madeUp := []cty.Value{
		cty.ObjectVal(map[string]cty.Value{
			"instance_type": cty.StringVal(n.Addr.String()),
			"ami":           cty.StringVal("ami-123456"),
			"deprecated":    cty.NullVal(cty.String),
			"instance_key":  cty.StringVal("foo"),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"instance_type": cty.StringVal(fmt.Sprintf("%s-v2", n.Addr.String())),
			"ami":           cty.StringVal("ami-654321"),
			"deprecated":    cty.NullVal(cty.String),
			"instance_key":  cty.StringVal("foo"),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"instance_type": cty.StringVal(fmt.Sprintf("%s-v3", n.Addr.String())),
			"ami":           cty.StringVal("ami-789012"),
			"deprecated":    cty.StringVal("foo"),
			"instance_key":  cty.NullVal(cty.String),
		}),
	}

	// Create identity values for the resources
	identities := []cty.Value{
		cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("i-v1"),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("i-v2"),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("i-v3"),
		}),
	}

	if configVal.GetAttr("filter").GetAttr("attr").AsString() != "empty" {
		ctx.State().SetListResourceInstance(n.Addr, &states.ResourceInstanceObject{
			Value:    cty.ListVal(madeUp),
			Identity: cty.ListVal(identities),
		})
	}
	return diags
}
