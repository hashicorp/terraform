package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/providers"
)

// Provider is an implementation of providers.Interface
type Provider struct {
	// Provider is the schema for the provider itself.
	Schema providers.Schema

	// DataSources maps the data source name to that data source's schema.
	DataSources map[string]providers.Schema
}

// NewProvider returns a new terraform provider
func NewProvider() *Provider {
	return &Provider{}
}

// GetSchema returns the complete schema for the provider.
func (p *Provider) GetSchema() providers.GetSchemaResponse {
	return providers.GetSchemaResponse{
		DataSources: map[string]providers.Schema{
			"terraform_remote_state": dataSourceRemoteStateGetSchema(),
		},
	}
}

// ValidateProviderConfig is used to validate the configuration values.
func (p *Provider) ValidateProviderConfig(providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
	// At this moment there is nothing to configure for the terraform provider,
	// so we will happily return without taking any action
	var res providers.ValidateProviderConfigResponse
	return res
}

// ValidateDataSourceConfig is used to validate the data source configuration values.
func (p *Provider) ValidateDataSourceConfig(providers.ValidateDataSourceConfigRequest) providers.ValidateDataSourceConfigResponse {
	// FIXME: move the backend configuration validate call that's currently
	// inside the read method  into here so that we can catch provider configuration
	// errors in terraform validate as well as during terraform plan.
	var res providers.ValidateDataSourceConfigResponse
	return res
}

// Configure configures and initializes the provider.
func (p *Provider) Configure(providers.ConfigureRequest) providers.ConfigureResponse {
	// At this moment there is nothing to configure for the terraform provider,
	// so we will happily return without taking any action
	var res providers.ConfigureResponse
	return res
}

// ReadDataSource returns the data source's current state.
func (p *Provider) ReadDataSource(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	// call function
	var res providers.ReadDataSourceResponse

	// This should not happen
	if req.TypeName != "terraform_remote_state" {
		res.Diagnostics.Append(fmt.Errorf("Error: unsupported data source %s", req.TypeName))
		return res
	}

	newState, diags := dataSourceRemoteStateRead(&req.Config)

	res.State = newState
	res.Diagnostics = diags

	return res
}

// Stop is called when the provider should halt any in-flight actions.
func (p *Provider) Stop() error {
	log.Println("[DEBUG] terraform provider cannot Stop")
	return nil
}

// All the Resource-specific functions are below.
// The terraform provider supplies a single data source, `terraform_remote_state`
// and no resources.

// UpgradeResourceState is called when the state loader encounters an
// instance state whose schema version is less than the one reported by the
// currently-used version of the corresponding provider, and the upgraded
// result is used for any further processing.
func (p *Provider) UpgradeResourceState(providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	panic("unimplemented - terraform_remote_state has no resources")
}

// ReadResource refreshes a resource and returns its current state.
func (p *Provider) ReadResource(providers.ReadResourceRequest) providers.ReadResourceResponse {
	panic("unimplemented - terraform_remote_state has no resources")
}

// PlanResourceChange takes the current state and proposed state of a
// resource, and returns the planned final state.
func (p *Provider) PlanResourceChange(providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	panic("unimplemented - terraform_remote_state has no resources")
}

// ApplyResourceChange takes the planned state for a resource, which may
// yet contain unknown computed values, and applies the changes returning
// the final state.
func (p *Provider) ApplyResourceChange(providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	panic("unimplemented - terraform_remote_state has no resources")
}

// ImportResourceState requests that the given resource be imported.
func (p *Provider) ImportResourceState(providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	panic("unimplemented - terraform_remote_state has no resources")
}

// ValidateResourceTypeConfig is used to to validate the resource configuration values.
func (p *Provider) ValidateResourceTypeConfig(providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
	// At this moment there is nothing to configure for the terraform provider,
	// so we will happily return without taking any action
	var res providers.ValidateResourceTypeConfigResponse
	return res
}
