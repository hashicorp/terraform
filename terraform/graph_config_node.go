package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// graphNodeConfig is an interface that all graph nodes for the
// configuration graph need to implement in order to build the variable
// dependencies properly.
type graphNodeConfig interface {
	dag.Vertex

	// Variables returns the full list of variables that this node
	// depends on. The values within the slice should map to the VarName()
	// values that are returned by any nodes.
	Variables() []string

	// VarName returns the name that is used to identify a variable
	// maps to this node. It should match the result of the
	// `VarName` function.
	VarName() string
}

// GraphNodeConfigModule represents a module within the configuration graph.
type GraphNodeConfigModule struct {
	Module *config.Module
}

func (n *GraphNodeConfigModule) Name() string {
	return fmt.Sprintf("module.%s", n.Module.Name)
}

func (n *GraphNodeConfigModule) Variables() []string {
	vars := n.Module.RawConfig.Variables
	result := make([]string, 0, len(vars))
	for _, v := range vars {
		result = append(result, varNameForVar(v))
	}

	return result
}

func (n *GraphNodeConfigModule) VarName() string {
	return n.Name()
}

// GraphNodeConfigProvider represents a configured provider within the
// configuration graph. These are only immediately in the graph when an
// explicit `provider` configuration block is in the configuration.
type GraphNodeConfigProvider struct {
	Provider *config.ProviderConfig
}

func (n *GraphNodeConfigProvider) Name() string {
	return fmt.Sprintf("provider.%s", n.Provider.Name)
}

func (n *GraphNodeConfigProvider) Variables() []string {
	vars := n.Provider.RawConfig.Variables
	result := make([]string, 0, len(vars))
	for _, v := range vars {
		result = append(result, varNameForVar(v))
	}

	return result
}

func (n *GraphNodeConfigProvider) VarName() string {
	return "never valid"
}

// GraphNodeConfigResource represents a resource within the config graph.
type GraphNodeConfigResource struct {
	Resource *config.Resource
}

func (n *GraphNodeConfigResource) Name() string {
	return n.Resource.Id()
}

func (n *GraphNodeConfigResource) Variables() []string {
	result := make([]string, len(n.Resource.DependsOn),
		len(n.Resource.RawCount.Variables)+
			len(n.Resource.RawConfig.Variables)+
			len(n.Resource.DependsOn))
	copy(result, n.Resource.DependsOn)
	for _, v := range n.Resource.RawCount.Variables {
		result = append(result, varNameForVar(v))
	}
	for _, v := range n.Resource.RawConfig.Variables {
		result = append(result, varNameForVar(v))
	}

	return result
}

func (n *GraphNodeConfigResource) VarName() string {
	return n.Resource.Id()
}
