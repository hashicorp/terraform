package terraform

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/hashicorp/terraform/version"
)

// EvalCompareDiff is an EvalNode implementation that compares two diffs
// and errors if the diffs are not equal.
type EvalCompareDiff struct {
	Addr     addrs.ResourceInstance
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
		log.Printf("[ERROR] %s: diffs didn't match", n.Addr)
		log.Printf("[ERROR] %s: reason: %s", n.Addr, reason)
		log.Printf("[ERROR] %s: diff one: %#v", n.Addr, one)
		log.Printf("[ERROR] %s: diff two: %#v", n.Addr, two)
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
			n.Addr, version.Version, n.Addr, reason, one, two)
	}

	return nil, nil
}

// EvalDiff is an EvalNode implementation that does a refresh for
// a resource.
type EvalDiff struct {
	Addr           addrs.ResourceInstance
	Config         *configs.Resource
	Provider       *ResourceProvider
	ProviderSchema **ProviderSchema
	State          **InstanceState
	PreviousDiff   **InstanceDiff

	OutputDiff  **InstanceDiff
	OutputValue *cty.Value
	OutputState **InstanceState

	Stub bool
}

// TODO: test
func (n *EvalDiff) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State
	config := *n.Config
	provider := *n.Provider
	providerSchema := *n.ProviderSchema

	var diags tfdiags.Diagnostics

	// The provider and hook APIs still expect our legacy InstanceInfo type.
	legacyInfo := NewInstanceInfo(n.Addr.Absolute(ctx.Path()).ContainingResource())

	// State still uses legacy-style internal ids, so we need to shim to get
	// a suitable key to use.
	stateId := NewLegacyResourceInstanceAddress(n.Addr.Absolute(ctx.Path())).stateId()

	// Call pre-diff hook
	if !n.Stub {
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreDiff(legacyInfo, state)
		})
		if err != nil {
			return nil, err
		}
	}

	// The state for the diff must never be nil
	diffState := state
	if diffState == nil {
		diffState = new(InstanceState)
	}
	diffState.init()

	// Evaluate the configuration
	schema := providerSchema.ResourceTypes[n.Addr.Resource.Type]
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Type)
	}
	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags.Err()
	}

	// The provider API still expects our legacy ResourceConfig type.
	legacyRC := NewResourceConfigShimmed(configVal, schema)

	// Diff!
	diff, err := provider.Diff(legacyInfo, diffState, legacyRC)
	if err != nil {
		return nil, err
	}
	if diff == nil {
		diff = new(InstanceDiff)
	}

	// Set DestroyDeposed if we have deposed instances
	_, err = readInstanceFromState(ctx, stateId, nil, func(rs *ResourceState) (*InstanceState, error) {
		if len(rs.Deposed) > 0 {
			diff.DestroyDeposed = true
		}

		return nil, nil
	})
	if err != nil {
		return nil, err
	}

	// Preserve the DestroyTainted flag
	if n.PreviousDiff != nil {
		diff.SetTainted((*n.PreviousDiff).GetDestroyTainted())
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

	// filter out ignored attributes
	if err := n.processIgnoreChanges(diff); err != nil {
		return nil, err
	}

	// Call post-refresh hook
	if !n.Stub {
		err = ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostDiff(legacyInfo, diff)
		})
		if err != nil {
			return nil, err
		}
	}

	// Update our output if we care
	if n.OutputDiff != nil {
		*n.OutputDiff = diff
	}

	if n.OutputValue != nil {
		*n.OutputValue = configVal
	}

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
	if diff == nil || n.Config == nil || n.Config.Managed == nil {
		return nil
	}
	ignoreChanges := n.Config.Managed.IgnoreChanges
	ignoreAll := n.Config.Managed.IgnoreAllChanges

	if len(ignoreChanges) == 0 && !ignoreAll {
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
	for _, ignoredTraversal := range ignoreChanges {
		ignoredKey := legacyFlatmapKeyForTraversal(ignoredTraversal)
		for k := range attrs {
			if ignoreAll || strings.HasPrefix(k, ignoredKey) {
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
			if v.keepDiff(ignorableAttrKeys) {
				// At least one key has changes, so list all the sibling keys
				// to keep in the diff
				for k := range v {
					keep[k] = true
					// this key may have been added by the user to ignore, but
					// if it's a subkey in a container, we need to un-ignore it
					// to keep the complete containter.
					delete(ignorableAttrKeys, k)
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
		log.Printf("[DEBUG] [EvalIgnoreChanges] %s: Ignoring diff attribute: %s", n.Addr.String(), k)
		diff.DelAttribute(k)
	}

	return nil
}

// legacyFlagmapKeyForTraversal constructs a key string compatible with what
// the flatmap package would generate for an attribute addressable by the given
// traversal.
//
// This is used only to shim references to attributes within the diff and
// state structures, which have not (at the time of writing) yet been updated
// to use the newer HCL-based representations.
func legacyFlatmapKeyForTraversal(traversal hcl.Traversal) string {
	var buf bytes.Buffer
	first := true
	for _, step := range traversal {
		if !first {
			buf.WriteByte('.')
		}
		switch ts := step.(type) {
		case hcl.TraverseRoot:
			buf.WriteString(ts.Name)
		case hcl.TraverseAttr:
			buf.WriteString(ts.Name)
		case hcl.TraverseIndex:
			val := ts.Key
			switch val.Type() {
			case cty.Number:
				bf := val.AsBigFloat()
				buf.WriteString(bf.String())
			case cty.String:
				s := val.AsString()
				buf.WriteString(s)
			default:
				// should never happen, since no other types appear in
				// traversals in practice.
				buf.WriteByte('?')
			}
		default:
			// should never happen, since we've covered all of the types
			// that show up in parsed traversals in practice.
			buf.WriteByte('?')
		}
		first = false
	}
	return buf.String()
}

// a group of key-*ResourceAttrDiff pairs from the same flatmapped container
type flatAttrDiff map[string]*ResourceAttrDiff

// we need to keep all keys if any of them have a diff that's not ignored
func (f flatAttrDiff) keepDiff(ignoreChanges map[string]bool) bool {
	for k, v := range f {
		ignore := false
		for attr := range ignoreChanges {
			if strings.HasPrefix(k, attr) {
				ignore = true
			}
		}

		if !v.Empty() && !v.NewComputed && !ignore {
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
	Addr   addrs.ResourceInstance
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

	// The provider and hook APIs still expect our legacy InstanceInfo type.
	legacyInfo := NewInstanceInfo(n.Addr.Absolute(ctx.Path()).ContainingResource())

	// Call pre-diff hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreDiff(legacyInfo, state)
	})
	if err != nil {
		return nil, err
	}

	// The diff
	diff := &InstanceDiff{Destroy: true}

	// Call post-diff hook
	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(legacyInfo, diff)
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
	Path addrs.ModuleInstance
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
