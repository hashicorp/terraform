// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodeValidatableAction represents an action that is used for validation only.
type NodeValidatableAction struct {
	*NodeAbstractAction
}

var (
	_ GraphNodeModuleInstance     = (*NodeValidatableAction)(nil)
	_ GraphNodeExecutable         = (*NodeValidatableAction)(nil)
	_ GraphNodeReferenceable      = (*NodeValidatableAction)(nil)
	_ GraphNodeReferencer         = (*NodeValidatableAction)(nil)
	_ GraphNodeConfigAction       = (*NodeValidatableAction)(nil)
	_ GraphNodeAttachActionSchema = (*NodeValidatableAction)(nil)
)

func (n *NodeValidatableAction) Path() addrs.ModuleInstance {
	// There is no expansion during validation, so we evaluate everything as
	// single module instances.
	return n.Addr.Module.UnkeyedInstanceShim()
}

func (n *NodeValidatableAction) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	keyData := EvalDataForNoInstanceKey

	switch {
	case n.Config.Count != nil:
		// If the config block has count, we'll evaluate with an unknown
		// number as count.index so we can still type check even though
		// we won't expand count until the plan phase.
		keyData = InstanceKeyEvalData{
			CountIndex: cty.UnknownVal(cty.Number),
		}

		// Basic type-checking of the count argument. More complete validation
		// of this will happen when we DynamicExpand during the plan walk.
		_, countDiags := evaluateCountExpressionValue(n.Config.Count, ctx)
		diags = diags.Append(countDiags)

	case n.Config.ForEach != nil:
		keyData = InstanceKeyEvalData{
			EachKey:   cty.UnknownVal(cty.String),
			EachValue: cty.UnknownVal(cty.DynamicPseudoType),
		}

		// Evaluate the for_each expression here so we can expose the diagnostics
		forEachDiags := newForEachEvaluator(n.Config.ForEach, ctx, false).ValidateActionValue()
		diags = diags.Append(forEachDiags)
	}

	schema := providerSchema.SchemaForActionType(n.Config.Type)
	if schema.ConfigSchema == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid action type",
			Detail:   fmt.Sprintf("The provider %s does not support action type %q.", n.Provider().ForDisplay(), n.Config.Type),
			Subject:  &n.Config.TypeRange,
		})
		return diags
	}

	config := n.Config.Config
	if n.Config.Config == nil {
		config = hcl.EmptyBody()
	}

	configVal, _, valDiags := ctx.EvaluateBlock(config, schema.ConfigSchema, nil, keyData)
	if valDiags.HasErrors() {
		// If there was no config block at all, we'll add a Context range to the returned diagnostic
		if n.Config.Config == nil {
			for _, diag := range valDiags.ToHCL() {
				diag.Context = &n.Config.DeclRange
				diags = diags.Append(diag)
			}
			return diags
		} else {
			diags = diags.Append(valDiags)
			return diags
		}
	}
	var deprecationDiags tfdiags.Diagnostics
	configVal, deprecationDiags = ctx.Deprecations().ValidateConfig(configVal, schema.ConfigSchema, n.ModulePath())
	diags = diags.Append(deprecationDiags.InConfigBody(n.Config.Config, n.Addr.String()))

	valDiags = validateResourceForbiddenEphemeralValues(ctx, configVal, schema.ConfigSchema)
	diags = diags.Append(valDiags.InConfigBody(config, n.Addr.String()))

	if diags.HasErrors() {
		return diags
	}

	// Use unmarked value for validate request
	unmarkedConfigVal, _ := configVal.UnmarkDeep()
	log.Printf("[TRACE] Validating config for %q", n.Addr)
	req := providers.ValidateActionConfigRequest{
		TypeName: n.Config.Type,
		Config:   unmarkedConfigVal,
	}

	resp := provider.ValidateActionConfig(req)
	diags = diags.Append(resp.Diagnostics.InConfigBody(n.Config.Config, n.Addr.String()))

	return diags
}
