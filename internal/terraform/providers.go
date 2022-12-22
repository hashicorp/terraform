package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// checkExternalProviders verifies that all of the explicitly-declared
// external provider configuration requirements in the root module are
// satisfied by the given instances, and also that all of the given
// instances belong to providers that the overall configuration at least
// uses somewhere.
//
// At the moment we only use external provider configurations for the
// "terraform test" command and so most normal use will not offer any
// externally-configured providers at all, and so the errors returned
// here are somewhat vague to accommodate being used both to describe
// an invalid test step and the problem of trying to plan and apply
// a module that wasn't intended to be a root module.
func checkExternalProviders(rootCfg *configs.Config, got map[addrs.RootProviderConfig]providers.Interface) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	allowedProviders := map[addrs.Provider]struct{}{}
	for _, addr := range rootCfg.ProviderTypes() {
		allowedProviders[addr] = struct{}{}
	}
	requiredConfigs := rootCfg.RequiredExternalProviderConfigs()

	// Passed-in provider configurations can only be for providers that this
	// configuration actually contains some use of.
	// (This is an imprecise way of rejecting undeclared provider configs;
	// we can't be precise because Terraform permits implicit default provider
	// configurations.)
	for cfgAddr := range got {
		if _, allowed := allowedProviders[cfgAddr.Provider]; !allowed {
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
	for _, cfgAddr := range rootCfg.RequiredExternalProviderConfigs() {
		if _, defined := got[cfgAddr]; !defined {
			if cfgAddr.Alias == "" {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Undefined provider configuration",
					fmt.Sprintf(
						"The root module declares that it requires the caller to pass a default (unaliased) configuration for provider %s.",
						cfgAddr.Provider,
					),
				))
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
// makes ConfigureProvider act as a no-op. This is a kinda-hacky way to
// deal with the fact that external providers are supposed to arrive
// pre-configured and so do not need to be configured again by Terraform Core.
type externalProviderWrapper struct {
	providers.Interface
}

var _ providers.Interface = externalProviderWrapper{}

func (pw externalProviderWrapper) ConfigureProvider(providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	log.Printf("[DEBUG] Skipping ConfigureProvider for external provider instance")
	return providers.ConfigureProviderResponse{}
}

func (pw externalProviderWrapper) Close() error {
	// It's not Terraform Core's responsibility to close a provider instance
	// that was provided by an external caller.
	log.Printf("[DEBUG] Skipping Close for external provider instance")
	return nil
}
