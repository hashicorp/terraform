// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
)

// TestProvidersTransformer is a GraphTransformer that gathers all the providers
// from the module configurations that the test runs depend on and attaches the
// required providers to the test run nodes.
type TestProvidersTransformer struct {
	Config    *configs.Config
	File      *moduletest.File
	Providers map[addrs.Provider]providers.Factory
}

func (t *TestProvidersTransformer) Transform(g *terraform.Graph) error {

	type tuple struct {
		configure *NodeProviderConfigure
		close     *NodeProviderClose
	}

	nodes := make(map[string]map[string]tuple)

	for _, config := range t.File.Config.Providers {
		provider := t.Config.ProviderForConfigAddr(config.Addr())

		factory, ok := t.Providers[provider]
		if !ok {
			return fmt.Errorf("unknown provider %s", provider)
		}

		impl, err := factory()
		if err != nil {
			return fmt.Errorf("could not create provider instance: %w", err)
		}
		var mock *providers.Mock

		if config.Mock {
			mock = &providers.Mock{
				Provider: impl,
				Data:     config.MockData,
			}
			impl = mock
		}

		addr := addrs.RootProviderConfig{
			Provider: provider,
			Alias:    config.Alias,
		}

		configure := &NodeProviderConfigure{
			name:     config.Name,
			alias:    config.Alias,
			Addr:     addr,
			File:     t.File,
			Config:   config,
			Provider: impl,
			Schema:   impl.GetProviderSchema(),
		}
		g.Add(configure)

		close := &NodeProviderClose{
			name:     config.Name,
			alias:    config.Alias,
			Addr:     addr,
			File:     t.File,
			Config:   config,
			Provider: impl,
		}
		g.Add(close)

		if _, exists := nodes[config.Name]; !exists {
			nodes[config.Name] = make(map[string]tuple)
		}
		nodes[config.Name][config.Alias] = tuple{
			configure: configure,
			close:     close,
		}

		// make sure the provider is only closed after the provider starts.
		g.Connect(dag.BasicEdge(close, configure))
	}

	for vertex := range g.VerticesSeq() {
		if vertex, ok := vertex.(*NodeTestRun); ok {
			// providers aren't referenceable so the automatic reference
			// transformer won't do this.

			if len(vertex.Run().Config.Providers) > 0 {
				for _, ref := range vertex.run.Config.Providers {
					if node, ok := nodes[ref.InParent.Name][ref.InParent.Alias]; ok {
						g.Connect(dag.BasicEdge(vertex, node.configure))
					}
				}
			} else {
				for provider := range requiredProviders(vertex.run.ModuleConfig) {
					name := t.Config.Module.LocalNameForProvider(provider.Provider)
					if node, ok := nodes[name][provider.Alias]; ok {
						g.Connect(dag.BasicEdge(vertex, node.configure))
					}
				}
			}
		}

		if vertex, ok := vertex.(*TeardownSubgraph); ok {
			for _, node := range nodes {
				for _, node := range node {
					// close all the providers after the states have been
					// cleaned up.
					g.Connect(dag.BasicEdge(node.close, vertex))
				}
			}
		}
	}
	return nil
}
