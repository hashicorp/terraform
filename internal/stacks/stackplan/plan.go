// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackplan

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// Plan is the main type in this package, representing an entire stack plan,
// or at least the subset of the information that Terraform needs to reliably
// apply the plan and detect any inconsistencies during the apply process.
//
// However, the process of _creating_ a plan doesn't actually produce a single
// object of this type, and instead produces fragments of it gradually as the
// planning process proceeds. The caller of the stack runtime must retain
// all of the raw parts in the order they were emitted and provide them back
// during the apply phase, and then we will finally construct a single instance
// of Plan covering the entire set of changes before we begin applying it.
type Plan struct {
	// Applyable is true for a plan that was successfully created in full and
	// is sufficient to be applied, or false if the plan is incomplete for
	// some reason, such as if an error occurred during planning and so
	// the planning process did not entirely run.
	Applyable bool

	// RootInputValues are the input variable values provided to calculate
	// the plan. We must use the same values during the apply step to
	// sure that the actions taken can be consistent with what was planned.
	RootInputValues map[stackaddrs.InputVariable]cty.Value

	// Components contains the separate plans for each of the compoonent
	// instances defined in the overall stack configuration, including any
	// nested component instances from embedded stacks.
	Components collections.Map[stackaddrs.AbsComponentInstance, *Component]
}
