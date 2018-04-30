package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
)

// NodeModuleRemoved represents a module that is no longer in the
// config.
type NodeModuleRemoved struct {
	Addr addrs.ModuleInstance
}

var (
	_ GraphNodeSubPath          = (*NodeModuleRemoved)(nil)
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
		Node: &EvalDeleteModule{
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

// EvalDeleteModule is an EvalNode implementation that removes an empty module
// entry from the state.
type EvalDeleteModule struct {
	Addr addrs.ModuleInstance
}

func (n *EvalDeleteModule) Eval(ctx EvalContext) (interface{}, error) {
	state, lock := ctx.State()
	if state == nil {
		return nil, nil
	}

	// Get a write lock so we can access this instance
	lock.Lock()
	defer lock.Unlock()

	// Make sure we have a clean state
	// Destroyed resources aren't deleted, they're written with an ID of "".
	state.prune()

	// find the module and delete it
Modules:
	for i, m := range state.Modules {
		// Since state is still using our old-style []string path representation,
		// comparison is a little awkward. This can be simplified once state
		// is updated to use addrs.ModuleInstance too.
		if len(m.Path) != len(n.Addr) {
			continue Modules
		}
		for i, step := range n.Addr {
			if step.InstanceKey != addrs.NoKey {
				// Old-style state path can't have keys anyway, so this can
				// never match.
				continue Modules
			}
			if step.Name != m.Path[i] {
				continue Modules
			}
		}

		if !m.Empty() {
			// a targeted apply may leave module resources even without a config,
			// so just log this and return.
			log.Printf("[DEBUG] not removing %s from state: not empty", n.Addr)
			break
		}
		state.Modules = append(state.Modules[:i], state.Modules[i+1:]...)
		break
	}

	return nil, nil
}
