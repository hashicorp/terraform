// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package testing

import (
	"fmt"

	"github.com/hashicorp/go-uuid"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	TestingResourceSchema = &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":    {Type: cty.String, Optional: true, Computed: true},
			"value": {Type: cty.String, Optional: true},
		},
	}

	TestingDataSourceSchema = &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":    {Type: cty.String, Required: true},
			"value": {Type: cty.String, Computed: true},
		},
	}
)

// MockProvider wraps the standard MockProvider with a simple in-memory
// data store for resources and data sources.
type MockProvider struct {
	*testing_provider.MockProvider

	ResourceStore *ResourceStore
}

// NewProvider returns a new MockProvider with an empty data store.
func NewProvider() *MockProvider {
	return NewProviderWithData(NewResourceStore())
}

// NewProviderWithData returns a new MockProvider with the given data store.
func NewProviderWithData(store *ResourceStore) *MockProvider {
	return &MockProvider{
		MockProvider: &testing_provider.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				ResourceTypes: map[string]providers.Schema{
					"testing_resource": {
						Block: TestingResourceSchema,
					},
				},
				DataSources: map[string]providers.Schema{
					"testing_data_source": {
						Block: TestingDataSourceSchema,
					},
				},
			},
			PlanResourceChangeFn: func(request providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
				if request.ProposedNewState.IsNull() {
					// Deleting, so just return the proposed change.
					return providers.PlanResourceChangeResponse{
						PlannedState: request.ProposedNewState,
					}
				}

				// We're creating or updating, so we need to return the new
				// state with any computed values filled in.

				value := request.ProposedNewState
				if id := value.GetAttr("id"); id.IsNull() {
					vals := value.AsValueMap()
					vals["id"] = cty.UnknownVal(cty.String)
					value = cty.ObjectVal(vals)
				}

				return providers.PlanResourceChangeResponse{
					PlannedState: value,
				}
			},
			ApplyResourceChangeFn: func(request providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
				if request.PlannedState.IsNull() {
					// Deleting, so just update the store and return.
					store.Delete(request.PlannedState.GetAttr("id").AsString())
					return providers.ApplyResourceChangeResponse{
						NewState: request.PlannedState,
					}
				}

				// Creating or updating, so update the store and return.

				// First, populate the computed value if we have to.
				value := request.PlannedState
				if id := value.GetAttr("id"); !id.IsKnown() {
					vals := value.AsValueMap()
					vals["id"] = cty.StringVal(mustGenerateUUID())
					value = cty.ObjectVal(vals)
				}

				// Finally, update the store and return.
				store.Set(value.GetAttr("id").AsString(), value)
				return providers.ApplyResourceChangeResponse{
					NewState: value,
				}
			},
			ReadResourceFn: func(request providers.ReadResourceRequest) providers.ReadResourceResponse {
				var diags tfdiags.Diagnostics

				id := request.PriorState.GetAttr("id").AsString()
				value, exists := store.Get(id)
				if !exists {
					diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "not found", fmt.Sprintf("%q not found", id)))
				}
				return providers.ReadResourceResponse{
					NewState:    value,
					Diagnostics: diags,
				}
			},
			ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
				var diags tfdiags.Diagnostics

				id := request.Config.GetAttr("id").AsString()
				value, exists := store.Get(id)
				if !exists {
					diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "not found", fmt.Sprintf("%q not found", id)))
				}
				return providers.ReadDataSourceResponse{
					State:       value,
					Diagnostics: diags,
				}
			},
		},
		ResourceStore: store,
	}
}

// mustGenerateUUID is a helper to generate a UUID and panic if it fails.
func mustGenerateUUID() string {
	val, err := uuid.GenerateUUID()
	if err != nil {
		panic(err)
	}
	return val
}
