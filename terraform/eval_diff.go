package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/config"
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
	oneId, _ := one.GetAttribute("id")
	twoId, _ := two.GetAttribute("id")
	one.DelAttribute("id")
	two.DelAttribute("id")
	defer func() {
		if oneId != nil {
			one.SetAttribute("id", oneId)
		}
		if twoId != nil {
			two.SetAttribute("id", twoId)
		}
	}()

	if same, reason := one.Same(two); !same {
		log.Printf("[ERROR] %s: diffs didn't match", n.Info.Id)
		log.Printf("[ERROR] %s: reason: %s", n.Info.Id, reason)
		log.Printf("[ERROR] %s: diff one: %#v", n.Info.Id, one)
		log.Printf("[ERROR] %s: diff two: %#v", n.Info.Id, two)
		return nil, fmt.Errorf(
			"%s: diffs didn't match during apply. This is a bug with "+
				"Terraform and should be reported as a GitHub Issue.\n"+
				"\n"+
				"Please include the following information in your report:\n"+
				"\n"+
				"    Terraform Version: %s\n"+
				"    Resource ID: %s\n"+
				"    Mismatch reason: %s\n"+
				"    Diff One (usually from plan): %#v\n"+
				"    Diff Two (usually from apply): %#v\n"+
				"\n"+
				"Also include as much context as you can about your config, state, "+
				"and the steps you performed to trigger this error.\n",
			n.Info.Id, Version, n.Info.Id, reason, one, two)
	}

	return nil, nil
}

// EvalDiff is an EvalNode implementation that does a refresh for
// a resource.
type EvalDiff struct {
	Info        *InstanceInfo
	Config      **ResourceConfig
	Provider    *ResourceProvider
	Diff        **InstanceDiff
	State       **InstanceState
	OutputDiff  **InstanceDiff
	OutputState **InstanceState

	// Resource is needed to fetch the ignore_changes list so we can
	// filter user-requested ignored attributes from the diff.
	Resource *config.Resource
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

	// Preserve the DestroyTainted flag
	if n.Diff != nil {
		diff.SetTainted((*n.Diff).GetDestroyTainted())
	}

	// Require a destroy if there is an ID and it requires new.
	if diff.RequiresNew() && state != nil && state.ID != "" {
		diff.SetDestroy(true)
	}

	// If we're creating a new resource, compute its ID
	if diff.RequiresNew() || state == nil || state.ID == "" {
		var oldID string
		if state != nil {
			oldID = state.Attributes["id"]
		}

		// Add diff to compute new ID
		diff.init()
		diff.SetAttribute("id", &ResourceAttrDiff{
			Old:         oldID,
			NewComputed: true,
			RequiresNew: true,
			Type:        DiffAttrOutput,
		})
	}

	if err := n.processIgnoreChanges(diff); err != nil {
		return nil, err
	}

	// Call post-refresh hook
	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(n.Info, diff)
	})
	if err != nil {
		return nil, err
	}

	// Update our output
	*n.OutputDiff = diff

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

func (n *EvalDiff) processIgnoreChanges(diff *InstanceDiff) error {
	if diff == nil || n.Resource == nil || n.Resource.Id() == "" {
		return nil
	}
	ignoreChanges := n.Resource.Lifecycle.IgnoreChanges

	if len(ignoreChanges) == 0 {
		return nil
	}

	changeType := diff.ChangeType()

	// If we're just creating the resource, we shouldn't alter the
	// Diff at all
	if changeType == DiffCreate {
		return nil
	}

	ignorableAttrKeys := make(map[string]bool)
	for _, ignoredKey := range ignoreChanges {
		for k := range diff.CopyAttributes() {
			if ignoredKey == "*" || strings.HasPrefix(k, ignoredKey) {
				ignorableAttrKeys[k] = true
			}
		}
	}

	// If we are replacing the resource, then we expect there to be a bunch of
	// extraneous attribute diffs we need to filter out for the other
	// non-requires-new attributes going from "" -> "configval" or "" ->
	// "<computed>". Filtering these out allows us to see if we might be able to
	// skip this diff altogether.
	if changeType == DiffDestroyCreate {
		for k, v := range diff.CopyAttributes() {
			if v.Empty() || v.NewComputed {
				ignorableAttrKeys[k] = true
			}
		}

		// Here we emulate the implementation of diff.RequiresNew() with one small
		// tweak, we ignore the "id" attribute diff that gets added by EvalDiff,
		// since that was added in reaction to RequiresNew being true.
		requiresNewAfterIgnores := false
		for k, v := range diff.CopyAttributes() {
			if k == "id" {
				continue
			}
			if _, ok := ignorableAttrKeys[k]; ok {
				continue
			}
			if v.RequiresNew == true {
				requiresNewAfterIgnores = true
			}
		}

		// If we still require resource replacement after ignores, we
		// can't touch the diff, as all of the attributes will be
		// required to process the replacement.
		if requiresNewAfterIgnores {
			return nil
		}

		// Here we undo the two reactions to RequireNew in EvalDiff - the "id"
		// attribute diff and the Destroy boolean field
		log.Printf("[DEBUG] Removing 'id' diff and setting Destroy to false " +
			"because after ignore_changes, this diff no longer requires replacement")
		diff.DelAttribute("id")
		diff.SetDestroy(false)
	}

	// If we didn't hit any of our early exit conditions, we can filter the diff.
	for k := range ignorableAttrKeys {
		log.Printf("[DEBUG] [EvalIgnoreChanges] %s - Ignoring diff attribute: %s",
			n.Resource.Id(), k)
		diff.DelAttribute(k)
	}

	return nil
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
		if input.GetDestroy() || input.RequiresNew() {
			result.SetDestroy(true)
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
