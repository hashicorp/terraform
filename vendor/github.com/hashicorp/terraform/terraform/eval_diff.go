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
	Name        string
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

	// Set DestroyDeposed if we have deposed instances
	_, err = readInstanceFromState(ctx, n.Name, nil, func(rs *ResourceState) (*InstanceState, error) {
		if len(rs.Deposed) > 0 {
			diff.DestroyDeposed = true
		}

		return nil, nil
	})
	if err != nil {
		return nil, err
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

	// filter out ignored resources
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

	// If we're just creating the resource, we shouldn't alter the
	// Diff at all
	if diff.ChangeType() == DiffCreate {
		return nil
	}

	// If the resource has been tainted then we don't process ignore changes
	// since we MUST recreate the entire resource.
	if diff.GetDestroyTainted() {
		return nil
	}

	attrs := diff.CopyAttributes()

	// get the complete set of keys we want to ignore
	ignorableAttrKeys := make(map[string]bool)
	for _, ignoredKey := range ignoreChanges {
		for k := range attrs {
			if ignoredKey == "*" || strings.HasPrefix(k, ignoredKey) {
				ignorableAttrKeys[k] = true
			}
		}
	}

	// If the resource was being destroyed, check to see if we can ignore the
	// reason for it being destroyed.
	if diff.GetDestroy() {
		for k, v := range attrs {
			if k == "id" {
				// id will always be changed if we intended to replace this instance
				continue
			}
			if v.Empty() || v.NewComputed {
				continue
			}

			// If any RequiresNew attribute isn't ignored, we need to keep the diff
			// as-is to be able to replace the resource.
			if v.RequiresNew && !ignorableAttrKeys[k] {
				return nil
			}
		}

		// Now that we know that we aren't replacing the instance, we can filter
		// out all the empty and computed attributes. There may be a bunch of
		// extraneous attribute diffs for the other non-requires-new attributes
		// going from "" -> "configval" or "" -> "<computed>".
		// We must make sure any flatmapped containers are filterred (or not) as a
		// whole.
		containers := groupContainers(diff)
		keep := map[string]bool{}
		for _, v := range containers {
			if v.keepDiff() {
				// At least one key has changes, so list all the sibling keys
				// to keep in the diff.
				for k := range v {
					keep[k] = true
				}
			}
		}

		for k, v := range attrs {
			if (v.Empty() || v.NewComputed) && !keep[k] {
				ignorableAttrKeys[k] = true
			}
		}
	}

	// Here we undo the two reactions to RequireNew in EvalDiff - the "id"
	// attribute diff and the Destroy boolean field
	log.Printf("[DEBUG] Removing 'id' diff and setting Destroy to false " +
		"because after ignore_changes, this diff no longer requires replacement")
	diff.DelAttribute("id")
	diff.SetDestroy(false)

	// If we didn't hit any of our early exit conditions, we can filter the diff.
	for k := range ignorableAttrKeys {
		log.Printf("[DEBUG] [EvalIgnoreChanges] %s - Ignoring diff attribute: %s",
			n.Resource.Id(), k)
		diff.DelAttribute(k)
	}

	return nil
}

// a group of key-*ResourceAttrDiff pairs from the same flatmapped container
type flatAttrDiff map[string]*ResourceAttrDiff

// we need to keep all keys if any of them have a diff
func (f flatAttrDiff) keepDiff() bool {
	for _, v := range f {
		if !v.Empty() && !v.NewComputed {
			return true
		}
	}
	return false
}

// sets, lists and maps need to be compared for diff inclusion as a whole, so
// group the flatmapped keys together for easier comparison.
func groupContainers(d *InstanceDiff) map[string]flatAttrDiff {
	isIndex := multiVal.MatchString
	containers := map[string]flatAttrDiff{}
	attrs := d.CopyAttributes()
	// we need to loop once to find the index key
	for k := range attrs {
		if isIndex(k) {
			// add the key, always including the final dot to fully qualify it
			containers[k[:len(k)-1]] = flatAttrDiff{}
		}
	}

	// loop again to find all the sub keys
	for prefix, values := range containers {
		for k, attrDiff := range attrs {
			// we include the index value as well, since it could be part of the diff
			if strings.HasPrefix(k, prefix) {
				values[k] = attrDiff
			}
		}
	}

	return containers
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
