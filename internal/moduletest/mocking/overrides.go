// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

// Overrides contains a summary of all the overrides that should apply for a
// test run.
//
// This requires us to deduplicate between run blocks and test files, and mock
// providers.
type Overrides struct {
	providerOverrides map[string]addrs.Map[addrs.Targetable, *configs.Override]
	localOverrides    addrs.Map[addrs.Targetable, *configs.Override]
}

func PackageOverrides(run *configs.TestRun, file *configs.TestFile, config *configs.Config) *Overrides {
	overrides := &Overrides{
		providerOverrides: make(map[string]addrs.Map[addrs.Targetable, *configs.Override]),
		localOverrides:    addrs.MakeMap[addrs.Targetable, *configs.Override](),
	}

	// The run block overrides have the highest priority, we always include all
	// of them.
	for _, elem := range run.Overrides.Elems {
		overrides.localOverrides.PutElement(elem)
	}

	// The file overrides are second, we include these as long as there isn't
	// a direct replacement in the current run block or the run block doesn't
	// override an entire module that a file override would be inside.
	for _, elem := range file.Overrides.Elems {
		target := elem.Key

		if overrides.localOverrides.Has(target) {
			// The run block provided a value already.
			continue
		}

		overrides.localOverrides.PutElement(elem)
	}

	// Finally, we want to include the overrides for any mock providers we have.
	for key, provider := range config.Module.ProviderConfigs {
		if !provider.Mock {
			// Only mock providers can supply overrides.
			continue
		}

		for _, elem := range provider.MockData.Overrides.Elems {
			target := elem.Key

			if overrides.localOverrides.Has(target) {
				// Then the file or the run block is providing an override with
				// higher precedence.
				continue
			}

			if _, exists := overrides.providerOverrides[key]; !exists {
				overrides.providerOverrides[key] = addrs.MakeMap[addrs.Targetable, *configs.Override]()
			}
			overrides.providerOverrides[key].PutElement(elem)
		}
	}

	return overrides
}

// IsOverridden returns true if the module is either overridden directly or
// nested within another module that is already being overridden.
//
// For this function, we know that overrides defined within mock providers
// cannot target modules directly. Therefore, we only need to check the local
// overrides within this function.
func (overrides *Overrides) IsOverridden(module addrs.ModuleInstance) bool {
	if module.Equal(addrs.RootModuleInstance) {
		// The root module is never overridden, so let's just short circuit
		// this.
		return false
	}

	if overrides.localOverrides.Has(module) {
		// Short circuit things, if we have an exact match just return now.
		return true
	}

	// Otherwise, check for parents.
	for _, elem := range overrides.localOverrides.Elems {
		if elem.Key.TargetContains(module) {
			// Then we have an ancestor of module being overridden instead of
			// module being overridden directly.
			return true
		}
	}

	return false
}

// GetResourceOverride checks the overrides for the given resource instance.
// If the provided address is instanced, then we will check the containing
// resource as well. This is because users can mark a resource instance as
// overridden by overriding the instance directly (eg. resource.foo[0]) or by
// overriding the containing resource (eg. resource.foo).
//
// If the resource is being supplied by a mock provider, then we need to check
// the overrides for that provider as well, as such the provider config is
// required so we know which mock provider to check.
func (overrides *Overrides) GetResourceOverride(inst addrs.AbsResourceInstance, provider addrs.AbsProviderConfig) (*configs.Override, bool) {
	if overrides.Empty() {
		// Short circuit any lookups if we have no overrides.
		return nil, false
	}

	// First check this specific resource.
	if override, ok := overrides.getResourceOverride(inst, provider); ok {
		return override, true
	}

	// Otherwise check the containing resource in case the user has set for all
	// the instances of a resource to be overridden.
	return overrides.getResourceOverride(inst.ContainingResource(), provider)
}

func (overrides *Overrides) getResourceOverride(target addrs.Targetable, provider addrs.AbsProviderConfig) (*configs.Override, bool) {
	// If we have a local override, then apply that first.
	if override, ok := overrides.localOverrides.GetOk(target); ok {
		return override, true
	}

	// Otherwise, check if we have overrides for this provider.
	providerOverrides, ok := overrides.ProviderMatch(provider)
	if ok {
		if override, ok := providerOverrides.GetOk(target); ok {
			return override, true
		}
	}

	// If we have no overrides, that's okay.
	return nil, false
}

// GetModuleOverride checks the overrides for the given module instance. This
// function automatically checks if the containing module has been overridden
// if the instance is instanced.
//
// Users can mark a module instance as overridden by overriding the instance
// directly (eg. module.foo[0]) or by overriding the containing module
// (eg. module.foo).
//
// Modules cannot be overridden by mock providers directly, so we don't need
// to know anything about providers for this function (in contrast to
// GetResourceOverride).
func (overrides *Overrides) GetModuleOverride(inst addrs.ModuleInstance) (*configs.Override, bool) {
	if len(inst) == 0 || overrides.Empty() {
		// The root module is never overridden, so let's just short circuit
		// this.
		return nil, false
	}

	// Otherwise check if this specific instance has been overridden.
	if override, ok := overrides.localOverrides.GetOk(inst); ok {
		// It has, so just return that.
		return override, true
	}

	// If this is an instanced address (eg. module.foo[0]), then we need to
	// check if the containing module has been overridden as we let users
	// override all instances of a module by overriding the containing module
	// (eg. module.foo).

	// Check if the last step is actually instanced, so we don't do extra work
	// needlessly.
	if inst[len(inst)-1].InstanceKey == addrs.NoKey {
		// Then we already checked the instance itself and it wasn't overridden.
		return nil, false
	}

	return overrides.localOverrides.GetOk(inst.ContainingModule())
}

// ProviderMatch returns true if we have overrides for the given provider.
//
// This is so that we can selectively apply overrides to resources that are
// being supplied by a given provider.
func (overrides *Overrides) ProviderMatch(provider addrs.AbsProviderConfig) (addrs.Map[addrs.Targetable, *configs.Override], bool) {
	if !provider.Module.IsRoot() {
		// We can only set mock providers within the root module.
		return addrs.Map[addrs.Targetable, *configs.Override]{}, false
	}

	name := provider.Provider.Type
	if len(provider.Alias) > 0 {
		name = fmt.Sprintf("%s.%s", name, provider.Alias)
	}

	data, exists := overrides.providerOverrides[name]
	return data, exists
}

// Empty returns true if we have no actual overrides.
//
// This is just a convenience function to make checking for overrides easier.
func (overrides *Overrides) Empty() bool {
	if overrides == nil {
		return true
	}

	if overrides.localOverrides.Len() > 0 {
		return false
	}

	for _, value := range overrides.providerOverrides {
		if value.Len() > 0 {
			return false
		}
	}

	return true
}
