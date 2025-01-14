// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
)

// ProviderTransformer is a GraphTransformer that adds all the test runs,
// and the variables defined in each run block, to the graph.
type ProviderTransformer struct {
	File       *moduletest.File
	config     *configs.Config
	globalVars map[string]backendrun.UnparsedVariableValue
}

func (t *ProviderTransformer) Transform(g *terraform.Graph) error {

	return nil
}

type nodeProvider struct {
}

// func (n *nodeProvider) Execute(ctx *hcltest.ExecContext, g *terraform.Graph) (tfdiags.Diagnostics, error) {
// 	runner.Suite.configLock.Lock()
// 	defer runner.Suite.configLock.Unlock()
// 	if _, exists := runner.Suite.configProviders[key]; exists {
// 		// Then we've processed this key before, so skip it.
// 		return
// 	}

// 	providers := make(map[string]bool)

// 	// First, let's look at the required providers first.
// 	for _, provider := range config.Module.ProviderRequirements.RequiredProviders {
// 		providers[provider.Name] = true
// 		for _, alias := range provider.Aliases {
// 			providers[alias.StringCompact()] = true
// 		}
// 	}

// 	// Second, we look at the defined provider configs.
// 	for _, provider := range config.Module.ProviderConfigs {
// 		providers[provider.Addr().StringCompact()] = true
// 	}

// 	// Third, we look at the resources and data sources.
// 	for _, resource := range config.Module.ManagedResources {
// 		if resource.ProviderConfigRef != nil {
// 			providers[resource.ProviderConfigRef.String()] = true
// 			continue
// 		}
// 		providers[resource.Provider.Type] = true
// 	}
// 	for _, datasource := range config.Module.DataResources {
// 		if datasource.ProviderConfigRef != nil {
// 			providers[datasource.ProviderConfigRef.String()] = true
// 			continue
// 		}
// 		providers[datasource.Provider.Type] = true
// 	}

// 	// Finally, we look at any module calls to see if any providers are used
// 	// in there.
// 	for _, module := range config.Module.ModuleCalls {
// 		for _, provider := range module.Providers {
// 			providers[provider.InParent.String()] = true
// 		}
// 	}

// 	runner.Suite.configProviders[key] = providers
// }
