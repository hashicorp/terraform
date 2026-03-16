// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/hashicorp/terraform/internal/addrs"
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
func BuildConfig(root *Module, walker ModuleWalker, loader MockDataLoader) (*Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	cfg := &Config{
		Module: root,
	}
	cfg.Root = cfg // Root module is self-referential.
	cfg.Children, diags = buildChildModules(cfg, walker)
	diags = append(diags, FinalizeConfig(cfg, walker, loader)...)

	return cfg, diags
}

// FinalizeConfig performs the post-load validation and setup steps that are
// shared by different configuration loaders.
//
// Callers must ensure cfg.Root is set correctly before calling this function.
func FinalizeConfig(cfg *Config, walker ModuleWalker, loader MockDataLoader) hcl.Diagnostics {
	var diags hcl.Diagnostics
	if cfg == nil {
		return diags
	}
	diags = append(diags, buildTestModules(cfg, walker)...)

	// Skip provider resolution if there are any errors, since the provider
	// configurations themselves may not be valid.
	if !diags.HasErrors() {
		// Now that the config is built, we can connect the provider names to all
		// the known types for validation.
		providers := cfg.resolveProviderTypes()
		cfg.resolveProviderTypesForTests(providers)

		if cfg.Module != nil && cfg.Module.StateStore != nil {
			stateProviderDiags := cfg.resolveStateStoreProviderType()
			diags = append(diags, stateProviderDiags...)
		}
	}

	diags = append(diags, validateProviderConfigs(nil, cfg, nil)...)
	diags = append(diags, validateProviderConfigsForTests(cfg)...)

	// Final step, let's side load any external mock data into our test files.
	diags = append(diags, installMockDataFiles(cfg, loader)...)

	return diags
}

func installMockDataFiles(root *Config, loader MockDataLoader) hcl.Diagnostics {
	var diags hcl.Diagnostics

	for _, file := range root.Module.Tests {
		for _, provider := range file.Providers {
			if !provider.Mock {
				// Don't try and process non-mocked providers.
				continue
			}

			data, dataDiags := loader.LoadMockData(provider)
			diags = append(diags, dataDiags...)
			if data != nil {
				// If we loaded some data, then merge the new data into the old
				// data. In this case we expect and accept collisions, so we
				// don't want the merge function warning us about them.
				diags = append(diags, provider.MockData.Merge(data, true)...)
			}
		}
	}

	return diags
}

func buildTestModules(root *Config, walker ModuleWalker) hcl.Diagnostics {
	var diags hcl.Diagnostics

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

			req := ModuleRequest{
				Name:              run.Name,
				Path:              path,
				SourceAddr:        run.Module.Source,
				SourceAddrRange:   run.Module.SourceDeclRange,
				VersionConstraint: run.Module.Version,
				Parent:            root,
				CallRange:         run.Module.DeclRange,
			}

			cfg, modDiags := loadModule(root, &req, walker)
			diags = append(diags, modDiags...)

			if cfg != nil {
				// To get the loader to work, we need to set a bunch of values
				// (like the name, path, and parent) as if the module was being
				// loaded as a child of the root config.
				//
				// In actuality, when this is executed it will be as if the
				// module was the root. So, we'll post-process some things to
				// get it to behave as expected later.

				// First, update the main module for this test run to behave as
				// if it is the root module.
				cfg.Parent = nil

				// Then we need to update the paths for this config and all
				// children, so they think they are all relative to the root
				// module we just created.
				rebaseChildModule(cfg, cfg)

				// Finally, link the new config back into our test run so
				// it can be retrieved later.
				run.ConfigUnderTest = cfg
			}
		}
	}

	return diags
}

// legacySourceHelper is used to decode module sources from the old-style
// string-only "source". It assumes that the expression does not contain any
// references and can be decoded without an evaluation context.
// In the long term, we want to get rid of this helper method.
func legacySourceHelper(expr hcl.Expression, haveVersionArg bool) (addrs.ModuleSource, hcl.Diagnostics) {
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

// legacyVersionHelper is used to decode version constraints from the old-style
// string-only "version". It assumes that the expression does not contain any
// references and can be decoded without an evaluation context.
// In the long term, we want to get rid of this helper method.
func legacyVersionHelper(expr hcl.Expression) (VersionConstraint, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var versionRaw string

	ret := VersionConstraint{
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

func buildChildModules(parent *Config, walker ModuleWalker) (map[string]*Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ret := map[string]*Config{}

	calls := parent.Module.ModuleCalls

	// We'll sort the calls by their local names so that they'll appear in a
	// predictable order in any logging that's produced during the walk.
	for _, callName := range slices.Sorted(maps.Keys(calls)) {
		call := calls[callName]
		path := slices.Clone(parent.Path)
		path = append(path, call.Name)

		sourceAddr, sourceDiags := legacySourceHelper(call.SourceExpr, call.VersionExpr != nil)
		diags = append(diags, sourceDiags...)
		if sourceDiags.HasErrors() {
			continue
		}

		var versionConstraint VersionConstraint
		if call.VersionExpr != nil {
			var versionDiags hcl.Diagnostics
			versionConstraint, versionDiags = legacyVersionHelper(call.VersionExpr)
			diags = append(diags, versionDiags...)
			if versionDiags.HasErrors() {
				continue
			}
		}

		req := ModuleRequest{
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

func loadModule(root *Config, req *ModuleRequest, walker ModuleWalker) (*Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	mod, ver, modDiags := walker.LoadModule(req)
	diags = append(diags, modDiags...)
	if mod == nil {
		// nil can be returned if the source address was invalid and so
		// nothing could be loaded whatsoever. LoadModule should've
		// returned at least one error diagnostic in that case.
		return nil, diags
	}

	cfg := &Config{
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

	if len(mod.Import) > 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import configuration",
			Detail:   fmt.Sprintf("An import block was detected in %q. Import blocks are only allowed in the root module.", cfg.Path),
			Subject:  mod.Import[0].DeclRange.Ptr(),
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

// rebaseChildModule updates cfg to make it act as if root is the base of the
// module tree.
//
// This is used for modules loaded directly from test files. In order to load
// them properly, and reuse the code for loading modules from normal
// configuration files, we pretend they are children of the main configuration
// object. Later, when it comes time for them to execute they will act as if
// they are the root module directly.
//
// This function updates cfg so that it treats the provided root as the actual
// root of this module tree. It then recurses into all the child modules and
// does the same for them.
func rebaseChildModule(cfg *Config, root *Config) {
	for _, child := range cfg.Children {
		rebaseChildModule(child, root)
	}

	cfg.Path = cfg.Path[len(root.Path):]
	cfg.Root = root
}

// A ModuleWalker knows how to find and load a child module given details about
// the module to be loaded and a reference to its partially-loaded parent
// Config.
type ModuleWalker interface {
	// LoadModule finds and loads a requested child module.
	//
	// If errors are detected during loading, implementations should return them
	// in the diagnostics object. If the diagnostics object contains any errors
	// then the caller will tolerate the returned module being nil or incomplete.
	// If no errors are returned, it should be non-nil and complete.
	//
	// Full validation need not have been performed but an implementation should
	// ensure that the basic file- and module-validations performed by the
	// LoadConfigDir function (valid syntax, no namespace collisions, etc) have
	// been performed before returning a module.
	LoadModule(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics)
}

// ModuleWalkerFunc is an implementation of ModuleWalker that directly wraps
// a callback function, for more convenient use of that interface.
type ModuleWalkerFunc func(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics)

// LoadModule implements ModuleWalker.
func (f ModuleWalkerFunc) LoadModule(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics) {
	return f(req)
}

// ModuleRequest is used with the ModuleWalker interface to describe a child
// module that must be loaded.
type ModuleRequest struct {
	// Name is the "logical name" of the module call within configuration.
	// This is provided in case the name is used as part of a storage key
	// for the module, but implementations must otherwise treat it as an
	// opaque string. It is guaranteed to have already been validated as an
	// HCL identifier and UTF-8 encoded.
	Name string

	// Path is a list of logical names that traverse from the root module to
	// this module. This can be used, for example, to form a lookup key for
	// each distinct module call in a configuration, allowing for multiple
	// calls with the same name at different points in the tree.
	Path addrs.Module

	// SourceAddr is the source address string provided by the user in
	// configuration.
	SourceAddr addrs.ModuleSource

	// SourceAddrRange is the source range for the SourceAddr value as it
	// was provided in configuration. This can and should be used to generate
	// diagnostics about the source address having invalid syntax, referring
	// to a non-existent object, etc.
	SourceAddrRange hcl.Range

	// VersionConstraint is the version constraint applied to the module in
	// configuration. This data structure includes the source range for
	// the constraint, which can and should be used to generate diagnostics
	// about constraint-related issues, such as constraints that eliminate all
	// available versions of a module whose source is otherwise valid.
	VersionConstraint VersionConstraint

	// Parent is the partially-constructed module tree node that the loaded
	// module will be added to. Callers may refer to any field of this
	// structure except Children, which is still under construction when
	// ModuleRequest objects are created and thus has undefined content.
	// The main reason this is provided is so that full module paths can
	// be constructed for uniqueness.
	Parent *Config

	// CallRange is the source range for the header of the "module" block
	// in configuration that prompted this request. This can be used as the
	// subject of an error diagnostic that relates to the module call itself,
	// rather than to either its source address or its version number.
	CallRange hcl.Range
}

// DisabledModuleWalker is a ModuleWalker that doesn't support
// child modules at all, and so will return an error if asked to load one.
//
// This is provided primarily for testing. There is no good reason to use this
// in the main application.
var DisabledModuleWalker ModuleWalker

func init() {
	DisabledModuleWalker = ModuleWalkerFunc(func(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics) {
		return nil, nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Child modules are not supported",
				Detail:   "Child module calls are not allowed in this context.",
				Subject:  &req.CallRange,
			},
		}
	})
}

// MockDataLoader provides an interface similar to loading modules, except it loads
// and returns MockData objects for the testing framework to consume.
type MockDataLoader interface {
	// LoadMockData accepts a path to a local directory that should contain a
	// set of .tfmock.hcl files that contain mock data that can be consumed by
	// a mock provider within the tewting framework.
	LoadMockData(provider *Provider) (*MockData, hcl.Diagnostics)
}

// MockDataLoaderFunc is an implementation of MockDataLoader that wraps a
// callback function, for more convenient use of that interface.
type MockDataLoaderFunc func(provider *Provider) (*MockData, hcl.Diagnostics)

// LoadMockData implements MockDataLoader.
func (f MockDataLoaderFunc) LoadMockData(provider *Provider) (*MockData, hcl.Diagnostics) {
	return f(provider)
}
