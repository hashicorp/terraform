// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package marks

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// valueMarks allow creating strictly typed values for use as cty.Value marks.
// Each distinct mark value must be a constant in this package whose value
// is a valueMark whose underlying string matches the name of the variable.
type valueMark string

func (m valueMark) GoString() string {
	return "marks." + string(m)
}

// Has returns true if and only if the cty.Value has the given mark.
func Has(val cty.Value, mark interface{}) bool {
	switch m := mark.(type) {
	case valueMark:
		return val.HasMark(m)

	// For value marks Has returns true if a mark of the type is present
	case DeprecationMark:
		for depMark := range val.Marks() {
			if _, ok := depMark.(DeprecationMark); ok {
				return true
			}
		}
		return false
	default:
		panic("Unknown mark type")
	}
}

// Contains returns true if the cty.Value or any any value within it contains
// the given mark.
func Contains(val cty.Value, mark interface{}) bool {
	ret := false
	cty.Walk(val, func(_ cty.Path, v cty.Value) (bool, error) {
		if Has(v, mark) {
			ret = true
			return false, nil
		}
		return true, nil
	})
	return ret
}

func FilterDeprecationMarks(marks cty.ValueMarks) []DeprecationMark {
	depMarks := []DeprecationMark{}
	for mark := range marks {
		if d, ok := mark.(DeprecationMark); ok {
			depMarks = append(depMarks, d)
		}
	}
	return depMarks
}

func GetDeprecationMarks(val cty.Value) []DeprecationMark {
	_, marks := val.UnmarkDeep()
	return FilterDeprecationMarks(marks)

}

func RemoveDeprecationMarks(val cty.Value) cty.Value {
	newVal, marks := val.Unmark()
	for mark := range marks {
		if _, ok := mark.(DeprecationMark); ok {
			continue
		}
		newVal = newVal.Mark(mark)
	}
	return newVal
}

// Sensitive indicates that this value is marked as sensitive in the context of
// Terraform.
const Sensitive = valueMark("Sensitive")

// Ephemeral indicates that a value exists only in memory during a single
// phase, and thus cannot persist between phases or between rounds.
//
// Ephemeral values can be used only in locations that don't require Terraform
// to persist them as part of artifacts such as state snapshots or saved plan
// files.
const Ephemeral = valueMark("Ephemeral")

// TypeType is used to indicate that the value contains a representation of
// another value's type. This is part of the implementation of the console-only
// `type` function.
const TypeType = valueMark("TypeType")

type DeprecationMark struct {
	Message string
	Origin  *hcl.Range
}

func (d DeprecationMark) GoString() string {
	return "marks.deprecation<" + d.Message + ">"
}

// Empty deprecation mark for usage in marks.Has / Contains / etc
var Deprecation = NewDeprecation("", nil)

func NewDeprecation(message string, origin *hcl.Range) DeprecationMark {
	return DeprecationMark{
		Message: message,
		Origin:  origin,
	}
}
