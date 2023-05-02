// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package checks

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// Status represents the status of an individual check associated with a
// checkable object.
type Status rune

//go:generate go run golang.org/x/tools/cmd/stringer -type=Status

const (
	// StatusUnknown represents that there is not yet a conclusive result
	// for the check, either because we haven't yet visited its associated
	// object or because the check condition itself depends on a value not
	// yet known during planning.
	StatusUnknown Status = 0
	// NOTE: Our implementation relies on StatusUnknown being the zero value
	// of Status.

	// StatusPass represents that Terraform Core has evaluated the check's
	// condition and it returned true, indicating success.
	StatusPass Status = 'P'

	// StatusFail represents that Terraform Core has evaluated the check's
	// condition and it returned false, indicating failure.
	StatusFail Status = 'F'

	// StatusError represents that Terraform Core tried to evaluate the check's
	// condition but encountered an error while evaluating the check expression.
	//
	// This is different than StatusFail because StatusFail indiciates that
	// the condition was valid and returned false, whereas StatusError
	// indicates that the condition was not valid at all.
	StatusError Status = 'E'
)

// StatusForCtyValue returns the Status value corresponding to the given
// cty Value, which must be one of either cty.True, cty.False, or
// cty.UnknownVal(cty.Bool) or else this function will panic.
//
// The current behavior of this function is:
//
//	cty.True                  StatusPass
//	cty.False                 StatusFail
//	cty.UnknownVal(cty.Bool)  StatusUnknown
//
// Any other input will panic. Note that there's no value that can produce
// StatusError, because in case of a condition error there will not typically
// be a result value at all.
func StatusForCtyValue(v cty.Value) Status {
	if !v.Type().Equals(cty.Bool) {
		panic(fmt.Sprintf("cannot use %s as check status", v.Type().FriendlyName()))
	}
	if v.IsNull() {
		panic("cannot use null as check status")
	}

	switch {
	case v == cty.True:
		return StatusPass
	case v == cty.False:
		return StatusFail
	case !v.IsKnown():
		return StatusUnknown
	default:
		// Should be impossible to get here unless something particularly
		// weird is going on, like a marked condition result.
		panic(fmt.Sprintf("cannot use %#v as check status", v))
	}
}
