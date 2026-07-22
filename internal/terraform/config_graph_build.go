// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"slices"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// BuildConfigWithGraph builds a configuration tree using the init graph so
// that module sources and versions can be resolved with full expression
// evaluation before loading descendant modules.
func BuildConfigWithGraph(rootMod *configs.Module, walker configs.ModuleWalker, vars InputValues, loader configs.MockDataLoader) (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

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

	testModuleLoader := func(root *configs.Config, req *configs.ModuleRequest) (*configs.Config, hcl.Diagnostics) {
		rootMod, _, loadDiags := walker.LoadModule(req)
		if loadDiags.HasErrors() || rootMod == nil {
			return nil, loadDiags
		}

		prefix := slices.Clone(req.Path)
		prefixedWalker := configs.ModuleWalkerFunc(func(childReq *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
			prefixedReq := *childReq
			prefixedReq.Path = append(slices.Clone(prefix), childReq.Path...)
			if childReq.Parent != nil {
				parent := *childReq.Parent
				parent.Path = append(slices.Clone(prefix), childReq.Parent.Path...)
				prefixedReq.Parent = &parent
			}
			return walker.LoadModule(&prefixedReq)
		})

		testCfg, testDiags := initConfigWithGraph(rootMod, prefixedWalker, vars)
		loadDiags = append(loadDiags, testDiags.ToHCL()...)
		return testCfg, loadDiags
	}

	finalDiags := configs.FinalizeConfigWithTestModuleLoader(cfg, walker, loader, testModuleLoader)
	diags = diags.Append(finalDiags)

	return cfg, diags
}

func initConfigWithGraph(rootMod *configs.Module, walker configs.ModuleWalker, vars InputValues) (*configs.Config, tfdiags.Diagnostics) {
	ctx, ctxDiags := NewContext(&ContextOpts{Parallelism: 1})
	if ctxDiags.HasErrors() {
		return nil, ctxDiags
	}
	return ctx.Init(rootMod, InitOpts{Walker: walker, SetVariables: vars})
}
