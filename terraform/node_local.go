package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
)

// NodeLocal represents a named local value in a particular module.
//
// Local value nodes only have one operation, common to all walk types:
// evaluate the result and place it in state.
type NodeLocal struct {
	Addr   addrs.AbsLocalValue
	Config *configs.Local
}

var (
	_ GraphNodeSubPath       = (*NodeLocal)(nil)
	_ RemovableIfNotTargeted = (*NodeLocal)(nil)
	_ GraphNodeReferenceable = (*NodeLocal)(nil)
	_ GraphNodeReferencer    = (*NodeLocal)(nil)
	_ GraphNodeEvalable      = (*NodeLocal)(nil)
	_ dag.GraphNodeDotter    = (*NodeLocal)(nil)
)

func (n *NodeLocal) Name() string {
	return n.Addr.String()
}

// GraphNodeSubPath
func (n *NodeLocal) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// RemovableIfNotTargeted
func (n *NodeLocal) RemoveIfNotTargeted() bool {
	return true
}

// GraphNodeReferenceable
func (n *NodeLocal) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.LocalValue}
}

// GraphNodeReferencer
func (n *NodeLocal) References() []*addrs.Reference {
	refs, _ := lang.ReferencesInExpr(n.Config.Expr)
	return appendResourceDestroyReferences(refs)
}

// GraphNodeEvalable
func (n *NodeLocal) EvalTree() EvalNode {
	return &EvalLocal{
		Addr: n.Addr.LocalValue,
		Expr: n.Config.Expr,
	}
}

// dag.GraphNodeDotter impl.
func (n *NodeLocal) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}
