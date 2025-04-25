// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

// QueryConfigTransformer is a GraphTransformer that adds all the lists
// from the query configuration to the graph.
type QueryConfigTransformer struct {
	// Module is the module to add resources from.
	Config *configs.Config

	// Mode will only add resources that match the given mode
	ModeFilter bool
	Mode       addrs.ResourceMode

	// generateConfigPathForImportTargets tells the graph where to write any
	// generated config for import targets that are not contained within config.
	//
	// If this is empty and an import target has no config, the graph will
	// simply import the state for the target and any follow-up operations will
	// try to delete the imported resource unless the config is updated
	// manually.
	generateConfigPathForImportTargets string
}

func (t *QueryConfigTransformer) Transform(g *Graph) error {
	// If no configuration is available, we don't do anything
	if t.Config == nil {
		return nil
	}

	// Start the transformation process
	return t.transform(g, t.Config)
}

func (t *QueryConfigTransformer) transform(g *Graph, config *configs.Config) error {
	// If no config, do nothing
	if config == nil {
		return nil
	}

	// Add our resources
	if err := t.transformSingle(g, config); err != nil {
		return err
	}

	// Skip nested modules for now
	// Transform all the children without generating config.
	// for _, c := range config.Children {
	// 	if err := t.transform(g, c); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func (t *QueryConfigTransformer) transformSingle(g *Graph, config *configs.Config) error {
	path := config.Path
	module := config.Module
	log.Printf("[TRACE] QueryConfigTransformer: Starting for path: %v", path)

	var allQueries []*configs.List
	for _, r := range module.Queries {
		for _, q := range r.Lists {
			allQueries = append(allQueries, q)
			node := &NodeQueryList{Config: q}
			g.Add(node)
		}
	}

	return nil
}
