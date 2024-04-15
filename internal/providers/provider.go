// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Interface represents the set of methods required for a complete resource
// provider plugin.
type Interface interface {
	// GetSchema returns the complete schema for the provider.
	GetProviderSchema() GetProviderSchemaResponse

	// ValidateProviderConfig allows the provider to validate the configuration.
	// The ValidateProviderConfigResponse.PreparedConfig field is unused. The
	// final configuration is not stored in the state, and any modifications
	// that need to be made must be made during the Configure method call.
	ValidateProviderConfig(ValidateProviderConfigRequest) ValidateProviderConfigResponse

	// ValidateResourceConfig allows the provider to validate the resource
	// configuration values.
	ValidateResourceConfig(ValidateResourceConfigRequest) ValidateResourceConfigResponse

	// ValidateDataResourceConfig allows the provider to validate the data source
	// configuration values.
	ValidateDataResourceConfig(ValidateDataResourceConfigRequest) ValidateDataResourceConfigResponse

	// UpgradeResourceState is called when the state loader encounters an
	// instance state whose schema version is less than the one reported by the
	// currently-used version of the corresponding provider, and the upgraded
	// result is used for any further processing.
	UpgradeResourceState(UpgradeResourceStateRequest) UpgradeResourceStateResponse

	// Configure configures and initialized the provider.
	ConfigureProvider(ConfigureProviderRequest) ConfigureProviderResponse

	// Stop is called when the provider should halt any in-flight actions.
	//
	// Stop should not block waiting for in-flight actions to complete. It
	// should take any action it wants and return immediately acknowledging it
	// has received the stop request. Terraform will not make any further API
	// calls to the provider after Stop is called.
	//
	// The error returned, if non-nil, is assumed to mean that signaling the
	// stop somehow failed and that the user should expect potentially waiting
	// a longer period of time.
	Stop() error

	// ReadResource refreshes a resource and returns its current state.
	ReadResource(ReadResourceRequest) ReadResourceResponse

	// PlanResourceChange takes the current state and proposed state of a
	// resource, and returns the planned final state.
	PlanResourceChange(PlanResourceChangeRequest) PlanResourceChangeResponse

	// ApplyResourceChange takes the planned state for a resource, which may
	// yet contain unknown computed values, and applies the changes returning
	// the final state.
	ApplyResourceChange(ApplyResourceChangeRequest) ApplyResourceChangeResponse

	// ImportResourceState requests that the given resource be imported.
	ImportResourceState(ImportResourceStateRequest) ImportResourceStateResponse

	// MoveResourceState retrieves the updated value for a resource after it
	// has moved resource types.
	MoveResourceState(MoveResourceStateRequest) MoveResourceStateResponse

	// ReadDataSource returns the data source's current state.
	ReadDataSource(ReadDataSourceRequest) ReadDataSourceResponse

	// CallFunction calls a provider-contributed function.
	CallFunction(CallFunctionRequest) CallFunctionResponse

	// Close shuts down the plugin process if applicable.
	Close() error
}

// GetProviderSchemaResponse is the return type for GetProviderSchema, and
// should only be used when handling a value for that method. The handling of
// of schemas in any other context should always use ProviderSchema, so that
// the in-memory representation can be more easily changed separately from the
// RCP protocol.
type GetProviderSchemaResponse struct {
	// Provider is the schema for the provider itself.
	Provider Schema

	// ProviderMeta is the schema for the provider's meta info in a module
	ProviderMeta Schema

	// ResourceTypes map the resource type name to that type's schema.
	ResourceTypes map[string]Schema

	// DataSources maps the data source name to that data source's schema.
	DataSources map[string]Schema

	// Functions maps from local function name (not including an namespace
	// prefix) to the declaration of a function.
	Functions map[string]FunctionDecl

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics

	// ServerCapabilities lists optional features supported by the provider.
	ServerCapabilities ServerCapabilities
}

// Schema pairs a provider or resource schema with that schema's version.
// This is used to be able to upgrade the schema in UpgradeResourceState.
//
// This describes the schema for a single object within a provider. Type
// "Schemas" (plural) instead represents the overall collection of schemas
// for everything within a particular provider.
type Schema struct {
	Version int64
	Block   *configschema.Block
}

// ServerCapabilities allows providers to communicate extra information
// regarding supported protocol features. This is used to indicate availability
// of certain forward-compatible changes which may be optional in a major
// protocol version, but cannot be tested for directly.
type ServerCapabilities struct {
	// PlanDestroy signals that this provider expects to receive a
	// PlanResourceChange call for resources that are to be destroyed.
	PlanDestroy bool

	// The GetProviderSchemaOptional capability indicates that this
	// provider does not require calling GetProviderSchema to operate
	// normally, and the caller can used a cached copy of the provider's
	// schema.
	GetProviderSchemaOptional bool

	// The MoveResourceState capability indicates that this provider supports
	// the MoveResourceState RPC.
	MoveResourceState bool
}

type ValidateProviderConfigRequest struct {
	// Config is the raw configuration value for the provider.
	Config cty.Value
}

type ValidateProviderConfigResponse struct {
	// PreparedConfig is unused and will be removed with support for plugin protocol v5.
	PreparedConfig cty.Value
	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

type ValidateResourceConfigRequest struct {
	// TypeName is the name of the resource type to validate.
	TypeName string

	// Config is the configuration value to validate, which may contain unknown
	// values.
	Config cty.Value
}

type ValidateResourceConfigResponse struct {
	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

type ValidateDataResourceConfigRequest struct {
	// TypeName is the name of the data source type to validate.
	TypeName string

	// Config is the configuration value to validate, which may contain unknown
	// values.
	Config cty.Value
}

type ValidateDataResourceConfigResponse struct {
	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

type UpgradeResourceStateRequest struct {
	// TypeName is the name of the resource type being upgraded
	TypeName string

	// Version is version of the schema that created the current state.
	Version int64

	// RawStateJSON and RawStateFlatmap contain the state that needs to be
	// upgraded to match the current schema version. Because the schema is
	// unknown, this contains only the raw data as stored in the state.
	// RawStateJSON is the current json state encoding.
	// RawStateFlatmap is the legacy flatmap encoding.
	// Only one of these fields may be set for the upgrade request.
	RawStateJSON    []byte
	RawStateFlatmap map[string]string
}

type UpgradeResourceStateResponse struct {
	// UpgradedState is the newly upgraded resource state.
	UpgradedState cty.Value

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

type ConfigureProviderRequest struct {
	// Terraform version is the version string from the running instance of
	// terraform. Providers can use TerraformVersion to verify compatibility,
	// and to store for informational purposes.
	TerraformVersion string

	// Config is the complete configuration value for the provider.
	Config cty.Value
}

type ConfigureProviderResponse struct {
	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

type ReadResourceRequest struct {
	// TypeName is the name of the resource type being read.
	TypeName string

	// PriorState contains the previously saved state value for this resource.
	PriorState cty.Value

	// Private is an opaque blob that will be stored in state along with the
	// resource. It is intended only for interpretation by the provider itself.
	Private []byte

	// ProviderMeta is the configuration for the provider_meta block for the
	// module and provider this resource belongs to. Its use is defined by
	// each provider, and it should not be used without coordination with
	// HashiCorp. It is considered experimental and subject to change.
	ProviderMeta cty.Value

	// DeferralAllowed signals that the provider is allowed to defer the
	// changes. If set the caller needs to handle the deferred response.
	DeferralAllowed bool
}

// DeferredReason is a string that describes why a resource was deferred.
// It differs from the protobuf enum in that it adds more cases
// since it's more widely used to represent the reason for deferral.
// Reasons like instance count unknown and deferred prereq are not
// relevant for providers but can occur in general.
type DeferredReason string

const (
	// DeferredReasonInvalid is used when the reason for deferring is
	// unknown or irrelevant.
	DeferredReasonInvalid DeferredReason = "invalid"

	// DeferredReasonInstanceCountUnknown is used when the reason for deferring
	// is that the count or for_each meta-attribute was unknown.
	DeferredReasonInstanceCountUnknown DeferredReason = "instance_count_unknown"

	// DeferredReasonResourceConfigUnknown is used when the reason for deferring
	// is that the resource configuration was unknown.
	DeferredReasonResourceConfigUnknown DeferredReason = "resource_config_unknown"

	// DeferredReasonProviderConfigUnknown is used when the reason for deferring
	// is that the provider configuration was unknown.
	DeferredReasonProviderConfigUnknown DeferredReason = "provider_config_unknown"

	// DeferredReasonAbsentPrereq is used when the reason for deferring is that
	// a required prerequisite resource was absent.
	DeferredReasonAbsentPrereq DeferredReason = "absent_prereq"

	// DeferredReasonDeferredPrereq is used when the reason for deferring is
	// that a required prerequisite resource was itself deferred.
	DeferredReasonDeferredPrereq DeferredReason = "deferred_prereq"
)

type Deferred struct {
	Reason DeferredReason
}

type ReadResourceResponse struct {
	// NewState contains the current state of the resource.
	NewState cty.Value

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics

	// Private is an opaque blob that will be stored in state along with the
	// resource. It is intended only for interpretation by the provider itself.
	Private []byte

	// Deferred if present signals that the provider was not able to fully
	// complete this operation and a susequent run is required.
	Deferred *Deferred
}

type PlanResourceChangeRequest struct {
	// TypeName is the name of the resource type to plan.
	TypeName string

	// PriorState is the previously saved state value for this resource.
	PriorState cty.Value

	// ProposedNewState is the expected state after the new configuration is
	// applied. This is created by directly applying the configuration to the
	// PriorState. The provider is then responsible for applying any further
	// changes required to create the proposed final state.
	ProposedNewState cty.Value

	// Config is the resource configuration, before being merged with the
	// PriorState. Any value not explicitly set in the configuration will be
	// null. Config is supplied for reference, but Provider implementations
	// should prefer the ProposedNewState in most circumstances.
	Config cty.Value

	// PriorPrivate is the previously saved private data returned from the
	// provider during the last apply.
	PriorPrivate []byte

	// ProviderMeta is the configuration for the provider_meta block for the
	// module and provider this resource belongs to. Its use is defined by
	// each provider, and it should not be used without coordination with
	// HashiCorp. It is considered experimental and subject to change.
	ProviderMeta cty.Value
}

type PlanResourceChangeResponse struct {
	// PlannedState is the expected state of the resource once the current
	// configuration is applied.
	PlannedState cty.Value

	// RequiresReplace is the list of the attributes that are requiring
	// resource replacement.
	RequiresReplace []cty.Path

	// PlannedPrivate is an opaque blob that is not interpreted by terraform
	// core. This will be saved and relayed back to the provider during
	// ApplyResourceChange.
	PlannedPrivate []byte

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics

	// LegacyTypeSystem is set only if the provider is using the legacy SDK
	// whose type system cannot be precisely mapped into the Terraform type
	// system. We use this to bypass certain consistency checks that would
	// otherwise fail due to this imprecise mapping. No other provider or SDK
	// implementation is permitted to set this.
	LegacyTypeSystem bool
}

type ApplyResourceChangeRequest struct {
	// TypeName is the name of the resource type being applied.
	TypeName string

	// PriorState is the current state of resource.
	PriorState cty.Value

	// Planned state is the state returned from PlanResourceChange, and should
	// represent the new state, minus any remaining computed attributes.
	PlannedState cty.Value

	// Config is the resource configuration, before being merged with the
	// PriorState. Any value not explicitly set in the configuration will be
	// null. Config is supplied for reference, but Provider implementations
	// should prefer the PlannedState in most circumstances.
	Config cty.Value

	// PlannedPrivate is the same value as returned by PlanResourceChange.
	PlannedPrivate []byte

	// ProviderMeta is the configuration for the provider_meta block for the
	// module and provider this resource belongs to. Its use is defined by
	// each provider, and it should not be used without coordination with
	// HashiCorp. It is considered experimental and subject to change.
	ProviderMeta cty.Value
}

type ApplyResourceChangeResponse struct {
	// NewState is the new complete state after applying the planned change.
	// In the event of an error, NewState should represent the most recent
	// known state of the resource, if it exists.
	NewState cty.Value

	// Private is an opaque blob that will be stored in state along with the
	// resource. It is intended only for interpretation by the provider itself.
	Private []byte

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics

	// LegacyTypeSystem is set only if the provider is using the legacy SDK
	// whose type system cannot be precisely mapped into the Terraform type
	// system. We use this to bypass certain consistency checks that would
	// otherwise fail due to this imprecise mapping. No other provider or SDK
	// implementation is permitted to set this.
	LegacyTypeSystem bool
}

type ImportResourceStateRequest struct {
	// TypeName is the name of the resource type to be imported.
	TypeName string

	// ID is a string with which the provider can identify the resource to be
	// imported.
	ID string
}

type ImportResourceStateResponse struct {
	// ImportedResources contains one or more state values related to the
	// imported resource. It is not required that these be complete, only that
	// there is enough identifying information for the provider to successfully
	// update the states in ReadResource.
	ImportedResources []ImportedResource

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

// ImportedResource represents an object being imported into Terraform with the
// help of a provider. An ImportedObject is a RemoteObject that has been read
// by the provider's import handler but hasn't yet been committed to state.
type ImportedResource struct {
	// TypeName is the name of the resource type associated with the
	// returned state. It's possible for providers to import multiple related
	// types with a single import request.
	TypeName string

	// State is the state of the remote object being imported. This may not be
	// complete, but must contain enough information to uniquely identify the
	// resource.
	State cty.Value

	// Private is an opaque blob that will be stored in state along with the
	// resource. It is intended only for interpretation by the provider itself.
	Private []byte
}

type MoveResourceStateRequest struct {
	// SourceProviderAddress is the address of the provider that the resource
	// is being moved from.
	SourceProviderAddress string

	// SourceTypeName is the name of the resource type that the resource is
	// being moved from.
	SourceTypeName string

	// SourceSchemaVersion is the schema version of the resource type that the
	// resource is being moved from.
	SourceSchemaVersion int64

	// SourceStateJSON contains the state of the resource that is being moved.
	// Because the schema is unknown, this contains only the raw data as stored
	// in the state.
	SourceStateJSON []byte

	// SourcePrivate contains the private state of the resource that is being
	// moved.
	SourcePrivate []byte

	// TargetTypeName is the name of the resource type that the resource is
	// being moved to.
	TargetTypeName string
}

type MoveResourceStateResponse struct {
	// TargetState is the state of the resource after it has been moved to the
	// new resource type.
	TargetState cty.Value

	// TargetPrivate is the private state of the resource after it has been
	// moved to the new resource type.
	TargetPrivate []byte

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

// AsInstanceObject converts the receiving ImportedObject into a
// ResourceInstanceObject that has status ObjectReady.
//
// The returned object does not know its own resource type, so the caller must
// retain the ResourceType value from the source object if this information is
// needed.
//
// The returned object also has no dependency addresses, but the caller may
// freely modify the direct fields of the returned object without affecting
// the receiver.
func (ir ImportedResource) AsInstanceObject() *states.ResourceInstanceObject {
	return &states.ResourceInstanceObject{
		Status:  states.ObjectReady,
		Value:   ir.State,
		Private: ir.Private,
	}
}

type ReadDataSourceRequest struct {
	// TypeName is the name of the data source type to Read.
	TypeName string

	// Config is the complete configuration for the requested data source.
	Config cty.Value

	// ProviderMeta is the configuration for the provider_meta block for the
	// module and provider this resource belongs to. Its use is defined by
	// each provider, and it should not be used without coordination with
	// HashiCorp. It is considered experimental and subject to change.
	ProviderMeta cty.Value

	// DeferralAllowed signals that the provider is allowed to defer the
	// changes. If set the caller needs to handle the deferred response.
	DeferralAllowed bool
}

type ReadDataSourceResponse struct {
	// State is the current state of the requested data source.
	State cty.Value

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics

	// Deferred if present signals that the provider was not able to fully
	// complete this operation and a susequent run is required.
	Deferred *Deferred
}

type CallFunctionRequest struct {
	// FunctionName is the local name of the function to call, as it was
	// declared by the provider in its schema and without any
	// externally-imposed namespace prefixes.
	FunctionName string

	// Arguments are the positional argument values given at the call site.
	//
	// Provider functions are required to behave as pure functions, and so
	// if all of the argument values are known then two separate calls with the
	// same arguments must always return an identical value, without performing
	// any externally-visible side-effects.
	Arguments []cty.Value
}

type CallFunctionResponse struct {
	// Result is the successful result of the function call.
	//
	// If all of the arguments in the call were known then the result must
	// also be known. If any arguments were unknown then the result may
	// optionally be unknown. The type of the returned value must conform
	// to the return type constraint for this function as declared in the
	// provider schema.
	//
	// If Diagnostics contains any errors, this field will be ignored and
	// so can be left as cty.NilVal to represent the absense of a value.
	Result cty.Value

	// Err is the error value from the function call. This may be an instance
	// of function.ArgError from the go-cty package to specify a problem with a
	// specific argument.
	Err error
}
