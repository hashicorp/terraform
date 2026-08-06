// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"path"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// BuildConfigWithGraph builds a configuration tree using the init graph so
// that module sources and versions can be resolved with full expression
// evaluation before loading descendant modules.
func BuildConfigWithGraph(rootMod *configs.Module, walker configs.ModuleWalker, vars InputValues, loader configs.MockDataLoader) (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	cfg, initDiags := initConfigWithGraph(rootMod, walker, vars, nil)
	diags = diags.Append(initDiags)
	if diags.HasErrors() {
		if cfg == nil && rootMod != nil {
			cfg = &configs.Config{Module: rootMod}
			cfg.Root = cfg
		}
		return cfg, diags
	}

	testDiags := loadTestModulesWithGraph(cfg, walker, vars)
	diags = diags.Append(testDiags)

	finalDiags := configs.FinalizeConfig(cfg, loader)
	diags = diags.Append(finalDiags)

	return cfg, diags
}

func loadTestModulesWithGraph(root *configs.Config, walker configs.ModuleWalker, vars InputValues) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for name, file := range root.Module.Tests {
		for _, run := range file.Runs {
			if run.Module == nil {
				continue
			}

			// We want to make sure the path for the testing modules are unique
			// so we create a dedicated path for them.
			//
			// Some examples:
			//    - file: main.tftest.hcl, run: setup - test.main.setup
			//    - file: tests/main.tftest.hcl, run: setup - test.tests.main.setup

			dir := path.Dir(name)
			base := path.Base(name)

			path := addrs.Module{}
			path = append(path, "test")
			if dir != "." {
				path = append(path, strings.Split(dir, "/")...)
			}
			path = append(path, strings.TrimSuffix(base, ".tftest.hcl"), run.Name)

			req := &configs.ModuleRequest{
				Name:              run.Name,
				Path:              path,
				SourceAddr:        run.Module.Source,
				SourceAddrRange:   run.Module.SourceDeclRange,
				VersionConstraint: run.Module.Version,
				Parent:            root,
				CallRange:         run.Module.DeclRange,
			}

			rootMod, version, loadDiags := walker.LoadModule(req)
			diags = diags.Append(loadDiags)
			if loadDiags.HasErrors() || rootMod == nil {
				continue
			}

			testCfg, testDiags := initConfigWithGraph(rootMod, walker, vars, path)
			diags = diags.Append(testDiags)
			if testCfg != nil {
				testCfg.CallRange = req.CallRange
				testCfg.SourceAddr = req.SourceAddr
				testCfg.SourceAddrRange = req.SourceAddrRange
				testCfg.Version = version
				run.ConfigUnderTest = testCfg
			}
		}
	}

	return diags
}

func initConfigWithGraph(rootMod *configs.Module, walker configs.ModuleWalker, vars InputValues, modulePathPrefix addrs.Module) (*configs.Config, tfdiags.Diagnostics) {
	ctx, ctxDiags := NewContext(&ContextOpts{
		Parallelism: 1,
	})
	if ctxDiags.HasErrors() {
		return nil, ctxDiags
	}

	return ctx.Init(rootMod, InitOpts{
		Walker:           walker,
		SetVariables:     vars,
		ModulePathPrefix: modulePathPrefix,
	})
}
