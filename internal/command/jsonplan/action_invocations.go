// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonplan

import (
	"github.com/hashicorp/terraform/internal/plans"
)

type ActionInvocation struct {
	// Address is the absolute action address
	Address string `json:"address,omitempty"`

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

func MarshalActionInvocations(actions []*plans.ActionInvocationInstanceSrc) ([]ActionInvocation, error) {
	ret := make([]ActionInvocation, 0, len(actions))

	for _, action := range actions {

		ai := ActionInvocation{
			Address:      action.Addr.String(),
			ProviderName: action.ProviderAddr.String(),

			// These fields are only used for non-CLI actions. We will need to find another format
			// once we support terraform invoke.
			ActionTriggerBlockIndex:   &action.ActionTriggerBlockIndex,
			ActionsListIndex:          &action.ActionsListIndex,
			TriggeringResourceAddress: action.TriggeringResourceAddr.String(),
			TriggerEvent:              action.TriggerEvent.String(),
		}

		ret = append(ret, ai)
	}

	return ret, nil
}
