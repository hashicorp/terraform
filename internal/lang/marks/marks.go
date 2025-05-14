// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package marks

import (
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
func Has(val cty.Value, mark valueMark) bool {
	return val.HasMark(mark)
}

// Contains returns true if the cty.Value or any any value within it contains
// the given mark.
func Contains(val cty.Value, mark valueMark) bool {
	ret := false
	cty.Walk(val, func(_ cty.Path, v cty.Value) (bool, error) {
		if v.HasMark(mark) {
			ret = true
			return false, nil
		}
		return true, nil
	})
	return ret
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
