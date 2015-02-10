package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// graphNodeConfig is an interface that all graph nodes for the
// configuration graph need to implement in order to build the variable
// dependencies properly.
type graphNodeConfig interface {
	dag.NamedVertex

	// All graph nodes should be dependent on other things, and able to
	// be depended on.
	GraphNodeDependable
	GraphNodeDependent
}

// GraphNodeConfigModule represents a module within the configuration graph.
type GraphNodeConfigModule struct {
	Path   []string
	Module *config.Module
	Tree   *module.Tree
}

func (n *GraphNodeConfigModule) DependableName() []string {
	return []string{n.Name()}
}

func (n *GraphNodeConfigModule) DependentOn() []string {
	vars := n.Module.RawConfig.Variables
	result := make([]string, 0, len(vars))
	for _, v := range vars {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}

	return result
}

func (n *GraphNodeConfigModule) Name() string {
	return fmt.Sprintf("module.%s", n.Module.Name)
}

// GraphNodeExpandable
func (n *GraphNodeConfigModule) Expand(b GraphBuilder) (*Graph, error) {
	return b.Build(n.Path)
}

// GraphNodeExpandable
func (n *GraphNodeConfigModule) ProvidedBy() []string {
	// Build up the list of providers by simply going over our configuration
	// to find the providers that are configured there as well as the
	// providers that the resources use.
	config := n.Tree.Config()
	providers := make(map[string]struct{})
	for _, p := range config.ProviderConfigs {
		providers[p.Name] = struct{}{}
	}
	for _, r := range config.Resources {
		providers[resourceProvider(r.Type)] = struct{}{}
	}

	// Turn the map into a string. This makes sure that the list is
	// de-dupped since we could be going over potentially many resources.
	result := make([]string, 0, len(providers))
	for p, _ := range providers {
		result = append(result, p)
	}

	return result
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

func (n *GraphNodeConfigProvider) DependableName() []string {
	return []string{n.Name()}
}

func (n *GraphNodeConfigProvider) DependentOn() []string {
	vars := n.Provider.RawConfig.Variables
	result := make([]string, 0, len(vars))
	for _, v := range vars {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}

	return result
}

// GraphNodeEvalable impl.
func (n *GraphNodeConfigProvider) EvalTree() EvalNode {
	return ProviderEvalTree(n.Provider.Name, n.Provider.RawConfig)
}

// GraphNodeProvider implementation
func (n *GraphNodeConfigProvider) ProviderName() string {
	return n.Provider.Name
}

// GraphNodeConfigResource represents a resource within the config graph.
type GraphNodeConfigResource struct {
	Resource *config.Resource
}

func (n *GraphNodeConfigResource) DependableName() []string {
	return []string{n.Resource.Id()}
}

// GraphNodeDependent impl.
func (n *GraphNodeConfigResource) DependentOn() []string {
	result := make([]string, len(n.Resource.DependsOn),
		len(n.Resource.RawCount.Variables)+
			len(n.Resource.RawConfig.Variables)+
			len(n.Resource.DependsOn))
	copy(result, n.Resource.DependsOn)
	for _, v := range n.Resource.RawCount.Variables {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}
	for _, v := range n.Resource.RawConfig.Variables {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}

	return result
}

func (n *GraphNodeConfigResource) Name() string {
	return n.Resource.Id()
}

// GraphNodeDynamicExpandable impl.
func (n *GraphNodeConfigResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	// Build the graph
	b := &BasicGraphBuilder{
		Steps: []GraphTransformer{
			&ResourceCountTransformer{Resource: n.Resource},
			&RootTransformer{},
		},
	}

	return b.Build(ctx.Path())
}

// GraphNodeEvalable impl.
func (n *GraphNodeConfigResource) EvalTree() EvalNode {
	return &EvalValidateCount{
		Resource: n.Resource,
	}
}

// GraphNodeProviderConsumer
func (n *GraphNodeConfigResource) ProvidedBy() []string {
	return []string{resourceProvider(n.Resource.Type)}
}

// GraphNodeProvisionerConsumer
func (n *GraphNodeConfigResource) ProvisionedBy() []string {
	result := make([]string, len(n.Resource.Provisioners))
	for i, p := range n.Resource.Provisioners {
		result[i] = p.Type
	}

	return result
}
