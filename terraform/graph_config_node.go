package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/depgraph2"
)

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

	// depMap and setDepMap are used to get and set the dependency map
	// for this node. This is used to modify the dependencies. The key of
	// this map should be the VarName() of graphNodeConfig.
	depMap() map[string]depgraph.Node
	setDepMap(map[string]depgraph.Node)
}

// graphNodeConfigBasicDepMap is a struct that provides the Deps(),
// depMap(), and setDepMap() functions to help satisfy the graphNodeConfig
// interface. This struct is meant to be embedded into other nodes to get
// these features for free.
type graphNodeConfigBasicDepMap struct {
	DepMap map[string]depgraph.Node
}

func (n *graphNodeConfigBasicDepMap) Deps() []depgraph.Node {
	r := make([]depgraph.Node, 0, len(n.DepMap))
	for _, v := range n.DepMap {
		if v != nil {
			r = append(r, v)
		}
	}

	return r
}

func (n *graphNodeConfigBasicDepMap) depMap() map[string]depgraph.Node {
	return n.DepMap
}

func (n *graphNodeConfigBasicDepMap) setDepMap(m map[string]depgraph.Node) {
	n.DepMap = m
}

// GraphNodeConfigProvider represents a resource within the config graph.
type GraphNodeConfigProvider struct {
	graphNodeConfigBasicDepMap

	Provider *config.ProviderConfig
}

func (n *GraphNodeConfigProvider) Name() string {
	return fmt.Sprintf("provider.%s", n.Provider.Name)
}

func (n *GraphNodeConfigProvider) Variables() map[string]config.InterpolatedVariable {
	return n.Provider.RawConfig.Variables
}

func (n *GraphNodeConfigProvider) VarName() string {
	return "never valid"
}

// GraphNodeConfigResource represents a resource within the config graph.
type GraphNodeConfigResource struct {
	graphNodeConfigBasicDepMap

	Resource *config.Resource
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
