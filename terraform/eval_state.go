package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
)

// EvalReadState is an EvalNode implementation that reads the
// primary InstanceState for a specific resource out of the state.
type EvalReadState struct {
	Name   string
	Output **InstanceState
}

func (n *EvalReadState) Eval(ctx EvalContext) (interface{}, error) {
	return readInstanceFromState(ctx, n.Name, n.Output, func(rs *ResourceState) (*InstanceState, error) {
		return rs.Primary, nil
	})
}

// EvalReadStateDeposed is an EvalNode implementation that reads the
// deposed InstanceState for a specific resource out of the state
type EvalReadStateDeposed struct {
	Name   string
	Output **InstanceState
	// Index indicates which instance in the Deposed list to target, or -1 for
	// the last item.
	Index int
}

func (n *EvalReadStateDeposed) Eval(ctx EvalContext) (interface{}, error) {
	return readInstanceFromState(ctx, n.Name, n.Output, func(rs *ResourceState) (*InstanceState, error) {
		// Get the index. If it is negative, then we get the last one
		idx := n.Index
		if idx < 0 {
			idx = len(rs.Deposed) - 1
		}
		if idx >= 0 && idx < len(rs.Deposed) {
			return rs.Deposed[idx], nil
		} else {
			return nil, fmt.Errorf("bad deposed index: %d, for resource: %#v", idx, rs)
		}
	})
}

// Does the bulk of the work for the various flavors of ReadState eval nodes.
// Each node just provides a reader function to get from the ResourceState to the
// InstanceState, and this takes care of all the plumbing.
func readInstanceFromState(
	ctx EvalContext,
	resourceName string,
	output **InstanceState,
	readerFn func(*ResourceState) (*InstanceState, error),
) (*InstanceState, error) {
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
	rs := mod.Resources[resourceName]
	if rs == nil {
		return nil, nil
	}

	// Use the delegate function to get the instance state from the resource state
	is, err := readerFn(rs)
	if err != nil {
		return nil, err
	}

	// Write the result to the output pointer
	if output != nil {
		*output = is
	}

	return is, nil
}

// EvalRequireState is an EvalNode implementation that early exits
// if the state doesn't have an ID.
type EvalRequireState struct {
	State **InstanceState
}

func (n *EvalRequireState) Eval(ctx EvalContext) (interface{}, error) {
	if n.State == nil {
		return nil, EvalEarlyExitError{}
	}

	state := *n.State
	if state == nil || state.ID == "" {
		return nil, EvalEarlyExitError{}
	}

	return nil, nil
}

// EvalUpdateStateHook is an EvalNode implementation that calls the
// PostStateUpdate hook with the current state.
type EvalUpdateStateHook struct{}

func (n *EvalUpdateStateHook) Eval(ctx EvalContext) (interface{}, error) {
	state, lock := ctx.State()

	// Get a full lock. Even calling something like WriteState can modify
	// (prune) the state, so we need the full lock.
	lock.Lock()
	defer lock.Unlock()

	// Call the hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostStateUpdate(state)
	})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// EvalWriteState is an EvalNode implementation that writes the
// primary InstanceState for a specific resource into the state.
type EvalWriteState struct {
	Name         string
	ResourceType string
	Provider     string
	Dependencies []string
	Config       *config.Resource
	State        **InstanceState
}

func (n *EvalWriteState) Eval(ctx EvalContext) (interface{}, error) {
	return writeInstanceToState(ctx, n,
		func(rs *ResourceState) **InstanceState {
			return &rs.Primary
		},
	)
}

// EvalWriteStateDeposed is an EvalNode implementation that writes
// an InstanceState out to the Deposed list of a resource in the state.
type EvalWriteStateDeposed struct {
	EvalWriteState
	// Index indicates which instance in the Deposed list to target, or -1 to append.
	Index int
}

func (n *EvalWriteStateDeposed) Eval(ctx EvalContext) (interface{}, error) {
	return writeInstanceToState(ctx, &n.EvalWriteState,
		func(rs *ResourceState) **InstanceState {
			if n.Index == -1 {
				rs.Deposed = append(rs.Deposed, nil)
				return &rs.Deposed[len(rs.Deposed)-1]
			}
			return &rs.Deposed[n.Index]
		},
	)
}

// Pulls together the common tasks of the EvalWriteState nodes.  All the args
// are passed directly down from the EvalNode along with a `getTarget` function
// that extracts the proper location to write the instance to.
func writeInstanceToState(
	ctx EvalContext,
	spec *EvalWriteState,
	getTarget func(*ResourceState) **InstanceState,
) (*InstanceState, error) {
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
	rs := mod.Resources[spec.Name]
	if rs == nil {
		rs = &ResourceState{}
		rs.init()
		mod.Resources[spec.Name] = rs
	}
	rs.Type = spec.ResourceType
	rs.Dependencies = spec.Dependencies
	rs.Provider = spec.Provider

	// We don't want to write no_store attributes into the state
	if *spec.State != nil && spec.Config != nil {
		for _, pattern := range spec.Config.Lifecycle.NoStore {
			if pattern == "*" {
				(*spec.State).Attributes = make(map[string]string)
				break
			}
			for key := range (*spec.State).Attributes {
				if strings.HasPrefix(key, pattern) {
					delete((*spec.State).Attributes, key)
				}
			}
		}
	}

	target := getTarget(rs)
	*target = *spec.State

	return nil, nil
}

// EvalClearPrimaryState is an EvalNode implementation that clears the primary
// instance from a resource state.
type EvalClearPrimaryState struct {
	Name string
}

func (n *EvalClearPrimaryState) Eval(ctx EvalContext) (interface{}, error) {
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

	// Clear primary from the resource state
	rs.Primary = nil

	return nil, nil
}

// EvalDeposeState is an EvalNode implementation that takes the primary
// out of a state and makes it Deposed. This is done at the beginning of
// create-before-destroy calls so that the create can create while preserving
// the old state of the to-be-destroyed resource.
type EvalDeposeState struct {
	Name string
}

// TODO: test
func (n *EvalDeposeState) Eval(ctx EvalContext) (interface{}, error) {
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

	// Depose
	rs.Deposed = append(rs.Deposed, rs.Primary)
	rs.Primary = nil

	return nil, nil
}

// EvalUndeposeState is an EvalNode implementation that reads the
// InstanceState for a specific resource out of the state.
type EvalUndeposeState struct {
	Name  string
	State **InstanceState
}

// TODO: test
func (n *EvalUndeposeState) Eval(ctx EvalContext) (interface{}, error) {
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

	// If we don't have any desposed resource, then we don't have anything to do
	if len(rs.Deposed) == 0 {
		return nil, nil
	}

	// Undepose
	idx := len(rs.Deposed) - 1
	rs.Primary = rs.Deposed[idx]
	rs.Deposed[idx] = *n.State

	return nil, nil
}
