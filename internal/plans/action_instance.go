// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

type ActionInvocationInstance struct {
	Addr addrs.AbsActionInstance // mildwonkey TODO: this will be a *trigger* instance when that pr merges

	// Provider is the address of the provider configuration that was used
	// to plan this action, and thus the configuration that must also be
	// used to apply it.
	ProviderAddr addrs.AbsProviderConfig

	// nil resources = unlinked action
	// single resource = lifecycle or linked
	// multiple resources = linked
	LinkedResources []ResourceInstanceActionChange
}

type ResourceInstanceActionChange struct {
	// Addr is the absolute address of the resource instance that the change
	// will apply to.
	Addr addrs.AbsResourceInstance

	// DeposedKey is the identifier for a deposed object associated with the
	// given instance, or states.NotDeposed if this change applies to the
	// current object.
	DeposedKey states.DeposedKey

	// Change is an embedded description of the change.
	//
	// Generic Actions have no "change", just a record of the triggering
	// resource, so this may be missing a field.
	Change
}

// Encode produces a variant of the receiver that has its change values
// serialized so it can be written to a plan file. Pass the implied type of the
// corresponding resource type schema for correct operation.
func (ai *ActionInvocationInstance) Encode(schema providers.Schema) (*ActionInvocationInstanceSrc, error) {
	resourceChanges := make([]ResourceInstanceActionChangeSrc, 0, len(ai.LinkedResources))

	for i, rc := range ai.LinkedResources {
		resourceChanges[i] = ResourceInstanceActionChangeSrc{
			Addr: rc.Addr,
		}

		cs, err := rc.Change.Encode(&schema)
		if err != nil {
			return nil, err
		}

		resourceChanges[i].ChangeSrc = *cs
	}

	return &ActionInvocationInstanceSrc{
		Addr:            ai.Addr,
		ProviderAddr:    ai.ProviderAddr,
		LinkedResources: resourceChanges,
	}, nil
}

type ActionInvocationInstances []*ActionInvocationInstance

func (ais ActionInvocationInstances) DeepCopy() ActionInvocationInstances {
	if ais == nil {
		return ais
	}

	ret := make(ActionInvocationInstances, len(ais))
	for i, ai := range ais {
		ret[i] = ai.DeepCopy()
	}
	return ret
}

func (ai *ActionInvocationInstance) DeepCopy() *ActionInvocationInstance {
	if ai == nil {
		return ai
	}

	ret := *ai
	if ai.LinkedResources != nil {
		ret.LinkedResources = make([]ResourceInstanceActionChange, len(ai.LinkedResources))
		copy(ai.LinkedResources, ret.LinkedResources)
	}
	return &ret
}
