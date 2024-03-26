// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package deferring

import (
	"github.com/zclconf/go-cty/cty"
)

// deferredPartialExpandedResource tracks placeholder information for an
// unbounded set of potential resource instances sharing a common known
// address prefix.
//
// This is for situations where we can't even predict which instances of
// a resource will be declared, due to a count or for_each argument being
// unknown. The unknown repetition argument could either be on the resource
// itself or on one of its ancestor module calls.
type deferredPartialExpandedResource struct {
	// valuePlaceholder is a placeholder value describes what all of the
	// potential instances in the unbounded set represented by this object
	// have in common, using unknown values for any parts where we cannot
	// guarantee that all instances will agree.
	valuePlaceholder cty.Value
}
