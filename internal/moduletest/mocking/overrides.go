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

// IsDeeplyOverridden returns true if an ancestor of this module is overridden
// but not if the module is overridden directly.
//
// This function doesn't consider an instanced module to be deeply overridden
// by the uninstanced reference to the same module. So,
// IsDeeplyOverridden("mod.child[0]") would return false if "mod.child" has been
// overridden.
//
// For this function, we know that overrides defined within mock providers
// cannot target modules directly. Therefore, we only need to check the local
// overrides within this function.
func (overrides *Overrides) IsDeeplyOverridden(module addrs.ModuleInstance) bool {
	for _, elem := range overrides.localOverrides.Elems {
		target := elem.Key

		if target.TargetContains(module) {
			// So we do think it contains it, but it could be matching here
			// because of equality or because we have an instanced module.
			if instance, ok := target.(addrs.ModuleInstance); ok {
				if instance.Equal(module) {
					// Then we're exactly equal, so not deeply nested.
					continue
				}

				if instance.Module().Equal(module.Module()) {
					// Then we're an instanced version of they other one, so
					// also not deeply nested by our definition of deeply.
					continue
				}

			}

			// Otherwise, it's deeply nested.
			return true
		}
	}
	return false
}

// GetOverrideInclProviders retrieves the override for target if it exists.
//
// This function also checks the provider specific overrides using the provider
// argument.
func (overrides *Overrides) GetOverrideInclProviders(target addrs.Targetable, provider addrs.AbsProviderConfig) (*configs.Override, bool) {
	// If we have a local override, then apply that first.
	if override, ok := overrides.GetOverride(target); ok {
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

// GetOverride retrieves the override for target from the local overrides if
// it exists.
func (overrides *Overrides) GetOverride(target addrs.Targetable) (*configs.Override, bool) {
	return overrides.localOverrides.GetOk(target)
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
