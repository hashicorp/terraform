package providers

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// Interface represents the set of methods required for a complete resource
// provider plugin.
type Interface interface {
	// GetSchema returns the complete schema for the provider.
	GetSchema() GetSchemaResponse

	// ValidateProviderConfig allows the provider to validate the configuration
	// values.
	ValidateProviderConfig(ValidateProviderConfigRequest) ValidateProviderConfigResponse

	// ValidateResourceTypeConfig allows the provider to validate the resource
	// configuration values.
	ValidateResourceTypeConfig(ValidateResourceTypeConfigRequest) ValidateResourceTypeConfigResponse

	// ValidateDataSource allows the provider to validate the data source
	// configuration values.
	ValidateDataSourceConfig(ValidateDataSourceConfigRequest) ValidateDataSourceConfigResponse

	// UpgradeResourceState is called when the state loader encounters an
	// instance state whose schema version is less than the one reported by the
	// currently-used version of the corresponding provider, and the upgraded
	// result is used for any further processing.
	UpgradeResourceState(UpgradeResourceStateRequest) UpgradeResourceStateResponse

	// Configure configures and initialized the provider.
	Configure(ConfigureRequest) ConfigureResponse

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

	// ReadDataSource returns the data source's current state.
	ReadDataSource(ReadDataSourceRequest) ReadDataSourceResponse

	// Close shuts down the plugin process if applicable.
	Close() error
}

type GetSchemaResponse struct {
	// Provider is the schema for the provider itself.
	Provider Schema

	// ResourceTypes map the resource type name to that type's schema.
	ResourceTypes map[string]Schema

	// DataSources maps the data source name to that data source's schema.
	DataSources map[string]Schema

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

// Schema pairs a provider or resource schema with that schema's version.
// This is used to be able to upgrade the schema in UpgradeResourceState.
type Schema struct {
	Version uint64
	Block   *configschema.Block
}

type ValidateProviderConfigRequest struct {
	// Config is the complete configuration value for the provider.
	Config cty.Value
}

type ValidateProviderConfigResponse struct {
	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

type ValidateResourceTypeConfigRequest struct {
	// TypeName is the name of the resource type to validate.
	TypeName string

	// Config is the configuration value to validate, which may contain unknown
	// values.
	Config cty.Value
}

type ValidateResourceTypeConfigResponse struct {
	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

type ValidateDataSourceConfigRequest struct {
	// TypeName is the name of the data source type to validate.
	TypeName string

	// Config is the configuration value to validate, which may contain unknown
	// values.
	Config cty.Value
}

type ValidateDataSourceConfigResponse struct {
	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

type UpgradeResourceStateRequest struct {
	// TypeName is the name of the resource type being upgraded
	TypeName string

	// Version is version of the schema that created the current state.
	Version int

	// RawStateJSON and RawStateFlatmap contiain the state that needs to be
	// upgraded to match the current schema version. Because the schema is
	// unknown, this contains only the raw data as stored in the state.
	// RawStateJSON is the current json state encoding.
	// RawStateFlatmap is the legacy flatmap encoding.
	// Only on of these fields may be set for the upgrade request.
	RawStateJSON    []byte
	RawStateFlatmap map[string]string
}

type UpgradeResourceStateResponse struct {
	// UpgradedState is the newly upgraded resource state.
	UpgradedState cty.Value

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

type ConfigureRequest struct {
	// Terraform version is the version string from the running instance of
	// terraform. Providers can use TerraformVersion to verify compatibility,
	// and to store for informational purposes.
	TerraformVersion string

	// Config is the complete configuration value for the provider.
	Config cty.Value
}

type ConfigureResponse struct {
	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

type ReadResourceRequest struct {
	// TypeName is the name of the resource type being read.
	TypeName string

	// PriorState contains the previously saved state value for this resource.
	PriorState cty.Value
}

type ReadResourceResponse struct {
	// NewState contains the current state of the resource.
	NewState cty.Value

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
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
}

type PlanResourceChangeResponse struct {
	// PlannedState is the expected state of the resource once the current
	// configuration is applied.
	PlannedState cty.Value

	// RequiresReplace is the list of thee attributes that are requiring
	// resource replacement.
	RequiresReplace []cty.Path

	// PlannedPrivate is an opaque blob that is not interpreted by terraform
	// core. This will be saved and relayed back to the provider during
	// ApplyResourceChange.
	PlannedPrivate []byte

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
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
}

type ReadDataSourceResponse struct {
	// State is the current state of the requested data source.
	State cty.Value

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}
