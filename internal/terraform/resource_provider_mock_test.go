// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"

	"github.com/zclconf/go-cty/cty"
)

// mockProviderWithConfigSchema is a test helper to concisely create a mock
// provider with the given schema for its own configuration.
func mockProviderWithConfigSchema(schema *configschema.Block) *testing_provider.MockProvider {
	return &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Block: schema},
		},
	}
}

// mockProviderWithResourceTypeSchema is a test helper to concisely create a mock
// provider with a schema containing a single resource type.
func mockProviderWithResourceTypeSchema(name string, schema *configschema.Block) *testing_provider.MockProvider {
	return &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"string": {
							Type:     cty.String,
							Optional: true,
						},
						"list": {
							Type:     cty.List(cty.String),
							Optional: true,
						},
						"root": {
							Type:     cty.Map(cty.String),
							Optional: true,
						},
					},
				},
			},
			ResourceTypes: map[string]providers.Schema{
				name: providers.Schema{Block: schema},
			},
		},
	}
}

// simpleMockProvider returns a MockProvider that is pre-configured
// with schema for its own config, for a resource type called "test_object" and
// for a data source also called "test_object".
//
// All three schemas have the same content as returned by function
// simpleTestSchema.
//
// For most reasonable uses the returned provider must be registered in a
// componentFactory under the name "test". Use simpleMockComponentFactory
// to obtain a pre-configured componentFactory containing the result of
// this function along with simpleMockProvisioner, both registered as "test".
//
// The returned provider has no other behaviors by default, but the caller may
// modify it in order to stub any other required functionality, or modify
// the default schema stored in the field GetSchemaReturn. Each new call to
// simpleTestProvider produces entirely new instances of all of the nested
// objects so that callers can mutate without affecting mock objects.
func simpleMockProvider() *testing_provider.MockProvider {
	return &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Block: simpleTestSchema()},
			ResourceTypes: map[string]providers.Schema{
				"test_object": providers.Schema{Block: simpleTestSchema()},
			},
			DataSources: map[string]providers.Schema{
				"test_object": providers.Schema{Block: simpleTestSchema()},
			},
		},
	}
}

// getProviderSchema is a helper to convert from the internal
// GetProviderSchemaResponse to a providerSchema.
func getProviderSchema(p *testing_provider.MockProvider) *providerSchema {
	if p.GetProviderSchemaResponse == nil {
		// Then just return an empty provider schema.
		return &providerSchema{
			ResourceTypes:              make(map[string]*configschema.Block),
			ResourceTypeSchemaVersions: make(map[string]uint64),
			DataSources:                make(map[string]*configschema.Block),
		}
	}

	resp := p.GetProviderSchemaResponse

	schema := &providerSchema{
		Provider:                   resp.Provider.Block,
		ProviderMeta:               resp.ProviderMeta.Block,
		ResourceTypes:              map[string]*configschema.Block{},
		DataSources:                map[string]*configschema.Block{},
		ResourceTypeSchemaVersions: map[string]uint64{},
	}

	for resType, s := range resp.ResourceTypes {
		schema.ResourceTypes[resType] = s.Block
		schema.ResourceTypeSchemaVersions[resType] = uint64(s.Version)
	}

	for dataSource, s := range resp.DataSources {
		schema.DataSources[dataSource] = s.Block
	}

	return schema
}

// the type was refactored out with all the functionality handled within the
// provider package, but we keep this here for a shim in existing tests.
type providerSchema struct {
	Provider                   *configschema.Block
	ProviderMeta               *configschema.Block
	ResourceTypes              map[string]*configschema.Block
	ResourceTypeSchemaVersions map[string]uint64
	DataSources                map[string]*configschema.Block
}

// getProviderSchemaResponseFromProviderSchema is a test helper to convert a
// providerSchema to a GetProviderSchemaResponse for use when building a mock provider.
func getProviderSchemaResponseFromProviderSchema(providerSchema *providerSchema) *providers.GetProviderSchemaResponse {
	resp := &providers.GetProviderSchemaResponse{
		Provider:      providers.Schema{Block: providerSchema.Provider},
		ProviderMeta:  providers.Schema{Block: providerSchema.ProviderMeta},
		ResourceTypes: map[string]providers.Schema{},
		DataSources:   map[string]providers.Schema{},
	}

	for name, schema := range providerSchema.ResourceTypes {
		resp.ResourceTypes[name] = providers.Schema{
			Block:   schema,
			Version: int64(providerSchema.ResourceTypeSchemaVersions[name]),
		}
	}

	for name, schema := range providerSchema.DataSources {
		resp.DataSources[name] = providers.Schema{Block: schema}
	}

	return resp
}
