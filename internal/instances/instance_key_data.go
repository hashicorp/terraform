// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package instances

import (
	"github.com/zclconf/go-cty/cty"
)

// RepetitionData represents the values available to identify individual
// repetitions of a particular object.
//
// This corresponds to the each.key, each.value, and count.index symbols in
// the configuration language.
type RepetitionData struct {
	// CountIndex is the value for count.index, or cty.NilVal if evaluating
	// in a context where the "count" argument is not active.
	//
	// For correct operation, this should always be of type cty.Number if not
	// nil.
	CountIndex cty.Value

	// EachKey and EachValue are the values for each.key and each.value
	// respectively, or cty.NilVal if evaluating in a context where the
	// "for_each" argument is not active. These must either both be set
	// or neither set.
	//
	// For correct operation, EachKey must always be either of type cty.String
	// or cty.Number if not nil.
	EachKey, EachValue cty.Value
}

// TotallyUnknownRepetitionData is a [RepetitionData] value for situations
// where don't even know yet what type of repetition will be used.
var TotallyUnknownRepetitionData = RepetitionData{
	CountIndex: cty.UnknownVal(cty.Number),
	EachKey:    cty.UnknownVal(cty.String),
	EachValue:  cty.DynamicVal,
}

// UnknownCountRepetitionData is a suitable [RepetitionData] value to use when
// evaluating the configuration of an object which has a count argument that
// is currently unknown.
var UnknownCountRepetitionData = RepetitionData{
	CountIndex: cty.UnknownVal(cty.Number),
}

// UnknownForEachRepetitionData generates a suitable [RepetitionData] value to
// use when evaluating the configuration of an object whose for_each argument
// is currently unknown.
//
// forEachType should be the type constraint of the unknown for_each argument
// value. This should be of a type that is valid to use in for_each, but
// if not then this will just return a very general RepetitionData that would
// be suitable (but less specific) for any valid for_each value.
func UnknownForEachRepetitionData(forEachType cty.Type) RepetitionData {
	switch {
	case forEachType.IsMapType():
		return RepetitionData{
			EachKey:   cty.UnknownVal(cty.String),
			EachValue: cty.UnknownVal(forEachType.ElementType()),
		}
	case forEachType.IsSetType() && forEachType.ElementType().Equals(cty.String):
		return RepetitionData{
			EachKey:   cty.UnknownVal(cty.String),
			EachValue: cty.UnknownVal(cty.String),
		}
	default:
		// We know that each.key is always a string, at least.
		return RepetitionData{
			EachKey:   cty.UnknownVal(cty.String),
			EachValue: cty.DynamicVal,
		}
	}
}
