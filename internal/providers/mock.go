// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var _ Interface = (*Mock)(nil)

// Mock is a mock provider that can be used by Terraform authors during test
// executions.
//
// The mock provider wraps an instance of an actual provider so it can return
// the correct schema and validate the configuration accurately. But, it
// intercepts calls to create resources or read data sources and instead reads
// and write the data to/from the state directly instead of needing to
// communicate with actual cloud providers.
//
// Callers can also specify the configs.MockData field to provide some preset
// data to return for any computed fields within the provider schema. The
// provider will make up random / junk data for any computed fields for which
// preset data is not available.
//
// This is distinct from the testing.MockProvider, which is a mock provider
// that is used by the Terraform core itself to test it's own behavior.
type Mock struct {
	Provider Interface
	Data     *configs.MockData

	schema *GetProviderSchemaResponse
}

func (m *Mock) GetProviderSchema() GetProviderSchemaResponse {
	if m.schema == nil {
		// Cache the schema, it's not changing.
		schema := m.Provider.GetProviderSchema()

		// Override the provider schema with the constant mock provider schema.
		// This is empty at the moment, check configs/mock_provider.go for the
		// actual schema.
		//
		// The GetProviderSchemaResponse is returned by value, so it should be
		// safe for us to modify directly, without affecting any shared state
		// that could be in use elsewhere.
		schema.Provider = Schema{
			Version: schema.Provider.Version,
			Block:   nil, // Empty - we support no blocks or attributes in mock provider configurations.
		}

		// Note, we leave the resource and data source schemas as they are since
		// we want to be able to validate those configurations against the real
		// provider schemas.

		m.schema = &schema
	}
	return *m.schema
}

func (m *Mock) ValidateProviderConfig(request ValidateProviderConfigRequest) (response ValidateProviderConfigResponse) {
	// The config for the mocked providers is consistent, and validated when we
	// parse the HCL directly. So we'll just make no change here.
	return ValidateProviderConfigResponse{
		PreparedConfig: request.Config,
	}
}

func (m *Mock) ValidateResourceConfig(request ValidateResourceConfigRequest) ValidateResourceConfigResponse {
	// We'll just pass this through to the underlying provider. The mock should
	// support the same resource syntax as the original provider and we can call
	// validate without needing to configure the provider first.
	return m.Provider.ValidateResourceConfig(request)
}

func (m *Mock) ValidateDataResourceConfig(request ValidateDataResourceConfigRequest) ValidateDataResourceConfigResponse {
	// We'll just pass this through to the underlying provider. The mock should
	// support the same data source syntax as the original provider and we can
	// call validate without needing to configure the provider first.
	return m.Provider.ValidateDataResourceConfig(request)
}

func (m *Mock) UpgradeResourceState(request UpgradeResourceStateRequest) (response UpgradeResourceStateResponse) {
	// We can't do this from a mocked provider, so we just return whatever state
	// is in the request back unchanged.

	schema := m.GetProviderSchema()
	response.Diagnostics = response.Diagnostics.Append(schema.Diagnostics)
	if schema.Diagnostics.HasErrors() {
		// We couldn't retrieve the schema for some reason, so the mock
		// provider can't really function.
		return response
	}

	resource, exists := schema.ResourceTypes[request.TypeName]
	if !exists {
		// This means something has gone wrong much earlier, we should have
		// failed a validation somewhere if a resource type doesn't exist.
		panic(fmt.Errorf("failed to retrieve schema for resource %s", request.TypeName))
	}

	schemaType := resource.Block.ImpliedType()

	var value cty.Value
	var err error

	switch {
	case request.RawStateFlatmap != nil:
		value, err = hcl2shim.HCL2ValueFromFlatmap(request.RawStateFlatmap, schemaType)
	case len(request.RawStateJSON) > 0:
		value, err = ctyjson.Unmarshal(request.RawStateJSON, schemaType)
	}

	if err != nil {
		// Generally, we shouldn't get an error here. The mocked providers are
		// only used in tests, and we can't use different versions of providers
		// within/between tests so the types should always match up. As such,
		// we're not gonna return a super detailed error here.
		response.Diagnostics = response.Diagnostics.Append(err)
		return response
	}
	response.UpgradedState = value
	return response
}

func (m *Mock) ConfigureProvider(request ConfigureProviderRequest) (response ConfigureProviderResponse) {
	// Do nothing here, we don't have anything to configure within the mocked
	// providers. We don't want to call the original providers from here as
	// they may try to talk to their underlying cloud providers and we
	// definitely don't have the right configuration or credentials for this.
	return response
}

func (m *Mock) Stop() error {
	// Just stop the original resource.
	return m.Provider.Stop()
}

func (m *Mock) ReadResource(request ReadResourceRequest) ReadResourceResponse {
	// For a mocked provider, reading a resource is just reading it from the
	// state. So we'll return what we have.
	return ReadResourceResponse{
		NewState: request.PriorState,
	}
}

func (m *Mock) PlanResourceChange(request PlanResourceChangeRequest) PlanResourceChangeResponse {
	if request.ProposedNewState.IsNull() {
		// Then we are deleting this resource - we don't need to do anything.
		return PlanResourceChangeResponse{
			PlannedState:   request.ProposedNewState,
			PlannedPrivate: []byte("destroy"),
		}
	}

	if request.PriorState.IsNull() {
		// Then we are creating this resource - we need to populate the computed
		// null fields with unknowns so Terraform will render them properly.

		var response PlanResourceChangeResponse

		schema := m.GetProviderSchema()
		response.Diagnostics = response.Diagnostics.Append(schema.Diagnostics)
		if schema.Diagnostics.HasErrors() {
			// We couldn't retrieve the schema for some reason, so the mock
			// provider can't really function.
			return response
		}

		resource, exists := schema.ResourceTypes[request.TypeName]
		if !exists {
			// This means something has gone wrong much earlier, we should have
			// failed a validation somewhere if a resource type doesn't exist.
			panic(fmt.Errorf("failed to retrieve schema for resource %s", request.TypeName))
		}

		value, diags := mocking.PlanComputedValuesForResource(request.ProposedNewState, resource.Block)
		response.Diagnostics = response.Diagnostics.Append(diags)
		response.PlannedState = value
		response.PlannedPrivate = []byte("create")
		return response
	}

	// Otherwise, we're just doing a simple update and we don't need to populate
	// any values for that.
	return PlanResourceChangeResponse{
		PlannedState:   request.ProposedNewState,
		PlannedPrivate: []byte("update"),
	}
}

func (m *Mock) ApplyResourceChange(request ApplyResourceChangeRequest) ApplyResourceChangeResponse {
	switch string(request.PlannedPrivate) {
	case "create":
		// A new resource that we've created might have computed fields we need
		// to populate.

		var response ApplyResourceChangeResponse

		schema := m.GetProviderSchema()
		response.Diagnostics = response.Diagnostics.Append(schema.Diagnostics)
		if schema.Diagnostics.HasErrors() {
			// We couldn't retrieve the schema for some reason, so the mock
			// provider can't really function.
			return response
		}

		resource, exists := schema.ResourceTypes[request.TypeName]
		if !exists {
			// This means something has gone wrong much earlier, we should have
			// failed a validation somewhere if a resource type doesn't exist.
			panic(fmt.Errorf("failed to retrieve schema for resource %s", request.TypeName))
		}

		replacement := mocking.MockedData{
			Value: cty.NilVal, // If we have no data then we use cty.NilVal.
		}
		if mockedResource, exists := m.Data.MockResources[request.TypeName]; exists {
			replacement.Value = mockedResource.Defaults
			replacement.Range = mockedResource.DefaultsRange
		}

		value, diags := mocking.ApplyComputedValuesForResource(request.PlannedState, replacement, resource.Block)
		response.Diagnostics = response.Diagnostics.Append(diags)
		response.NewState = value
		return response

	default:
		// For update or destroy operations, we don't have to create any values
		// so we'll just return the planned state directly.
		return ApplyResourceChangeResponse{
			NewState: request.PlannedState,
		}
	}
}

func (m *Mock) ImportResourceState(request ImportResourceStateRequest) (response ImportResourceStateResponse) {
	// You can't import via mock providers. The users should write specific
	// `override_resource` blocks for any resources they want to import, so we
	// just make them think about it rather than performing a blanket import
	// of all resources that are backed by mock providers.
	response.Diagnostics = response.Diagnostics.Append(tfdiags.Sourceless(tfdiags.Error, "Invalid import request", "Cannot import resources from mock providers. Use an `override_resource` block to targeting the specific resource being imported instead."))
	return response
}

func (m *Mock) MoveResourceState(request MoveResourceStateRequest) MoveResourceStateResponse {
	// The MoveResourceState operation happens offline, so we can just hand this
	// off to the underlying provider.
	return m.Provider.MoveResourceState(request)
}

func (m *Mock) ReadDataSource(request ReadDataSourceRequest) ReadDataSourceResponse {
	var response ReadDataSourceResponse

	schema := m.GetProviderSchema()
	response.Diagnostics = response.Diagnostics.Append(schema.Diagnostics)
	if schema.Diagnostics.HasErrors() {
		// We couldn't retrieve the schema for some reason, so the mock
		// provider can't really function.
		return response
	}

	datasource, exists := schema.DataSources[request.TypeName]
	if !exists {
		// This means something has gone wrong much earlier, we should have
		// failed a validation somewhere if a data source type doesn't exist.
		panic(fmt.Errorf("failed to retrieve schema for data source %s", request.TypeName))
	}

	mockedData := mocking.MockedData{
		Value: cty.NilVal, // If we have no mocked data we use cty.NilVal.
	}
	if mockedDataSource, exists := m.Data.MockDataSources[request.TypeName]; exists {
		mockedData.Value = mockedDataSource.Defaults
		mockedData.Range = mockedDataSource.DefaultsRange
	}

	value, diags := mocking.ComputedValuesForDataSource(request.Config, mockedData, datasource.Block)
	response.Diagnostics = response.Diagnostics.Append(diags)
	response.State = value
	return response
}

func (m *Mock) CallFunction(request CallFunctionRequest) CallFunctionResponse {
	return m.Provider.CallFunction(request)
}

func (m *Mock) Close() error {
	return m.Provider.Close()
}
