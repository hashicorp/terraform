// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

type ActionInvocationInstance struct {
	Addr                   addrs.AbsActionInstance
	TriggeringResourceAddr addrs.AbsResourceInstance

	// Information about the trigger
	// The event that triggered this action invocation.
	TriggerEvent configs.ActionTriggerEvent
	// The index of the action_trigger block that triggered this invocation.
	ActionTriggerBlockIndex int
	// The index of the action in the evens list of the action_trigger block
	ActionsListIndex int

	// Provider is the address of the provider configuration that was used
	// to plan this action, and thus the configuration that must also be
	// used to apply it.
	ProviderAddr addrs.AbsProviderConfig
}

// Encode produces a variant of the receiver that has its change values
// serialized so it can be written to a plan file. Pass the implied type of the
// corresponding resource type schema for correct operation.
func (ai *ActionInvocationInstance) Encode() (*ActionInvocationInstanceSrc, error) {
	return &ActionInvocationInstanceSrc{
		Addr:                    ai.Addr,
		TriggeringResourceAddr:  ai.TriggeringResourceAddr,
		TriggerEvent:            ai.TriggerEvent,
		ActionTriggerBlockIndex: ai.ActionTriggerBlockIndex,
		ActionsListIndex:        ai.ActionsListIndex,
		ProviderAddr:            ai.ProviderAddr,
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
	return &ret
}
