// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mnptu

import (
	"fmt"
	"log"

	"github.com/hashicorp/mnptu/internal/providers"
)

// Provider is an implementation of providers.Interface
type Provider struct{}

// NewProvider returns a new mnptu provider
func NewProvider() providers.Interface {
	return &Provider{}
}

// GetSchema returns the complete schema for the provider.
func (p *Provider) GetProviderSchema() providers.GetProviderSchemaResponse {
	return providers.GetProviderSchemaResponse{
		DataSources: map[string]providers.Schema{
			"mnptu_remote_state": dataSourceRemoteStateGetSchema(),
		},
		ResourceTypes: map[string]providers.Schema{
			"mnptu_data": dataStoreResourceSchema(),
		},
	}
}

// ValidateProviderConfig is used to validate the configuration values.
func (p *Provider) ValidateProviderConfig(req providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
	// At this moment there is nothing to configure for the mnptu provider,
	// so we will happily return without taking any action
	var res providers.ValidateProviderConfigResponse
	res.PreparedConfig = req.Config
	return res
}

// ValidateDataResourceConfig is used to validate the data source configuration values.
func (p *Provider) ValidateDataResourceConfig(req providers.ValidateDataResourceConfigRequest) providers.ValidateDataResourceConfigResponse {
	// FIXME: move the backend configuration validate call that's currently
	// inside the read method  into here so that we can catch provider configuration
	// errors in mnptu validate as well as during mnptu plan.
	var res providers.ValidateDataResourceConfigResponse

	// This should not happen
	if req.TypeName != "mnptu_remote_state" {
		res.Diagnostics.Append(fmt.Errorf("Error: unsupported data source %s", req.TypeName))
		return res
	}

	diags := dataSourceRemoteStateValidate(req.Config)
	res.Diagnostics = diags

	return res
}

// Configure configures and initializes the provider.
func (p *Provider) ConfigureProvider(providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	// At this moment there is nothing to configure for the mnptu provider,
	// so we will happily return without taking any action
	var res providers.ConfigureProviderResponse
	return res
}

// ReadDataSource returns the data source's current state.
func (p *Provider) ReadDataSource(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	// call function
	var res providers.ReadDataSourceResponse

	// This should not happen
	if req.TypeName != "mnptu_remote_state" {
		res.Diagnostics.Append(fmt.Errorf("Error: unsupported data source %s", req.TypeName))
		return res
	}

	newState, diags := dataSourceRemoteStateRead(req.Config)

	res.State = newState
	res.Diagnostics = diags

	return res
}

// Stop is called when the provider should halt any in-flight actions.
func (p *Provider) Stop() error {
	log.Println("[DEBUG] mnptu provider cannot Stop")
	return nil
}

// All the Resource-specific functions are below.
// The mnptu provider supplies a single data source, `mnptu_remote_state`
// and no resources.

// UpgradeResourceState is called when the state loader encounters an
// instance state whose schema version is less than the one reported by the
// currently-used version of the corresponding provider, and the upgraded
// result is used for any further processing.
func (p *Provider) UpgradeResourceState(req providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	return upgradeDataStoreResourceState(req)
}

// ReadResource refreshes a resource and returns its current state.
func (p *Provider) ReadResource(req providers.ReadResourceRequest) providers.ReadResourceResponse {
	return readDataStoreResourceState(req)
}

// PlanResourceChange takes the current state and proposed state of a
// resource, and returns the planned final state.
func (p *Provider) PlanResourceChange(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	return planDataStoreResourceChange(req)
}

// ApplyResourceChange takes the planned state for a resource, which may
// yet contain unknown computed values, and applies the changes returning
// the final state.
func (p *Provider) ApplyResourceChange(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	return applyDataStoreResourceChange(req)
}

// ImportResourceState requests that the given resource be imported.
func (p *Provider) ImportResourceState(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	if req.TypeName == "mnptu_data" {
		return importDataStore(req)
	}

	panic("unimplemented - mnptu_remote_state has no resources")
}

// ValidateResourceConfig is used to to validate the resource configuration values.
func (p *Provider) ValidateResourceConfig(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
	return validateDataStoreResourceConfig(req)
}

// Close is a noop for this provider, since it's run in-process.
func (p *Provider) Close() error {
	return nil
}
