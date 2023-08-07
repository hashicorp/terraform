// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package testing

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	ProviderSchema = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"data_prefix":     {Type: cty.String, Optional: true},
					"resource_prefix": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_resource": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":              {Type: cty.String, Optional: true, Computed: true},
						"value":           {Type: cty.String, Optional: true},
						"interrupt_count": {Type: cty.Number, Optional: true},
					},
				},
			},
		},
		DataSources: map[string]providers.Schema{
			"test_data_source": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":              {Type: cty.String, Required: true},
						"value":           {Type: cty.String, Computed: true},
						"interrupt_count": {Type: cty.Number, Computed: true},
					},
				},
			},
		},
	}
)

// TestProvider is a wrapper around terraform.MockProvider that defines dynamic
// schemas, and keeps track of the resources and data sources that it contains.
type TestProvider struct {
	Provider *terraform.MockProvider

	data, resource cty.Value

	Interrupt chan<- struct{}

	Store *ResourceStore
}

func NewProvider(store *ResourceStore) *TestProvider {
	if store == nil {
		store = &ResourceStore{
			Data: make(map[string]cty.Value),
		}
	}

	provider := &TestProvider{
		Provider: new(terraform.MockProvider),
		Store:    store,
	}

	provider.Provider.GetProviderSchemaResponse = ProviderSchema
	provider.Provider.ConfigureProviderFn = provider.ConfigureProvider
	provider.Provider.PlanResourceChangeFn = provider.PlanResourceChange
	provider.Provider.ApplyResourceChangeFn = provider.ApplyResourceChange
	provider.Provider.ReadResourceFn = provider.ReadResource
	provider.Provider.ReadDataSourceFn = provider.ReadDataSource

	return provider
}

func (provider *TestProvider) DataPrefix() string {
	var prefix string
	if !provider.data.IsNull() && provider.data.IsKnown() {
		prefix = provider.data.AsString()
	}
	return prefix
}

func (provider *TestProvider) SetDataPrefix(prefix string) {
	provider.data = cty.StringVal(prefix)
}

func (provider *TestProvider) GetDataKey(id string) string {
	if !provider.data.IsNull() && provider.data.IsKnown() {
		return path.Join(provider.data.AsString(), id)
	}
	return id
}

func (provider *TestProvider) ResourcePrefix() string {
	var prefix string
	if !provider.resource.IsNull() && provider.resource.IsKnown() {
		prefix = provider.resource.AsString()
	}
	return prefix
}

func (provider *TestProvider) SetResourcePrefix(prefix string) {
	provider.resource = cty.StringVal(prefix)
}

func (provider *TestProvider) GetResourceKey(id string) string {
	if !provider.resource.IsNull() && provider.resource.IsKnown() {
		return path.Join(provider.resource.AsString(), id)
	}
	return id
}

func (provider *TestProvider) ResourceString() string {
	return provider.string(provider.ResourcePrefix())
}

func (provider *TestProvider) ResourceCount() int {
	return provider.count(provider.ResourcePrefix())
}

func (provider *TestProvider) DataSourceString() string {
	return provider.string(provider.DataPrefix())
}

func (provider *TestProvider) DataSourceCount() int {
	return provider.count(provider.DataPrefix())
}

func (provider *TestProvider) count(prefix string) int {
	if len(prefix) == 0 {
		return len(provider.Store.Data)
	}

	count := 0
	for key := range provider.Store.Data {
		if strings.HasPrefix(key, prefix) {
			count++
		}
	}
	return count
}

func (provider *TestProvider) string(prefix string) string {
	var keys []string
	for key := range provider.Store.Data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return strings.Join(keys, ", ")
}

func (provider *TestProvider) ConfigureProvider(request providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	provider.resource = request.Config.GetAttr("resource_prefix")
	provider.data = request.Config.GetAttr("data_prefix")
	return providers.ConfigureProviderResponse{}
}

func (provider *TestProvider) PlanResourceChange(request providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	if request.ProposedNewState.IsNull() {
		// Then this is a delete operation.
		return providers.PlanResourceChangeResponse{
			PlannedState: request.ProposedNewState,
		}
	}

	resource := request.ProposedNewState
	if id := resource.GetAttr("id"); !id.IsKnown() || id.IsNull() {
		vals := resource.AsValueMap()
		vals["id"] = cty.UnknownVal(cty.String)
		resource = cty.ObjectVal(vals)
	}

	return providers.PlanResourceChangeResponse{
		PlannedState: resource,
	}
}

func (provider *TestProvider) ApplyResourceChange(request providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	if request.PlannedState.IsNull() {
		// Then this is a delete operation.
		provider.Store.Delete(provider.GetResourceKey(request.PriorState.GetAttr("id").AsString()))
		return providers.ApplyResourceChangeResponse{
			NewState: request.PlannedState,
		}
	}

	resource := request.PlannedState
	id := resource.GetAttr("id")
	if !id.IsKnown() {
		val, err := uuid.GenerateUUID()
		if err != nil {
			panic(fmt.Errorf("failed to generate uuid: %v", err))
		}

		id = cty.StringVal(val)

		vals := resource.AsValueMap()
		vals["id"] = id
		resource = cty.ObjectVal(vals)
	}

	interrupts := resource.GetAttr("interrupt_count")
	if !interrupts.IsNull() && interrupts.IsKnown() && provider.Interrupt != nil {
		count, _ := interrupts.AsBigFloat().Int64()
		for ix := 0; ix < int(count); ix++ {
			provider.Interrupt <- struct{}{}
		}

		// Wait for a second to make sure the interrupts are processed by
		// Terraform before the provider finishes. This is an attempt to ensure
		// the output of any tests that rely on this behaviour is deterministic.
		time.Sleep(time.Second)
	}

	provider.Store.Put(provider.GetResourceKey(id.AsString()), resource)
	return providers.ApplyResourceChangeResponse{
		NewState: resource,
	}
}

func (provider *TestProvider) ReadResource(request providers.ReadResourceRequest) providers.ReadResourceResponse {
	var diags tfdiags.Diagnostics

	id := request.PriorState.GetAttr("id").AsString()
	resource := provider.Store.Get(provider.GetResourceKey(id))
	if resource == cty.NilVal {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "not found", fmt.Sprintf("%s does not exist", id)))
	}

	return providers.ReadResourceResponse{
		NewState:    resource,
		Diagnostics: diags,
	}
}

func (provider *TestProvider) ReadDataSource(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	var diags tfdiags.Diagnostics

	id := request.Config.GetAttr("id").AsString()
	resource := provider.Store.Get(provider.GetDataKey(id))
	if resource == cty.NilVal {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "not found", fmt.Sprintf("%s does not exist", id)))
	}

	return providers.ReadDataSourceResponse{
		State:       resource,
		Diagnostics: diags,
	}
}

// ResourceStore manages a set of cty.Value resources that can be shared between
// TestProvider providers.
type ResourceStore struct {
	Data map[string]cty.Value
}

func (store *ResourceStore) Delete(key string) cty.Value {
	if resource, ok := store.Data[key]; ok {
		delete(store.Data, key)
		return resource
	}
	return cty.NilVal
}

func (store *ResourceStore) Get(key string) cty.Value {
	if resource, ok := store.Data[key]; ok {
		return resource
	}
	return cty.NilVal
}

func (store *ResourceStore) Put(key string, resource cty.Value) cty.Value {
	old := store.Get(key)
	store.Data[key] = resource
	return old
}
