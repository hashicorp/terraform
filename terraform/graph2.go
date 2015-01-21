package terraform

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/depgraph2"
)

// Graph takes a module tree and builds a logical graph of all the nodes
// in that module.
func Graph2(mod *module.Tree) (*depgraph.Graph, error) {
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
		(len(config.Modules)+len(config.Resources))*2)

	// Write all the resources out
	for _, r := range config.Resources {
		nodes = append(nodes, &GraphNodeConfigResource{
			Resource: r,
		})
	}

	// Write all the modules out
	// TODO

	// Build the full map of the var names to the nodes.
	fullMap := make(map[string]depgraph.Node)
	for _, n := range nodes {
		fullMap[n.VarName()] = n
	}

	// Go through all the nodes and build up the actual dependency map. We
	// do this by getting the variables that each node depends on and then
	// building the dep map based on the fullMap which contains the mapping
	// of var names to the actual node with that name.
	for _, n := range nodes {
		m := make(map[string]depgraph.Node)
		for _, v := range n.Variables() {
			id := varNameForVar(v)
			m[id] = fullMap[id]
		}

		n.setDepMap(m)
	}

	// Build the graph and return it
	g := &depgraph.Graph{Nodes: make([]depgraph.Node, 0, len(nodes))}
	for _, n := range nodes {
		g.Nodes = append(g.Nodes, n)
	}

	return g, nil
}

// graphNodeConfig is an interface that all graph nodes for the
// configuration graph need to implement in order to build the variable
// dependencies properly.
type graphNodeConfig interface {
	depgraph.Node

	// Variables returns the full list of variables that this node
	// depends on.
	Variables() map[string]config.InterpolatedVariable

	// VarName returns the name that is used to identify a variable
	// maps to this node. It should match the result of the
	// `VarName` function.
	VarName() string

	// setDepMap sets the dependency map for this node. If the node is
	// nil, then it wasn't found.
	setDepMap(map[string]depgraph.Node)
}

// GraphNodeConfigResource represents a resource within the configuration
// graph.
type GraphNodeConfigResource struct {
	Resource *config.Resource
	DepMap   map[string]depgraph.Node
}

func (n *GraphNodeConfigResource) Deps() []depgraph.Node {
	r := make([]depgraph.Node, 0, len(n.DepMap))
	for _, v := range n.DepMap {
		if v != nil {
			r = append(r, v)
		}
	}

	return r
}

func (n *GraphNodeConfigResource) Name() string {
	return n.Resource.Id()
}

func (n *GraphNodeConfigResource) Variables() map[string]config.InterpolatedVariable {
	var m map[string]config.InterpolatedVariable
	if n.Resource != nil {
		m = make(map[string]config.InterpolatedVariable)
		for k, v := range n.Resource.RawCount.Variables {
			m[k] = v
		}
		for k, v := range n.Resource.RawConfig.Variables {
			m[k] = v
		}
	}

	return m
}

func (n *GraphNodeConfigResource) VarName() string {
	return n.Resource.Id()
}

func (n *GraphNodeConfigResource) setDepMap(m map[string]depgraph.Node) {
	n.DepMap = m
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
