// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"sort"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

// Plan is the top-level type representing a planned set of changes.
//
// A plan is a summary of the set of changes required to move from a current
// state to a goal state derived from configuration. The described changes
// are not applied directly, but contain an approximation of the final
// result that will be completed during apply by resolving any values that
// cannot be predicted.
//
// A plan must always be accompanied by the configuration it was built from,
// since the plan does not itself include all of the information required to
// make the changes indicated.
type Plan struct {
	// Mode is the mode under which this plan was created.
	//
	// This is only recorded to allow for UI differences when presenting plans
	// to the end-user, and so it must not be used to influence apply-time
	// behavior. The actions during apply must be described entirely by
	// the Changes field, regardless of how the plan was created.
	//
	// FIXME: destroy operations still rely on DestroyMode being set, because
	// there is no other source of this information in the plan. New behavior
	// should not be added based on this flag, and changing the flag should be
	// checked carefully against existing destroy behaviors.
	UIMode Mode

	// VariableValues, VariableMarks, and ApplyTimeVariables together describe
	// how Terraform should decide the input variable values for the apply
	// phase if this plan is to be applied.
	//
	// VariableValues and VariableMarks describe persisted (non-ephemeral)
	// values that were set as part of the planning options and are to be
	// re-used during the apply phase. VariableValues can potentially contain
	// unknown values for a speculative plan, but the variable values must
	// all be known for a plan that will subsequently be applied.
	//
	// ApplyTimeVariables retains the names of any ephemeral variables that were
	// set (non-null) during the planning phase and must therefore be
	// re-supplied by the caller (potentially with different values) during
	// the apply phase. Ephemeral input variables are intended for populating
	// arguments for other ephemeral objects in the configuration, such as
	// provider configurations. Although the values for these variables can
	// change between plan and apply, their "nullness" may not.
	VariableValues     map[string]DynamicValue
	VariableMarks      map[string][]cty.PathValueMarks
	ApplyTimeVariables collections.Set[string]

	Changes           *ChangesSrc
	DriftedResources  []*ResourceInstanceChangeSrc
	DeferredResources []*DeferredResourceInstanceChangeSrc
	TargetAddrs       []addrs.Targetable
	ForceReplaceAddrs []addrs.AbsResourceInstance
	Backend           Backend

	// Complete is true if Terraform considers this to be a "complete" plan,
	// which is to say that it includes a planned action (even if no-op)
	// for every resource instance object that was mentioned across both
	// the desired state and prior state.
	//
	// If Complete is false then the plan might still be applyable (check
	// [Plan.Applyable]) but after applying it the operator should be reminded
	// to plan and apply again to hopefully make more progress towards
	// convergence.
	//
	// For an incomplete plan, other fields of this type may give more context
	// about why the plan is incomplete, which a UI layer could present to
	// the user as part of a warning that the plan is incomplete.
	Complete bool

	// Applyable is true if both Terraform was able to create a plan
	// successfully and if the plan calls for making some sort of meaningful
	// change.
	//
	// If [Plan.Errored] is also set then that means the plan is non-applyable
	// due to an error. If not then the plan was created successfully but found
	// no material differences between desired and prior state, and so
	// applying this plan would achieve nothing.
	Applyable bool

	// Errored is true if the Changes information is incomplete because
	// the planning operation failed. An errored plan cannot be applied,
	// but can be cautiously inspected for debugging purposes.
	Errored bool

	// Checks captures a snapshot of the (probably-incomplete) check results
	// at the end of the planning process.
	//
	// If this plan is applyable (that is, if the planning process completed
	// without errors) then the set of checks here should be complete even
	// though some of them will likely have StatusUnknown where the check
	// condition depends on values we won't know until the apply step.
	Checks *states.CheckResults

	// RelevantAttributes is a set of resource instance addresses and
	// attributes that are either directly affected by proposed changes or may
	// have indirectly contributed to them via references in expressions.
	//
	// This is the result of a heuristic and is intended only as a hint to
	// the UI layer in case it wants to emphasize or de-emphasize certain
	// resources. Don't use this to drive any non-cosmetic behavior, especially
	// including anything that would be subject to compatibility constraints.
	RelevantAttributes []globalref.ResourceAttr

	// PrevRunState and PriorState both describe the situation that the plan
	// was derived from:
	//
	// PrevRunState is a representation of the outcome of the previous
	// Terraform operation, without any updates from the remote system but
	// potentially including some changes that resulted from state upgrade
	// actions.
	//
	// PriorState is a representation of the current state of remote objects,
	// which will differ from PrevRunState if the "refresh" step returned
	// different data, which might reflect drift.
	//
	// PriorState is the main snapshot we use for actions during apply.
	// PrevRunState is only here so that we can diff PriorState against it in
	// order to report to the user any out-of-band changes we've detected.
	PrevRunState *states.State
	PriorState   *states.State

	// ExternalReferences are references that are being made to resources within
	// the plan from external sources.
	//
	// This is never recorded outside of Terraform. It is not written into the
	// binary plan file, and it is not written into the JSON structured outputs.
	// The testing framework never writes the plans out but holds everything in
	// memory as it executes, so there is no need to add any kind of
	// serialization for this field. This does mean that you shouldn't rely on
	// this field existing unless you have just generated the plan.
	ExternalReferences []*addrs.Reference

	// Overrides contains the set of overrides that were applied while making
	// this plan. We need to provide the same set of overrides when applying
	// the plan so we preserve them here. As with  ExternalReferences, this is
	// only used by the testing framework and so isn't written into any external
	// representation of the plan.
	Overrides *mocking.Overrides

	// Timestamp is the record of truth for when the plan happened.
	Timestamp time.Time

	// ProviderFunctionResults stores hashed results from all provider
	// function calls, so that calls during apply can be checked for
	// consistency.
	ProviderFunctionResults []providers.FunctionHash
}

// ProviderAddrs returns a list of all of the provider configuration addresses
// referenced throughout the receiving plan.
//
// The result is de-duplicated so that each distinct address appears only once.
func (p *Plan) ProviderAddrs() []addrs.AbsProviderConfig {
	if p == nil || p.Changes == nil {
		return nil
	}

	m := map[string]addrs.AbsProviderConfig{}
	for _, rc := range p.Changes.Resources {
		m[rc.ProviderAddr.String()] = rc.ProviderAddr
	}
	if len(m) == 0 {
		return nil
	}

	// This is mainly just so we'll get stable results for testing purposes.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ret := make([]addrs.AbsProviderConfig, len(keys))
	for i, key := range keys {
		ret[i] = m[key]
	}

	return ret
}

// Backend represents the backend-related configuration and other data as it
// existed when a plan was created.
type Backend struct {
	// Type is the type of backend that the plan will apply against.
	Type string

	// Config is the configuration of the backend, whose schema is decided by
	// the backend Type.
	Config DynamicValue

	// Workspace is the name of the workspace that was active when the plan
	// was created. It is illegal to apply a plan created for one workspace
	// to the state of another workspace.
	// (This constraint is already enforced by the statefile lineage mechanism,
	// but storing this explicitly allows us to return a better error message
	// in the situation where the user has the wrong workspace selected.)
	Workspace string
}

func NewBackend(typeName string, config cty.Value, configSchema *configschema.Block, workspaceName string) (*Backend, error) {
	dv, err := NewDynamicValue(config, configSchema.ImpliedType())
	if err != nil {
		return nil, err
	}

	return &Backend{
		Type:      typeName,
		Config:    dv,
		Workspace: workspaceName,
	}, nil
}
