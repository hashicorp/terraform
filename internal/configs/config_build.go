// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
)

// FinalizeConfig performs the post-load validation and setup steps that are
// shared by different configuration loaders.
//
// Callers must ensure cfg.Root is set correctly before calling this function.
func FinalizeConfig(cfg *Config, loader MockDataLoader) hcl.Diagnostics {
	var diags hcl.Diagnostics
	if cfg == nil {
		return diags
	}

	// Now that the config is built, connect provider names to all known types
	// for validation.
	providers := cfg.resolveProviderTypes()
	cfg.resolveProviderTypesForTests(providers)

	if cfg.Module != nil && cfg.Module.StateStore != nil {
		stateProviderDiags := cfg.resolveStateStoreProviderType()
		diags = append(diags, stateProviderDiags...)
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
