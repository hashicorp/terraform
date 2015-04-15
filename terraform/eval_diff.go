package terraform

import (
	"fmt"
	"log"
)

// EvalCompareDiff is an EvalNode implementation that compares two diffs
// and errors if the diffs are not equal.
type EvalCompareDiff struct {
	Info     *InstanceInfo
	One, Two **InstanceDiff
}

// TODO: test
func (n *EvalCompareDiff) Eval(ctx EvalContext) (interface{}, error) {
	one, two := *n.One, *n.Two

	// If either are nil, let them be empty
	if one == nil {
		one = new(InstanceDiff)
		one.init()
	}
	if two == nil {
		two = new(InstanceDiff)
		two.init()
	}
	oneId := one.Attributes["id"]
	twoId := two.Attributes["id"]
	delete(one.Attributes, "id")
	delete(two.Attributes, "id")
	defer func() {
		if oneId != nil {
			one.Attributes["id"] = oneId
		}
		if twoId != nil {
			two.Attributes["id"] = twoId
		}
	}()

	if same, reason := one.Same(two); !same {
		log.Printf("[ERROR] %s: diffs didn't match", n.Info.Id)
		log.Printf("[ERROR] %s: reason: %s", n.Info.Id, reason)
		log.Printf("[ERROR] %s: diff one: %#v", n.Info.Id, one)
		log.Printf("[ERROR] %s: diff two: %#v", n.Info.Id, two)
		return nil, fmt.Errorf(
			"%s: diffs didn't match during apply. This is a bug with "+
				"Terraform and should be reported.", n.Info.Id)
	}

	return nil, nil
}

// EvalDiff is an EvalNode implementation that does a refresh for
// a resource.
type EvalDiff struct {
	Info        *InstanceInfo
	Config      **ResourceConfig
	Provider    *ResourceProvider
	State       **InstanceState
	Output      **InstanceDiff
	OutputState **InstanceState
}

// TODO: test
func (n *EvalDiff) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State
	config := *n.Config
	provider := *n.Provider

	// Call pre-diff hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreDiff(n.Info, state)
	})
	if err != nil {
		return nil, err
	}

	// The state for the diff must never be nil
	diffState := state
	if diffState == nil {
		diffState = new(InstanceState)
	}
	diffState.init()

	// Diff!
	diff, err := provider.Diff(n.Info, diffState, config)
	if err != nil {
		return nil, err
	}
	if diff == nil {
		diff = new(InstanceDiff)
	}

	// Require a destroy if there is no ID and it requires new.
	if diff.RequiresNew() && state != nil && state.ID != "" {
		diff.Destroy = true
	}

	// If we're creating a new resource, compute its ID
	if diff.RequiresNew() || state == nil || state.ID == "" {
		var oldID string
		if state != nil {
			oldID = state.Attributes["id"]
		}

		// Add diff to compute new ID
		diff.init()
		diff.Attributes["id"] = &ResourceAttrDiff{
			Old:         oldID,
			NewComputed: true,
			RequiresNew: true,
			Type:        DiffAttrOutput,
		}
	}

	// Call post-refresh hook
	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(n.Info, diff)
	})
	if err != nil {
		return nil, err
	}

	// Update our output
	*n.Output = diff

	// Update the state if we care
	if n.OutputState != nil {
		*n.OutputState = state

		// Merge our state so that the state is updated with our plan
		if !diff.Empty() && n.OutputState != nil {
			*n.OutputState = state.MergeDiff(diff)
		}
	}

	return nil, nil
}

// EvalDiffDestroy is an EvalNode implementation that returns a plain
// destroy diff.
type EvalDiffDestroy struct {
	Info   *InstanceInfo
	State  **InstanceState
	Output **InstanceDiff
}

// TODO: test
func (n *EvalDiffDestroy) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State

	// If there is no state or we don't have an ID, we're already destroyed
	if state == nil || state.ID == "" {
		return nil, nil
	}

	// Call pre-diff hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreDiff(n.Info, state)
	})
	if err != nil {
		return nil, err
	}

	// The diff
	diff := &InstanceDiff{Destroy: true}

	// Call post-diff hook
	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(n.Info, diff)
	})
	if err != nil {
		return nil, err
	}

	// Update our output
	*n.Output = diff

	return nil, nil
}

// EvalDiffDestroyModule is an EvalNode implementation that writes the diff to
// the full diff.
type EvalDiffDestroyModule struct {
	Path []string
}

// TODO: test
func (n *EvalDiffDestroyModule) Eval(ctx EvalContext) (interface{}, error) {
	diff, lock := ctx.Diff()

	// Acquire the lock so that we can do this safely concurrently
	lock.Lock()
	defer lock.Unlock()

	// Write the diff
	modDiff := diff.ModuleByPath(n.Path)
	if modDiff == nil {
		modDiff = diff.AddModule(n.Path)
	}
	modDiff.Destroy = true

	return nil, nil
}

// EvalDiffTainted is an EvalNode implementation that writes the diff to
// the full diff.
type EvalDiffTainted struct {
	Name string
	Diff **InstanceDiff
}

// TODO: test
func (n *EvalDiffTainted) Eval(ctx EvalContext) (interface{}, error) {
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

	// If we have tainted, then mark it on the diff
	if len(rs.Tainted) > 0 {
		(*n.Diff).DestroyTainted = true
	}

	return nil, nil
}

// EvalFilterDiff is an EvalNode implementation that filters the diff
// according to some filter.
type EvalFilterDiff struct {
	// Input and output
	Diff   **InstanceDiff
	Output **InstanceDiff

	// Destroy, if true, will only include a destroy diff if it is set.
	Destroy bool
}

func (n *EvalFilterDiff) Eval(ctx EvalContext) (interface{}, error) {
	if *n.Diff == nil {
		return nil, nil
	}

	input := *n.Diff
	result := new(InstanceDiff)

	if n.Destroy {
		if input.Destroy || input.RequiresNew() {
			result.Destroy = true
		}
	}

	if n.Output != nil {
		*n.Output = result
	}

	return nil, nil
}

// EvalReadDiff is an EvalNode implementation that writes the diff to
// the full diff.
type EvalReadDiff struct {
	Name string
	Diff **InstanceDiff
}

func (n *EvalReadDiff) Eval(ctx EvalContext) (interface{}, error) {
	diff, lock := ctx.Diff()

	// Acquire the lock so that we can do this safely concurrently
	lock.Lock()
	defer lock.Unlock()

	// Write the diff
	modDiff := diff.ModuleByPath(ctx.Path())
	if modDiff == nil {
		return nil, nil
	}

	*n.Diff = modDiff.Resources[n.Name]

	return nil, nil
}

// EvalWriteDiff is an EvalNode implementation that writes the diff to
// the full diff.
type EvalWriteDiff struct {
	Name string
	Diff **InstanceDiff
}

// TODO: test
func (n *EvalWriteDiff) Eval(ctx EvalContext) (interface{}, error) {
	diff, lock := ctx.Diff()

	// The diff to write, if its empty it should write nil
	var diffVal *InstanceDiff
	if n.Diff != nil {
		diffVal = *n.Diff
	}
	if diffVal.Empty() {
		diffVal = nil
	}

	// Acquire the lock so that we can do this safely concurrently
	lock.Lock()
	defer lock.Unlock()

	// Write the diff
	modDiff := diff.ModuleByPath(ctx.Path())
	if modDiff == nil {
		modDiff = diff.AddModule(ctx.Path())
	}
	if diffVal != nil {
		modDiff.Resources[n.Name] = diffVal
	} else {
		delete(modDiff.Resources, n.Name)
	}

	return nil, nil
}
