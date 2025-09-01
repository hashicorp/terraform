// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
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

	Less(other ActionTrigger) bool
}

var (
	_ ActionTrigger = (*LifecycleActionTrigger)(nil)
	_ ActionTrigger = (*InvokeActionTrigger)(nil)
)

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

func (t *LifecycleActionTrigger) TriggerEvent() configs.ActionTriggerEvent {
	return t.ActionTriggerEvent
}

func (t *LifecycleActionTrigger) actionTriggerSigil() {}

func (t *LifecycleActionTrigger) String() string {
	return t.TriggeringResourceAddr.String()
}

func (t *LifecycleActionTrigger) Equals(other ActionTrigger) bool {
	o, ok := other.(*LifecycleActionTrigger)
	if !ok {
		return false
	}

	return t.TriggeringResourceAddr.Equal(o.TriggeringResourceAddr) &&
		t.ActionTriggerBlockIndex == o.ActionTriggerBlockIndex &&
		t.ActionsListIndex == o.ActionsListIndex
}

func (t *LifecycleActionTrigger) Less(other ActionTrigger) bool {
	o, ok := other.(*LifecycleActionTrigger)
	if !ok {
		return false // We always want to show non-lifecycle actions first
	}

	return t.TriggeringResourceAddr.Less(o.TriggeringResourceAddr) ||
		(t.TriggeringResourceAddr.Equal(o.TriggeringResourceAddr) &&
			t.ActionTriggerBlockIndex < o.ActionTriggerBlockIndex) ||
		(t.TriggeringResourceAddr.Equal(o.TriggeringResourceAddr) &&
			t.ActionTriggerBlockIndex == o.ActionTriggerBlockIndex &&
			t.ActionsListIndex < o.ActionsListIndex)
}

type InvokeActionTrigger struct{}

func (t *InvokeActionTrigger) actionTriggerSigil() {}

func (t *InvokeActionTrigger) String() string {
	return "CLI"
}

func (t *InvokeActionTrigger) TriggerEvent() configs.ActionTriggerEvent {
	return configs.Invoke
}

func (t *InvokeActionTrigger) Equals(other ActionTrigger) bool {
	_, ok := other.(*InvokeActionTrigger)
	if !ok {
		return false
	}

	return true // InvokeActionTriggers are always considered equal
}

func (t *InvokeActionTrigger) Less(other ActionTrigger) bool {
	// always return true, actions that are equal are already ordered by
	// address externally. these actions should go first anyway.
	return true
}

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

		unmarkedConfigValue, pvms := ai.ConfigValue.UnmarkDeepWithPaths()
		sensitivePaths, otherMarks := marks.PathsWithMark(pvms, marks.Sensitive)
		if len(otherMarks) > 0 {
			return nil, fmt.Errorf("%s: error serializing action invocation with unexpected marks on config value: %#v. This is a bug in Terraform.", tfdiags.FormatCtyPath(otherMarks[0].Path), otherMarks[0].Marks)
		}

		var err error
		ret.ConfigValue, err = NewDynamicValue(unmarkedConfigValue, ty)
		ret.SensitiveConfigPaths = sensitivePaths
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

// PartialActionTrigger is the equivalent of ActionTrigger but allows the
// triggering address to be only partially expanded. This is used during earlier
// phases of planning when (for example) count/for_each expansions are not yet
// fully resolved.
type PartialActionTrigger interface {
	partialActionTriggerSigil()

	TriggerEvent() configs.ActionTriggerEvent

	String() string

	Equals(other PartialActionTrigger) bool
}

// PartialLifecycleActionTrigger is the partial-expanded form of
// LifecycleActionTrigger. It differs only in that it stores a partial-expanded
// resource instance address for the triggering resource.
type PartialLifecycleActionTrigger struct {
	TriggeringResourceAddr  addrs.PartialExpandedResource
	ActionTriggerEvent      configs.ActionTriggerEvent
	ActionTriggerBlockIndex int
	ActionsListIndex        int
}

func (t PartialLifecycleActionTrigger) partialActionTriggerSigil() {}

func (t PartialLifecycleActionTrigger) TriggerEvent() configs.ActionTriggerEvent {
	return t.ActionTriggerEvent
}

func (t PartialLifecycleActionTrigger) String() string {
	return t.TriggeringResourceAddr.String()
}

func (t PartialLifecycleActionTrigger) Equals(other PartialActionTrigger) bool {
	o, ok := other.(*PartialLifecycleActionTrigger)
	if !ok {
		return false
	}
	pomt, tIsPartial := t.TriggeringResourceAddr.PartialExpandedModule()
	pemo, oIsPartial := o.TriggeringResourceAddr.PartialExpandedModule()

	if tIsPartial != oIsPartial {
		return false
	}

	return pomt.MatchesPartial(pemo) && t.TriggeringResourceAddr.Resource().Equal(o.TriggeringResourceAddr.Resource()) &&
		t.ActionTriggerEvent == o.ActionTriggerEvent &&
		t.ActionTriggerBlockIndex == o.ActionTriggerBlockIndex &&
		t.ActionsListIndex == o.ActionsListIndex
}

var _ PartialActionTrigger = (*PartialLifecycleActionTrigger)(nil)

// PartialExpandedActionInvocationInstance mirrors ActionInvocationInstance
// but keeps the action and/or trigger resource addresses in a
// partial-expanded form until all dynamic expansions (count, for_each, etc.)
// are resolved.
type PartialExpandedActionInvocationInstance struct {
	Addr          addrs.PartialExpandedAction
	ActionTrigger PartialActionTrigger
	ProviderAddr  addrs.AbsProviderConfig
	ConfigValue   cty.Value
}

// DeepCopy creates a defensive copy of the partial-expanded invocation.
func (pii *PartialExpandedActionInvocationInstance) DeepCopy() *PartialExpandedActionInvocationInstance {
	if pii == nil {
		return pii
	}
	ret := *pii
	return &ret
}

// Equals compares two partial-expanded invocation instances.
func (pii *PartialExpandedActionInvocationInstance) Equals(other *PartialExpandedActionInvocationInstance) bool {
	if pii == nil || other == nil {
		return pii == other
	}
	// We compare the (partial) action address and the trigger (which may also
	// embed a partial address).
	addrEqual := pii.Addr.Equal(other.Addr)
	triggerEqual := false
	if pii.ActionTrigger == nil && other.ActionTrigger == nil {
		triggerEqual = true
	} else if pii.ActionTrigger != nil && other.ActionTrigger != nil {
		triggerEqual = pii.ActionTrigger.Equals(other.ActionTrigger)
	}
	return addrEqual && triggerEqual
}

type PartialExpandedActionInvocationInstanceSrc struct {
	Addr                 addrs.PartialExpandedAction
	ActionTrigger        PartialActionTrigger
	ProviderAddr         addrs.AbsProviderConfig
	ConfigValue          DynamicValue
	SensitiveConfigPaths []cty.Path
}

// Encode produces a variant of the receiver that has its config value
// serialized so it can be written to a plan file while action and trigger
// addresses are still in their partial-expanded form. Pass the implied type
// of the corresponding action schema for correct operation.
func (pii *PartialExpandedActionInvocationInstance) Encode(schema *providers.ActionSchema) (*PartialExpandedActionInvocationInstanceSrc, error) {
	ret := &PartialExpandedActionInvocationInstanceSrc{
		Addr:          pii.Addr,
		ActionTrigger: pii.ActionTrigger,
		ProviderAddr:  pii.ProviderAddr,
	}

	if pii.ConfigValue != cty.NilVal {
		ty := cty.DynamicPseudoType
		if schema != nil {
			ty = schema.ConfigSchema.ImpliedType()
		}

		unmarkedConfigValue, pvms := pii.ConfigValue.UnmarkDeepWithPaths()
		sensitivePaths, otherMarks := marks.PathsWithMark(pvms, marks.Sensitive)
		if len(otherMarks) > 0 {
			return nil, fmt.Errorf("%s: error serializing partial-expanded action invocation with unexpected marks on config value: %#v. This is a bug in Terraform.", tfdiags.FormatCtyPath(otherMarks[0].Path), otherMarks[0].Marks)
		}

		var err error
		ret.ConfigValue, err = NewDynamicValue(unmarkedConfigValue, ty)
		ret.SensitiveConfigPaths = sensitivePaths
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

// Decode produces an in-memory form of the serialized partial-expanded action
// invocation instance using the provided schema to infer the original config
// value type.
func (src *PartialExpandedActionInvocationInstanceSrc) Decode(schema *providers.ActionSchema) (*PartialExpandedActionInvocationInstance, error) {
	ret := &PartialExpandedActionInvocationInstance{
		Addr:          src.Addr,
		ActionTrigger: src.ActionTrigger,
		ProviderAddr:  src.ProviderAddr,
	}

	if src.ConfigValue != nil {
		ty := cty.DynamicPseudoType
		if schema != nil {
			ty = schema.ConfigSchema.ImpliedType()
		}

		val, err := src.ConfigValue.Decode(ty)
		if err != nil {
			return nil, err
		}

		if len(src.SensitiveConfigPaths) > 0 {
			val = marks.MarkPaths(val, marks.Sensitive, src.SensitiveConfigPaths)
		}

		ret.ConfigValue = val
	}

	return ret, nil
}
