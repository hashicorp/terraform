package differ

import "github.com/hashicorp/terraform/internal/command/jsonformat/differ/replace"

// ChangeSlice is a Change that represents a Tuple, Set, or List type, and has
// converted the relevant interfaces into slices for easier access.
type ChangeSlice struct {
	// Before contains the value before the proposed change.
	Before []interface{}

	// After contains the value after the proposed change.
	After []interface{}

	// Unknown contains the unknown status of any elements of this list/set.
	Unknown []interface{}

	// BeforeSensitive contains the before sensitive status of any elements of
	//this list/set.
	BeforeSensitive []interface{}

	// AfterSensitive contains the after sensitive status of any elements of
	//this list/set.
	AfterSensitive []interface{}

	// ReplacePaths matches the same attributes in Change exactly.
	ReplacePaths replace.ForcesReplacement
}

func (change Change) asSlice() ChangeSlice {
	return ChangeSlice{
		Before:          genericToSlice(change.Before),
		After:           genericToSlice(change.After),
		Unknown:         genericToSlice(change.Unknown),
		BeforeSensitive: genericToSlice(change.BeforeSensitive),
		AfterSensitive:  genericToSlice(change.AfterSensitive),
		ReplacePaths:    change.ReplacePaths,
	}
}

func (s ChangeSlice) getChild(beforeIx, afterIx int) Change {
	before, beforeExplicit := getFromGenericSlice(s.Before, beforeIx)
	after, afterExplicit := getFromGenericSlice(s.After, afterIx)
	unknown, _ := getFromGenericSlice(s.Unknown, afterIx)
	beforeSensitive, _ := getFromGenericSlice(s.BeforeSensitive, beforeIx)
	afterSensitive, _ := getFromGenericSlice(s.AfterSensitive, afterIx)

	return Change{
		BeforeExplicit:  beforeExplicit,
		AfterExplicit:   afterExplicit,
		Before:          before,
		After:           after,
		Unknown:         unknown,
		BeforeSensitive: beforeSensitive,
		AfterSensitive:  afterSensitive,
		ReplacePaths:    s.ReplacePaths.GetChildWithIndex(beforeIx),
	}
}

func getFromGenericSlice(generic []interface{}, ix int) (interface{}, bool) {
	if generic == nil {
		return nil, false
	}
	if ix < 0 || ix >= len(generic) {
		return nil, false
	}
	return generic[ix], true
}

func genericToSlice(generic interface{}) []interface{} {
	if concrete, ok := generic.([]interface{}); ok {
		return concrete
	}
	return nil
}
