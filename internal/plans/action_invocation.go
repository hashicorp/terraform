// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
)

type ActionInvocationInstance struct {
	Addr addrs.AbsActionInstance

	ActionTrigger ActionTrigger

	// Provider is the address of the provider configuration that was used
	// to plan this action, and thus the configuration that must also be
	// used to apply it.
	ProviderAddr addrs.AbsProviderConfig

	ConfigValue cty.Value
}

func (ai *ActionInvocationInstance) Equals(other *ActionInvocationInstance) bool {
	// Since the trigger can be the same if it's a CLI invocation we also compare the action addr
	return ai.Addr.Equal(other.Addr) && ai.ActionTrigger.Equals(other.ActionTrigger)
}

type ActionTrigger interface {
	actionTriggerSigil()

	TriggerEvent() configs.ActionTriggerEvent

	String() string

	Equals(to ActionTrigger) bool
}

type LifecycleActionTrigger struct {
	TriggeringResourceAddr addrs.AbsResourceInstance
	// Information about the trigger
	// The event that triggered this action invocation.
	ActionTriggerEvent configs.ActionTriggerEvent
	// The index of the action_trigger block that triggered this invocation.
	ActionTriggerBlockIndex int
	// The index of the action in the events list of the action_trigger block
	ActionsListIndex int
}

func (t LifecycleActionTrigger) TriggerEvent() configs.ActionTriggerEvent {
	return t.ActionTriggerEvent
}

func (t LifecycleActionTrigger) actionTriggerSigil() {}

func (t LifecycleActionTrigger) String() string {
	return t.TriggeringResourceAddr.String()
}

func (t LifecycleActionTrigger) Equals(other ActionTrigger) bool {
	o, ok := other.(LifecycleActionTrigger)
	if !ok {
		return false
	}

	return t.TriggeringResourceAddr.Equal(o.TriggeringResourceAddr) &&
		t.ActionTriggerBlockIndex == o.ActionTriggerBlockIndex &&
		t.ActionsListIndex == o.ActionsListIndex
}

var _ ActionTrigger = (*LifecycleActionTrigger)(nil)

// Encode produces a variant of the receiver that has its change values
// serialized so it can be written to a plan file. Pass the implied type of the
// corresponding resource type schema for correct operation.
func (ai *ActionInvocationInstance) Encode(schema *providers.ActionSchema) (*ActionInvocationInstanceSrc, error) {

	ret := &ActionInvocationInstanceSrc{
		Addr:          ai.Addr,
		ActionTrigger: ai.ActionTrigger,
		ProviderAddr:  ai.ProviderAddr,
	}

	if ai.ConfigValue != cty.NilVal {
		ty := cty.DynamicPseudoType
		if schema != nil {
			ty = schema.ConfigSchema.ImpliedType()
		}

		var err error
		ret.ConfigValue, err = NewDynamicValue(ai.ConfigValue, ty)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil

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
