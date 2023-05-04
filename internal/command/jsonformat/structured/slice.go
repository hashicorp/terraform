// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package structured

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured/attribute_path"
)

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
	ReplacePaths attribute_path.Matcher

	// RelevantAttributes matches the same attributes in Change exactly.
	RelevantAttributes attribute_path.Matcher
}

// AsSlice converts the Change into a slice representation by converting the
// internal Before, After, Unknown, BeforeSensitive, and AfterSensitive data
// structures into generic slices.
func (change Change) AsSlice() ChangeSlice {
	return ChangeSlice{
		Before:             genericToSlice(change.Before),
		After:              genericToSlice(change.After),
		Unknown:            genericToSlice(change.Unknown),
		BeforeSensitive:    genericToSlice(change.BeforeSensitive),
		AfterSensitive:     genericToSlice(change.AfterSensitive),
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}
}

// GetChild safely packages up a Change object for the given child, handling
// all the cases where the data might be null or a static boolean.
func (s ChangeSlice) GetChild(beforeIx, afterIx int) Change {
	before, beforeExplicit := getFromGenericSlice(s.Before, beforeIx)
	after, afterExplicit := getFromGenericSlice(s.After, afterIx)
	unknown, _ := getFromGenericSlice(s.Unknown, afterIx)
	beforeSensitive, _ := getFromGenericSlice(s.BeforeSensitive, beforeIx)
	afterSensitive, _ := getFromGenericSlice(s.AfterSensitive, afterIx)

	mostRelevantIx := beforeIx
	if beforeIx < 0 || beforeIx >= len(s.Before) {
		mostRelevantIx = afterIx
	}

	return Change{
		BeforeExplicit:     beforeExplicit,
		AfterExplicit:      afterExplicit,
		Before:             before,
		After:              after,
		Unknown:            unknown,
		BeforeSensitive:    beforeSensitive,
		AfterSensitive:     afterSensitive,
		ReplacePaths:       s.ReplacePaths.GetChildWithIndex(mostRelevantIx),
		RelevantAttributes: s.RelevantAttributes.GetChildWithIndex(mostRelevantIx),
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
