package terraform

import (
	"fmt"
	"log"
	"reflect"
)

// NodeModuleRemoved represents a module that is no longer in the
// config.
type NodeModuleRemoved struct {
	PathValue []string
}

func (n *NodeModuleRemoved) Name() string {
	return fmt.Sprintf("%s (removed)", modulePrefixStr(n.PathValue))
}

// GraphNodeSubPath
func (n *NodeModuleRemoved) Path() []string {
	return n.PathValue
}

// GraphNodeEvalable
func (n *NodeModuleRemoved) EvalTree() EvalNode {
	return &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkApply, walkDestroy},
		Node: &EvalDeleteModule{
			PathValue: n.PathValue,
		},
	}
}

// EvalDeleteModule is an EvalNode implementation that removes an empty module
// entry from the state.
type EvalDeleteModule struct {
	PathValue []string
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
	for i, m := range state.Modules {
		if reflect.DeepEqual(m.Path, n.PathValue) {
			if !m.Empty() {
				// a targeted apply may leave module resources even without a config,
				// so just log this and return.
				log.Printf("[DEBUG] cannot remove module %s, not empty", modulePrefixStr(n.PathValue))
				break
			}
			tail := len(state.Modules) - 1
			state.Modules[i] = state.Modules[tail]
			state.Modules = state.Modules[:tail]
			break
		}
	}

	return nil, nil
}
