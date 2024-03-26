// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package deferring

import (
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/zclconf/go-cty/cty"
)

// deferredResourceInstance tracks information about a resource instance whose
// address is precisely known but whose planned action has been deferred for
// some other reason.
type deferredResourceInstance struct {
	// plannedAction is the action that Terraform expects to take for this
	// resource instance in a future round.
	//
	// This can be set to plans.Undecided in situations where there isn't
	// even enough information to decide what the action would be.
	plannedAction plans.Action

	// plannedValue is an approximation of the value that Terraform expects
	// to plan for this resource instance in a future round, using unknown
	// values in locations where a concrete value cannot yet be decided.
	//
	// In the most extreme case, plannedValue could be cty.DynamicVal to
	// reflect that we know nothing at all about the resource instance, or
	// an unknown value of the resource instance's schema type if the values
	// are completely unknown but we've at least got enough information to
	// approximate the type of the value.
	//
	// However, ideally this should be a known object value that potentially
	// has unknown values for individual attributes inside, since that gives
	// the most context to aid in finding errors that would definitely arise
	// on a future round, and thus shorten the iteration time to find that
	// problem.
	plannedValue cty.Value
}
