package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// ConcreteProviderNodeFunc is a callback type used to convert an
// abstract provider to a concrete one of some type.
type ConcreteProviderNodeFunc func(*NodeAbstractProvider) dag.Vertex

// NodeAbstractProvider represents a provider that has no associated operations.
// It registers all the common interfaces across operations for providers.
type NodeAbstractProvider struct {
	NameValue string
	PathValue []string

	// The fields below will be automatically set using the Attach
	// interfaces if you're running those transforms, but also be explicitly
	// set if you already have that information.

	Config *config.ProviderConfig
}

func ResolveProviderName(name string, path []string) string {
	name = fmt.Sprintf("provider.%s", name)
	if len(path) >= 1 {
		name = fmt.Sprintf("%s.%s", modulePrefixStr(path), name)
	}

	return name
}

func (n *NodeAbstractProvider) Name() string {
	return ResolveProviderName(n.NameValue, n.PathValue)
}

// GraphNodeSubPath
func (n *NodeAbstractProvider) Path() []string {
	return n.PathValue
}

// RemovableIfNotTargeted
func (n *NodeAbstractProvider) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeReferencer
func (n *NodeAbstractProvider) References() []string {
	if n.Config == nil {
		return nil
	}

	return ReferencesFromConfig(n.Config.RawConfig)
}

// GraphNodeProvider
func (n *NodeAbstractProvider) ProviderName() string {
	return n.NameValue
}

// GraphNodeProvider
func (n *NodeAbstractProvider) ProviderConfig() *config.ProviderConfig {
	if n.Config == nil {
		return nil
	}

	return n.Config
}

// GraphNodeAttachProvider
func (n *NodeAbstractProvider) AttachProvider(c *config.ProviderConfig) {
	n.Config = c
}

// GraphNodeDotter impl.
func (n *NodeAbstractProvider) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "diamond",
		},
	}
}
