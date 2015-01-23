package terraform

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// Graph takes a module tree and builds a logical graph of all the nodes
// in that module.
func Graph2(mod *module.Tree) (*dag.Graph, error) {
	// A module is required and also must be completely loaded.
	if mod == nil {
		return nil, errors.New("module must not be nil")
	}
	if !mod.Loaded() {
		return nil, errors.New("module must be loaded")
	}

	// Get the configuration for this module
	config := mod.Config()

	// Create the node list we'll use for the graph
	nodes := make([]graphNodeConfig, 0,
		(len(config.ProviderConfigs)+len(config.Modules)+len(config.Resources))*2)

	// Write all the provider configs out
	for _, pc := range config.ProviderConfigs {
		nodes = append(nodes, &GraphNodeConfigProvider{Provider: pc})
	}

	// Write all the resources out
	for _, r := range config.Resources {
		nodes = append(nodes, &GraphNodeConfigResource{Resource: r})
	}

	// Write all the modules out
	for _, m := range config.Modules {
		nodes = append(nodes, &GraphNodeConfigModule{Module: m})
	}

	// Build the full map of the var names to the nodes.
	fullMap := make(map[string]dag.Vertex)
	for _, n := range nodes {
		fullMap[n.VarName()] = n
	}

	// Build the graph vertices
	var g dag.Graph
	for _, n := range nodes {
		g.Add(n)
	}

	// Go through all the nodes and build up the actual graph edges. We
	// do this by getting the variables that each node depends on and then
	// building the dep map based on the fullMap which contains the mapping
	// of var names to the actual node with that name.
	for _, n := range nodes {
		for _, id := range n.Variables() {
			if target, ok := fullMap[id]; ok {
				g.Connect(dag.BasicEdge(n, target))
			}
		}
	}

	return &g, nil
}

// varNameForVar returns the VarName value for an interpolated variable.
// This value is compared to the VarName() value for the nodes within the
// graph to build the graph edges.
func varNameForVar(raw config.InterpolatedVariable) string {
	switch v := raw.(type) {
	case *config.ModuleVariable:
		return fmt.Sprintf("module.%s", v.Name)
	case *config.ResourceVariable:
		return v.ResourceId()
	default:
		return ""
	}
}
