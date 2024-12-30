// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package testing

import (
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/providers/testing"
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
						"id":                   {Type: cty.String, Optional: true, Computed: true},
						"value":                {Type: cty.String, Optional: true},
						"interrupt_count":      {Type: cty.Number, Optional: true},
						"destroy_fail":         {Type: cty.Bool, Optional: true, Computed: true},
						"create_wait_seconds":  {Type: cty.Number, Optional: true},
						"destroy_wait_seconds": {Type: cty.Number, Optional: true},
					},
				},
			},
		},
		DataSources: map[string]providers.Schema{
			"test_data_source": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":    {Type: cty.String, Required: true},
						"value": {Type: cty.String, Computed: true},

						// We never actually reference these values from a data
						// source, but we have tests that use the same cty.Value
						// to represent a test_resource and a test_data_source
						// so the schemas have to match.

						"interrupt_count":      {Type: cty.Number, Computed: true},
						"destroy_fail":         {Type: cty.Bool, Computed: true},
						"create_wait_seconds":  {Type: cty.Number, Computed: true},
						"destroy_wait_seconds": {Type: cty.Number, Computed: true},
					},
				},
			},
		},
		EphemeralResourceTypes: map[string]providers.Schema{
			"test_ephemeral_resource": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:     cty.String,
							Computed: true,
						},
					},
				},
			},
		},
		Functions: map[string]providers.FunctionDecl{
			"is_true": {
				Parameters: []providers.FunctionParam{
					{
						Name:               "input",
						Type:               cty.Bool,
						AllowNullValue:     false,
						AllowUnknownValues: false,
					},
				},
				ReturnType: cty.Bool,
			},
		},
	}
)

// TestProvider is a wrapper around terraform.MockProvider that defines dynamic
// schemas, and keeps track of the resources and data sources that it contains.
type TestProvider struct {
	Provider *testing.MockProvider

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
		Provider: new(testing.MockProvider),
		Store:    store,
	}

	provider.Provider.GetProviderSchemaResponse = ProviderSchema
	provider.Provider.ConfigureProviderFn = provider.ConfigureProvider
	provider.Provider.PlanResourceChangeFn = provider.PlanResourceChange
	provider.Provider.ApplyResourceChangeFn = provider.ApplyResourceChange
	provider.Provider.ReadResourceFn = provider.ReadResource
	provider.Provider.ReadDataSourceFn = provider.ReadDataSource
	provider.Provider.CallFunctionFn = provider.CallFunction
	provider.Provider.OpenEphemeralResourceFn = provider.OpenEphemeralResource
	provider.Provider.CloseEphemeralResourceFn = provider.CloseEphemeralResource

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
	provider.Store.mutex.RLock()
	defer provider.Store.mutex.RUnlock()

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
	provider.Store.mutex.RLock()
	defer provider.Store.mutex.RUnlock()

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

	if destryFail := resource.GetAttr("destroy_fail"); !destryFail.IsKnown() || destryFail.IsNull() {
		vals := resource.AsValueMap()
		vals["destroy_fail"] = cty.UnknownVal(cty.Bool)
		resource = cty.ObjectVal(vals)
	}

	return providers.PlanResourceChangeResponse{
		PlannedState: resource,
	}
}

func (provider *TestProvider) ApplyResourceChange(request providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	if request.PlannedState.IsNull() {
		// Then this is a delete operation.

		if destroyFail := request.PriorState.GetAttr("destroy_fail"); destroyFail.IsKnown() && destroyFail.True() {
			var diags tfdiags.Diagnostics
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Failed to destroy resource", "destroy_fail is set to true"))
			return providers.ApplyResourceChangeResponse{
				Diagnostics: diags,
			}
		}

		if wait := request.PriorState.GetAttr("destroy_wait_seconds"); !wait.IsNull() && wait.IsKnown() {
			waitTime, _ := wait.AsBigFloat().Int64()
			time.Sleep(time.Second * time.Duration(waitTime))
		}

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

	if interrupts := resource.GetAttr("interrupt_count"); !interrupts.IsNull() && interrupts.IsKnown() && provider.Interrupt != nil {
		count, _ := interrupts.AsBigFloat().Int64()
		for ix := 0; ix < int(count); ix++ {
			provider.Interrupt <- struct{}{}
		}

		// Wait for a second to make sure the interrupts are processed by
		// Terraform before the provider finishes. This is an attempt to ensure
		// the output of any tests that rely on this behaviour is deterministic.
		time.Sleep(time.Second)
	}

	if wait := resource.GetAttr("create_wait_seconds"); !wait.IsNull() && wait.IsKnown() {
		waitTime, _ := wait.AsBigFloat().Int64()
		time.Sleep(time.Second * time.Duration(waitTime))
	}

	if destroyFail := resource.GetAttr("destroy_fail"); !destroyFail.IsKnown() {
		vals := resource.AsValueMap()
		vals["destroy_fail"] = cty.False
		resource = cty.ObjectVal(vals)
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

func (provider *TestProvider) CallFunction(request providers.CallFunctionRequest) providers.CallFunctionResponse {
	switch request.FunctionName {
	case "is_true":
		return providers.CallFunctionResponse{
			Result: request.Arguments[0],
		}
	default:
		return providers.CallFunctionResponse{
			Err: fmt.Errorf("unknown function %q", request.FunctionName),
		}
	}
}

func (provider *TestProvider) OpenEphemeralResource(providers.OpenEphemeralResourceRequest) (resp providers.OpenEphemeralResourceResponse) {
	resp.Result = cty.ObjectVal(map[string]cty.Value{
		"value": cty.StringVal("bar"),
	})
	return resp
}

func (provider *TestProvider) CloseEphemeralResource(providers.CloseEphemeralResourceRequest) (resp providers.CloseEphemeralResourceResponse) {
	return resp
}

// ResourceStore manages a set of cty.Value resources that can be shared between
// TestProvider providers.
type ResourceStore struct {
	mutex sync.RWMutex

	Data map[string]cty.Value
}

func (store *ResourceStore) Delete(key string) cty.Value {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if resource, ok := store.Data[key]; ok {
		delete(store.Data, key)
		return resource
	}
	return cty.NilVal
}

func (store *ResourceStore) Get(key string) cty.Value {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	return store.get(key)
}

func (store *ResourceStore) Put(key string, resource cty.Value) cty.Value {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	old := store.get(key)
	store.Data[key] = resource
	return old
}

func (store *ResourceStore) get(key string) cty.Value {
	if resource, ok := store.Data[key]; ok {
		return resource
	}
	return cty.NilVal
}
