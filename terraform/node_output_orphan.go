package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
)

// NodeOutputOrphan represents an output that is an orphan.
type NodeOutputOrphan struct {
	Addr addrs.AbsOutputValue
}

var (
	_ GraphNodeSubPath          = (*NodeOutputOrphan)(nil)
	_ GraphNodeReferenceable    = (*NodeOutputOrphan)(nil)
	_ GraphNodeReferenceOutside = (*NodeOutputOrphan)(nil)
	_ GraphNodeEvalable         = (*NodeOutputOrphan)(nil)
)

func (n *NodeOutputOrphan) Name() string {
	return fmt.Sprintf("%s (orphan)", n.Addr.String())
}

// GraphNodeReferenceOutside implementation
func (n *NodeOutputOrphan) ReferenceOutside() (selfPath, referencePath addrs.ModuleInstance) {
	return referenceOutsideForOutput(n.Addr)
}

// GraphNodeReferenceable
func (n *NodeOutputOrphan) ReferenceableAddrs() []addrs.Referenceable {
	return referenceableAddrsForOutput(n.Addr)
}

// GraphNodeSubPath
func (n *NodeOutputOrphan) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// GraphNodeEvalable
func (n *NodeOutputOrphan) EvalTree() EvalNode {
	return &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkApply, walkDestroy},
		Node: &EvalDeleteOutput{
			Addr: n.Addr.OutputValue,
		},
	}
}
