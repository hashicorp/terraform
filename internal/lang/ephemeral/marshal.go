// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

// RemoveEphemeralValues takes a value that possibly contains ephemeral
// values and returns an equal value without ephemeral values. If an attribute contains
// an ephemeral value it will be set to null.
func RemoveEphemeralValues(value cty.Value) cty.Value {
	// We currently have no error case, so we can ignore the error
	val, _ := cty.Transform(value, func(p cty.Path, v cty.Value) (cty.Value, error) {
		_, givenMarks := v.Unmark()
		if _, isEphemeral := givenMarks[marks.Ephemeral]; isEphemeral {
			// We'll strip the ephemeral mark but retain any other marks
			// that might be present on the input.
			delete(givenMarks, marks.Ephemeral)
			if !v.IsKnown() {
				// If the source value is unknown then we must leave it
				// unknown because its final type might be more precise
				// than the associated type constraint and returning a
				// typed null could therefore over-promise on what the
				// final result type will be.
				// We're deliberately constructing a fresh unknown value
				// here, rather than returning the one we were given,
				// because we need to discard any refinements that the
				// unknown value might be carrying that definitely won't
				// be honored when we force the final result to be null.
				return cty.UnknownVal(v.Type()).WithMarks(givenMarks), nil
			}
			return cty.NullVal(v.Type()).WithMarks(givenMarks), nil
		}
		return v, nil
	})
	return val
}
