package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
)

// NodeModuleRemoved represents a module that is no longer in the
// config.
type NodeModuleRemoved struct {
	Addr addrs.ModuleInstance
}

var (
	_ GraphNodeSubPath          = (*NodeModuleRemoved)(nil)
	_ RemovableIfNotTargeted    = (*NodeModuleRemoved)(nil)
	_ GraphNodeEvalable         = (*NodeModuleRemoved)(nil)
	_ GraphNodeReferencer       = (*NodeModuleRemoved)(nil)
	_ GraphNodeReferenceOutside = (*NodeModuleRemoved)(nil)
)

func (n *NodeModuleRemoved) Name() string {
	return fmt.Sprintf("%s (removed)", n.Addr.String())
}

// GraphNodeSubPath
func (n *NodeModuleRemoved) Path() addrs.ModuleInstance {
	return n.Addr
}

// GraphNodeEvalable
func (n *NodeModuleRemoved) EvalTree() EvalNode {
	return &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkApply, walkDestroy},
		Node: &EvalCheckModuleRemoved{
			Addr: n.Addr,
		},
	}
}

func (n *NodeModuleRemoved) ReferenceOutside() (selfPath, referencePath addrs.ModuleInstance) {
	// Our "References" implementation indicates that this node depends on
	// the call to the module it represents, which implicitly depends on
	// everything inside the module. That reference must therefore be
	// interpreted in terms of our parent module.
	return n.Addr, n.Addr.Parent()
}

func (n *NodeModuleRemoved) References() []*addrs.Reference {
	// We depend on the call to the module we represent, because that
	// implicitly then depends on everything inside that module.
	// Our ReferenceOutside implementation causes this to be interpreted
	// within the parent module.

	_, call := n.Addr.CallInstance()
	return []*addrs.Reference{
		{
			Subject: call,

			// No source range here, because there's nothing reasonable for
			// us to return.
		},
	}
}

// RemovableIfNotTargeted
func (n *NodeModuleRemoved) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// EvalCheckModuleRemoved is an EvalNode implementation that verifies that
// a module has been removed from the state as expected.
type EvalCheckModuleRemoved struct {
	Addr addrs.ModuleInstance
}

func (n *EvalCheckModuleRemoved) Eval(ctx EvalContext) (interface{}, error) {
	mod := ctx.State().Module(n.Addr)
	if mod != nil {
		// If we get here then that indicates a bug either in the states
		// module or in an earlier step of the graph walk, since we should've
		// pruned out the module when the last resource was removed from it.
		return nil, fmt.Errorf("leftover module %s in state that should have been removed; this is a bug in Terraform and should be reported", n.Addr)
	}
	return nil, nil
}
