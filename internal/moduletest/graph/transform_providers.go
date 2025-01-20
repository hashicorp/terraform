// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/terraform"
)

// TestProvidersTransformer is a GraphTransformer that gathers all the providers
// from the module configurations that the test runs depend on and attaches the
// required providers to the test run nodes.
type TestProvidersTransformer struct{}

func (t *TestProvidersTransformer) Transform(g *terraform.Graph) error {
	var errs []error
	configsProviderMap := make(map[string]map[string]bool)

	for _, v := range g.Vertices() {
		node, ok := v.(*NodeTestRun)
		if !ok {
			continue
		}

		// Get the providers that the test run depends on
		configKey := node.run.GetModuleConfigID()
		if _, ok := configsProviderMap[configKey]; !ok {
			providers := t.transformSingleConfig(node.run.ModuleConfig)
			configsProviderMap[configKey] = providers
		}

		providers, ok := configsProviderMap[configKey]
		if !ok {
			// This should not happen
			errs = append(errs, fmt.Errorf("missing providers for module config %q", configKey))
			continue
		}

		// Add the required providers for the test run node
		node.requiredProviders = providers
	}
	return errors.Join(errs...)
}

func (t *TestProvidersTransformer) transformSingleConfig(config *configs.Config) map[string]bool {
	providers := make(map[string]bool)

	// First, let's look at the required providers first.
	for _, provider := range config.Module.ProviderRequirements.RequiredProviders {
		providers[provider.Name] = true
		for _, alias := range provider.Aliases {
			providers[alias.StringCompact()] = true
		}
	}

	// Second, we look at the defined provider configs.
	for _, provider := range config.Module.ProviderConfigs {
		providers[provider.Addr().StringCompact()] = true
	}

	// Third, we look at the resources and data sources.
	for _, resource := range config.Module.ManagedResources {
		if resource.ProviderConfigRef != nil {
			providers[resource.ProviderConfigRef.String()] = true
			continue
		}
		providers[resource.Provider.Type] = true
	}
	for _, datasource := range config.Module.DataResources {
		if datasource.ProviderConfigRef != nil {
			providers[datasource.ProviderConfigRef.String()] = true
			continue
		}
		providers[datasource.Provider.Type] = true
	}

	// Finally, we look at any module calls to see if any providers are used
	// in there.
	for _, module := range config.Module.ModuleCalls {
		for _, provider := range module.Providers {
			providers[provider.InParent.String()] = true
		}
	}

	return providers
}
