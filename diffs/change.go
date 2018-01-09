package diffs

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// Change represents a change to a complex object, such as a resource.
//
// Rather than instantiating this struct directly, prefer instead to use the
// NewCreate, NewRead, NewUpdate, NewDelete, and NewReplace methods, which
// ensure that expected invariants are met.
type Change struct {
	// Action identifies the kind of action represented by this change.
	Action Action

	// Old and New are the expected values before and after the action
	// respectively. The usage of these varies by Action.
	//
	// For Create, New is the value that will be created and Old is a null
	// value of the same type.
	//
	// For Read, New describes the object that will be read. A New value
	// for Read will generally consist primarily of unknown values, thus
	// indicating the expected structure of the result even though the values
	// are not yet known. Old may either be null, indicating that the value
	// is being read for the first time, or may be a prior value that the
	// read value will replace.
	//
	// For Update and Replace, Old describes the expected existing value to
	// update and New is its replacement. For Replace, ForceNew is also
	// populated.
	//
	// For Delete, Old is the expected value to destroy and New must be a null
	// value of the change type.
	//
	// In no case may the Old value be unknown or contain unknown values. The
	// New value may contain unknowns, however.
	Old, New cty.Value

	// ForcedReplace is populated for changes with action Replace to indicate
	// which paths within the value prompted the change to be a Replace rather
	// than an Update.
	//
	// The set may be populated with paths from the old or new value, or both,
	// depending on the nature of the change being described. A diff renderer
	// should annotate its rendering of a particular path with an
	// indication that it prompted replacement if that path is present in
	// this set.
	//
	// ForceNew is nil for all other actions and will panic if accessed.
	ForcedReplace PathSet
}

// NewNoAction returns a NoAction change for the given value.
//
// This should be used only to add context elements to a diff for a sequence
// of objects where some objects have not changed.
func NewNoAction(v cty.Value) *Change {
	return &Change{
		Action: NoAction,
		Old:    v,
		New:    v,
	}
}

// NewCreate returns a Create change for the given value.
func NewCreate(v cty.Value) *Change {
	return &Change{
		Action: Create,
		Old:    cty.NullVal(v.Type()),
		New:    v,
	}
}

// NewRead returns a Read change for the given value.
//
// If a value is being read for the first time, pass prev as a null value of
// the same type as next, or use NewFirstRead to achieve the same result.
func NewRead(prev, next cty.Value) *Change {
	return &Change{
		Action: Read,
		Old:    prev,
		New:    next,
	}
}

// NewFirstRead is a convenience wrapper around NewRead that sets the previous
// value to null, indicating that a value is being read for the first time and
// so there is no prior value it is replacing.
func NewFirstRead(next cty.Value) *Change {
	return NewRead(cty.NullVal(next.Type()), next)
}

// NewUpdate returns an Update change for the given value.
func NewUpdate(old, new cty.Value) *Change {
	return &Change{
		Action: Update,
		Old:    old,
		New:    new,
	}
}

// NewReplace returns a Replace change for the given value.
func NewReplace(old, new cty.Value, forcedReplace PathSet) *Change {
	return &Change{
		Action:        Replace,
		Old:           old,
		New:           new,
		ForcedReplace: forcedReplace,
	}
}

// NewDelete returns a Delete change for the given value.
func NewDelete(v cty.Value) *Change {
	return &Change{
		Action: Delete,
		Old:    v,
		New:    cty.NullVal(v.Type()),
	}
}

// CheckConsistency tests if the receiver is consistent with the given other
// change.
//
// The reciever is consistent if both "Old" and "New" values are equal and
// both have the same action and type. The reciever is also consistent if its
// "New" value has known values at paths where the given other change has
// unknown values, suggesting that the receiver is an updated version of the
// other given change once additional information became available.
//
// The "ForcedReplace" set is not considered during a consistency check, since
// it is additional contextual information that does not directly affect the
// meaning of a change.
//
// If the receiver is consistent, nil is returned. Otherwise, an error is
// returned describing the first detected inconsistency. This error may be
// a cty.PathError whose path refers to a location within the New value
// where an inconsistency was detected.
//
// This method is intended for detecting implementation errors in Terraform
// and its providers, so the errors returned are Terraform-developer-oriented
// rather than user-oriented. If any returned errors are included in the user
// interface, they should be annotated with language like "This is a bug in
// Terraform that should be reported".
func (c *Change) CheckConsistency(other *Change) error {
	if c.Action != other.Action {
		return fmt.Errorf("new change has action %q while old change had action %q", c.Action, other.Action)
	}

	{
		eq := c.Old.Equals(other.Old)
		if !eq.IsKnown() || eq.False() {
			return fmt.Errorf("new change has old value %#v while old change had old value %#v", c.Old, other.Old)
		}
	}

	// Checking the "New" values is more complex since we need to dig into
	// collection and structural types to permit changes to sub-paths that
	// had unknown values in the other given change.
	var path cty.Path
	return checkValueConsistency(c.New, other.New, path)
}

func checkValueConsistency(old, new cty.Value, path cty.Path) error {
	// It may make sense to move this "consistency" idea out into cty itself,
	// since it could be generally useful for other callers. For now though
	// we implement it inline here.

	if !old.IsKnown() {
		// If the old value was unknown then any new value is considered
		// consistent, including another unknown.
		return nil
	}

	// Most of our error outcomes return messages of this form
	const wrongValueFmt = "new change has value %#v where old change had value %#v"

	if old.IsKnown() && !new.IsKnown() {
		return path.NewErrorf("new change has unknown value where old change had known value %#v", old)
	}

	if old.IsNull() && new.IsNull() {
		// If both values are null then we allow it even if the types don't
		// match, since we use untyped nulls pretty liberally in HCL.
		return nil
	}

	if !old.Type().Equals(new.Type()) {
		return path.NewErrorf(wrongValueFmt, new, old)
	}

	ty := new.Type()

	switch {
	case !new.IsKnown():
		return nil

		// We can assume that both old and new are known after this point

	case ty.IsCollectionType() || ty.IsTupleType():
		if old.LengthInt() != new.LengthInt() {
			return path.NewErrorf(wrongValueFmt, new, old)
		}

		it := old.ElementIterator()
		for it.Next() {
			key, oldValue := it.Element()
			if !new.HasIndex(key).True() {
				return path.NewErrorf("new change lacks element key %#v which is present in old change")
			}
			newValue := new.Index(key)
			path := append(path, cty.IndexStep{
				Key: key,
			})
			err := checkValueConsistency(oldValue, newValue, path)
			if err != nil {
				return err
			}
		}

	case ty.IsObjectType():
		for attrName := range ty.AttributeTypes() {
			path := append(path, cty.GetAttrStep{
				Name: attrName,
			})
			oldValue := old.GetAttr(attrName)
			newValue := new.GetAttr(attrName)
			err := checkValueConsistency(oldValue, newValue, path)
			if err != nil {
				return err
			}
		}

	default:
		if !new.RawEquals(old) {
			return path.NewErrorf(wrongValueFmt, new, old)
		}
	}

	return nil
}
