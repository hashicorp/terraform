// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonplan

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

type ActionInvocation struct {
	// Address is the absolute action address
	Address string `json:"address,omitempty"`
	// Type is the type of the action
	Type string `json:"type,omitempty"`
	// Name is the name of the action
	Name string `json:"name,omitempty"`

	// ConfigValues is the JSON representation of the values in the config block of the action
	ConfigValues    map[string]json.RawMessage `json:"config_values,omitempty"`
	ConfigSensitive json.RawMessage            `json:"config_sensitive,omitempty"`

	// ProviderName allows the property "type" to be interpreted unambiguously
	// in the unusual situation where a provider offers a type whose
	// name does not start with its own name, such as the "googlebeta" provider
	// offering "google_compute_instance".
	ProviderName string `json:"provider_name,omitempty"`

	LifecycleActionTrigger *LifecycleActionTrigger `json:"lifecycle_action_trigger,omitempty"`
	InvokeCmdActionTrigger *InvokeCmdActionTrigger `json:"invoke_cmd_action_trigger,omitempty"`
}

type LifecycleActionTrigger struct {
	TriggeringResourceAddress string `json:"triggering_resource_address,omitempty"`
	ActionTriggerEvent        string `json:"action_trigger_event,omitempty"`
	ActionTriggerBlockIndex   int    `json:"action_trigger_block_index"`
	ActionsListIndex          int    `json:"actions_list_index"`
}

type InvokeCmdActionTrigger struct {
	ActionTriggerEvent string `json:"action_trigger_event,omitempty"`
}

func ActionInvocationCompare(a, b ActionInvocation) int {
	if a.LifecycleActionTrigger != nil && b.LifecycleActionTrigger != nil {
		latA := *a.LifecycleActionTrigger
		latB := *b.LifecycleActionTrigger

		if latA.TriggeringResourceAddress < latB.TriggeringResourceAddress {
			return -1
		} else if latA.TriggeringResourceAddress > latB.TriggeringResourceAddress {
			return 1
		}

		if latA.ActionTriggerBlockIndex < latB.ActionTriggerBlockIndex {
			return -1
		} else if latA.ActionTriggerBlockIndex > latB.ActionTriggerBlockIndex {
			return 1
		}

		if latA.ActionsListIndex < latB.ActionsListIndex {
			return -1

		} else if latA.ActionsListIndex > latB.ActionsListIndex {
			return 1
		}
	}

	return 0
}

func marshalConfigValues(value cty.Value) map[string]json.RawMessage {
	// unmark our value to show all values
	v, _ := value.UnmarkDeep()

	if v == cty.NilVal || v.IsNull() {
		return nil
	}

	ret := make(map[string]json.RawMessage)
	it := v.ElementIterator()
	for it.Next() {
		k, v := it.Element()
		vJSON, _ := ctyjson.Marshal(v, v.Type())
		ret[k.AsString()] = json.RawMessage(vJSON)
	}
	return ret
}

func MarshalActionInvocations(actions []*plans.ActionInvocationInstanceSrc, schemas *terraform.Schemas) ([]ActionInvocation, error) {
	ret := make([]ActionInvocation, 0, len(actions))
	for _, action := range actions {
		ai, err := MarshalActionInvocation(action, schemas)
		if err != nil {
			return ret, fmt.Errorf("failed to decode action %s: %w", action.Addr, err)
		}
		ret = append(ret, ai)
	}
	return ret, nil
}

func MarshalActionInvocation(action *plans.ActionInvocationInstanceSrc, schemas *terraform.Schemas) (ActionInvocation, error) {
	ai := ActionInvocation{
		Address:      action.Addr.String(),
		Type:         action.Addr.Action.Action.Type,
		Name:         action.Addr.Action.Action.Name,
		ProviderName: action.ProviderAddr.Provider.String(),
	}
	schema := schemas.ActionTypeConfig(
		action.ProviderAddr.Provider,
		action.Addr.Action.Action.Type,
	)
	if schema.ConfigSchema == nil {
		return ai, fmt.Errorf("no schema found for %s (in provider %s)", action.Addr.Action.Action.Type, action.ProviderAddr.Provider)
	}

	actionDec, err := action.Decode(&schema)
	if err != nil {
		return ai, fmt.Errorf("failed to decode action %s: %w", action.Addr, err)
	}

	switch at := action.ActionTrigger.(type) {
	case plans.LifecycleActionTrigger:
		ai.LifecycleActionTrigger = &LifecycleActionTrigger{
			TriggeringResourceAddress: at.TriggeringResourceAddr.String(),
			ActionTriggerEvent:        at.TriggerEvent().String(),
			ActionTriggerBlockIndex:   at.ActionTriggerBlockIndex,
			ActionsListIndex:          at.ActionsListIndex,
		}
	default:
		return ai, fmt.Errorf("unsupported action trigger type: %T", at)
	}

	if actionDec.ConfigValue != cty.NilVal {
		_, pvms := actionDec.ConfigValue.UnmarkDeepWithPaths()
		sensitivePaths, otherMarks := marks.PathsWithMark(pvms, marks.Sensitive)
		ephemeralPaths, otherMarks := marks.PathsWithMark(otherMarks, marks.Ephemeral)
		if len(ephemeralPaths) > 0 {
			return ai, fmt.Errorf("action %s has ephemeral config values, which are not supported in action invocations", action.Addr)
		}
		if len(otherMarks) > 0 {
			return ai, fmt.Errorf("action %s has config values with unsupported marks: %v", action.Addr, otherMarks)
		}

		configValue := actionDec.ConfigValue
		if !configValue.IsWhollyKnown() {
			configValue = omitUnknowns(actionDec.ConfigValue)
		}
		cs := jsonstate.SensitiveAsBool(marks.MarkPaths(configValue, marks.Sensitive, sensitivePaths))
		configSensitive, err := ctyjson.Marshal(cs, cs.Type())
		if err != nil {
			return ai, err
		}

		ai.ConfigValues = marshalConfigValues(configValue)
		ai.ConfigSensitive = configSensitive
	}
	return ai, nil
}

// DeferredActionInvocation is a description of an action invocation that has been
// deferred for some reason.
type DeferredActionInvocation struct {
	// Reason is the reason why this action was deferred.
	Reason string `json:"reason"`

	// Change contains any information we have about the deferred change.
	ActionInvocation ActionInvocation `json:"action_invocation"`
}

func MarshalDeferredActionInvocations(dais []*plans.DeferredActionInvocationSrc, schemas *terraform.Schemas) ([]DeferredActionInvocation, error) {
	var deferredInvocations []DeferredActionInvocation

	sortedActions := append([]*plans.DeferredActionInvocationSrc{}, dais...)
	sort.Slice(sortedActions, func(i, j int) bool {
		return sortedActions[i].ActionInvocationInstanceSrc.Less(sortedActions[j].ActionInvocationInstanceSrc)
	})

	for _, daiSrc := range dais {
		ai, err := MarshalActionInvocation(daiSrc.ActionInvocationInstanceSrc, schemas)
		if err != nil {
			return nil, err
		}

		dai := DeferredActionInvocation{
			ActionInvocation: ai,
		}
		switch daiSrc.DeferredReason {
		case providers.DeferredReasonInstanceCountUnknown:
			dai.Reason = DeferredReasonInstanceCountUnknown
		case providers.DeferredReasonResourceConfigUnknown:
			dai.Reason = DeferredReasonResourceConfigUnknown
		case providers.DeferredReasonProviderConfigUnknown:
			dai.Reason = DeferredReasonProviderConfigUnknown
		case providers.DeferredReasonAbsentPrereq:
			dai.Reason = DeferredReasonAbsentPrereq
		case providers.DeferredReasonDeferredPrereq:
			dai.Reason = DeferredReasonDeferredPrereq
		default:
			// If we find a reason we don't know about, we'll just mark it as
			// unknown. This is a bit of a safety net to ensure that we don't
			// break if new reasons are introduced in future versions of the
			// provider protocol.
			dai.Reason = DeferredReasonUnknown
		}

		deferredInvocations = append(deferredInvocations, dai)
	}
	return deferredInvocations, nil
}
