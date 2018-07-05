package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/configs"

	"github.com/hashicorp/terraform/dag"
)

// ConcreteProviderNodeFunc is a callback type used to convert an
// abstract provider to a concrete one of some type.
type ConcreteProviderNodeFunc func(*NodeAbstractProvider) dag.Vertex

// NodeAbstractProvider represents a provider that has no associated operations.
// It registers all the common interfaces across operations for providers.
type NodeAbstractProvider struct {
	Addr addrs.AbsProviderConfig

	// The fields below will be automatically set using the Attach
	// interfaces if you're running those transforms, but also be explicitly
	// set if you already have that information.

	Config *configs.Provider
	Schema *configschema.Block
}

var (
	_ GraphNodeSubPath                    = (*NodeAbstractProvider)(nil)
	_ RemovableIfNotTargeted              = (*NodeAbstractProvider)(nil)
	_ GraphNodeReferencer                 = (*NodeAbstractProvider)(nil)
	_ GraphNodeProvider                   = (*NodeAbstractProvider)(nil)
	_ GraphNodeAttachProvider             = (*NodeAbstractProvider)(nil)
	_ GraphNodeAttachProviderConfigSchema = (*NodeAbstractProvider)(nil)
	_ dag.GraphNodeDotter                 = (*NodeAbstractProvider)(nil)
)

func (n *NodeAbstractProvider) Name() string {
	return n.Addr.String()
}

// GraphNodeSubPath
func (n *NodeAbstractProvider) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// RemovableIfNotTargeted
func (n *NodeAbstractProvider) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeReferencer
func (n *NodeAbstractProvider) References() []*addrs.Reference {
	if n.Config == nil || n.Schema == nil {
		return nil
	}

	return ReferencesFromConfig(n.Config.Config, n.Schema)
}

// GraphNodeProvider
func (n *NodeAbstractProvider) ProviderAddr() addrs.AbsProviderConfig {
	return n.Addr
}

// GraphNodeProvider
func (n *NodeAbstractProvider) ProviderConfig() *configs.Provider {
	if n.Config == nil {
		return nil
	}

	return n.Config
}

// GraphNodeAttachProvider
func (n *NodeAbstractProvider) AttachProvider(c *configs.Provider) {
	n.Config = c
}

// GraphNodeAttachProviderConfigSchema impl.
func (n *NodeAbstractProvider) AttachProviderConfigSchema(schema *configschema.Block) {
	n.Schema = schema
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
