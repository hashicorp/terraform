// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackstate

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateProducer is an interface of an object that can produce a state file
// and required it to be into AppliedChange objects.
type StateProducer interface {
	Addr() stackaddrs.AbsComponentInstance

	// ResourceSchema returns the schema for a resource type from a provider.
	ResourceSchema(ctx context.Context, providerTypeAddr addrs.Provider, mode addrs.ResourceMode, resourceType string) (*configschema.Block, error)
}

func FromState(ctx context.Context, state *states.State, plan *stackplan.Component, applyTimeInputs cty.Value, affectedResources addrs.Set[addrs.AbsResourceInstanceObject], producer StateProducer) ([]AppliedChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var changes []AppliedChange

	addr := producer.Addr()

	for _, rioAddr := range affectedResources {
		os := state.ResourceInstanceObjectSrc(rioAddr)
		var providerConfigAddr addrs.AbsProviderConfig
		var schema *configschema.Block
		if os != nil {
			rAddr := rioAddr.ResourceInstance.ContainingResource()
			rs := state.Resource(rAddr)
			if rs == nil {
				// We should not get here: it should be impossible to
				// have state for a resource instance object without
				// also having state for its containing resource, because
				// the object is nested inside the resource state.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Inconsistent updated state for resource",
					fmt.Sprintf(
						"There is a state for %s specifically, but somehow no state for its containing resource %s. This is a bug in Terraform.",
						rioAddr, rAddr,
					),
				))
				continue
			}
			providerConfigAddr = rs.ProviderConfig

			var err error
			schema, err = producer.ResourceSchema(
				ctx,
				rs.ProviderConfig.Provider,
				rAddr.Resource.Mode,
				rAddr.Resource.Type,
			)
			if err != nil {
				// It shouldn't be possible to get here because we would've
				// used the same schema we were just trying to retrieve
				// to encode the dynamic data in this states.State object
				// in the first place. If we _do_ get here then we won't
				// actually be able to save the updated state, which will
				// force the user to manually clean things up.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Can't fetch provider schema to save new state",
					fmt.Sprintf(
						"Failed to retrieve the schema for %s from provider %s: %s. This is a bug in Terraform.\n\nThe new state for this object cannot be saved. If this object was only just created, you may need to delete it manually in the target system to reconcile with the Terraform state before trying again.",
						rAddr, rs.ProviderConfig.Provider, err,
					),
				))
				continue
			}
		} else {
			// Our model doesn't have any way to represent the absense
			// of a provider configuration, so if we're trying to describe
			// just that the object has been deleted then we'll just
			// use a synthetic provider config address, this won't get
			// used for anything significant anyway.
			providerAddr := addrs.ImpliedProviderForUnqualifiedType(rioAddr.ResourceInstance.Resource.Resource.ImpliedProvider())
			providerConfigAddr = addrs.AbsProviderConfig{
				Module:   addrs.RootModule,
				Provider: providerAddr,
			}
		}

		var previousAddress *stackaddrs.AbsResourceInstanceObject
		if plannedChange := plan.ResourceInstancePlanned.Get(rioAddr); plannedChange != nil && plannedChange.Moved() {
			// If we moved the resource instance object, we need to record
			// the previous address in the applied change. The planned
			// change might be nil if the resource instance object was
			// deleted.
			previousAddress = &stackaddrs.AbsResourceInstanceObject{
				Component: addr,
				Item: addrs.AbsResourceInstanceObject{
					ResourceInstance: plannedChange.PrevRunAddr,
					DeposedKey:       addrs.NotDeposed,
				},
			}
		}

		changes = append(changes, &AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
				Component: addr,
				Item:      rioAddr,
			},
			PreviousResourceInstanceObjectAddr: previousAddress,
			NewStateSrc:                        os,
			ProviderConfigAddr:                 providerConfigAddr,
			Schema:                             schema,
		})
	}

	destroyPlan := plan.PlannedAction == plans.Delete || plan.PlannedAction == plans.Forget
	if plan.PlanComplete && destroyPlan && state.Empty() && !diags.HasErrors() {

		// We'll publish a special change type for the case where the
		// component instance was deleted and the state is now empty.
		//
		// We check here that we:
		//   - were planning to delete the component instance
		//   - have a complete plan (so no changes were deferred)
		//   - the state is now empty (so everything was actually deleted)
		//   - there were no errors in the diagnostics (so we published all changes)
		//
		// If all of the above are true, we'll happily publish this special
		// change type to indicate that the component instance was deleted.
		//
		// If the above weren't true then we'll publish the normal update
		// change type, which will mean this component stays in state for
		// now and will be tidied up properly in a follow-up change.

		changes = append(changes, &AppliedChangeComponentInstanceRemoved{
			ComponentAddr: stackaddrs.AbsComponent{
				Stack: addr.Stack,
				Item:  addr.Item.Component,
			},
			ComponentInstanceAddr: addr,
		})
	} else {
		ourChange := &AppliedChangeComponentInstance{
			ComponentAddr: stackaddrs.AbsComponent{
				Stack: addr.Stack,
				Item:  addr.Item.Component,
			},
			ComponentInstanceAddr: addr,
			Dependents:            plan.Dependents,
			Dependencies:          plan.Dependencies,
			OutputValues:          make(map[addrs.OutputValue]cty.Value, len(state.RootOutputValues)),
			InputVariables:        make(map[addrs.InputVariable]cty.Value, len(applyTimeInputs.Type().AttributeTypes())),
		}
		for name, os := range state.RootOutputValues {
			val := os.Value
			if os.Sensitive {
				val = val.Mark(marks.Sensitive)
			}
			ourChange.OutputValues[addrs.OutputValue{Name: name}] = val
		}
		for name, value := range applyTimeInputs.AsValueMap() {
			ourChange.InputVariables[addrs.InputVariable{Name: name}] = value
		}
		changes = append(changes, ourChange)
	}

	return changes, diags
}
