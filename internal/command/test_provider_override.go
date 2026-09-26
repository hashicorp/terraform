// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// testProviderVersionOverride is a provider version constraint that should
// replace the module-under-test requirement for a single terraform test run.
type testProviderVersionOverride struct {
	Addr       addrs.Provider
	Constraint providerreqs.VersionConstraints
	Origin     string
}

// collectTestProviderVersionOverrides gathers version overrides from matching
// test files and from -override-provider command-line flags. CLI flags replace
// file-level constraints for the same provider.
func collectTestProviderVersionOverrides(config *configs.Config, cliOverrides []arguments.ProviderVersionOverride, filters []string) []testProviderVersionOverride {
	if config == nil || config.Module == nil {
		return cliOverridesToTestOverrides(cliOverrides)
	}

	merged := make(map[addrs.Provider]testProviderVersionOverride)

	for name, file := range config.Module.Tests {
		if !testFileMatchesFilter(name, filters) {
			continue
		}
		if file == nil || file.RequiredProviders == nil {
			continue
		}
		for _, rp := range file.RequiredProviders.RequiredProviders {
			if rp == nil || rp.Type.IsZero() {
				continue
			}

			var constraint providerreqs.VersionConstraints
			if rp.Requirement.Required != nil {
				parsed, err := providerreqs.ParseVersionConstraints(rp.Requirement.Required.String())
				if err != nil {
					// Invalid constraints are already reported during config load.
					continue
				}
				constraint = parsed
			}

			if existing, ok := merged[rp.Type]; ok {
				existing.Constraint = append(existing.Constraint, constraint...)
				existing.Origin = existing.Origin + " and " + name
				merged[rp.Type] = existing
				continue
			}

			merged[rp.Type] = testProviderVersionOverride{
				Addr:       rp.Type,
				Constraint: constraint,
				Origin:     name,
			}
		}
	}

	for _, override := range cliOverrides {
		merged[override.Addr] = testProviderVersionOverride{
			Addr:       override.Addr,
			Constraint: override.Constraint,
			Origin:     fmt.Sprintf("-override-provider=%s", override.Raw),
		}
	}

	if len(merged) == 0 {
		return nil
	}

	ret := make([]testProviderVersionOverride, 0, len(merged))
	for _, override := range merged {
		ret = append(ret, override)
	}
	return ret
}

func cliOverridesToTestOverrides(cliOverrides []arguments.ProviderVersionOverride) []testProviderVersionOverride {
	if len(cliOverrides) == 0 {
		return nil
	}
	ret := make([]testProviderVersionOverride, 0, len(cliOverrides))
	for _, override := range cliOverrides {
		ret = append(ret, testProviderVersionOverride{
			Addr:       override.Addr,
			Constraint: override.Constraint,
			Origin:     fmt.Sprintf("-override-provider=%s", override.Raw),
		})
	}
	return ret
}

func testFileMatchesFilter(name string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if filter == name || filepath.ToSlash(filter) == filepath.ToSlash(name) {
			return true
		}
	}
	return false
}

// applyTestProviderVersionOverrides installs the requested provider versions
// for this test run and updates opts.Providers to use them. The on-disk
// dependency lock file is left unchanged.
func (m *Meta) applyTestProviderVersionOverrides(opts *terraform.ContextOpts, overrides []testProviderVersionOverride) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if len(overrides) == 0 || opts == nil {
		return diags
	}

	// Command tests inject mock providers through testingOverrides. Those
	// factories already ignore lock-file versions, so there is nothing to
	// install.
	if m.testingOverrides != nil {
		return diags
	}

	locks, lockDiags := m.lockedDependencies()
	diags = diags.Append(lockDiags)
	if lockDiags.HasErrors() {
		return diags
	}

	locks = locks.DeepCopy()
	reqs := make(getproviders.Requirements)

	for addr, lock := range locks.AllProviders() {
		reqs[addr] = providerreqs.MustParseVersionConstraints("=" + lock.Version().String())
	}

	var summary []string
	for _, override := range overrides {
		reqs[override.Addr] = override.Constraint
		if depsfile.ProviderIsLockable(override.Addr) {
			locks.RemoveProvider(override.Addr)
		}
		constraint := providerreqs.VersionConstraintsString(override.Constraint)
		if constraint == "" {
			constraint = "any"
		}
		summary = append(summary, fmt.Sprintf("- %s (%s) from %s", override.Addr.ForDisplay(), constraint, override.Origin))
	}
	sort.Strings(summary)

	inst := m.providerInstaller()
	newLocks, err := inst.EnsureProviderVersions(context.Background(), locks, reqs, providercache.InstallNewProvidersOnly)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to install overridden provider version",
			fmt.Sprintf("Terraform could not install the provider version requested for this test run: %s.", err),
		))
		return diags
	}

	factories, err := m.ProviderFactoriesFromLocks(newLocks)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to load overridden provider version",
			fmt.Sprintf("Terraform installed an overridden provider version but could not load it: %s.", err),
		))
		return diags
	}
	opts.Providers = factories

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Warning,
		"Provider version overrides are in effect",
		fmt.Sprintf("terraform test is using provider versions that differ from the dependency lock file. The lock file has not been updated.\n\n%s", strings.Join(summary, "\n")),
	))

	return diags
}
