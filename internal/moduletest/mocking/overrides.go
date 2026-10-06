// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// Overrides contains a summary of all the overrides that should apply for a
// test run.
//
// This requires us to deduplicate between run blocks and test files, and mock
// providers.
type Overrides struct {
	providerOverrides map[addrs.RootProviderConfig]addrs.Map[addrs.TargetPattern, *configs.Override]
	localOverrides    addrs.Map[addrs.TargetPattern, *configs.Override]
}

func PackageOverrides(ctx *hcl.EvalContext, run *configs.TestRun, file *configs.TestFile, mocks map[addrs.RootProviderConfig]*configs.MockData) (*Overrides, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	overrides := &Overrides{
		providerOverrides: make(map[addrs.RootProviderConfig]addrs.Map[addrs.TargetPattern, *configs.Override]),
		localOverrides:    addrs.MakeMap[addrs.TargetPattern, *configs.Override](),
	}

	// helper function to evaluate each override values, returning any error encountered.
	evalAndPut := func(container addrs.Map[addrs.TargetPattern, *configs.Override], target addrs.TargetPattern, override *configs.Override) tfdiags.Diagnostics {

		override.Values = cty.NilVal
		if override.RawExpr != nil {
			values, hclDiags := override.RawExpr.Value(ctx)
			if values != cty.NilVal && !values.Type().IsObjectType() {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid outputs attribute",
					Detail:   fmt.Sprintf("%s blocks must specify an outputs attribute that is an object.", override.BlockName),
					Subject:  override.ValuesRange.Ptr(),
				})
				return diags
			}
			override.Values = values
			diags = diags.Append(hclDiags)
		}

		container.Put(target, override)
		return diags
	}

	// The run block overrides have the highest priority, we always include all
	// of them.
	for _, elem := range run.Overrides.Elems {
		if diags := evalAndPut(overrides.localOverrides, elem.Key, elem.Value); diags.HasErrors() {
			return overrides, diags
		}
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

		if diags := evalAndPut(overrides.localOverrides, elem.Key, elem.Value); diags.HasErrors() {
			return overrides, diags
		}
	}

	// Finally, we want to include the overrides for any mock providers we have.
	for key, data := range mocks {
		for _, elem := range data.Overrides.Elems {
			target := elem.Key

			if overrides.localOverrides.Has(target) {
				// Then the file or the run block is providing an override with
				// higher precedence.
				continue
			}

			if _, exists := overrides.providerOverrides[key]; !exists {
				overrides.providerOverrides[key] = addrs.MakeMap[addrs.TargetPattern, *configs.Override]()
			}

			if diags := evalAndPut(overrides.providerOverrides[key], elem.Key, elem.Value); diags.HasErrors() {
				return overrides, diags
			}
		}
	}

	return overrides, diags
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

	for _, elem := range overrides.localOverrides.Elems {
		if elem.Key.Contains(module) {
			// Then either module or one of its ancestors is being overridden.
			return true
		}
	}

	return false
}

// GetResourceOverride checks the overrides for the given resource instance.
// Users can mark a resource instance as overridden by overriding the instance
// directly (eg. resource.foo[0]) or by overriding the containing resource (eg.
// resource.foo), and the most specific of the matching overrides is used.
//
// If the resource is being supplied by a mock provider, then we need to check
// the overrides for that provider as well, as such the provider config is
// required so we know which mock provider to check.
func (overrides *Overrides) GetResourceOverride(inst addrs.AbsResourceInstance, provider addrs.AbsProviderConfig) (*configs.Override, bool) {
	if overrides.Empty() {
		// Short circuit any lookups if we have no overrides.
		return nil, false
	}

	// An override applies if it selects this instance. The targets must also
	// be within the same configuration resource, to exclude module overrides
	// which would contain the instance too.
	matches := func(target addrs.TargetPattern) bool {
		return target.Contains(inst) && inst.ConfigResource().Contains(target)
	}

	// Local overrides are listed first, so they take precedence over any
	// mock provider overrides for the same instances.
	candidates := matchingOverrides(overrides.localOverrides, matches)
	if providerOverrides, ok := overrides.ProviderMatch(provider); ok {
		candidates = append(candidates, matchingOverrides(providerOverrides, matches)...)
	}

	return mostSpecificOverride(candidates)
}

// GetModuleOverride checks the overrides for the given module instance.
//
// Users can mark a module instance as overridden by overriding the instance
// directly (eg. module.foo[0]) or by overriding the containing module
// (eg. module.foo), and the most specific of the matching overrides is used.
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

	// An override applies if it selects this module instance. The targets
	// must also be within the same configuration module, to exclude overrides
	// of ancestor modules, which are handled by IsOverridden.
	return mostSpecificOverride(matchingOverrides(overrides.localOverrides, func(target addrs.TargetPattern) bool {
		return target.Contains(inst) && inst.Module().Contains(target)
	}))
}

// matchingOverrides returns the overrides with targets accepted by the match
// function, ordered by target so that the results are deterministic.
func matchingOverrides(overrides addrs.Map[addrs.TargetPattern, *configs.Override], match func(addrs.TargetPattern) bool) []addrs.MapElem[addrs.TargetPattern, *configs.Override] {
	var ret []addrs.MapElem[addrs.TargetPattern, *configs.Override]
	for _, elem := range overrides.Elems {
		if match(elem.Key) {
			ret = append(ret, elem)
		}
	}
	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i].Key.String() < ret[j].Key.String()
	})
	return ret
}

// mostSpecificOverride returns the candidate with the narrowest target, which
// is contained by the targets of the others. If targets overlap without
// either containing the other, the earliest candidate takes precedence.
func mostSpecificOverride(candidates []addrs.MapElem[addrs.TargetPattern, *configs.Override]) (*configs.Override, bool) {
	if len(candidates) == 0 {
		return nil, false
	}

	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if best.Key.Contains(candidate.Key) && !candidate.Key.Contains(best.Key) {
			best = candidate
		}
	}
	return best.Value, true
}

// ProviderMatch returns true if we have overrides for the given provider.
//
// This is so that we can selectively apply overrides to resources that are
// being supplied by a given provider.
func (overrides *Overrides) ProviderMatch(provider addrs.AbsProviderConfig) (addrs.Map[addrs.TargetPattern, *configs.Override], bool) {
	if !provider.Module.IsRoot() {
		// We can only set mock providers within the root module.
		return addrs.Map[addrs.TargetPattern, *configs.Override]{}, false
	}

	data, exists := overrides.providerOverrides[addrs.RootProviderConfig{
		Provider: provider.Provider,
		Alias:    provider.Alias,
	}]
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
