// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackplan

import (
	"time"

	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/plans"
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

	// Complete is true for a plan that shouldn't need any follow-up plans to
	// converge.
	Complete bool

	// Mode is the original mode of the plan.
	Mode plans.Mode

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

	// DeletedInputVariables tracks the set of input variables that are being
	// deleted by this plan. The apply operation will miss any values
	// that are not defined in the configuration, but should still emit
	// deletion events to remove them from the state.
	DeletedInputVariables collections.Set[stackaddrs.InputVariable]

	// DeletedOutputValues tracks the set of output values that are being
	// deleted by this plan. The apply operation will miss any output values
	// that are not defined in the configuration, but should still emit
	// deletion events to remove them from the state. Output values not being
	// deleted will be recomputed during the apply so are not needed.
	DeletedOutputValues collections.Set[stackaddrs.OutputValue]

	// Components contains the separate plans for each of the compoonent
	// instances defined in the overall stack configuration, including any
	// nested component instances from embedded stacks.
	Components collections.Map[stackaddrs.AbsComponentInstance, *Component]

	// DeletedComponents are a set of components that are in the state that
	// should just be removed without any apply operation. This is typically
	// because they are not referenced in the configuration and have no
	// associated resources.
	DeletedComponents collections.Set[stackaddrs.AbsComponentInstance]

	// ProviderFunctionResults is a shared table of results from calling
	// provider functions. This is stored and loaded from during the planning
	// stage to use during apply operations.
	ProviderFunctionResults []providers.FunctionHash

	// PlanTimestamp is the time at which the plan was created.
	PlanTimestamp time.Time
}

// ComponentInstances returns a set of the component instances that belong to
// the given component.
func (p *Plan) ComponentInstances(addr stackaddrs.AbsComponent) collections.Set[stackaddrs.ComponentInstance] {
	ret := collections.NewSet[stackaddrs.ComponentInstance]()
	for elem := range p.Components.All() {
		if elem.Stack.String() != addr.Stack.String() {
			// Then
			continue
		}
		if elem.Item.Component.Name != addr.Item.Name {
			continue
		}
		ret.Add(elem.Item)
	}
	return ret
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
