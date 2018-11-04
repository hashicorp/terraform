package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
)

// NodeRootVariable represents a root variable input.
type NodeRootVariable struct {
	Addr   addrs.InputVariable
	Config *configs.Variable
}

var (
	_ GraphNodeSubPath       = (*NodeRootVariable)(nil)
	_ GraphNodeReferenceable = (*NodeRootVariable)(nil)
	_ dag.GraphNodeDotter    = (*NodeApplyableModuleVariable)(nil)
)

func (n *NodeRootVariable) Name() string {
	return n.Addr.String()
}

// GraphNodeSubPath
func (n *NodeRootVariable) Path() addrs.ModuleInstance {
	return addrs.RootModuleInstance
}

// GraphNodeReferenceable
func (n *NodeRootVariable) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr}
}

// dag.GraphNodeDotter impl.
func (n *NodeRootVariable) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}
