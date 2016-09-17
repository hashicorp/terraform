package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// NodeDestroyResource represents a resource that is to be destroyed.
type NodeApplyableResource struct {
	Addr *ResourceAddress // Addr is the address for this resource
}

func (n *NodeApplyableResource) Name() string {
	return n.Addr.String()
}

// GraphNodeSubPath
func (n *NodeApplyableResource) Path() []string {
	return n.Addr.Path
}

// GraphNodeReferenceable
func (n *NodeApplyableResource) ReferenceableName() []string {
	if n.Config == nil {
		return nil
	}

	return []string{n.Config.Id()}
}

// GraphNodeProviderConsumer
func (n *NodeApplyableResource) ProvidedBy() []string {
	// If we have a config we prefer that above all else
	if n.Config != nil {
		return []string{resourceProvider(n.Config.Type, n.Config.Provider)}
	}

	// If we have state, then we will use the provider from there
	if n.ResourceState != nil {
		return []string{n.ResourceState.Provider}
	}

	// Use our type
	return []string{resourceProvider(n.Addr.Type, "")}
}

// GraphNodeEvalable
func (n *NodeApplyableResource) EvalTree() EvalNode {
	return nil
}
