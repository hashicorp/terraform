// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackplan

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/plans/planproto"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/version"
)

func LoadFromProto(msgs []*anypb.Any) (*Plan, error) {
	ret := &Plan{
		RootInputValues: make(map[stackaddrs.InputVariable]cty.Value),
		Components:      collections.NewMap[stackaddrs.AbsComponentInstance, *Component](),
	}

	foundHeader := false
	for i, rawMsg := range msgs {
		msg, err := anypb.UnmarshalNew(rawMsg, proto.UnmarshalOptions{
			// Just the default unmarshalling options
		})
		if err != nil {
			return nil, fmt.Errorf("invalid raw message %d: %w", i, err)
		}

		// The references to specific message types below ensure that
		// the protobuf descriptors for these types are included in the
		// compiled program, and thus available in the global protobuf
		// registry that anypb.UnmarshalNew relies on above.
		switch msg := msg.(type) {

		case *tfstackdata1.PlanHeader:
			wantVersion := version.SemVer.String()
			gotVersion := msg.TerraformVersion
			if gotVersion != wantVersion {
				return nil, fmt.Errorf("plan was created by Terraform %s, but this is Terraform %s", gotVersion, wantVersion)
			}
			ret.PrevRunStateRaw = msg.PrevRunStateRaw
			foundHeader = true

		case *tfstackdata1.PlanApplyable:
			ret.Applyable = msg.Applyable

		case *tfstackdata1.PlanRootInputValue:
			addr := stackaddrs.InputVariable{
				Name: msg.Name,
			}
			dv := plans.DynamicValue(msg.Value.Msgpack)
			val, err := dv.Decode(cty.DynamicPseudoType)
			if err != nil {
				return nil, fmt.Errorf("invalid stored value for %s: %w", addr, err)
			}
			ret.RootInputValues[addr] = val

		case *tfstackdata1.PlanComponentInstance:
			addr, diags := stackaddrs.ParseAbsComponentInstanceStr(msg.ComponentInstanceAddr)
			if diags.HasErrors() {
				// Should not get here because the address we're parsing
				// should've been produced by this same version of Terraform.
				return nil, fmt.Errorf("invalid component instance address syntax in %q", msg.ComponentInstanceAddr)
			}

			dependencies := collections.NewSet[stackaddrs.AbsComponent]()
			for _, rawAddr := range msg.DependsOnComponentAddrs {
				// NOTE: We're using the component _instance_ address parser
				// here, but we really want just components, so we'll need to
				// check afterwards to make sure we don't have an instance key.
				addr, diags := stackaddrs.ParseAbsComponentInstanceStr(rawAddr)
				if diags.HasErrors() {
					return nil, fmt.Errorf("invalid component address syntax in %q", rawAddr)
				}
				if addr.Item.Key != addrs.NoKey {
					return nil, fmt.Errorf("invalid component address syntax in %q: is actually a component instance address", rawAddr)
				}
				realAddr := stackaddrs.AbsComponent{
					Stack: addr.Stack,
					Item:  addr.Item.Component,
				}
				dependencies.Add(realAddr)
			}

			plannedAction, err := planproto.FromAction(msg.PlannedAction)
			if err != nil {
				return nil, fmt.Errorf("decoding plan for %s: %w", addr, err)
			}

			outputVals := make(map[addrs.OutputValue]cty.Value)
			for name, rawVal := range msg.PlannedOutputValues {
				v, err := tfstackdata1.DynamicValueFromTFStackData1(rawVal, cty.DynamicPseudoType)
				if err != nil {
					return nil, fmt.Errorf("decoding output value %q for %s: %w", name, addr, err)
				}
				outputVals[addrs.OutputValue{Name: name}] = v
			}

			if !ret.Components.HasKey(addr) {
				ret.Components.Put(addr, &Component{
					PlannedAction:       plannedAction,
					PlanApplyable:       msg.PlanApplyable,
					PlanComplete:        msg.PlanComplete,
					Dependencies:        dependencies,
					Dependents:          collections.NewSet[stackaddrs.AbsComponent](),
					PlannedOutputValues: outputVals,

					ResourceInstancePlanned:        addrs.MakeMap[addrs.AbsResourceInstanceObject, *plans.ResourceInstanceChangeSrc](),
					ResourceInstancePriorState:     addrs.MakeMap[addrs.AbsResourceInstanceObject, *states.ResourceInstanceObjectSrc](),
					ResourceInstanceProviderConfig: addrs.MakeMap[addrs.AbsResourceInstanceObject, addrs.AbsProviderConfig](),
				})
			}
			c := ret.Components.Get(addr)
			err = c.PlanTimestamp.UnmarshalText([]byte(msg.PlanTimestamp))
			if err != nil {
				return nil, fmt.Errorf("invalid plan timestamp %q for %s", msg.PlanTimestamp, addr)
			}

		case *tfstackdata1.PlanResourceInstanceChangePlanned:
			cAddr, diags := stackaddrs.ParseAbsComponentInstanceStr(msg.ComponentInstanceAddr)
			if diags.HasErrors() {
				return nil, fmt.Errorf("invalid component instance address syntax in %q", msg.ComponentInstanceAddr)
			}
			riAddr, diags := addrs.ParseAbsResourceInstanceStr(msg.ResourceInstanceAddr)
			if diags.HasErrors() {
				return nil, fmt.Errorf("invalid resource instance address syntax in %q", msg.ResourceInstanceAddr)
			}
			var deposedKey addrs.DeposedKey
			if msg.DeposedKey != "" {
				deposedKey, err = addrs.ParseDeposedKey(msg.DeposedKey)
				if err != nil {
					return nil, fmt.Errorf("invalid deposed key syntax in %q", msg.DeposedKey)
				}
			}
			providerConfigAddr, diags := addrs.ParseAbsProviderConfigStr(msg.ProviderConfigAddr)
			if diags.HasErrors() {
				return nil, fmt.Errorf("invalid provider configuration address syntax in %q", msg.ProviderConfigAddr)
			}
			fullAddr := addrs.AbsResourceInstanceObject{
				ResourceInstance: riAddr,
				DeposedKey:       deposedKey,
			}
			c, ok := ret.Components.GetOk(cAddr)
			if !ok {
				return nil, fmt.Errorf("resource instance change for unannounced component instance %s", cAddr)
			}

			c.ResourceInstanceProviderConfig.Put(fullAddr, providerConfigAddr)

			var riPlan *plans.ResourceInstanceChangeSrc
			// Not all "planned changes" for resource instances are actually
			// changes in the plans.Change sense, confusingly: sometimes the
			// "change" we're recording is just to overwrite the state entry
			// with a refreshed copy, in which case riPlan is nil and
			// msg.PriorState is the main content of this change, handled below.
			if msg.Change != nil {
				riPlan, err = planfile.ResourceChangeFromProto(msg.Change)
				if err != nil {
					return nil, fmt.Errorf("invalid resource instance change: %w", err)
				}
				// We currently have some redundant information in the nested
				// "change" object due to having reused some protobuf message
				// types from the traditional Terraform CLI planproto format.
				// We'll make sure the redundant information is consistent
				// here because otherwise they're likely to cause
				// difficult-to-debug problems downstream.
				if !riPlan.Addr.Equal(fullAddr.ResourceInstance) && riPlan.DeposedKey == fullAddr.DeposedKey {
					return nil, fmt.Errorf("planned change has inconsistent address to its containing object")
				}
				if !riPlan.ProviderAddr.Equal(providerConfigAddr) {
					return nil, fmt.Errorf("planned change has inconsistent provider configuration address to its containing object")
				}

				c.ResourceInstancePlanned.Put(fullAddr, riPlan)
			}

			if msg.PriorState != nil {
				stateSrc, err := stackstate.DecodeProtoResourceInstanceObject(msg.PriorState)
				if err != nil {
					return nil, fmt.Errorf("invalid prior state for %s: %w", fullAddr, err)
				}
				c.ResourceInstancePriorState.Put(fullAddr, stateSrc)
			} else {
				// We'll record an explicit nil just to affirm that there's
				// intentionally no prior state for this resource instance
				// object.
				c.ResourceInstancePriorState.Put(fullAddr, nil)
			}

		default:
			// Should not get here, because a stack plan can only be loaded by
			// the same version of Terraform that created it, and the above
			// should cover everything this version of Terraform can possibly
			// emit during PlanStackChanges.
			return nil, fmt.Errorf("unsupported raw message type %T at index %d", msg, i)
		}
	}

	// If we got through all of the messages without encountering at least
	// one *PlanHeader then we'll abort because we may have lost part of the
	// plan sequence somehow.
	if !foundHeader {
		return nil, fmt.Errorf("missing PlanHeader")
	}

	// Before we return we'll calculate the reverse dependency information
	// based on the forward dependency information we loaded above.
	for _, elem := range ret.Components.Elems() {
		dependentInstAddr := elem.K
		dependentAddr := stackaddrs.AbsComponent{
			Stack: dependentInstAddr.Stack,
			Item:  dependentInstAddr.Item.Component,
		}

		for _, dependencyAddr := range elem.V.Dependencies.Elems() {
			// FIXME: This is very inefficient because the current data structure doesn't
			// allow looking up all of the component instances that have a particular
			// component. This'll be okay as long as the number of components is
			// small, but we'll need to improve this if we ever want to support stacks
			// with a large number of components.
			for _, elem := range ret.Components.Elems() {
				maybeDependencyInstAddr := elem.K
				maybeDependencyAddr := stackaddrs.AbsComponent{
					Stack: maybeDependencyInstAddr.Stack,
					Item:  maybeDependencyInstAddr.Item.Component,
				}
				if dependencyAddr.UniqueKey() == maybeDependencyAddr.UniqueKey() {
					elem.V.Dependents.Add(dependentAddr)
				}
			}
		}
	}

	return ret, nil
}
