// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackplan

import (
	"time"

	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/providers"
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

	// The raw representation of the raw state that was provided in the request
	// to create the plan. We use this primarily to perform mundane state
	// data structure maintenence operations, such as discarding keys that
	// are no longer needed or replacing data in old formats with the
	// equivalent new representations.
	PrevRunStateRaw map[string]*anypb.Any

	// RootInputValues are the input variable values provided to calculate
	// the plan. We must use the same values during the apply step to
	// sure that the actions taken can be consistent with what was planned.
	RootInputValues map[stackaddrs.InputVariable]cty.Value

	// ApplyTimeInputVariables are the names of the root input variable
	// values whose values must be re-supplied during the apply phase,
	// instead of being persisted in [Plan.RootInputValues].
	ApplyTimeInputVariables collections.Set[stackaddrs.InputVariable]

	// Components contains the separate plans for each of the compoonent
	// instances defined in the overall stack configuration, including any
	// nested component instances from embedded stacks.
	Components collections.Map[stackaddrs.AbsComponentInstance, *Component]

	// ProviderFunctionResults is a shared table of results from calling
	// provider functions. This is stored and loaded from during the planning
	// stage to use during apply operations.
	ProviderFunctionResults []providers.FunctionHash

	// PlanTimestamp is the time at which the plan was created.
	PlanTimestamp time.Time
}

// RequiredProviderInstances returns a description of all of the provider
// instance slots that are required to satisfy the resource instances
// belonging to the given component instance.
//
// See also stackeval.ComponentConfig.RequiredProviderInstances for a similar
// function that operates on the configuration of a component instance rather
// than the plan of one.
func (p *Plan) RequiredProviderInstances(addr stackaddrs.AbsComponentInstance) addrs.Set[addrs.RootProviderConfig] {
	component, ok := p.Components.GetOk(addr)
	if !ok {
		return addrs.MakeSet[addrs.RootProviderConfig]()
	}
	return component.RequiredProviderInstances()
}
