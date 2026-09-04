// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackmodule

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

// BuildConfig constructs a Config from a root module by loading all of its
// descendant modules via the given ModuleWalker. This function also side loads
// and installs any mock data files needed by the testing framework via the
// MockDataLoader.
//
// The result is a module tree that has so far only had basic module- and
// file-level invariants validated. If the returned diagnostics contains errors,
// the returned module tree may be incomplete but can still be used carefully
// for static analysis.
func BuildConfig(root *configs.Module, walker configs.ModuleWalker, loader configs.MockDataLoader) (*configs.Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	cfg := &configs.Config{
		Module: root,
	}
	cfg.Root = cfg // Root module is self-referential.
	cfg.Children, diags = buildChildModules(cfg, walker)
	diags = append(diags, configs.FinalizeConfig(cfg, loader)...)

	return cfg, diags
}

// sourceHelper is used to decode module sources from the old-style
// string-only "source". It assumes that the expression does not contain any
// references and can be decoded without an evaluation context.
// In the long term, we want to get rid of this helper method.
func sourceHelper(expr hcl.Expression, haveVersionArg bool) (addrs.ModuleSource, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var sourceAddrRaw string
	var addr addrs.ModuleSource

	valDiags := gohcl.DecodeExpression(expr, nil, &sourceAddrRaw)
	diags = append(diags, valDiags...)
	if !valDiags.HasErrors() {
		var err error
		if haveVersionArg {
			addr, err = moduleaddrs.ParseModuleSourceRegistry(sourceAddrRaw)
		} else {
			addr, err = moduleaddrs.ParseModuleSource(sourceAddrRaw)
		}
		if err != nil {
			// NOTE: We leave addr as nil for any situation where the
			// source attribute is invalid, so any code which tries to carefully
			// use the partial result of a failed config decode must be
			// resilient to that.
			addr = nil

			// NOTE: In practice it's actually very unlikely to end up here,
			// because our source address parser can turn just about any string
			// into some sort of remote package address, and so for most errors
			// we'll detect them only during module installation. There are
			// still a _few_ purely-syntax errors we can catch at parsing time,
			// though, mostly related to remote package sub-paths and local
			// paths.
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
					// In this case we'll include some extra context that
					// we assumed a registry source address due to the
					// version argument.
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
		}
	}

	return addr, diags
}

// versionHelper is used to decode version constraints from the old-style
// string-only "version". It assumes that the expression does not contain any
// references and can be decoded without an evaluation context.
// In the long term, we want to get rid of this helper method.
func versionHelper(expr hcl.Expression) (configs.VersionConstraint, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var versionRaw string

	ret := configs.VersionConstraint{
		DeclRange: expr.Range(),
	}

	valDiags := gohcl.DecodeExpression(expr, nil, &versionRaw)
	diags = append(diags, valDiags...)
	if !valDiags.HasErrors() {
		constraints, err := version.NewConstraint(versionRaw)
		if err != nil {
			// NewConstraint doesn't return user-friendly errors, so we'll just
			// ignore the provided error and produce our own generic one.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid version constraint",
				Detail:   "This string does not use correct version constraint syntax.", // Not very actionable :(
				Subject:  expr.Range().Ptr(),
			})
			return ret, diags
		}
		ret.Required = constraints
	}

	return ret, diags
}

func buildChildModules(parent *configs.Config, walker configs.ModuleWalker) (map[string]*configs.Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ret := map[string]*configs.Config{}

	calls := parent.Module.ModuleCalls

	// We'll sort the calls by their local names so that they'll appear in a
	// predictable order in any logging that's produced during the walk.
	for _, callName := range slices.Sorted(maps.Keys(calls)) {
		call := calls[callName]
		path := slices.Clone(parent.Path)
		path = append(path, call.Name)

		sourceAddr, sourceDiags := sourceHelper(call.SourceExpr, call.VersionExpr != nil)
		diags = append(diags, sourceDiags...)
		if sourceDiags.HasErrors() {
			continue
		}

		var versionConstraint configs.VersionConstraint
		if call.VersionExpr != nil {
			var versionDiags hcl.Diagnostics
			versionConstraint, versionDiags = versionHelper(call.VersionExpr)
			diags = append(diags, versionDiags...)
			if versionDiags.HasErrors() {
				continue
			}
		}

		req := configs.ModuleRequest{
			Name:              call.Name,
			Path:              path,
			SourceAddr:        sourceAddr,
			SourceAddrRange:   call.SourceExpr.Range(),
			VersionConstraint: versionConstraint,
			Parent:            parent,
			CallRange:         call.DeclRange,
		}
		child, modDiags := loadModule(parent.Root, &req, walker)
		diags = append(diags, modDiags...)
		if child == nil {
			// This means an error occurred, there should be diagnostics within
			// modDiags for this.
			continue
		}

		ret[call.Name] = child
	}

	return ret, diags
}

func loadModule(root *configs.Config, req *configs.ModuleRequest, walker configs.ModuleWalker) (*configs.Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	mod, ver, modDiags := walker.LoadModule(req)
	diags = append(diags, modDiags...)
	if mod == nil {
		// nil can be returned if the source address was invalid and so
		// nothing could be loaded whatsoever. LoadModule should've
		// returned at least one error diagnostic in that case.
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

	cfg.Children, modDiags = buildChildModules(cfg, walker)
	diags = append(diags, modDiags...)

	if mod.Backend != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Backend configuration ignored",
			Detail:   "Any selected backend applies to the entire configuration, so Terraform expects backend configurations only in the root module.\n\nThis is a warning rather than an error because it's sometimes convenient to temporarily call a root module as a child module for testing purposes, but this backend configuration block will have no effect.",
			Subject:  mod.Backend.DeclRange.Ptr(),
		})
	}

	if mod.CloudConfig != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Cloud configuration ignored",
			Detail:   "A cloud configuration block applies to the entire configuration, so Terraform expects 'cloud' blocks to only be in the root module.\n\nThis is a warning rather than an error because it's sometimes convenient to temporarily call a root module as a child module for testing purposes, but this cloud configuration block will have no effect.",
			Subject:  mod.CloudConfig.DeclRange.Ptr(),
		})
	}

	if len(mod.ListResources) > 0 {
		first := slices.Collect(maps.Values(mod.ListResources))[0]
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid list configuration",
			Detail:   fmt.Sprintf("A list block was detected in %q. List blocks are only allowed in the root module.", cfg.Path),
			Subject:  first.DeclRange.Ptr(),
		})
	}

	return cfg, diags
}
