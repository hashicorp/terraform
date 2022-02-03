package componentstree

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/terraform-ng/internal/ngaddrs"
	"github.com/hashicorp/terraform/internal/terraform-ng/internal/tfcomponents"
)

// Node represents a single node in a components tree. Each node corresponds
// with a single component group.
type Node struct {
	// Parent refers to the parent node of this node, or nil if this is the
	// root node of the tree.
	Parent *Node

	// Root refers to the root node of the tree. It's a self-reference when
	// inside the root node of the tree already.
	Root *Node

	// CallPath is the sequence of static component group calls leading to
	// this node. For the root node in a tree, this has length zero.
	CallPath []ngaddrs.ComponentGroupCall

	// Children is a mapping of the child nodes created by component group
	// calls in the configuration belonging to this node, identified by their
	// call address.
	Children map[ngaddrs.ComponentGroupCall]*Node

	// Config is the component group configuration belonging to this node in
	// the component group tree.
	Config *tfcomponents.Config

	// SourceAddr is the address where the components configuration was
	// loaded from. If this is a local source address then it's relative
	// to the parent's SourceAddr, or to the current working directory if
	// there is no parent.
	SourceAddr addrs.ModuleSource
}

func (n *Node) ChildCallConfig(addr ngaddrs.ComponentGroupCall) *tfcomponents.ComponentGroup {
	return n.Config.Groups[addr.Name]
}
