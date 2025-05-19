// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestProvidersTransformer is a GraphTransformer that gathers all the providers
// from the module configurations that the test runs depend on and attaches the
// required providers to the test run nodes.
type TestProvidersTransformer struct{}

func (t *TestProvidersTransformer) Transform(g *terraform.Graph) error {
	configsProviderMap := make(map[string]map[string]bool)
	runProviderMap := make(map[*NodeTestRun]map[string]bool)

	// a root provider node that will add the providers to the context
	rootProviderNode := t.createRootNode(g, runProviderMap)

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
		runProviderMap[node] = configsProviderMap[configKey]

		// Add an edge from the test run node to the root provider node
		g.Connect(dag.BasicEdge(v, rootProviderNode))
	}

	return nil
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

func (t *TestProvidersTransformer) createRootNode(g *terraform.Graph, providerMap map[*NodeTestRun]map[string]bool) *dynamicNode {
	node := &dynamicNode{
		eval: func(ctx *EvalContext) tfdiags.Diagnostics {
			for node, providers := range providerMap {
				ctx.SetProviders(node.run, providers)
			}
			return nil
		},
	}
	g.Add(node)
	return node
}
