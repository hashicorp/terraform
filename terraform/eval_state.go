package terraform

import (
	"fmt"
)

// EvalReadState is an EvalNode implementation that reads the
// InstanceState for a specific resource out of the state.
type EvalReadState struct {
	Name         string
	Tainted      bool
	TaintedIndex int
}

func (n *EvalReadState) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

// TODO: test
func (n *EvalReadState) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	state, lock := ctx.State()

	// Get a read lock so we can access this instance
	lock.RLock()
	defer lock.RUnlock()

	// Look for the module state. If we don't have one, then it doesn't matter.
	mod := state.ModuleByPath(ctx.Path())
	if mod == nil {
		return nil, nil
	}

	// Look for the resource state. If we don't have one, then it is okay.
	rs := mod.Resources[n.Name]
	if rs == nil {
		return nil, nil
	}

	if !n.Tainted {
		// Return the primary
		return rs.Primary, nil
	} else {
		// Return the proper tainted resource
		return rs.Tainted[n.TaintedIndex], nil
	}
}

func (n *EvalReadState) Type() EvalType {
	return EvalTypeInstanceState
}

// EvalWriteState is an EvalNode implementation that reads the
// InstanceState for a specific resource out of the state.
type EvalWriteState struct {
	Name         string
	ResourceType string
	Dependencies []string
	State        EvalNode
	Tainted      bool
	TaintedIndex int
}

func (n *EvalWriteState) Args() ([]EvalNode, []EvalType) {
	return []EvalNode{n.State}, []EvalType{EvalTypeInstanceState}
}

// TODO: test
func (n *EvalWriteState) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	var instanceState *InstanceState
	if args[0] != nil {
		instanceState = args[0].(*InstanceState)
	}

	state, lock := ctx.State()
	if state == nil {
		return nil, fmt.Errorf("cannot write state to nil state")
	}

	// Get a write lock so we can access this instance
	lock.Lock()
	defer lock.Unlock()

	// Look for the module state. If we don't have one, create it.
	mod := state.ModuleByPath(ctx.Path())
	if mod == nil {
		mod = state.AddModule(ctx.Path())
	}

	// Look for the resource state.
	rs := mod.Resources[n.Name]
	if rs == nil {
		rs = &ResourceState{}
		rs.init()
		mod.Resources[n.Name] = rs
	}
	rs.Type = n.ResourceType
	rs.Dependencies = n.Dependencies

	if n.Tainted {
		if n.TaintedIndex != -1 {
			rs.Tainted[n.TaintedIndex] = instanceState
		}
	} else {
		// Set the primary state
		rs.Primary = instanceState
	}

	// Prune because why not, we can clear out old useless entries now
	rs.prune()
	return nil, nil
}

func (n *EvalWriteState) Type() EvalType {
	return EvalTypeNull
}
