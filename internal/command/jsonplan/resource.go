package jsonplan

import (
	"encoding/json"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Resource is the representation of a resource in the json plan
type resource struct {
	// Address is the absolute resource address
	Address string `json:"address,omitempty"`

	// Mode can be "managed" or "data"
	Mode string `json:"mode,omitempty"`

	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`

	// Index is omitted for a resource not using `count` or `for_each`
	Index addrs.InstanceKey `json:"index,omitempty"`

	// ProviderName allows the property "type" to be interpreted unambiguously
	// in the unusual situation where a provider offers a resource type whose
	// name does not start with its own name, such as the "googlebeta" provider
	// offering "google_compute_instance".
	ProviderName string `json:"provider_name,omitempty"`

	// SchemaVersion indicates which version of the resource type schema the
	// "values" property conforms to.
	SchemaVersion uint64 `json:"schema_version"`

	// AttributeValues is the JSON representation of the attribute values of the
	// resource, whose structure depends on the resource type schema. Any
	// unknown values are omitted or set to null, making them indistinguishable
	// from absent values.
	AttributeValues attributeValues `json:"values,omitempty"`

	// SensitiveValues is similar to AttributeValues, but with all sensitive
	// values replaced with true, and all non-sensitive leaf values omitted.
	SensitiveValues json.RawMessage `json:"sensitive_values,omitempty"`
}

// resourceChange is a description of an individual change action that Terraform
// plans to use to move from the prior state to a new state matching the
// configuration.
type resourceChange struct {
	// Address is the absolute resource address
	Address string `json:"address,omitempty"`

	// PreviousAddress is the absolute address that this resource instance had
	// at the conclusion of a previous run.
	//
	// This will typically be omitted, but will be present if the previous
	// resource instance was subject to a "moved" block that we handled in the
	// process of creating this plan.
	//
	// Note that this behavior diverges from the internal plan data structure,
	// where the previous address is set equal to the current address in the
	// common case, rather than being omitted.
	PreviousAddress string `json:"previous_address,omitempty"`

	// ModuleAddress is the module portion of the above address. Omitted if the
	// instance is in the root module.
	ModuleAddress string `json:"module_address,omitempty"`

	// "managed" or "data"
	Mode string `json:"mode,omitempty"`

	Type         string            `json:"type,omitempty"`
	Name         string            `json:"name,omitempty"`
	Index        addrs.InstanceKey `json:"index,omitempty"`
	ProviderName string            `json:"provider_name,omitempty"`

	// "deposed", if set, indicates that this action applies to a "deposed"
	// object of the given instance rather than to its "current" object. Omitted
	// for changes to the current object.
	Deposed string `json:"deposed,omitempty"`

	// Change describes the change that will be made to this object
	Change change `json:"change,omitempty"`

	// ActionReason is a keyword representing some optional extra context
	// for why the actions in Change.Actions were chosen.
	//
	// This extra detail is only for display purposes, to help a UI layer
	// present some additional explanation to a human user. The possible
	// values here might grow and change over time, so any consumer of this
	// information should be resilient to encountering unrecognized values
	// and treat them as an unspecified reason.
	ActionReason string `json:"action_reason,omitempty"`
}
