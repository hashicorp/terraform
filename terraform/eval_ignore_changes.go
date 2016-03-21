package terraform

import (
	"log"
	"strings"

	"github.com/hashicorp/terraform/config"
)

// EvalIgnoreChanges is an EvalNode implementation that removes diff
// attributes if their name matches names provided by the resource's
// IgnoreChanges lifecycle.
type EvalIgnoreChanges struct {
	Resource      *config.Resource
	Diff          **InstanceDiff
	WasChangeType *DiffChangeType
}

func (n *EvalIgnoreChanges) Eval(ctx EvalContext) (interface{}, error) {
	if n.Diff == nil || *n.Diff == nil || n.Resource == nil || n.Resource.Id() == "" {
		return nil, nil
	}

	diff := *n.Diff
	ignoreChanges := n.Resource.Lifecycle.IgnoreChanges

	if len(ignoreChanges) == 0 {
		return nil, nil
	}

	changeType := diff.ChangeType()
	// Let the passed in change type pointer override what the diff currently has.
	if n.WasChangeType != nil && *n.WasChangeType != DiffInvalid {
		changeType = *n.WasChangeType
	}

	// If we're just creating the resource, we shouldn't alter the
	// Diff at all
	if changeType == DiffCreate {
		return nil, nil
	}

	for _, ignoredName := range ignoreChanges {
		for name := range diff.Attributes {
			if strings.HasPrefix(name, ignoredName) {
				delete(diff.Attributes, name)
			}
		}
	}

	// If we are replacing the resource, then we expect there to be a bunch of
	// extraneous attribute diffs we need to filter out for the other
	// non-requires-new attributes going from "" -> "configval" or "" ->
	// "<computed>". Filtering these out allows us to see if we might be able to
	// skip this diff altogether.
	if changeType == DiffDestroyCreate {
		for k, v := range diff.Attributes {
			if v.Empty() || v.NewComputed {
				delete(diff.Attributes, k)
			}
		}

		// Here we emulate the implementation of diff.RequiresNew() with one small
		// tweak, we ignore the "id" attribute diff that gets added by EvalDiff,
		// since that was added in reaction to RequiresNew being true.
		requiresNewAfterIgnores := false
		for k, v := range diff.Attributes {
			if k == "id" {
				continue
			}
			if v.RequiresNew == true {
				requiresNewAfterIgnores = true
			}
		}

		// Here we undo the two reactions to RequireNew in EvalDiff - the "id"
		// attribute diff and the Destroy boolean field
		if !requiresNewAfterIgnores {
			log.Printf("[DEBUG] Removing 'id' diff and setting Destroy to false " +
				"because after ignore_changes, this diff no longer requires replacement")
			delete(diff.Attributes, "id")
			diff.Destroy = false
		}
	}

	return nil, nil
}
