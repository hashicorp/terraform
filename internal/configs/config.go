// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"log"
	"sort"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
)

// A Config is a node in the tree of modules within a configuration.
//
// The module tree is constructed by following ModuleCall instances recursively
// through the root module transitively into descendant modules.
//
// A module tree described in *this* package represents the static tree
// represented by configuration. During evaluation a static ModuleNode may
// expand into zero or more module instances depending on the use of count and
// for_each configuration attributes within each call.
type Config struct {
	// RootModule points to the Config for the root module within the same
	// module tree as this module. If this module _is_ the root module then
	// this is self-referential.
	Root *Config

	// ParentModule points to the Config for the module that directly calls
	// this module. If this is the root module then this field is nil.
	Parent *Config

	// Path is a sequence of module logical names that traverse from the root
	// module to this config. Path is empty for the root module.
	//
	// This should only be used to display paths to the end-user in rare cases
	// where we are talking about the static module tree, before module calls
	// have been resolved. In most cases, an addrs.ModuleInstance describing
	// a node in the dynamic module tree is better, since it will then include
	// any keys resulting from evaluating "count" and "for_each" arguments.
	Path addrs.Module

	// ChildModules points to the Config for each of the direct child modules
	// called from this module. The keys in this map match the keys in
	// Module.ModuleCalls.
	Children map[string]*Config

	// Module points to the object describing the configuration for the
	// various elements (variables, resources, etc) defined by this module.
	Module *Module

	// CallRange is the source range for the header of the module block that
	// requested this module.
	//
	// This field is meaningless for the root module, where its contents are undefined.
	CallRange hcl.Range

	// SourceAddr is the source address that the referenced module was requested
	// from, as specified in configuration. SourceAddrRaw is the same
	// information, but as the raw string the user originally entered.
	//
	// These fields are meaningless for the root module, where their contents are undefined.
	SourceAddr    addrs.ModuleSource
	SourceAddrRaw string

	// SourceAddrRange is the location in the configuration source where the
	// SourceAddr value was set, for use in diagnostic messages.
	//
	// This field is meaningless for the root module, where its contents are undefined.
	SourceAddrRange hcl.Range

	// Version is the specific version that was selected for this module,
	// based on version constraints given in configuration.
	//
	// This field is nil if the module was loaded from a non-registry source,
	// since versions are not supported for other sources.
	//
	// This field is meaningless for the root module, where it will always
	// be nil.
	Version *version.Version
}

// ModuleRequirements represents the provider requirements for an individual
// module, along with references to any child modules. This is used to
// determine which modules require which providers.
type ModuleRequirements struct {
	Name         string
	SourceAddr   addrs.ModuleSource
	SourceDir    string
	Requirements providerreqs.Requirements
	Children     map[string]*ModuleRequirements
	Tests        map[string]*TestFileModuleRequirements
}

// TestFileModuleRequirements maps the runs for a given test file to the module
// requirements for that run block.
type TestFileModuleRequirements struct {
	Requirements providerreqs.Requirements
	Runs         map[string]*ModuleRequirements
}

// NewEmptyConfig constructs a single-node configuration tree with an empty
// root module. This is generally a pretty useless thing to do, so most callers
// should instead use BuildConfig.
func NewEmptyConfig() *Config {
	ret := &Config{}
	ret.Root = ret
	ret.Children = make(map[string]*Config)
	ret.Module = &Module{}
	return ret
}

// Depth returns the number of "hops" the receiver is from the root of its
// module tree, with the root module having a depth of zero.
func (c *Config) Depth() int {
	ret := 0
	this := c
	for this.Parent != nil {
		ret++
		this = this.Parent
	}
	return ret
}

// DeepEach calls the given function once for each module in the tree, starting
// with the receiver.
//
// A parent is always called before its children and children of a particular
// node are visited in lexicographic order by their names.
func (c *Config) DeepEach(cb func(c *Config)) {
	cb(c)

	names := make([]string, 0, len(c.Children))
	for name := range c.Children {
		names = append(names, name)
	}

	for _, name := range names {
		c.Children[name].DeepEach(cb)
	}
}

// AllModules returns a slice of all the receiver and all of its descendant
// nodes in the module tree, in the same order they would be visited by
// DeepEach.
func (c *Config) AllModules() []*Config {
	var ret []*Config
	c.DeepEach(func(c *Config) {
		ret = append(ret, c)
	})
	return ret
}

// Descendant returns the descendant config that has the given path beneath
// the receiver, or nil if there is no such module.
//
// The path traverses the static module tree, prior to any expansion to handle
// count and for_each arguments.
//
// An empty path will just return the receiver, and is therefore pointless.
func (c *Config) Descendant(path addrs.Module) *Config {
	current := c
	for _, name := range path {
		current = current.Children[name]
		if current == nil {
			return nil
		}
	}
	return current
}

// DescendantForInstance is like Descendant except that it accepts a path
// to a particular module instance in the dynamic module graph, returning
// the node from the static module graph that corresponds to it.
//
// All instances created by a particular module call share the same
// configuration, so the keys within the given path are disregarded.
func (c *Config) DescendantForInstance(path addrs.ModuleInstance) *Config {
	current := c
	for _, step := range path {
		current = current.Children[step.Name]
		if current == nil {
			return nil
		}
	}
	return current
}

// TargetExists returns true if it's possible for the provided target to exist
// within the configuration.
//
// This doesn't consider instance expansion, so we're only making sure the
// target could exist if the instance expansion expands correctly.
func (c *Config) TargetExists(target addrs.Targetable) bool {
	switch target.AddrType() {
	case addrs.ConfigResourceAddrType:
		addr := target.(addrs.ConfigResource)
		module := c.Descendant(addr.Module)
		if module != nil {
			return module.Module.ResourceByAddr(addr.Resource) != nil
		} else {
			return false
		}
	case addrs.AbsResourceInstanceAddrType:
		addr := target.(addrs.AbsResourceInstance)
		module := c.DescendantForInstance(addr.Module)
		if module != nil {
			return module.Module.ResourceByAddr(addr.Resource.Resource) != nil
		} else {
			return false
		}
	case addrs.AbsResourceAddrType:
		addr := target.(addrs.AbsResource)
		module := c.DescendantForInstance(addr.Module)
		if module != nil {
			return module.Module.ResourceByAddr(addr.Resource) != nil
		} else {
			return false
		}
	case addrs.ModuleAddrType:
		return c.Descendant(target.(addrs.Module)) != nil
	case addrs.ModuleInstanceAddrType:
		return c.DescendantForInstance(target.(addrs.ModuleInstance)) != nil
	default:
		panic(fmt.Errorf("unrecognized targetable type: %d", target.AddrType()))
	}
}

// EntersNewPackage returns true if this call is to an external module, either
// directly via a remote source address or indirectly via a registry source
// address.
//
// Other behaviors in Terraform may treat package crossings as a special
// situation, because that indicates that the caller and callee can change
// independently of one another and thus we should disallow using any features
// where the caller assumes anything about the callee other than its input
// variables, required provider configurations, and output values.
//
// It's not meaningful to ask if the Config representing the root module enters
// a new package because the root module is always outside of all module
// packages, and so this function will arbitrarily return false in that case.
func (c *Config) EntersNewPackage() bool {
	return moduleSourceAddrEntersNewPackage(c.SourceAddr)
}

// VerifyDependencySelections checks whether the given locked dependencies
// are acceptable for all of the version constraints reported in the
// configuration tree represented by the reciever.
//
// This function will errors only if any of the locked dependencies are out of
// range for corresponding constraints in the configuration. If there are
// multiple inconsistencies then it will attempt to describe as many of them
// as possible, rather than stopping at the first problem.
//
// It's typically the responsibility of "terraform init" to change the locked
// dependencies to conform with the configuration, and so
// VerifyDependencySelections is intended for other commands to check whether
// it did so correctly and to catch if anything has changed in configuration
// since the last "terraform init" which requires re-initialization. However,
// it's up to the caller to decide how to advise users recover from these
// errors, because the advise can vary depending on what operation the user
// is attempting.
func (c *Config) VerifyDependencySelections(depLocks *depsfile.Locks) []error {
	var errs []error

	reqs, diags := c.ProviderRequirements()
	if diags.HasErrors() {
		// It should be very unusual to get here, but unfortunately we can
		// end up here in some edge cases where the config loader doesn't
		// process version constraint strings in exactly the same way as
		// the requirements resolver. (See the addProviderRequirements method
		// for more information.)
		errs = append(errs, fmt.Errorf("failed to determine the configuration's provider requirements: %s", diags.Error()))
	}

	for providerAddr, constraints := range reqs {
		if !depsfile.ProviderIsLockable(providerAddr) {
			continue // disregard builtin providers, and such
		}
		if depLocks != nil && depLocks.ProviderIsOverridden(providerAddr) {
			// The "overridden" case is for unusual special situations like
			// dev overrides, so we'll explicitly note it in the logs just in
			// case we see bug reports with these active and it helps us
			// understand why we ended up using the "wrong" plugin.
			log.Printf("[DEBUG] Config.VerifyDependencySelections: skipping %s because it's overridden by a special configuration setting", providerAddr)
			continue
		}

		var lock *depsfile.ProviderLock
		if depLocks != nil { // Should always be true in main code, but unfortunately sometimes not true in old tests that don't fill out arguments completely
			lock = depLocks.Provider(providerAddr)
		}
		if lock == nil {
			log.Printf("[TRACE] Config.VerifyDependencySelections: provider %s has no lock file entry to satisfy %q", providerAddr, providerreqs.VersionConstraintsString(constraints))
			errs = append(errs, fmt.Errorf("provider %s: required by this configuration but no version is selected", providerAddr))
			continue
		}

		selectedVersion := lock.Version()
		allowedVersions := providerreqs.MeetingConstraints(constraints)
		log.Printf("[TRACE] Config.VerifyDependencySelections: provider %s has %s to satisfy %q", providerAddr, selectedVersion.String(), providerreqs.VersionConstraintsString(constraints))
		if !allowedVersions.Has(selectedVersion) {
			// The most likely cause of this is that the author of a module
			// has changed its constraints, but this could also happen in
			// some other unusual situations, such as the user directly
			// editing the lock file to record something invalid. We'll
			// distinguish those cases here in order to avoid the more
			// specific error message potentially being a red herring in
			// the edge-cases.
			currentConstraints := providerreqs.VersionConstraintsString(constraints)
			lockedConstraints := providerreqs.VersionConstraintsString(lock.VersionConstraints())
			switch {
			case currentConstraints != lockedConstraints:
				errs = append(errs, fmt.Errorf("provider %s: locked version selection %s doesn't match the updated version constraints %q", providerAddr, selectedVersion.String(), currentConstraints))
			default:
				errs = append(errs, fmt.Errorf("provider %s: version constraints %q don't match the locked version selection %s", providerAddr, currentConstraints, selectedVersion.String()))
			}
		}
	}

	// Return multiple errors in an arbitrary-but-deterministic order.
	sort.Slice(errs, func(i, j int) bool {
		return errs[i].Error() < errs[j].Error()
	})

	return errs
}

// ProviderRequirements searches the full tree of modules under the receiver
// for both explicit and implicit dependencies on providers.
//
// The result is a full manifest of all of the providers that must be available
// in order to work with the receiving configuration.
//
// If the returned diagnostics includes errors then the resulting Requirements
// may be incomplete.
func (c *Config) ProviderRequirements() (providerreqs.Requirements, hcl.Diagnostics) {
	reqs := make(providerreqs.Requirements)
	diags := c.addProviderRequirements(reqs, true, true)

	return reqs, diags
}

// ProviderRequirementsConfigOnly searches the full tree of configuration
// files for all providers. This function does not consider any test files.
func (c *Config) ProviderRequirementsConfigOnly() (providerreqs.Requirements, hcl.Diagnostics) {
	reqs := make(providerreqs.Requirements)
	diags := c.addProviderRequirements(reqs, true, false)

	return reqs, diags
}

// ProviderRequirementsShallow searches only the direct receiver for explicit
// and implicit dependencies on providers. Descendant modules are ignored.
//
// If the returned diagnostics includes errors then the resulting Requirements
// may be incomplete.
func (c *Config) ProviderRequirementsShallow() (providerreqs.Requirements, hcl.Diagnostics) {
	reqs := make(providerreqs.Requirements)
	diags := c.addProviderRequirements(reqs, false, true)

	return reqs, diags
}

// ProviderRequirementsByModule searches the full tree of modules under the
// receiver for both explicit and implicit dependencies on providers,
// constructing a tree where the requirements are broken out by module.
//
// If the returned diagnostics includes errors then the resulting Requirements
// may be incomplete.
func (c *Config) ProviderRequirementsByModule() (*ModuleRequirements, hcl.Diagnostics) {
	reqs := make(providerreqs.Requirements)
	diags := c.addProviderRequirements(reqs, false, false)

	children := make(map[string]*ModuleRequirements)
	for name, child := range c.Children {
		childReqs, childDiags := child.ProviderRequirementsByModule()
		childReqs.Name = name
		children[name] = childReqs
		diags = append(diags, childDiags...)
	}

	tests := make(map[string]*TestFileModuleRequirements)
	for name, test := range c.Module.Tests {
		testReqs := &TestFileModuleRequirements{
			Requirements: make(providerreqs.Requirements),
			Runs:         make(map[string]*ModuleRequirements),
		}

		for _, run := range test.Runs {
			if run.ConfigUnderTest == nil {
				continue
			}

			runReqs, runDiags := run.ConfigUnderTest.ProviderRequirementsByModule()
			runReqs.Name = run.Name
			testReqs.Runs[run.Name] = runReqs
			diags = append(diags, runDiags...)
		}

		tests[name] = testReqs
	}

	ret := &ModuleRequirements{
		SourceAddr:   c.SourceAddr,
		SourceDir:    c.Module.SourceDir,
		Requirements: reqs,
		Children:     children,
		Tests:        tests,
	}

	return ret, diags
}

// addProviderRequirements is the main part of the ProviderRequirements
// implementation, gradually mutating a shared requirements object to
// eventually return. If the recurse argument is true, the requirements will
// include all descendant modules; otherwise, only the specified module.
func (c *Config) addProviderRequirements(reqs providerreqs.Requirements, recurse, tests bool) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// First we'll deal with the requirements directly in _our_ module...
	if c.Module.ProviderRequirements != nil {
		for _, providerReqs := range c.Module.ProviderRequirements.RequiredProviders {
			fqn := providerReqs.Type
			if _, ok := reqs[fqn]; !ok {
				// We'll at least have an unconstrained dependency then, but might
				// add to this in the loop below.
				reqs[fqn] = nil
			}
			// The model of version constraints in this package is still the
			// old one using a different upstream module to represent versions,
			// so we'll need to shim that out here for now. The two parsers
			// don't exactly agree in practice 🙄 so this might produce new errors.
			// TODO: Use the new parser throughout this package so we can get the
			// better error messages it produces in more situations.
			constraints, err := providerreqs.ParseVersionConstraints(providerReqs.Requirement.Required.String())
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid version constraint",
					// The errors returned by ParseVersionConstraint already include
					// the section of input that was incorrect, so we don't need to
					// include that here.
					Detail:  fmt.Sprintf("Incorrect version constraint syntax: %s.", err.Error()),
					Subject: providerReqs.Requirement.DeclRange.Ptr(),
				})
			}
			reqs[fqn] = append(reqs[fqn], constraints...)
		}
	}

	// Each resource in the configuration creates an *implicit* provider
	// dependency, though we'll only record it if there isn't already
	// an explicit dependency on the same provider.
	for _, rc := range c.Module.ManagedResources {
		fqn := rc.Provider
		if _, exists := reqs[fqn]; exists {
			// Explicit dependency already present
			continue
		}
		reqs[fqn] = nil
	}

	for _, rc := range c.Module.DataResources {
		fqn := rc.Provider
		if _, exists := reqs[fqn]; exists {
			// Explicit dependency already present
			continue
		}
		reqs[fqn] = nil
	}

	for _, rc := range c.Module.EphemeralResources {
		fqn := rc.Provider
		if _, exists := reqs[fqn]; exists {
			// Explicit dependency already present
			continue
		}
		reqs[fqn] = nil
	}

	// Import blocks that are generating config may have a custom provider
	// meta-argument. Like the provider meta-argument used in resource blocks,
	// we use this opportunity to load any implicit providers.
	//
	// We'll also use this to validate that import blocks and targeted resource
	// blocks agree on which provider they should be using. If they don't agree,
	// this will be because the user has written explicit provider arguments
	// that don't agree and we'll get them to fix it.
	for _, i := range c.Module.Import {
		if len(i.ToResource.Module) > 0 {
			// All provider information for imports into modules should come
			// from the module block, so we don't need to load anything for
			// import targets within modules.
			continue
		}

		if target, exists := c.Module.ManagedResources[i.ToResource.Resource.String()]; exists {
			// This means the information about the provider for this import
			// should come from the resource block itself and not the import
			// block.
			//
			// In general, we say that you shouldn't set the provider attribute
			// on import blocks in this case. But to make config generation
			// easier, we will say that if it is set in both places and it's the
			// same then that is okay.

			if i.ProviderConfigRef != nil {
				if target.ProviderConfigRef == nil {
					// This means we have a provider specified in the import
					// block and not in the resource block. This isn't the right
					// way round so let's consider this a failure.
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid import provider argument",
						Detail:   "The provider argument can only be specified in import blocks that will generate configuration.\n\nUse the provider argument in the target resource block to configure the provider for a resource with explicit provider configuration.",
						Subject:  i.ProviderDeclRange.Ptr(),
					})
					continue
				}

				if i.ProviderConfigRef.Name != target.ProviderConfigRef.Name || i.ProviderConfigRef.Alias != target.ProviderConfigRef.Alias {
					// This means we have a provider specified in both the
					// import block and the resource block, and they disagree.
					// This is bad as Terraform now has different instructions
					// about which provider to use.
					//
					// The general guidance is that only the resource should be
					// specifying the provider as the import block provider
					// attribute is just for generating config. So, let's just
					// tell the user to only set the provider argument in the
					// resource.
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid import provider argument",
						Detail:   "The provider argument can only be specified in import blocks that will generate configuration.\n\nUse the provider argument in the target resource block to configure the provider for a resource with explicit provider configuration.",
						Subject:  i.ProviderDeclRange.Ptr(),
					})
					continue
				}
			}

			// All the provider information should come from the target resource
			// which has already been processed, so skip the rest of this
			// processing.
			continue
		}

		// Otherwise we are generating config for the resource being imported,
		// so all the provider information must come from this import block.
		fqn := i.Provider
		if _, exists := reqs[fqn]; exists {
			// Explicit dependency already present
			continue
		}
		reqs[fqn] = nil
	}

	// "provider" block can also contain version constraints
	for _, provider := range c.Module.ProviderConfigs {
		moreDiags := c.addProviderRequirementsFromProviderBlock(reqs, provider)
		diags = append(diags, moreDiags...)
	}

	// We may have provider blocks and required_providers set in some testing
	// files.
	if tests && recurse {
		for _, file := range c.Module.Tests {
			// Then we'll also look for requirements in testing modules.
			for _, run := range file.Runs {
				if run.ConfigUnderTest != nil {
					moreDiags := run.ConfigUnderTest.addProviderRequirements(reqs, true, false)
					diags = append(diags, moreDiags...)
				}
			}
		}
	}

	if recurse {
		for _, childConfig := range c.Children {
			moreDiags := childConfig.addProviderRequirements(reqs, true, false)
			diags = append(diags, moreDiags...)
		}
	}

	return diags
}

func (c *Config) addProviderRequirementsFromProviderBlock(reqs providerreqs.Requirements, provider *Provider) hcl.Diagnostics {
	var diags hcl.Diagnostics

	fqn := c.Module.ProviderForLocalConfig(addrs.LocalProviderConfig{LocalName: provider.Name})
	if _, ok := reqs[fqn]; !ok {
		// We'll at least have an unconstrained dependency then, but might
		// add to this in the loop below.
		reqs[fqn] = nil
	}
	if provider.Version.Required != nil {
		// The model of version constraints in this package is still the
		// old one using a different upstream module to represent versions,
		// so we'll need to shim that out here for now. The two parsers
		// don't exactly agree in practice 🙄 so this might produce new errors.
		// TODO: Use the new parser throughout this package so we can get the
		// better error messages it produces in more situations.
		constraints, err := providerreqs.ParseVersionConstraints(provider.Version.Required.String())
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid version constraint",
				// The errors returned by ParseVersionConstraint already include
				// the section of input that was incorrect, so we don't need to
				// include that here.
				Detail:  fmt.Sprintf("Incorrect version constraint syntax: %s.", err.Error()),
				Subject: provider.Version.DeclRange.Ptr(),
			})
		}
		reqs[fqn] = append(reqs[fqn], constraints...)
	}

	return diags
}

// resolveProviderTypes walks through the providers in the module and ensures
// the true types are assigned based on the provider requirements for the
// module.
func (c *Config) resolveProviderTypes() map[string]addrs.Provider {
	for _, child := range c.Children {
		child.resolveProviderTypes()
	}

	// collect the required_providers, and then add any missing default providers
	providers := map[string]addrs.Provider{}
	for name, p := range c.Module.ProviderRequirements.RequiredProviders {
		providers[name] = p.Type
	}

	// ensure all provider configs know their correct type
	for _, p := range c.Module.ProviderConfigs {
		addr, required := providers[p.Name]
		if required {
			p.providerType = addr
		} else {
			addr := addrs.NewDefaultProvider(p.Name)
			p.providerType = addr
			providers[p.Name] = addr
		}
	}

	// connect module call providers to the correct type
	for _, mod := range c.Module.ModuleCalls {
		for _, p := range mod.Providers {
			if addr, known := providers[p.InParent.Name]; known {
				p.InParent.providerType = addr
			}
		}
	}

	// fill in parent module calls too
	if c.Parent != nil {
		for _, mod := range c.Parent.Module.ModuleCalls {
			for _, p := range mod.Providers {
				if addr, known := providers[p.InChild.Name]; known {
					p.InChild.providerType = addr
				}
			}
		}
	}

	return providers
}

// resolveProviderTypesForTests matches resolveProviderTypes except it uses
// the information from resolveProviderTypes to resolve the provider types for
// providers defined within the configs test files.
func (c *Config) resolveProviderTypesForTests(providers map[string]addrs.Provider) {

	for _, test := range c.Module.Tests {

		// testProviders contains the configuration blocks for all the providers
		// defined by this test file. It is keyed by the name of the provider
		// and the values are a slice of provider configurations which contains
		// all the definitions of a named provider of which there can be
		// multiple because of aliases.
		testProviders := make(map[string][]*Provider)
		for _, provider := range test.Providers {
			testProviders[provider.Name] = append(testProviders[provider.Name], provider)
		}

		// matchedProviders maps the names of providers from testProviders to
		// the provider type we have identified for them so far. If during the
		// course of resolving the types we find a run block is attempting to
		// reuse a provider that has already been assigned a different type,
		// then this is an error that we can raise now.
		matchedProviders := make(map[string]addrs.Provider)

		// First, we primarily draw our provider types from the main
		// configuration under test. The providers for the main configuration
		// are provided to us in the argument.

		// We've now set provider types for all the providers required by the
		// main configuration. But we can have modules with their own required
		// providers referenced by the run blocks. We also have passed provider
		// configs that can affect the types of providers when the names don't
		// match, so we'll do that here.

		for _, run := range test.Runs {

			// If this run block is executing against our main configuration, we
			// want to use the external providers passed in. If we are executing
			// against a different module then we need to resolve the provider
			// types for that first, and then use those providers.
			providers := providers
			if run.ConfigUnderTest != nil {
				providers = run.ConfigUnderTest.resolveProviderTypes()
			}

			// We now check to see what providers this run block is actually
			// using, and we can then assign types back to the

			if len(run.Providers) > 0 {
				// This provider is only using the subset of providers specified
				// within the provider block.

				for _, p := range run.Providers {
					addr, exists := providers[p.InChild.Name]
					if !exists {
						// If this provider wasn't explicitly defined in the
						// target module, then we'll set it to the default.
						addr = addrs.NewDefaultProvider(p.InChild.Name)
					}

					// The child type is always just derived from the providers
					// within the config this run block is using.
					p.InChild.providerType = addr

					// If we have previously assigned a type to the provider
					// for the parent reference, then we use that for the
					// parent type.
					if addr, exists := matchedProviders[p.InParent.Name]; exists {
						p.InParent.providerType = addr
						continue
					}

					// Otherwise, we'll define the parent type based on the
					// child and reference that backwards.
					p.InParent.providerType = p.InChild.providerType

					if aliases, exists := testProviders[p.InParent.Name]; exists {
						matchedProviders[p.InParent.Name] = p.InParent.providerType
						for _, alias := range aliases {
							alias.providerType = p.InParent.providerType
						}
					}
				}

			} else {
				// This provider is going to load all the providers it can using
				// simple name matching.

				for name, addr := range providers {

					if _, exists := matchedProviders[name]; exists {
						// Then we've already handled providers of this type
						// previously.
						continue
					}

					if aliases, exists := testProviders[name]; exists {
						// Then this provider has been defined within our test
						// config. Let's give it the appropriate type.
						matchedProviders[name] = addr
						for _, alias := range aliases {
							alias.providerType = addr
						}

						continue
					}

					// If we get here then it means we don't actually have a
					// provider block for this provider name within our test
					// file. This is fine, it just means we don't have to do
					// anything and the test will use the default provider for
					// that name.

				}
			}

		}

		// Now, we've analysed all the test runs for this file. If any providers
		// have not been claimed then we'll just give them the default provider
		// for their name.
		for name, aliases := range testProviders {
			if _, exists := matchedProviders[name]; exists {
				// Then this provider has a type already.
				continue
			}

			addr := addrs.NewDefaultProvider(name)
			matchedProviders[name] = addr

			for _, alias := range aliases {
				alias.providerType = addr
			}
		}

	}

}

// ProviderTypes returns the FQNs of each distinct provider type referenced
// in the receiving configuration.
//
// This is a helper for easily determining which provider types are required
// to fully interpret the configuration, though it does not include version
// information and so callers are expected to have already dealt with
// provider version selection in an earlier step and have identified suitable
// versions for each provider.
func (c *Config) ProviderTypes() []addrs.Provider {
	// Ignore diagnostics here because they relate to version constraints
	reqs, _ := c.ProviderRequirements()

	ret := make([]addrs.Provider, 0, len(reqs))
	for k := range reqs {
		ret = append(ret, k)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].String() < ret[j].String()
	})
	return ret
}

// ResolveAbsProviderAddr returns the AbsProviderConfig represented by the given
// ProviderConfig address, which must not be nil or this method will panic.
//
// If the given address is already an AbsProviderConfig then this method returns
// it verbatim, and will always succeed. If it's a LocalProviderConfig then
// it will consult the local-to-FQN mapping table for the given module
// to find the absolute address corresponding to the given local one.
//
// The module address to resolve local addresses in must be given in the second
// argument, and must refer to a module that exists under the receiver or
// else this method will panic.
func (c *Config) ResolveAbsProviderAddr(addr addrs.ProviderConfig, inModule addrs.Module) addrs.AbsProviderConfig {
	switch addr := addr.(type) {

	case addrs.AbsProviderConfig:
		return addr

	case addrs.LocalProviderConfig:
		// Find the descendant Config that contains the module that this
		// local config belongs to.
		mc := c.Descendant(inModule)
		if mc == nil {
			panic(fmt.Sprintf("ResolveAbsProviderAddr with non-existent module %s", inModule.String()))
		}

		var provider addrs.Provider
		if providerReq, exists := c.Module.ProviderRequirements.RequiredProviders[addr.LocalName]; exists {
			provider = providerReq.Type
		} else {
			provider = addrs.ImpliedProviderForUnqualifiedType(addr.LocalName)
		}

		return addrs.AbsProviderConfig{
			Module:   inModule,
			Provider: provider,
			Alias:    addr.Alias,
		}

	default:
		panic(fmt.Sprintf("cannot ResolveAbsProviderAddr(%v, ...)", addr))
	}

}

// ProviderForConfigAddr returns the FQN for a given addrs.ProviderConfig, first
// by checking for the provider in module.ProviderRequirements and falling
// back to addrs.NewDefaultProvider if it is not found.
func (c *Config) ProviderForConfigAddr(addr addrs.LocalProviderConfig) addrs.Provider {
	if provider, exists := c.Module.ProviderRequirements.RequiredProviders[addr.LocalName]; exists {
		return provider.Type
	}
	return c.ResolveAbsProviderAddr(addr, addrs.RootModule).Provider
}

// RequiredProviderConfig represents a provider configuration that is required
// by a module, either explicitly or implicitly.
//
// An explicit provider means the LocalName within the addrs.LocalProviderConfig
// was defined directly within the configuration via a required_providers block
// instead of implied due to the name of a resource or data block.
//
// This helps callers of the EffectiveRequiredProviderConfigs function tailor
// error messages around implied or explicit provider types.
type RequiredProviderConfig struct {
	Local    addrs.LocalProviderConfig
	Explicit bool
}

// EffectiveRequiredProviderConfigs returns a set of all of the provider
// configurations this config's direct module expects to have passed in
// (explicitly or implicitly) by its caller. This method only makes sense
// to call on the object representing the root module.
//
// This includes both provider configurations declared explicitly using
// configuration_aliases in the required_providers block _and_ configurations
// that are implied to be required by declaring something that belongs to
// an configuration for a provider even when there is no such declaration
// inside the module itself.
//
// Terraform Core treats root modules differently than downstream modules in
// that it will implicitly create empty provider configurations for any provider
// config addresses that are implied in the configuration but not explicitly
// configured. This function assumes those implied empty configurations don't
// exist and so therefore any provider configuration without an explicit
// "provider" block is a required provider config. In practice that means that
// the answer is appropriate for downstream modules but not for root modules,
// unless a root module is being used in a context where it is treated as if
// a shared module, such as when directly testing a shared module or when
// using a shared module as the root of the module tree of a stack component.
//
// This function assumes that the configuration is valid. It may produce under-
// or over-constrained results if called on an invalid configuration.
func (c *Config) EffectiveRequiredProviderConfigs() addrs.Map[addrs.RootProviderConfig, RequiredProviderConfig] {
	// The Terraform language has accumulated so many different ways to imply
	// the need for a provider configuration that answering this is quite a
	// complicated process that ends up potentially needing to visit the
	// entire subtree of modules even though we're only actually answering
	// about the current node's requirements. In the happy explicit case we
	// can avoid any recursion, but that case is rare in practice.

	if c == nil {
		return addrs.MakeMap[addrs.RootProviderConfig, RequiredProviderConfig]()
	}

	// We'll start by visiting all of the "provider" blocks in the module and
	// figuring out which provider configuration address they each declare. Any
	// configuration addresses we find here cannot be "required" provider
	// configs because the module instantiates them itself.
	selfConfigured := addrs.MakeSet[addrs.RootProviderConfig]()
	for _, pc := range c.Module.ProviderConfigs {
		localAddr := pc.Addr()
		sourceAddr := c.Module.ProviderForLocalConfig(localAddr)
		selfConfigured.Add(addrs.RootProviderConfig{
			Provider: sourceAddr,
			Alias:    localAddr.Alias,
		})
	}
	ret := addrs.MakeMap[addrs.RootProviderConfig, RequiredProviderConfig]()

	// maybePut looks up the default local provider for the given root provider.
	maybePut := func(addr addrs.RootProviderConfig) {
		localName := c.Module.LocalNameForProvider(addr.Provider)
		localAddr := addrs.LocalProviderConfig{
			LocalName: localName,
			Alias:     addr.Alias,
		}
		if !selfConfigured.Has(addr) && !ret.Has(addr) {
			ret.Put(addr, RequiredProviderConfig{
				Local: localAddr,

				// Since we look at the required providers first below, and only
				// the required providers can set explicit local names, this
				// will always be false as the map entry will already have been
				// set if this would be true.
				Explicit: false,
			})
		}
	}

	// maybePutLocal looks up the default provider for the given local provider
	// address.
	maybePutLocal := func(localAddr addrs.LocalProviderConfig, explicit bool) {
		// Caution: this function is only correct to use for LocalProviderConfig
		// in the _current_ module c.Module. It will produce incorrect results
		// if used for addresses from any child module.
		addr := addrs.RootProviderConfig{
			Provider: c.Module.ProviderForLocalConfig(localAddr),
			Alias:    localAddr.Alias,
		}
		if !selfConfigured.Has(addr) && !ret.Has(addr) {
			ret.Put(addr, RequiredProviderConfig{
				Local:    localAddr,
				Explicit: explicit,
			})
		}
	}

	if c.Module.ProviderRequirements != nil {
		for _, req := range c.Module.ProviderRequirements.RequiredProviders {
			for _, addr := range req.Aliases {
				// The RequiredProviders block always produces explicit provider
				// names.
				maybePutLocal(addr, true)
			}
		}
	}
	for _, rc := range c.Module.ManagedResources {
		maybePutLocal(rc.ProviderConfigAddr(), false)
	}
	for _, rc := range c.Module.DataResources {
		maybePutLocal(rc.ProviderConfigAddr(), false)
	}
	for _, ic := range c.Module.Import {
		if ic.ProviderConfigRef != nil {
			maybePutLocal(addrs.LocalProviderConfig{
				LocalName: ic.ProviderConfigRef.Name,
				Alias:     ic.ProviderConfigRef.Alias,
			}, false)
		} else {
			maybePut(addrs.RootProviderConfig{
				Provider: ic.Provider,
			})
		}
	}
	for _, mc := range c.Module.ModuleCalls {
		for _, pp := range mc.Providers {
			maybePutLocal(pp.InParent.Addr(), false)
		}
		// If there aren't any explicitly-passed providers then
		// the module implicitly requires a default configuration
		// for each provider the child module mentions, since
		// that would get implicitly passed into the child by
		// Terraform Core.
		// (We don't need to visit the child module at all if
		// the call has an explicit "providers" argument, because
		// we require that to be exhaustive when present.)
		if len(mc.Providers) == 0 {
			child := c.Children[mc.Name]
			childReqs := child.EffectiveRequiredProviderConfigs()
			for _, childReq := range childReqs.Keys() {
				if childReq.Alias != "" {
					continue // only default provider configs are eligible for this implicit treatment
				}
				// We must reinterpret the child address to appear as
				// if written in its parent (our current module).
				maybePut(addrs.RootProviderConfig{
					Provider: childReq.Provider,
				})
			}
		}
	}

	return ret
}

func (c *Config) CheckCoreVersionRequirements() hcl.Diagnostics {
	var diags hcl.Diagnostics

	diags = diags.Extend(c.Module.CheckCoreVersionRequirements(c.Path, c.SourceAddr))

	for _, c := range c.Children {
		childDiags := c.CheckCoreVersionRequirements()
		diags = diags.Extend(childDiags)
	}

	return diags
}
