// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/didyoumean"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/ephemeral"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodeValidatableResource represents a resource that is used for validation
// only.
type NodeValidatableResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeModuleInstance            = (*NodeValidatableResource)(nil)
	_ GraphNodeExecutable                = (*NodeValidatableResource)(nil)
	_ GraphNodeReferenceable             = (*NodeValidatableResource)(nil)
	_ GraphNodeReferencer                = (*NodeValidatableResource)(nil)
	_ GraphNodeConfigResource            = (*NodeValidatableResource)(nil)
	_ GraphNodeAttachResourceConfig      = (*NodeValidatableResource)(nil)
	_ GraphNodeAttachProviderMetaConfigs = (*NodeValidatableResource)(nil)
)

func (n *NodeValidatableResource) Path() addrs.ModuleInstance {
	// There is no expansion during validation, so we evaluate everything as
	// single module instances.
	return n.Addr.Module.UnkeyedInstanceShim()
}

// GraphNodeEvalable
func (n *NodeValidatableResource) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	// this is done first since there may not be config if we are generating it
	diags = diags.Append(n.validateImportTargets(ctx))

	if n.Config == nil {
		return diags
	}

	diags = diags.Append(n.validateResource(ctx))

	diags = diags.Append(n.validateCheckRules(ctx, n.Config))

	if managed := n.Config.Managed; managed != nil {
		// Validate all the provisioners
		for _, p := range managed.Provisioners {
			diags = diags.Append(n.validateProvisioner(ctx, p, n.Config.Managed.Connection))
			if diags.HasErrors() {
				return diags
			}
		}
	}
	return diags
}

// validateProvisioner validates the configuration of a provisioner belonging to
// a resource. The provisioner config is expected to contain the merged
// connection configurations.
func (n *NodeValidatableResource) validateProvisioner(ctx EvalContext, p *configs.Provisioner, baseConn *configs.Connection) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	provisioner, err := ctx.Provisioner(p.Type)
	if err != nil {
		diags = diags.Append(err)
		return diags
	}

	if provisioner == nil {
		return diags.Append(fmt.Errorf("provisioner %s not initialized", p.Type))
	}
	provisionerSchema, err := ctx.ProvisionerSchema(p.Type)
	if err != nil {
		return diags.Append(fmt.Errorf("failed to read schema for provisioner %s: %s", p.Type, err))
	}
	if provisionerSchema == nil {
		return diags.Append(fmt.Errorf("provisioner %s has no schema", p.Type))
	}

	// Validate the provisioner's own config first
	configVal, _, configDiags := n.evaluateBlock(ctx, p.Config, provisionerSchema)
	diags = diags.Append(configDiags)

	if configVal == cty.NilVal {
		// Should never happen for a well-behaved EvaluateBlock implementation
		return diags.Append(fmt.Errorf("EvaluateBlock returned nil value"))
	}

	// Use unmarked value for validate request
	unmarkedConfigVal, _ := configVal.UnmarkDeep()
	req := provisioners.ValidateProvisionerConfigRequest{
		Config: unmarkedConfigVal,
	}

	resp := provisioner.ValidateProvisionerConfig(req)
	diags = diags.Append(resp.Diagnostics)

	if p.Connection != nil {
		// We can't comprehensively validate the connection config since its
		// final structure is decided by the communicator and we can't instantiate
		// that until we have a complete instance state. However, we *can* catch
		// configuration keys that are not valid for *any* communicator, catching
		// typos early rather than waiting until we actually try to run one of
		// the resource's provisioners.

		cfg := p.Connection.Config
		if baseConn != nil {
			// Merge the local config into the base connection config, if we
			// both specified.
			cfg = configs.MergeBodies(baseConn.Config, cfg)
		}

		_, _, connDiags := n.evaluateBlock(ctx, cfg, connectionBlockSupersetSchema)
		diags = diags.Append(connDiags)
	} else if baseConn != nil {
		// Just validate the baseConn directly.
		_, _, connDiags := n.evaluateBlock(ctx, baseConn.Config, connectionBlockSupersetSchema)
		diags = diags.Append(connDiags)

	}
	return diags
}

func (n *NodeValidatableResource) evaluateBlock(ctx EvalContext, body hcl.Body, schema *configschema.Block) (cty.Value, hcl.Body, tfdiags.Diagnostics) {
	keyData, selfAddr := n.stubRepetitionData(n.Config.Count != nil, n.Config.ForEach != nil)

	return ctx.EvaluateBlock(body, schema, selfAddr, keyData)
}

// connectionBlockSupersetSchema is a schema representing the superset of all
// possible arguments for "connection" blocks across all supported connection
// types.
//
// This currently lives here because we've not yet updated our communicator
// subsystem to be aware of schema itself. Once that is done, we can remove
// this and use a type-specific schema from the communicator to validate
// exactly what is expected for a given connection type.
var connectionBlockSupersetSchema = &configschema.Block{
	Attributes: map[string]*configschema.Attribute{
		// NOTE: "type" is not included here because it's treated special
		// by the config loader and stored away in a separate field.

		// Common attributes for both connection types
		"host": {
			Type:     cty.String,
			Required: true,
		},
		"type": {
			Type:     cty.String,
			Optional: true,
		},
		"user": {
			Type:     cty.String,
			Optional: true,
		},
		"password": {
			Type:     cty.String,
			Optional: true,
		},
		"port": {
			Type:     cty.Number,
			Optional: true,
		},
		"timeout": {
			Type:     cty.String,
			Optional: true,
		},
		"script_path": {
			Type:     cty.String,
			Optional: true,
		},
		// For type=ssh only (enforced in ssh communicator)
		"target_platform": {
			Type:     cty.String,
			Optional: true,
		},
		"private_key": {
			Type:     cty.String,
			Optional: true,
		},
		"certificate": {
			Type:     cty.String,
			Optional: true,
		},
		"host_key": {
			Type:     cty.String,
			Optional: true,
		},
		"agent": {
			Type:     cty.Bool,
			Optional: true,
		},
		"agent_identity": {
			Type:     cty.String,
			Optional: true,
		},
		"proxy_scheme": {
			Type:     cty.String,
			Optional: true,
		},
		"proxy_host": {
			Type:     cty.String,
			Optional: true,
		},
		"proxy_port": {
			Type:     cty.Number,
			Optional: true,
		},
		"proxy_user_name": {
			Type:     cty.String,
			Optional: true,
		},
		"proxy_user_password": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_host": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_host_key": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_port": {
			Type:     cty.Number,
			Optional: true,
		},
		"bastion_user": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_password": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_private_key": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_certificate": {
			Type:     cty.String,
			Optional: true,
		},

		// For type=winrm only (enforced in winrm communicator)
		"https": {
			Type:     cty.Bool,
			Optional: true,
		},
		"insecure": {
			Type:     cty.Bool,
			Optional: true,
		},
		"cacert": {
			Type:     cty.String,
			Optional: true,
		},
		"use_ntlm": {
			Type:     cty.Bool,
			Optional: true,
		},
	},
}

func (n *NodeValidatableResource) validateResource(ctx EvalContext) tfdiags.Diagnostics {
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
		countDiags := validateCount(ctx, n.Config.Count)
		diags = diags.Append(countDiags)

	case n.Config.ForEach != nil:
		keyData = InstanceKeyEvalData{
			EachKey:   cty.UnknownVal(cty.String),
			EachValue: cty.UnknownVal(cty.DynamicPseudoType),
		}

		// Evaluate the for_each expression here so we can expose the diagnostics
		forEachDiags := newForEachEvaluator(n.Config.ForEach, ctx, false).ValidateResourceValue()
		diags = diags.Append(forEachDiags)
	}

	diags = diags.Append(validateDependsOn(ctx, n.Config.DependsOn))

	// Provider entry point varies depending on resource mode, because
	// managed resources and data resources are two distinct concepts
	// in the provider abstraction.
	switch n.Config.Mode {
	case addrs.ManagedResourceMode:
		schema, _ := providerSchema.SchemaForResourceType(n.Config.Mode, n.Config.Type)
		if schema == nil {
			var suggestion string
			if dSchema, _ := providerSchema.SchemaForResourceType(addrs.DataResourceMode, n.Config.Type); dSchema != nil {
				suggestion = fmt.Sprintf("\n\nDid you intend to use the data source %q? If so, declare this using a \"data\" block instead of a \"resource\" block.", n.Config.Type)
			} else if len(providerSchema.ResourceTypes) > 0 {
				suggestions := make([]string, 0, len(providerSchema.ResourceTypes))
				for name := range providerSchema.ResourceTypes {
					suggestions = append(suggestions, name)
				}
				if suggestion = didyoumean.NameSuggestion(n.Config.Type, suggestions); suggestion != "" {
					suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
				}
			}

			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid resource type",
				Detail:   fmt.Sprintf("The provider %s does not support resource type %q.%s", n.Provider().ForDisplay(), n.Config.Type, suggestion),
				Subject:  &n.Config.TypeRange,
			})
			return diags
		}

		configVal, _, valDiags := ctx.EvaluateBlock(n.Config.Config, schema, nil, keyData)
		diags = diags.Append(valDiags)
		if valDiags.HasErrors() {
			return diags
		}
		diags = diags.Append(
			validateResourceForbiddenEphemeralValues(ctx, configVal, schema).InConfigBody(n.Config.Config, n.Addr.String()),
		)

		if n.Config.Managed != nil { // can be nil only in tests with poorly-configured mocks
			for _, traversal := range n.Config.Managed.IgnoreChanges {
				// validate the ignore_changes traversals apply.
				moreDiags := schema.StaticValidateTraversal(traversal)
				diags = diags.Append(moreDiags)

				// ignore_changes cannot be used for Computed attributes,
				// unless they are also Optional.
				// If the traversal was valid, convert it to a cty.Path and
				// use that to check whether the Attribute is Computed and
				// non-Optional.
				if !diags.HasErrors() {
					path := traversalToPath(traversal)

					attrSchema := schema.AttributeByPath(path)

					if attrSchema != nil && !attrSchema.Optional && attrSchema.Computed {
						// ignore_changes uses absolute traversal syntax in config despite
						// using relative traversals, so we strip the leading "." added by
						// FormatCtyPath for a better error message.
						attrDisplayPath := strings.TrimPrefix(tfdiags.FormatCtyPath(path), ".")

						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagWarning,
							Summary:  "Redundant ignore_changes element",
							Detail:   fmt.Sprintf("Adding an attribute name to ignore_changes tells Terraform to ignore future changes to the argument in configuration after the object has been created, retaining the value originally configured.\n\nThe attribute %s is decided by the provider alone and therefore there can be no configured value to compare with. Including this attribute in ignore_changes has no effect. Remove the attribute from ignore_changes to quiet this warning.", attrDisplayPath),
							Subject:  &n.Config.TypeRange,
						})
					}
				}
			}
		}

		// Use unmarked value for validate request
		unmarkedConfigVal, _ := configVal.UnmarkDeep()
		req := providers.ValidateResourceConfigRequest{
			TypeName: n.Config.Type,
			Config:   unmarkedConfigVal,
		}

		resp := provider.ValidateResourceConfig(req)
		diags = diags.Append(resp.Diagnostics.InConfigBody(n.Config.Config, n.Addr.String()))

	case addrs.DataResourceMode:
		schema, _ := providerSchema.SchemaForResourceType(n.Config.Mode, n.Config.Type)
		if schema == nil {
			var suggestion string
			if dSchema, _ := providerSchema.SchemaForResourceType(addrs.ManagedResourceMode, n.Config.Type); dSchema != nil {
				suggestion = fmt.Sprintf("\n\nDid you intend to use the managed resource type %q? If so, declare this using a \"resource\" block instead of a \"data\" block.", n.Config.Type)
			} else if len(providerSchema.DataSources) > 0 {
				suggestions := make([]string, 0, len(providerSchema.DataSources))
				for name := range providerSchema.DataSources {
					suggestions = append(suggestions, name)
				}
				if suggestion = didyoumean.NameSuggestion(n.Config.Type, suggestions); suggestion != "" {
					suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
				}
			}

			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid data source",
				Detail:   fmt.Sprintf("The provider %s does not support data source %q.%s", n.Provider().ForDisplay(), n.Config.Type, suggestion),
				Subject:  &n.Config.TypeRange,
			})
			return diags
		}

		configVal, _, valDiags := ctx.EvaluateBlock(n.Config.Config, schema, nil, keyData)
		diags = diags.Append(valDiags)
		if valDiags.HasErrors() {
			return diags
		}
		diags = diags.Append(
			validateResourceForbiddenEphemeralValues(ctx, configVal, schema).InConfigBody(n.Config.Config, n.Addr.String()),
		)

		// Use unmarked value for validate request
		unmarkedConfigVal, _ := configVal.UnmarkDeep()
		req := providers.ValidateDataResourceConfigRequest{
			TypeName: n.Config.Type,
			Config:   unmarkedConfigVal,
		}

		resp := provider.ValidateDataResourceConfig(req)
		diags = diags.Append(resp.Diagnostics.InConfigBody(n.Config.Config, n.Addr.String()))
	}

	return diags
}

func (n *NodeValidatableResource) evaluateExpr(ctx EvalContext, expr hcl.Expression, wantTy cty.Type, self addrs.Referenceable, keyData instances.RepetitionData) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRef, expr)
	diags = diags.Append(refDiags)

	scope := ctx.EvaluationScope(self, nil, keyData)

	hclCtx, moreDiags := scope.EvalContext(refs)
	diags = diags.Append(moreDiags)

	result, hclDiags := expr.Value(hclCtx)
	diags = diags.Append(hclDiags)

	return result, diags
}

func (n *NodeValidatableResource) stubRepetitionData(hasCount, hasForEach bool) (instances.RepetitionData, addrs.Referenceable) {
	keyData := EvalDataForNoInstanceKey
	selfAddr := n.ResourceAddr().Resource.Instance(addrs.NoKey)

	if n.Config.Count != nil {
		// For a resource that has count, we allow count.index but don't
		// know at this stage what it will return.
		keyData = InstanceKeyEvalData{
			CountIndex: cty.UnknownVal(cty.Number),
		}

		// "self" can't point to an unknown key, but we'll force it to be
		// key 0 here, which should return an unknown value of the
		// expected type since none of these elements are known at this
		// point anyway.
		selfAddr = n.ResourceAddr().Resource.Instance(addrs.IntKey(0))
	} else if n.Config.ForEach != nil {
		// For a resource that has for_each, we allow each.value and each.key
		// but don't know at this stage what it will return.
		keyData = InstanceKeyEvalData{
			EachKey:   cty.UnknownVal(cty.String),
			EachValue: cty.DynamicVal,
		}

		// "self" can't point to an unknown key, but we'll force it to be
		// key "" here, which should return an unknown value of the
		// expected type since none of these elements are known at
		// this point anyway.
		selfAddr = n.ResourceAddr().Resource.Instance(addrs.StringKey(""))
	}

	return keyData, selfAddr
}

func (n *NodeValidatableResource) validateCheckRules(ctx EvalContext, config *configs.Resource) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	keyData, selfAddr := n.stubRepetitionData(n.Config.Count != nil, n.Config.ForEach != nil)

	for _, cr := range config.Preconditions {
		_, conditionDiags := n.evaluateExpr(ctx, cr.Condition, cty.Bool, nil, keyData)
		diags = diags.Append(conditionDiags)

		_, errorMessageDiags := n.evaluateExpr(ctx, cr.ErrorMessage, cty.Bool, nil, keyData)
		diags = diags.Append(errorMessageDiags)
	}

	for _, cr := range config.Postconditions {
		_, conditionDiags := n.evaluateExpr(ctx, cr.Condition, cty.Bool, selfAddr, keyData)
		diags = diags.Append(conditionDiags)

		_, errorMessageDiags := n.evaluateExpr(ctx, cr.ErrorMessage, cty.Bool, selfAddr, keyData)
		diags = diags.Append(errorMessageDiags)
	}

	return diags
}

// validateImportTargets checks that the import block expressions are valid.
func (n *NodeValidatableResource) validateImportTargets(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if len(n.importTargets) == 0 {
		return diags
	}

	diags = diags.Append(n.validateConfigGen(ctx))

	// Import blocks are only valid within the root module, and must be
	// evaluated within that context
	ctx = evalContextForModuleInstance(ctx, addrs.RootModuleInstance)

	for _, imp := range n.importTargets {
		if imp.Config == nil {
			// if we have a legacy addr, it will be supplied on the command line so
			// there is nothing to check now and we need to wait for plan.
			continue
		}

		diags = diags.Append(validateImportSelfRef(n.Addr.Resource, imp.Config.ID))
		if diags.HasErrors() {
			return diags
		}

		// Resource config might be nil here since we are also validating config generation.
		expanded := n.Config != nil && (n.Config.ForEach != nil || n.Config.Count != nil)

		if imp.Config.ForEach != nil {
			if !expanded {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Use of import for_each in an invalid context",
					Detail:   "Use of for_each in import requires a resource using count or for_each.",
					// FIXME: minor issue, but this points to the for_each expression rather than for_each itself.
					Subject: imp.Config.ForEach.Range().Ptr(),
				})
			}

			forEachData, _, forEachDiags := newForEachEvaluator(imp.Config.ForEach, ctx, true).ImportValues()
			diags = diags.Append(forEachDiags)
			if forEachDiags.HasErrors() {
				return diags
			}

			for _, keyData := range forEachData {
				to, evalDiags := evalImportToExpression(imp.Config.To, keyData)
				diags = diags.Append(evalDiags)
				if diags.HasErrors() {
					return diags
				}
				diags = diags.Append(validateImportTargetExpansion(n.Config, to, imp.Config.To))
				if diags.HasErrors() {
					return diags
				}
				_, evalDiags = evaluateImportIdExpression(imp.Config.ID, ctx, keyData, true)
				diags = diags.Append(evalDiags)
				if diags.HasErrors() {
					return diags
				}
			}
		} else {
			traversal, hds := hcl.AbsTraversalForExpr(imp.Config.To)
			diags = diags.Append(hds)
			to, tds := addrs.ParseAbsResourceInstance(traversal)
			diags = diags.Append(tds)
			if diags.HasErrors() {
				return diags
			}

			diags = diags.Append(validateImportTargetExpansion(n.Config, to, imp.Config.To))
			if diags.HasErrors() {
				return diags
			}

			_, evalDiags := evaluateImportIdExpression(imp.Config.ID, ctx, EvalDataForNoInstanceKey, true)
			diags = diags.Append(evalDiags)
			if diags.HasErrors() {
				return diags
			}
		}
	}

	return diags
}

// validateImportTargetExpansion ensures that the To address key and resource expansion mode both agree.
func validateImportTargetExpansion(cfg *configs.Resource, to addrs.AbsResourceInstance, toExpr hcl.Expression) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	forEach := cfg != nil && cfg.ForEach != nil
	count := cfg != nil && cfg.Count != nil

	switch to.Resource.Key.(type) {
	case addrs.StringKey:
		if !forEach {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid import 'to' expression",
				Detail:   "The target resource does not use for_each.",
				Subject:  toExpr.Range().Ptr(),
			})
		}
	case addrs.IntKey:
		if !count {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid import 'to' expression",
				Detail:   "The target resource does not use count.",
				Subject:  toExpr.Range().Ptr(),
			})
		}
	default:
		if forEach {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid import 'to' expression",
				Detail:   "The target resource is using for_each.",
				Subject:  toExpr.Range().Ptr(),
			})
		}

		if count {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid import 'to' expression",
				Detail:   "The target resource is using count.",
				Subject:  toExpr.Range().Ptr(),
			})
		}
	}

	return diags
}

// validate imports with no config for possible config generation
func (n *NodeValidatableResource) validateConfigGen(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if n.Config != nil {
		return diags
	}

	// We won't have the config generation output path during validate, so only
	// check if generation is at all possible.

	for _, imp := range n.importTargets {
		if !n.Addr.Module.IsRoot() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Configuration for import target does not exist",
				Detail:   fmt.Sprintf("Resource %s not found. Only resources within the root module are eligible for config generation.", n.Addr),
				Subject:  imp.Config.To.Range().Ptr(),
			})
			continue
		}

		var toDiags tfdiags.Diagnostics
		traversal, hd := hcl.AbsTraversalForExpr(imp.Config.To)
		toDiags = toDiags.Append(hd)
		to, td := addrs.ParseAbsResourceInstance(traversal)
		toDiags = toDiags.Append(td)

		if toDiags.HasErrors() {
			// these will be caught elsewhere with better context
			continue
		}

		if to.Resource.Key != addrs.NoKey || imp.Config.ForEach != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Configuration for import target does not exist",
				Detail:   "The given import block is not compatible with config generation. The -generate-config-out option cannot be used with import blocks which use for_each, or resources which use for_each or count.",
				Subject:  imp.Config.To.Range().Ptr(),
			})
		}
	}
	return diags
}

func validateCount(ctx EvalContext, expr hcl.Expression) (diags tfdiags.Diagnostics) {
	val, countDiags := evaluateCountExpressionValue(expr, ctx)
	// If the value isn't known then that's the best we can do for now, but
	// we'll check more thoroughly during the plan walk
	if !val.IsKnown() {
		return diags
	}

	if countDiags.HasErrors() {
		diags = diags.Append(countDiags)
	}

	return diags
}

func validateDependsOn(ctx EvalContext, dependsOn []hcl.Traversal) (diags tfdiags.Diagnostics) {
	for _, traversal := range dependsOn {
		ref, refDiags := addrs.ParseRef(traversal)
		diags = diags.Append(refDiags)
		if !refDiags.HasErrors() && len(ref.Remaining) != 0 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid depends_on reference",
				Detail:   "References in depends_on must be to a whole object (resource, etc), not to an attribute of an object.",
				Subject:  ref.Remaining.SourceRange().Ptr(),
			})
		}

		// The ref must also refer to something that exists. To test that,
		// we'll just eval it and count on the fact that our evaluator will
		// detect references to non-existent objects.
		if !diags.HasErrors() {
			scope := ctx.EvaluationScope(nil, nil, EvalDataForNoInstanceKey)
			if scope != nil { // sometimes nil in tests, due to incomplete mocks
				_, refDiags = scope.EvalReference(ref, cty.DynamicPseudoType)
				diags = diags.Append(refDiags)
			}
		}
	}
	return diags
}

// validateResourceForbiddenEphemeralValues returns an error diagnostic for each
// value anywhere inside the given value that is marked as ephemeral, for
// situations where ephemeral values are not permitted.
//
// All returned diagnostics are contextual diagnostics that must be finalized
// by calling [tfdiags.Diagnostics.InConfigBody] before returning them to
// any caller that expects fully-resolved diagnostics.
func validateResourceForbiddenEphemeralValues(ctx EvalContext, value cty.Value, schema *configschema.Block) (diags tfdiags.Diagnostics) {
	// NOTE: We take a schema argument in anticipation of a future feature
	// that might allow managed resources to declare certain attributes as
	// being "write-only", which would create a little nested island where
	// ephemeral values are permitted in return for providers accepting that
	// those values will not be preserved between plan and apply or between
	// sequential plan/apply rounds. But we aren't doing that yet, so we
	// just ignore that argument for now.

	for _, path := range ephemeral.EphemeralValuePaths(value) {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid use of ephemeral value",
			"Ephemeral values are not valid in resource arguments, because resource instances must persist between Terraform phases.",
			path,
		))
	}
	return diags
}
