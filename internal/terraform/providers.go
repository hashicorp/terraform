// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// checkExternalProviders verifies that all of the explicitly-declared
// external provider configuration requirements in the root module are
// satisfied by the given instances, and also that all of the given
// instances belong to providers that the overall configuration at least
// uses somewhere.
//
// At the moment we only use external provider configurations for module
// trees acting as Stack components and most other use will not offer any
// externally-configured providers at all, and so the errors returned
// here are somewhat vague to accommodate being used both to describe
// an invalid component configuration and the problem of trying to plan and
// apply a module that wasn't intended to be a root module.
func checkExternalProviders(rootCfg *configs.Config, plan *plans.Plan, state *states.State, got map[addrs.RootProviderConfig]providers.Interface) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	allowedProviders := make(map[addrs.Provider]bool)
	for _, addr := range rootCfg.ProviderTypes() {
		allowedProviders[addr] = true
	}
	if state != nil {
		for _, addr := range state.ProviderAddrs() {
			allowedProviders[addr.Provider] = true
		}
	}
	if plan != nil {
		for _, addr := range plan.ProviderAddrs() {
			allowedProviders[addr.Provider] = true
		}
	}
	requiredConfigs := rootCfg.EffectiveRequiredProviderConfigs().Keys()

	// Passed-in provider configurations can only be for providers that this
	// configuration actually contains some use of.
	// (This is an imprecise way of rejecting undeclared provider configs;
	// we can't be precise because Terraform permits implicit default provider
	// configurations.)
	for cfgAddr := range got {
		if !allowedProviders[cfgAddr.Provider] {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unexpected provider configuration",
				fmt.Sprintf("The plan options include a configuration for provider %s, which is not used anywhere in this configuration.", cfgAddr.Provider),
			))
		} else if cfgAddr.Alias != "" && !requiredConfigs.Has(cfgAddr) {
			// Additional (aliased) provider configurations must always be
			// explicitly declared.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unexpected provider configuration",
				fmt.Sprintf("The plan options include a configuration for provider %s with alias %q, which is not declared by the root module.", cfgAddr.Provider, cfgAddr.Alias),
			))
		}
	}

	// The caller _must_ pass external provider configurations for any address
	// that's been explicitly declared as required in the required_providers
	// block.
	for _, cfgAddr := range requiredConfigs {
		if _, defined := got[cfgAddr]; !defined {
			if cfgAddr.Alias == "" {
				// We can't actually return an error here because it's valid
				// to leave a default provider configuration implied as long
				// as the provider itself will accept an all-null configuration,
				// which we won't know until we actually start evaluating.
				continue
			} else {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Undefined provider configuration",
					fmt.Sprintf(
						"The root module declares that it requires the caller to pass a configuration for provider %s with alias %q.",
						cfgAddr.Provider, cfgAddr.Alias,
					),
				))
			}
		}
	}

	// It isn't valid to pass in a provider for an address that is associated
	// with an explicit "provider" block in the root module, since that would
	// make it ambiguous whether we're using the passed in one or the declared
	// one.
	for _, pc := range rootCfg.Module.ProviderConfigs {
		absAddr := rootCfg.ResolveAbsProviderAddr(pc.Addr(), addrs.RootModule)
		rootAddr := addrs.RootProviderConfig{
			Provider: absAddr.Provider,
			Alias:    absAddr.Alias,
		}
		if _, defined := got[rootAddr]; defined {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unexpected provider configuration",
				fmt.Sprintf("The plan options include provider configuration %s, but that conflicts with the explicitly-defined provider configuration at %s.", rootAddr, pc.DeclRange.String()),
			))
		}
	}

	return diags
}

// externalProviderWrapper is a wrapper around a provider instance that
// intercepts methods that don't make sense to call on a provider instance
// passed in by an external caller which we assume is owned by the caller
// and pre-configured.
//
// This is a kinda-hacky way to deal with the fact that Terraform Core
// logic tends to assume it is responsible for the full lifecycle of a
// provider instance, which isn't true for externally-provided ones.
type externalProviderWrapper struct {
	providers.Interface
}

var _ providers.Interface = externalProviderWrapper{}

// ConfigureProvider does nothing because external providers are supposed to
// be pre-configured before passing them to Terraform Core.
func (pw externalProviderWrapper) ConfigureProvider(providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	return providers.ConfigureProviderResponse{}
}

// Close does nothing because the caller which provided an external provider
// client is the one responsible for eventually closing it.
func (pw externalProviderWrapper) Close() error {
	return nil
}
