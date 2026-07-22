// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package staticmodule loads the static module trees used by stack components.
// Stacks do not support dynamic module source or version expressions.
package staticmodule

import (
	"fmt"
	"maps"
	"slices"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/getmodules/moduleaddrs"
)

// BuildConfig constructs a static configuration tree by loading descendant
// modules through walker. Module source and version expressions must be
// literal values because stacks do not perform early variable evaluation.
func BuildConfig(root *configs.Module, walker configs.ModuleWalker) (*configs.Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	cfg := &configs.Config{
		Module: root,
	}
	cfg.Root = cfg
	cfg.Children, diags = buildChildModules(cfg, walker)
	diags = append(diags, configs.FinalizeConfig(cfg, nil)...)
	return cfg, diags
}

func buildChildModules(parent *configs.Config, walker configs.ModuleWalker) (map[string]*configs.Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ret := map[string]*configs.Config{}

	for _, callName := range slices.Sorted(maps.Keys(parent.Module.ModuleCalls)) {
		call := parent.Module.ModuleCalls[callName]
		modulePath := append(slices.Clone(parent.Path), call.Name)

		sourceAddr, sourceDiags := decodeSource(call.SourceExpr, call.VersionExpr != nil)
		diags = append(diags, sourceDiags...)
		if sourceDiags.HasErrors() {
			continue
		}

		var versionConstraint configs.VersionConstraint
		if call.VersionExpr != nil {
			var versionDiags hcl.Diagnostics
			versionConstraint, versionDiags = decodeVersion(call.VersionExpr)
			diags = append(diags, versionDiags...)
			if versionDiags.HasErrors() {
				continue
			}
		}

		req := &configs.ModuleRequest{
			Name:              call.Name,
			Path:              modulePath,
			SourceAddr:        sourceAddr,
			SourceAddrRange:   call.SourceExpr.Range(),
			VersionConstraint: versionConstraint,
			Parent:            parent,
			CallRange:         call.DeclRange,
		}
		child, moduleDiags := loadModule(parent.Root, req, walker)
		diags = append(diags, moduleDiags...)
		if child != nil {
			ret[call.Name] = child
		}
	}

	return ret, diags
}

func loadModule(root *configs.Config, req *configs.ModuleRequest, walker configs.ModuleWalker) (*configs.Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	mod, ver, moduleDiags := walker.LoadModule(req)
	diags = append(diags, moduleDiags...)
	if mod == nil {
		return nil, diags
	}

	cfg := &configs.Config{
		Parent:          req.Parent,
		Root:            root,
		Path:            req.Path,
		Module:          mod,
		CallRange:       req.CallRange,
		SourceAddr:      req.SourceAddr,
		SourceAddrRange: req.SourceAddrRange,
		Version:         ver,
	}
	cfg.Children, moduleDiags = buildChildModules(cfg, walker)
	diags = append(diags, moduleDiags...)

	if mod.Backend != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Backend configuration ignored",
			Detail:   "Any selected backend applies to the entire configuration, so Terraform expects backend configurations only in the root module.\n\nThis is a warning rather than an error because it's sometimes convenient to temporarily call a root module as a child module for testing purposes, but this backend configuration block will have no effect.",
			Subject:  mod.Backend.DeclRange.Ptr(),
		})
	}

	if mod.CloudConfig != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Cloud configuration ignored",
			Detail:   "A cloud configuration block applies to the entire configuration, so Terraform expects 'cloud' blocks to only be in the root module.\n\nThis is a warning rather than an error because it's sometimes convenient to temporarily call a root module as a child module for testing purposes, but this cloud configuration block will have no effect.",
			Subject:  mod.CloudConfig.DeclRange.Ptr(),
		})
	}

	if len(mod.ListResources) > 0 {
		first := slices.Collect(maps.Values(mod.ListResources))[0]
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid list configuration",
			Detail:   fmt.Sprintf("A list block was detected in %q. List blocks are only allowed in the root module.", cfg.Path),
			Subject:  first.DeclRange.Ptr(),
		})
	}

	return cfg, diags
}

func decodeSource(expr hcl.Expression, haveVersionArg bool) (ret addrs.ModuleSource, diags hcl.Diagnostics) {
	var sourceAddrRaw string
	valDiags := gohcl.DecodeExpression(expr, nil, &sourceAddrRaw)
	diags = append(diags, valDiags...)
	if valDiags.HasErrors() {
		return nil, diags
	}

	var err error
	if haveVersionArg {
		ret, err = moduleaddrs.ParseModuleSourceRegistry(sourceAddrRaw)
	} else {
		ret, err = moduleaddrs.ParseModuleSource(sourceAddrRaw)
	}
	if err == nil {
		return ret, diags
	}

	ret = nil
	switch err := err.(type) {
	case *moduleaddrs.MaybeRelativePathErr:
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid module source address",
			Detail: fmt.Sprintf(
				"Terraform failed to determine your intended installation method for remote module package %q.\n\nIf you intended this as a path relative to the current module, use \"./%s\" instead. The \"./\" prefix indicates that the address is a relative filesystem path.",
				err.Addr, err.Addr,
			),
			Subject: expr.Range().Ptr(),
		})
	default:
		if haveVersionArg {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid registry module source address",
				Detail:   fmt.Sprintf("Failed to parse module registry address: %s.\n\nTerraform assumed that you intended a module registry source address because you also set the argument \"version\", which applies only to registry modules.", err),
				Subject:  expr.Range().Ptr(),
			})
		} else {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid module source address",
				Detail:   fmt.Sprintf("Failed to parse module source address: %s.", err),
				Subject:  expr.Range().Ptr(),
			})
		}
	}
	return ret, diags
}

func decodeVersion(expr hcl.Expression) (configs.VersionConstraint, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var versionRaw string
	ret := configs.VersionConstraint{DeclRange: expr.Range()}

	valDiags := gohcl.DecodeExpression(expr, nil, &versionRaw)
	diags = append(diags, valDiags...)
	if valDiags.HasErrors() {
		return ret, diags
	}

	constraints, err := version.NewConstraint(versionRaw)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid version constraint",
			Detail:   "This string does not use correct version constraint syntax.",
			Subject:  expr.Range().Ptr(),
		})
		return ret, diags
	}
	ret.Required = constraints
	return ret, diags
}
