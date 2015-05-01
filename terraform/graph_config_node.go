package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/dot"
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

	// ConfigType returns the type of thing in the configuration that
	// this node represents, such as a resource, module, etc.
	ConfigType() GraphNodeConfigType
}

// GraphNodeAddressable is an interface that all graph nodes for the
// configuration graph need to implement in order to be be addressed / targeted
// properly.
type GraphNodeAddressable interface {
	graphNodeConfig

	ResourceAddress() *ResourceAddress
}

// GraphNodeTargetable is an interface for graph nodes to implement when they
// need to be told about incoming targets. This is useful for nodes that need
// to respect targets as they dynamically expand. Note that the list of targets
// provided will contain every target provided, and each implementing graph
// node must filter this list to targets considered relevant.
type GraphNodeTargetable interface {
	GraphNodeAddressable

	SetTargets([]ResourceAddress)
}

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
func (n *GraphNodeConfigProvider) DotNode(name string, opts *GraphDotOpts) *dot.Node {
	return dot.NewNode(name, map[string]string{
		"label": n.Name(),
		"shape": "diamond",
	})
}

// GraphNodeDotterOrigin impl.
func (n *GraphNodeConfigProvider) DotOrigin() bool {
	return true
}
