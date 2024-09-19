// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackplan

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// PlanProducer is an interface of an object that can produce a plan and
// require it to be converted into PlannedChange objects.
type PlanProducer interface {
	Addr() stackaddrs.AbsComponentInstance

	// RequiredComponents returns the static set of components that this
	// component depends on. Static in this context means based on the
	// configuration, so this result shouldn't change based on the type of
	// plan.
	//
	// Normal and destroy plans should return the same set of components,
	// with dependents and dependencies computed from this set during the
	// apply phase.
	RequiredComponents(ctx context.Context) collections.Set[stackaddrs.AbsComponent]

	// ResourceSchema returns the schema for a resource type from a provider.
	ResourceSchema(ctx context.Context, providerTypeAddr addrs.Provider, mode addrs.ResourceMode, resourceType string) (*configschema.Block, error)
}

func FromPlan(ctx context.Context, config *configs.Config, plan *plans.Plan, refreshPlan *plans.Plan, action plans.Action, producer PlanProducer) ([]PlannedChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var changes []PlannedChange

	var outputs map[string]cty.Value
	if refreshPlan != nil {
		// we're going to be a little cheeky and publish the outputs as being
		// the results from the refresh part of the plan. This will then be
		// consumed by the apply part of the plan to ensure that the outputs
		// are correctly updated. The refresh plan should only be present if the
		// main plan was a destroy plan in which case the outputs that the
		// apply needs do actually come from the refresh.
		outputs = OutputsFromPlan(config, refreshPlan)
	} else {
		outputs = OutputsFromPlan(config, plan)
	}

	// We must always at least announce that the component instance exists,
	// and that must come before any resource instance changes referring to it.
	changes = append(changes, &PlannedChangeComponentInstance{
		Addr: producer.Addr(),

		Action:                         action,
		Mode:                           plan.UIMode,
		PlanApplyable:                  plan.Applyable,
		PlanComplete:                   plan.Complete,
		RequiredComponents:             producer.RequiredComponents(ctx),
		PlannedInputValues:             plan.VariableValues,
		PlannedInputValueMarks:         plan.VariableMarks,
		PlannedOutputValues:            outputs,
		PlannedCheckResults:            plan.Checks,
		PlannedProviderFunctionResults: plan.ProviderFunctionResults,

		// We must remember the plan timestamp so that the plantimestamp
		// function can return a consistent result during a later apply phase.
		PlanTimestamp: plan.Timestamp,
	})

	seenObjects := addrs.MakeSet[addrs.AbsResourceInstanceObject]()
	for _, rsrcChange := range plan.Changes.Resources {
		schema, err := producer.ResourceSchema(
			ctx,
			rsrcChange.ProviderAddr.Provider,
			rsrcChange.Addr.Resource.Resource.Mode,
			rsrcChange.Addr.Resource.Resource.Type,
		)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Can't fetch provider schema to save plan",
				fmt.Sprintf(
					"Failed to retrieve the schema for %s from provider %s: %s. This is a bug in Terraform.",
					rsrcChange.Addr, rsrcChange.ProviderAddr.Provider, err,
				),
			))
			continue
		}

		objAddr := addrs.AbsResourceInstanceObject{
			ResourceInstance: rsrcChange.Addr,
			DeposedKey:       rsrcChange.DeposedKey,
		}
		var priorStateSrc *states.ResourceInstanceObjectSrc
		if plan.PriorState != nil {
			priorStateSrc = plan.PriorState.ResourceInstanceObjectSrc(objAddr)
		}

		changes = append(changes, &PlannedChangeResourceInstancePlanned{
			ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
				Component: producer.Addr(),
				Item:      objAddr,
			},
			ChangeSrc:          rsrcChange,
			Schema:             schema,
			PriorStateSrc:      priorStateSrc,
			ProviderConfigAddr: rsrcChange.ProviderAddr,

			// TODO: Also provide the previous run state, if it's
			// different from the prior state, and signal whether the
			// difference from previous run seems "notable" per
			// Terraform Core's heuristics. Only the external plan
			// description needs that info, to populate the
			// "changes outside of Terraform" part of the plan UI;
			// the raw plan only needs the prior state.
		})
		seenObjects.Add(objAddr)
	}

	// We need to keep track of the deferred changes as well
	for _, dr := range plan.DeferredResources {
		rsrcChange := dr.ChangeSrc
		objAddr := addrs.AbsResourceInstanceObject{
			ResourceInstance: rsrcChange.Addr,
			DeposedKey:       rsrcChange.DeposedKey,
		}
		var priorStateSrc *states.ResourceInstanceObjectSrc
		if plan.PriorState != nil {
			priorStateSrc = plan.PriorState.ResourceInstanceObjectSrc(objAddr)
		}

		schema, err := producer.ResourceSchema(
			ctx,
			rsrcChange.ProviderAddr.Provider,
			rsrcChange.Addr.Resource.Resource.Mode,
			rsrcChange.Addr.Resource.Resource.Type,
		)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Can't fetch provider schema to save plan",
				fmt.Sprintf(
					"Failed to retrieve the schema for %s from provider %s: %s. This is a bug in Terraform.",
					rsrcChange.Addr, rsrcChange.ProviderAddr.Provider, err,
				),
			))
			continue
		}

		plannedChangeResourceInstance := PlannedChangeResourceInstancePlanned{
			ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
				Component: producer.Addr(),
				Item:      objAddr,
			},
			ChangeSrc:          rsrcChange,
			Schema:             schema,
			PriorStateSrc:      priorStateSrc,
			ProviderConfigAddr: rsrcChange.ProviderAddr,
		}
		changes = append(changes, &PlannedChangeDeferredResourceInstancePlanned{
			DeferredReason:          dr.DeferredReason,
			ResourceInstancePlanned: plannedChangeResourceInstance,
		})
		seenObjects.Add(objAddr)
	}

	// We also need to catch any objects that exist in the "prior state"
	// but don't have any actions planned, since we still need to capture
	// the prior state part in case it was updated by refreshing during
	// the plan walk.
	if priorState := plan.PriorState; priorState != nil {
		for _, addr := range priorState.AllResourceInstanceObjectAddrs() {
			if seenObjects.Has(addr) {
				// We're only interested in objects that didn't appear
				// in the plan, such as data resources whose read has
				// completed during the plan phase.
				continue
			}

			rs := priorState.Resource(addr.ResourceInstance.ContainingResource())
			os := priorState.ResourceInstanceObjectSrc(addr)
			schema, err := producer.ResourceSchema(
				ctx,
				rs.ProviderConfig.Provider,
				addr.ResourceInstance.Resource.Resource.Mode,
				addr.ResourceInstance.Resource.Resource.Type,
			)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Can't fetch provider schema to save plan",
					fmt.Sprintf(
						"Failed to retrieve the schema for %s from provider %s: %s. This is a bug in Terraform.",
						addr, rs.ProviderConfig.Provider, err,
					),
				))
				continue
			}

			changes = append(changes, &PlannedChangeResourceInstancePlanned{
				ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
					Component: producer.Addr(),
					Item:      addr,
				},
				Schema:             schema,
				PriorStateSrc:      os,
				ProviderConfigAddr: rs.ProviderConfig,
				// We intentionally omit ChangeSrc, because we're not actually
				// planning to change this object during the apply phase, only
				// to update its state data.
			})
			seenObjects.Add(addr)
		}
	}

	prevRunState := plan.PrevRunState
	if refreshPlan != nil {
		// If we executed a refresh plan as part of this, then the true
		// previous run state is the one from the refresh plan, because
		// the later plan used the output of the refresh plan as the
		// previous state.
		prevRunState = refreshPlan.PrevRunState
	}

	// We also have one more unusual case to deal with: if an object
	// existed at the end of the previous run but was found to have
	// been deleted when we refreshed during planning then it will
	// not be present in either the prior state _or_ the plan, but
	// we still need to include a stubby object for it in the plan
	// so we can remember to discard it from the state during the
	// apply phase.
	if prevRunState != nil {
		for _, addr := range prevRunState.AllResourceInstanceObjectAddrs() {
			if seenObjects.Has(addr) {
				// We're only interested in objects that didn't appear
				// in the plan, such as data resources whose read has
				// completed during the plan phase.
				continue
			}

			rs := prevRunState.Resource(addr.ResourceInstance.ContainingResource())

			changes = append(changes, &PlannedChangeResourceInstancePlanned{
				ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
					Component: producer.Addr(),
					Item:      addr,
				},
				ProviderConfigAddr: rs.ProviderConfig,
				// Everything except the addresses are omitted in this case,
				// which represents that we should just delete the object
				// from the state when applied, and not take any other
				// action.
			})
			seenObjects.Add(addr)
		}
	}

	return changes, diags
}

func OutputsFromPlan(config *configs.Config, plan *plans.Plan) map[string]cty.Value {
	if plan == nil {
		return nil
	}

	// We need to vary our behavior here slightly depending on what action
	// we're planning to take with this overall component: normally we want
	// to use the "planned new state"'s output values, but if we're actually
	// planning to destroy all of the infrastructure managed by this
	// component then the planned new state has no output values at all,
	// so we'll use the prior state's output values instead just in case
	// we also need to plan destroying another component instance
	// downstream of this one which will make use of this instance's
	// output values _before_ we destroy it.
	//
	// FIXME: We're using UIMode for this decision, despite its doc comment
	// saying we shouldn't, because this behavior is an offshoot of the
	// already-documented annoying exception to that rule where various
	// parts of Terraform use UIMode == DestroyMode in particular to deal
	// with necessary variations during a "full destroy". Hopefully we'll
	// eventually find a more satisfying solution for that, in which case
	// we should update the following to use that solution too.
	attrs := make(map[string]cty.Value)
	switch plan.UIMode {
	case plans.DestroyMode:
		// The "prior state" of the plan includes any new information we
		// learned by "refreshing" before we planned to destroy anything,
		// and so should be as close as possible to the current
		// (pre-destroy) state of whatever infrastructure this component
		// instance is managing.
		for _, os := range plan.PriorState.RootOutputValues {
			v := os.Value
			if os.Sensitive {
				// For our purposes here, a static sensitive flag on the
				// output value is indistinguishable from the value having
				// been dynamically marked as sensitive.
				v = v.Mark(marks.Sensitive)
			}
			attrs[os.Addr.OutputValue.Name] = v
		}
	default:
		for _, changeSrc := range plan.Changes.Outputs {
			if len(changeSrc.Addr.Module) > 0 {
				// Only include output values of the root module as part
				// of the component.
				continue
			}

			name := changeSrc.Addr.OutputValue.Name
			change, err := changeSrc.Decode()
			if err != nil {
				attrs[name] = cty.DynamicVal
				continue
			}

			if changeSrc.Sensitive {
				// For our purposes here, a static sensitive flag on the
				// output value is indistinguishable from the value having
				// been dynamically marked as sensitive.
				attrs[name] = change.After.Mark(marks.Sensitive)
				continue
			}

			// Otherwise, just use the value as-is.
			attrs[name] = change.After
		}
	}

	if config != nil {
		// If the plan only ran partially then we might be missing
		// some planned changes for output values, which could
		// cause "attrs" to have an incomplete set of attributes.
		// To avoid confusing downstream errors we'll insert unknown
		// values for any declared output values that don't yet
		// have a final value.
		for name := range config.Module.Outputs {
			if _, ok := attrs[name]; !ok {
				// We can't do any better than DynamicVal because
				// output values in the modules language don't
				// have static type constraints.
				attrs[name] = cty.DynamicVal
			}
		}
		// In the DestroyMode case above we might also find ourselves
		// with some remnant additional output values that have since
		// been removed from the configuration, but yet remain in the
		// state. Destroying with a different configuration than was
		// most recently applied is not guaranteed to work, but we
		// can make it more likely to work by dropping anything that
		// isn't currently declared, since referring directly to these
		// would be a static validation error anyway, and including
		// them might cause aggregate operations like keys(component.foo)
		// to produce broken results.
		for name := range attrs {
			_, declared := config.Module.Outputs[name]
			if !declared {
				// (deleting map elements during iteration is valid in Go,
				// unlike some other languages.)
				delete(attrs, name)
			}
		}
	}

	return attrs
}
