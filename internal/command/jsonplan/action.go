// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonplan

import (
	"encoding/json"
)

// ActionInvocation is the representation of an action invocation in the json plan
type ActionInvocation struct {
	// ActionAddress is the absolute action address
	ActionAddress string `json:"action_address,omitempty"`

	// Config is the JSON representation of the attribute values of the
	// action, whose structure depends on the action type schema. Any
	// unknown values are omitted or set to null, making them indistinguishable
	// from absent values.
	Config attributeValues `json:"config,omitempty"`

	// SensitiveConfig is similar to AttributeConfig, but with all sensitive
	// values replaced with true, and all non-sensitive leaf values omitted.
	SensitiveConfig json.RawMessage `json:"sensitive_values,omitempty"`

	// TriggeredBy is set to "cli" if the action was triggered by the CLI, or to "lifecycle" if
	// a lifecycle trigger block was used to trigger the action.
	TriggeredBy string `json:"triggered_by,omitempty"`

	// TriggeringResourceAddress is the absolute resource address of the resource that triggered the action. (only set if TriggeredBy = "lifecycle")
	TriggeringResourceAddress string `json:"triggering_resource_address,omitempty"`

	// TriggeringEvent is the path to the triggering event in the plan. (only set if TriggeredBy = "lifecycle")
	TriggeringEvent string `json:"triggering_event,omitempty"`
}
