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
	Output       **InstanceState
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

	var result *InstanceState
	if !n.Tainted {
		// Return the primary
		result = rs.Primary
	} else {
		// Get the index. If it is negative, then we get the last one
		idx := n.TaintedIndex
		if idx < 0 {
			idx = len(rs.Tainted) - 1
		}

		if idx < len(rs.Tainted) {
			// Return the proper tainted resource
			result = rs.Tainted[n.TaintedIndex]
		}
	}

	// Write the result to the output pointer
	if n.Output != nil {
		*n.Output = result
	}

	return result, nil
}

func (n *EvalReadState) Type() EvalType {
	return EvalTypeInstanceState
}

// EvalWriteState is an EvalNode implementation that reads the
// InstanceState for a specific resource out of the state.
type EvalWriteState struct {
	Name                string
	ResourceType        string
	Dependencies        []string
	State               **InstanceState
	Tainted             *bool
	TaintedIndex        int
	TaintedClearPrimary bool
}

func (n *EvalWriteState) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

// TODO: test
func (n *EvalWriteState) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
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

	if n.Tainted != nil && *n.Tainted {
		if n.TaintedIndex != -1 {
			rs.Tainted[n.TaintedIndex] = *n.State
		} else {
			rs.Tainted = append(rs.Tainted, *n.State)
		}

		if n.TaintedClearPrimary {
			rs.Primary = nil
		}
	} else {
		// Set the primary state
		rs.Primary = *n.State
	}
	println(fmt.Sprintf("%#v", rs))

	return nil, nil
}

func (n *EvalWriteState) Type() EvalType {
	return EvalTypeNull
}

// EvalDeposeState is an EvalNode implementation that reads the
// InstanceState for a specific resource out of the state.
type EvalDeposeState struct {
	Name string
}

func (n *EvalDeposeState) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

// TODO: test
func (n *EvalDeposeState) Eval(
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

	// If we don't have a primary, we have nothing to depose
	if rs.Primary == nil {
		return nil, nil
	}

	// Depose to the tainted
	rs.Tainted = append(rs.Tainted, rs.Primary)
	rs.Primary = nil

	return nil, nil
}

func (n *EvalDeposeState) Type() EvalType {
	return EvalTypeNull
}

// EvalUndeposeState is an EvalNode implementation that reads the
// InstanceState for a specific resource out of the state.
type EvalUndeposeState struct {
	Name string
}

func (n *EvalUndeposeState) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

// TODO: test
func (n *EvalUndeposeState) Eval(
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

	// If we don't have any tainted, then we don't have anything to do
	if len(rs.Tainted) == 0 {
		return nil, nil
	}

	// Undepose to the tainted
	idx := len(rs.Tainted) - 1
	rs.Primary = rs.Tainted[idx]
	rs.Tainted[idx] = nil

	return nil, nil
}

func (n *EvalUndeposeState) Type() EvalType {
	return EvalTypeNull
}
