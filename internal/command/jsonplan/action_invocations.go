// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonplan

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/zclconf/go-cty/cty"
)

type ActionInvocation struct {
	// Address is the absolute action address
	Address string `json:"address,omitempty"`
	// Type is the type of the action
	Type string `json:"type,omitempty"`
	// Name is the name of the action
	Name string `json:"name,omitempty"`

	// ConfigValues is the JSON representation of the values in the config block of the action
	ConfigValues attributeValues `json:"config_values,omitempty"`

	// ProviderName allows the property "type" to be interpreted unambiguously
	// in the unusual situation where a provider offers a type whose
	// name does not start with its own name, such as the "googlebeta" provider
	// offering "google_compute_instance".
	ProviderName string `json:"provider_name,omitempty"`

	// These fields below are used for actions invoked during plan / apply, they are not applicable
	// for terraform invoke.

	// ActionTriggerBlockIndex is the index of the action trigger block
	ActionTriggerBlockIndex *int `json:"action_trigger_block_index,omitempty"`
	// ActionsListIndex is the index of the action in the actions list
	ActionsListIndex *int `json:"actions_list_index,omitempty"`
	// TriggeringResourceAddress is the address of the resource that triggered the action
	TriggeringResourceAddress string `json:"triggering_resource_address,omitempty"`
	// TriggerEvent is the event that triggered the action
	TriggerEvent string `json:"trigger_event,omitempty"`
}

func ActionInvocationCompare(a, b ActionInvocation) int {
	if a.TriggeringResourceAddress < b.TriggeringResourceAddress {
		return -1
	} else if a.TriggeringResourceAddress > b.TriggeringResourceAddress {
		return 1
	}

	if a.ActionTriggerBlockIndex != nil && b.ActionTriggerBlockIndex != nil {
		if *a.ActionTriggerBlockIndex < *b.ActionTriggerBlockIndex {
			return -1
		} else if *a.ActionTriggerBlockIndex > *b.ActionTriggerBlockIndex {
			return 1
		}
	}

	if a.ActionsListIndex != nil && b.ActionsListIndex != nil {
		if *a.ActionsListIndex < *b.ActionsListIndex {
			return -1

		} else if *a.ActionsListIndex > *b.ActionsListIndex {
			return 1
		}
	}

	return 0
}

func MarshalActionInvocations(actions []*plans.ActionInvocationInstanceSrc, schemas *terraform.Schemas) ([]ActionInvocation, error) {
	ret := make([]ActionInvocation, 0, len(actions))

	for _, action := range actions {

		schema := schemas.ActionTypeConfig(
			action.ProviderAddr.Provider,
			action.Addr.Action.Action.Type,
		)
		if schema.ConfigSchema == nil {
			return ret, fmt.Errorf("no schema found for %s (in provider %s)", action.Addr.Action.Action.Type, action.ProviderAddr.Provider)
		}

		actionDec, err := action.Decode(&schema)
		if err != nil {
			return ret, fmt.Errorf("failed to decode action %s: %w", action.Addr, err)
		}

		ai := ActionInvocation{
			Address:      action.Addr.String(),
			Type:         action.Addr.Action.Action.Type,
			Name:         action.Addr.Action.Action.Name,
			ProviderName: action.ProviderAddr.String(),

			// These fields are only used for non-CLI actions. We will need to find another format
			// once we support terraform invoke.
			ActionTriggerBlockIndex:   &action.ActionTriggerBlockIndex,
			ActionsListIndex:          &action.ActionsListIndex,
			TriggeringResourceAddress: action.TriggeringResourceAddr.String(),
			TriggerEvent:              action.TriggerEvent.String(),
		}

		if actionDec.ConfigValue != cty.NilVal {
			if actionDec.ConfigValue.IsWhollyKnown() {
				ai.ConfigValues = marshalAttributeValues(actionDec.ConfigValue)
			} else {
				knowns := omitUnknowns(actionDec.ConfigValue)
				ai.ConfigValues = marshalAttributeValues(knowns)
			}
		}
		ret = append(ret, ai)
	}

	return ret, nil
}
