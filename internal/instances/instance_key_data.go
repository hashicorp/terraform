// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
