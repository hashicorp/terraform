package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeConfigProvider represents a configured provider within the
// configuration graph. These are only immediately in the graph when an
// explicit `provider` configuration block is in the configuration.
type GraphNodeConfigProvider struct {
	Provider *config.ProviderConfig
}

func (n *GraphNodeConfigProvider) Name() string {
	return fmt.Sprintf("provider.%s", n.ProviderName())
}

func (n *GraphNodeConfigProvider) ConfigType() GraphNodeConfigType {
	return GraphNodeConfigTypeProvider
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
	return ProviderEvalTree(n.ProviderName(), n.Provider.RawConfig)
}

// GraphNodeProvider implementation
func (n *GraphNodeConfigProvider) ProviderName() string {
	if n.Provider.Alias == "" {
		return n.Provider.Name
	} else {
		return fmt.Sprintf("%s.%s", n.Provider.Name, n.Provider.Alias)
	}
}

// GraphNodeProvider implementation
func (n *GraphNodeConfigProvider) ProviderConfig() *config.RawConfig {
	return n.Provider.RawConfig
}

// GraphNodeDotter impl.
func (n *GraphNodeConfigProvider) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "diamond",
		},
	}
}

// GraphNodeDotterOrigin impl.
func (n *GraphNodeConfigProvider) DotOrigin() bool {
	return true
}

// GraphNodeFlattenable impl.
func (n *GraphNodeConfigProvider) Flatten(p []string) (dag.Vertex, error) {
	return &GraphNodeConfigProviderFlat{
		GraphNodeConfigProvider: n,
		PathValue:               p,
	}, nil
}

// Same as GraphNodeConfigProvider, but for flattening
type GraphNodeConfigProviderFlat struct {
	*GraphNodeConfigProvider

	PathValue []string
}

func (n *GraphNodeConfigProviderFlat) Name() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue), n.GraphNodeConfigProvider.Name())
}

func (n *GraphNodeConfigProviderFlat) Path() []string {
	return n.PathValue
}

func (n *GraphNodeConfigProviderFlat) DependableName() []string {
	return modulePrefixList(
		n.GraphNodeConfigProvider.DependableName(),
		modulePrefixStr(n.PathValue))
}

func (n *GraphNodeConfigProviderFlat) DependentOn() []string {
	prefixed := modulePrefixList(
		n.GraphNodeConfigProvider.DependentOn(),
		modulePrefixStr(n.PathValue))

	result := make([]string, len(prefixed), len(prefixed)+1)
	copy(result, prefixed)

	// If we're in a module, then depend on our parent's provider
	if len(n.PathValue) > 1 {
		prefix := modulePrefixStr(n.PathValue[:len(n.PathValue)-1])
		if prefix != "" {
			prefix += "."
		}

		result = append(result, fmt.Sprintf(
			"%s%s",
			prefix, n.GraphNodeConfigProvider.Name()))
	}

	return result
}

func (n *GraphNodeConfigProviderFlat) ProviderName() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue),
		n.GraphNodeConfigProvider.ProviderName())
}
