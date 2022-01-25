package terraform

import (
	"encoding/json"
	"sync"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform/internal/providers"
)

var _ providers.Interface = (*MockProvider)(nil)

// MockProvider implements providers.Interface but mocks out all the
// calls for testing purposes.
type MockProvider struct {
	sync.Mutex

	// Anything you want, in case you need to store extra data with the mock.
	Meta interface{}

	GetSchemaCalled bool
	GetSchemaReturn *ProviderSchema // This is using ProviderSchema directly rather than providers.GetProviderSchemaResponse for compatibility with old tests

	ValidateProviderConfigCalled   bool
	ValidateProviderConfigResponse providers.ValidateProviderConfigResponse
	ValidateProviderConfigRequest  providers.ValidateProviderConfigRequest
	ValidateProviderConfigFn       func(providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse

	ValidateResourceConfigCalled   bool
	ValidateResourceConfigTypeName string
	ValidateResourceConfigResponse providers.ValidateResourceConfigResponse
	ValidateResourceConfigRequest  providers.ValidateResourceConfigRequest
	ValidateResourceConfigFn       func(providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse

	ValidateDataResourceConfigCalled   bool
	ValidateDataResourceConfigTypeName string
	ValidateDataResourceConfigResponse providers.ValidateDataResourceConfigResponse
	ValidateDataResourceConfigRequest  providers.ValidateDataResourceConfigRequest
	ValidateDataResourceConfigFn       func(providers.ValidateDataResourceConfigRequest) providers.ValidateDataResourceConfigResponse

	UpgradeResourceStateCalled   bool
	UpgradeResourceStateTypeName string
	UpgradeResourceStateResponse providers.UpgradeResourceStateResponse
	UpgradeResourceStateRequest  providers.UpgradeResourceStateRequest
	UpgradeResourceStateFn       func(providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse

	ConfigureProviderCalled   bool
	ConfigureProviderResponse providers.ConfigureProviderResponse
	ConfigureProviderRequest  providers.ConfigureProviderRequest
	ConfigureProviderFn       func(providers.ConfigureProviderRequest) providers.ConfigureProviderResponse

	StopCalled   bool
	StopFn       func() error
	StopResponse error

	ReadResourceCalled   bool
	ReadResourceResponse providers.ReadResourceResponse
	ReadResourceRequest  providers.ReadResourceRequest
	ReadResourceFn       func(providers.ReadResourceRequest) providers.ReadResourceResponse

	PlanResourceChangeCalled   bool
	PlanResourceChangeResponse providers.PlanResourceChangeResponse
	PlanResourceChangeRequest  providers.PlanResourceChangeRequest
	PlanResourceChangeFn       func(providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse

	ApplyResourceChangeCalled   bool
	ApplyResourceChangeResponse providers.ApplyResourceChangeResponse
	ApplyResourceChangeRequest  providers.ApplyResourceChangeRequest
	ApplyResourceChangeFn       func(providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse

	ImportResourceStateCalled   bool
	ImportResourceStateResponse providers.ImportResourceStateResponse
	ImportResourceStateRequest  providers.ImportResourceStateRequest
	ImportResourceStateFn       func(providers.ImportResourceStateRequest) providers.ImportResourceStateResponse
	// Legacy return type for existing tests, which will be shimmed into an
	// ImportResourceStateResponse if set
	ImportStateReturn []*InstanceState

	ReadDataSourceCalled   bool
	ReadDataSourceResponse providers.ReadDataSourceResponse
	ReadDataSourceRequest  providers.ReadDataSourceRequest
	ReadDataSourceFn       func(providers.ReadDataSourceRequest) providers.ReadDataSourceResponse

	CloseCalled bool
	CloseError  error
}

func (p *MockProvider) GetProviderSchema() providers.GetProviderSchemaResponse {
	p.Lock()
	defer p.Unlock()
	p.GetSchemaCalled = true
	return p.getSchema()
}

func (p *MockProvider) getSchema() providers.GetProviderSchemaResponse {
	// This version of getSchema doesn't do any locking, so it's suitable to
	// call from other methods of this mock as long as they are already
	// holding the lock.

	ret := providers.GetProviderSchemaResponse{
		Provider:      providers.Schema{},
		DataSources:   map[string]providers.Schema{},
		ResourceTypes: map[string]providers.Schema{},
	}
	if p.GetSchemaReturn != nil {
		ret.Provider.Block = p.GetSchemaReturn.Provider
		ret.ProviderMeta.Block = p.GetSchemaReturn.ProviderMeta
		for n, s := range p.GetSchemaReturn.DataSources {
			ret.DataSources[n] = providers.Schema{
				Block: s,
			}
		}
		for n, s := range p.GetSchemaReturn.ResourceTypes {
			ret.ResourceTypes[n] = providers.Schema{
				Version: int64(p.GetSchemaReturn.ResourceTypeSchemaVersions[n]),
				Block:   s,
			}
		}
	}

	return ret
}

func (p *MockProvider) ValidateProviderConfig(r providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
	p.Lock()
	defer p.Unlock()

	p.ValidateProviderConfigCalled = true
	p.ValidateProviderConfigRequest = r
	if p.ValidateProviderConfigFn != nil {
		return p.ValidateProviderConfigFn(r)
	}
	return p.ValidateProviderConfigResponse
}

func (p *MockProvider) ValidateResourceConfig(r providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
	p.Lock()
	defer p.Unlock()

	p.ValidateResourceConfigCalled = true
	p.ValidateResourceConfigRequest = r

	if p.ValidateResourceConfigFn != nil {
		return p.ValidateResourceConfigFn(r)
	}

	return p.ValidateResourceConfigResponse
}

func (p *MockProvider) ValidateDataResourceConfig(r providers.ValidateDataResourceConfigRequest) providers.ValidateDataResourceConfigResponse {
	p.Lock()
	defer p.Unlock()

	p.ValidateDataResourceConfigCalled = true
	p.ValidateDataResourceConfigRequest = r

	if p.ValidateDataResourceConfigFn != nil {
		return p.ValidateDataResourceConfigFn(r)
	}

	return p.ValidateDataResourceConfigResponse
}

func (p *MockProvider) UpgradeResourceState(r providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	p.Lock()
	defer p.Unlock()

	schemas := p.getSchema()
	schema := schemas.ResourceTypes[r.TypeName]
	schemaType := schema.Block.ImpliedType()

	p.UpgradeResourceStateCalled = true
	p.UpgradeResourceStateRequest = r

	if p.UpgradeResourceStateFn != nil {
		return p.UpgradeResourceStateFn(r)
	}

	resp := p.UpgradeResourceStateResponse

	if resp.UpgradedState == cty.NilVal {
		switch {
		case r.RawStateFlatmap != nil:
			v, err := hcl2shim.HCL2ValueFromFlatmap(r.RawStateFlatmap, schemaType)
			if err != nil {
				resp.Diagnostics = resp.Diagnostics.Append(err)
				return resp
			}
			resp.UpgradedState = v
		case len(r.RawStateJSON) > 0:
			v, err := ctyjson.Unmarshal(r.RawStateJSON, schemaType)

			if err != nil {
				resp.Diagnostics = resp.Diagnostics.Append(err)
				return resp
			}
			resp.UpgradedState = v
		}
	}
	return resp
}

func (p *MockProvider) ConfigureProvider(r providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	p.Lock()
	defer p.Unlock()

	p.ConfigureProviderCalled = true
	p.ConfigureProviderRequest = r

	if p.ConfigureProviderFn != nil {
		return p.ConfigureProviderFn(r)
	}

	return p.ConfigureProviderResponse
}

func (p *MockProvider) Stop() error {
	// We intentionally don't lock in this one because the whole point of this
	// method is to be called concurrently with another operation that can
	// be cancelled.  The provider itself is responsible for handling
	// any concurrency concerns in this case.

	p.StopCalled = true
	if p.StopFn != nil {
		return p.StopFn()
	}

	return p.StopResponse
}

func (p *MockProvider) ReadResource(r providers.ReadResourceRequest) providers.ReadResourceResponse {
	p.Lock()
	defer p.Unlock()

	p.ReadResourceCalled = true
	p.ReadResourceRequest = r

	if p.ReadResourceFn != nil {
		return p.ReadResourceFn(r)
	}

	resp := p.ReadResourceResponse
	if resp.NewState != cty.NilVal {
		// make sure the NewState fits the schema
		// This isn't always the case for the existing tests
		newState, err := p.GetSchemaReturn.ResourceTypes[r.TypeName].CoerceValue(resp.NewState)
		if err != nil {
			panic(err)
		}
		resp.NewState = newState
		return resp
	}

	// just return the same state we received
	resp.NewState = r.PriorState
	return resp
}

func (p *MockProvider) PlanResourceChange(r providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	p.Lock()
	defer p.Unlock()

	p.PlanResourceChangeCalled = true
	p.PlanResourceChangeRequest = r

	if p.PlanResourceChangeFn != nil {
		return p.PlanResourceChangeFn(r)
	}

	return p.PlanResourceChangeResponse
}

func (p *MockProvider) ApplyResourceChange(r providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	p.Lock()
	p.ApplyResourceChangeCalled = true
	p.ApplyResourceChangeRequest = r
	p.Unlock()

	if p.ApplyResourceChangeFn != nil {
		return p.ApplyResourceChangeFn(r)
	}

	return p.ApplyResourceChangeResponse
}

func (p *MockProvider) ImportResourceState(r providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	p.Lock()
	defer p.Unlock()

	if p.ImportStateReturn != nil {
		for _, is := range p.ImportStateReturn {
			if is.Attributes == nil {
				is.Attributes = make(map[string]string)
			}
			is.Attributes["id"] = is.ID

			typeName := is.Ephemeral.Type
			// Use the requested type if the resource has no type of it's own.
			// We still return the empty type, which will error, but this prevents a panic.
			if typeName == "" {
				typeName = r.TypeName
			}

			schema := p.GetSchemaReturn.ResourceTypes[typeName]
			if schema == nil {
				panic("no schema found for " + typeName)
			}

			private, err := json.Marshal(is.Meta)
			if err != nil {
				panic(err)
			}

			state, err := hcl2shim.HCL2ValueFromFlatmap(is.Attributes, schema.ImpliedType())
			if err != nil {
				panic(err)
			}

			state, err = schema.CoerceValue(state)
			if err != nil {
				panic(err)
			}

			p.ImportResourceStateResponse.ImportedResources = append(
				p.ImportResourceStateResponse.ImportedResources,
				providers.ImportedResource{
					TypeName: is.Ephemeral.Type,
					State:    state,
					Private:  private,
				})
		}
	}

	p.ImportResourceStateCalled = true
	p.ImportResourceStateRequest = r
	if p.ImportResourceStateFn != nil {
		return p.ImportResourceStateFn(r)
	}

	return p.ImportResourceStateResponse
}

func (p *MockProvider) ReadDataSource(r providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	p.Lock()
	defer p.Unlock()

	p.ReadDataSourceCalled = true
	p.ReadDataSourceRequest = r

	if p.ReadDataSourceFn != nil {
		return p.ReadDataSourceFn(r)
	}

	return p.ReadDataSourceResponse
}

func (p *MockProvider) Close() error {
	p.CloseCalled = true
	return p.CloseError
}
