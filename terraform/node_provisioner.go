package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
)

// NodeProvisioner represents a provider that has no associated operations.
// It registers all the common interfaces across operations for providers.
type NodeProvisioner struct {
	NameValue string
	PathValue addrs.ModuleInstance
}

var (
	_ GraphNodeSubPath     = (*NodeProvisioner)(nil)
	_ GraphNodeProvisioner = (*NodeProvisioner)(nil)
	_ GraphNodeEvalable    = (*NodeProvisioner)(nil)
)

func (n *NodeProvisioner) Name() string {
	result := fmt.Sprintf("provisioner.%s", n.NameValue)
	if len(n.PathValue) > 0 {
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
