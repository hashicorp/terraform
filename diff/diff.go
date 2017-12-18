package diff

import (
	"github.com/zclconf/go-cty/cty"
)

// Diff represents a sequence of changes to transform one Value into another
// Value.
//
// This type has various helper functions for appending a change to create
// a new diff. These are equivalent to using the Go built-in append function,
// and so may directly modify the array that backs the diff. They should
// generally be used only be the caller that is constructing a diff, and not
// used at all by callers that are merely consuming diffs.
type Diff []Change

// Apply attempts to apply the recieving diff to the given value, producing
// a new value.
//
// If any of the contextual information in the diff does not match, a
// ConflictError is returned describing the first such inconsistency.
func (d Diff) Apply(val cty.Value) (cty.Value, error) {
	for _, change := range d {
		var err error
		val, err = change.Apply(val)
		if err != nil {
			return cty.NilVal, err
		}
	}
	return val, nil
}

// Replace appends a change that replaces an existing value with an entirely
// new value.
//
// When adding a new element to a map value, this change type should be used
// with "old" set to a null value of the appropriate type.
func (d Diff) Replace(path cty.Path, old, new cty.Value) Diff {
	return d.append(ReplaceChange{
		Path:     path,
		OldValue: old,
		NewValue: new,
	})
}

// Delete appends a change that removes an element from an indexable collection.
//
// For a list type, if the deleted element is not the final element in
// the list then the resulting "gap" is closed by renumbering all subsequent
// items. Therefore a Diff containing a sequence of DeleteChange operations
// on the same list must be careful to consider the new state of the element
// indices after each step, or present the deletions in reverse order to
// avoid such complexity.
//
// Delete is not used for removing items from sets. For sets, use Remove.
func (d Diff) Delete(path cty.Path, old cty.Value) Diff {
	return d.append(DeleteChange{
		Path:     path,
		OldValue: old,
	})
}

// Insert appends a change that inserts an element into a list.
//
// When appending to a list, the Path should be to the not-yet-existing index
// and BeforeValue should be a null of the appropriate type.
func (d Diff) Insert(path cty.Path, new, before cty.Value) Diff {
	return d.append(InsertChange{
		Path:        path,
		NewValue:    new,
		BeforeValue: before,
	})
}

// Add appends a change that inserts an element into a set.
//
// The given path is the set itself, and new is the value to insert.
func (d Diff) Add(path cty.Path, new cty.Value) Diff {
	return d.append(AddChange{
		Path:     path,
		NewValue: new,
	})
}

// Remove appends a change that removes an element from a set.
//
// The given path is the set itself, and old is the value to remove.
func (d Diff) Remove(path cty.Path, old cty.Value) Diff {
	return d.append(RemoveChange{
		Path:     path,
		OldValue: old,
	})
}

// Nested appends a change that applies the given diff to a sub-path.
//
// This is similar to Replace but it allows the new value to be produced
// by gradually transforming the old value using the given diff. This is
// particularly useful for making changes to objects nested inside sets.
func (d Diff) Nested(path cty.Path, diff Diff) Diff {
	return d.append(NestedDiff{
		Path: path,
		Diff: diff,
	})
}

// Context appends a context check.
//
// Context is a funny sort of "change" that doesn't actually change anything
// but that fails if the given value is not present at the given path.
func (d Diff) Context(path cty.Path, want cty.Value) Diff {
	return d.append(Context{
		Path:      path,
		WantValue: want,
	})
}

func (d Diff) append(change Change) Diff {
	return append(d, change)
}
