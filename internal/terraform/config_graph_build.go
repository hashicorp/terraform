// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// BuildConfigWithGraph builds a configuration tree using the init graph so
// that module sources and versions can be resolved with full expression
// evaluation before loading descendant modules.
func BuildConfigWithGraph(rootMod *configs.Module, walker configs.ModuleWalker, vars InputValues, loader configs.MockDataLoader) (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Resolve any dynamic provider sources that reference const variables
	// before building the config tree, so that all provider types are
	// available for resource-to-provider mapping and validation.
	diags = diags.Append(resolveModuleProviderSources(rootMod, vars))
	if diags.HasErrors() {
		return nil, diags
	}

	ctx, ctxDiags := NewContext(&ContextOpts{
		Parallelism: 1,
	})
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return nil, diags
	}

	cfg, initDiags := ctx.Init(rootMod, InitOpts{
		Walker:       walker,
		SetVariables: vars,
	})
	diags = diags.Append(initDiags)
	if diags.HasErrors() {
		if cfg == nil && rootMod != nil {
			cfg = &configs.Config{Module: rootMod}
			cfg.Root = cfg
		}
		return cfg, diags
	}

	finalDiags := configs.FinalizeConfig(cfg, walker, loader)
	diags = diags.Append(finalDiags)

	return cfg, diags
}

// resolveModuleProviderSources resolves any dynamic provider source/version
// expressions in the module's required_providers block using const variable
// values extracted from the given InputValues.
func resolveModuleProviderSources(mod *configs.Module, vars InputValues) hcl.Diagnostics {
	if mod == nil || mod.ProviderRequirements == nil {
		return nil
	}

	ctx := constVarEvalContext(vars, mod.Variables)
	if ctx == nil {
		return nil
	}

	resolveDiags := mod.ProviderRequirements.ResolveProviderSources(ctx)
	if !resolveDiags.HasErrors() {
		// Re-assign resource Provider fields now that provider types
		// are resolved, since they were initially set before dynamic
		// provider resolution occurred.
		mod.ResolveResourceProviders()
	}
	return resolveDiags
}

// constVarEvalContext builds a hcl.EvalContext containing only the values
// of const variables. Returns nil if there are no const variables with values.
func constVarEvalContext(vars InputValues, decls map[string]*configs.Variable) *hcl.EvalContext {
	constVals := make(map[string]cty.Value)
	for name, decl := range decls {
		if !decl.Const {
			continue
		}
		if iv, ok := vars[name]; ok && iv.Value != cty.NilVal {
			constVals[name] = iv.Value
		} else if decl.Default != cty.NilVal {
			constVals[name] = decl.Default
		}
	}
	if len(constVals) == 0 {
		return nil
	}
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(constVals),
		},
	}
}
