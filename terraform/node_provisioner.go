package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config"
)

// NodeProvisioner represents a provider that has no associated operations.
// It registers all the common interfaces across operations for providers.
type NodeProvisioner struct {
	NameValue string
	PathValue addrs.ModuleInstance

	// The fields below will be automatically set using the Attach
	// interfaces if you're running those transforms, but also be explicitly
	// set if you already have that information.

	Config *config.ProviderConfig
}

var (
	_ GraphNodeSubPath     = (*NodeProvisioner)(nil)
	_ GraphNodeProvisioner = (*NodeProvisioner)(nil)
	_ GraphNodeEvalable    = (*NodeProvisioner)(nil)
)

func (n *NodeProvisioner) Name() string {
	result := fmt.Sprintf("provisioner.%s", n.NameValue)
	if len(n.PathValue) > 1 {
		result = fmt.Sprintf("%s.%s", n.PathValue.String(), result)
	}

	return result
}

// GraphNodeSubPath
func (n *NodeProvisioner) Path() addrs.ModuleInstance {
	return n.PathValue
}

// GraphNodeProvisioner
func (n *NodeProvisioner) ProvisionerName() string {
	return n.NameValue
}

// GraphNodeEvalable impl.
func (n *NodeProvisioner) EvalTree() EvalNode {
	return &EvalInitProvisioner{Name: n.NameValue}
}
