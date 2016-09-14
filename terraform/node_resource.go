package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// NodeApplyableResource represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodeApplyableResource struct {
	Addr          *ResourceAddress // Addr is the address for this resource
	Config        *config.Resource // Config is the resource in the config
	ResourceState *ResourceState   // ResourceState is the ResourceState for this
}

func (n *NodeApplyableResource) Name() string {
	return n.Addr.String()
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
