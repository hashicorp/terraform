package terraform

import (
	"fmt"
	"sync"

	ctyjson "github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/configs/hcl2shim"
	"github.com/hashicorp/terraform/providers"
)

var _ providers.Interface = (*MockProvider)(nil)

// MockProvider implements providers.Interface but mocks out all the
// calls for testing purposes.
type MockProvider struct {
	sync.Mutex

	// Anything you want, in case you need to store extra data with the mock.
	Meta interface{}

	GetSchemaCalled   bool
	GetSchemaResponse *providers.GetSchemaResponse

	PrepareProviderConfigCalled   bool
	PrepareProviderConfigResponse *providers.PrepareProviderConfigResponse
	PrepareProviderConfigRequest  providers.PrepareProviderConfigRequest
	PrepareProviderConfigFn       func(providers.PrepareProviderConfigRequest) providers.PrepareProviderConfigResponse

	ValidateResourceTypeConfigCalled   bool
	ValidateResourceTypeConfigTypeName string
	ValidateResourceTypeConfigResponse *providers.ValidateResourceTypeConfigResponse
	ValidateResourceTypeConfigRequest  providers.ValidateResourceTypeConfigRequest
	ValidateResourceTypeConfigFn       func(providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse

	ValidateDataSourceConfigCalled   bool
	ValidateDataSourceConfigTypeName string
	ValidateDataSourceConfigResponse *providers.ValidateDataSourceConfigResponse
	ValidateDataSourceConfigRequest  providers.ValidateDataSourceConfigRequest
	ValidateDataSourceConfigFn       func(providers.ValidateDataSourceConfigRequest) providers.ValidateDataSourceConfigResponse

	UpgradeResourceStateCalled   bool
	UpgradeResourceStateTypeName string
	UpgradeResourceStateResponse *providers.UpgradeResourceStateResponse
	UpgradeResourceStateRequest  providers.UpgradeResourceStateRequest
	UpgradeResourceStateFn       func(providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse

	ConfigureCalled   bool
	ConfigureResponse *providers.ConfigureResponse
	ConfigureRequest  providers.ConfigureRequest
	ConfigureFn       func(providers.ConfigureRequest) providers.ConfigureResponse

	StopCalled   bool
	StopFn       func() error
	StopResponse error

	ReadResourceCalled   bool
	ReadResourceResponse *providers.ReadResourceResponse
	ReadResourceRequest  providers.ReadResourceRequest
	ReadResourceFn       func(providers.ReadResourceRequest) providers.ReadResourceResponse

	PlanResourceChangeCalled   bool
	PlanResourceChangeResponse *providers.PlanResourceChangeResponse
	PlanResourceChangeRequest  providers.PlanResourceChangeRequest
	PlanResourceChangeFn       func(providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse

	ApplyResourceChangeCalled   bool
	ApplyResourceChangeResponse *providers.ApplyResourceChangeResponse
	ApplyResourceChangeRequest  providers.ApplyResourceChangeRequest
	ApplyResourceChangeFn       func(providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse

	ImportResourceStateCalled   bool
	ImportResourceStateResponse *providers.ImportResourceStateResponse
	ImportResourceStateRequest  providers.ImportResourceStateRequest
	ImportResourceStateFn       func(providers.ImportResourceStateRequest) providers.ImportResourceStateResponse

	ReadDataSourceCalled   bool
	ReadDataSourceResponse *providers.ReadDataSourceResponse
	ReadDataSourceRequest  providers.ReadDataSourceRequest
	ReadDataSourceFn       func(providers.ReadDataSourceRequest) providers.ReadDataSourceResponse

	CloseCalled bool
	CloseError  error
}

func (p *MockProvider) GetSchema() providers.GetSchemaResponse {
	p.Lock()
	defer p.Unlock()
	p.GetSchemaCalled = true
	return p.getSchema()
}

func (p *MockProvider) getSchema() providers.GetSchemaResponse {
	// This version of getSchema doesn't do any locking, so it's suitable to
	// call from other methods of this mock as long as they are already
	// holding the lock.
	if p.GetSchemaResponse != nil {
		return *p.GetSchemaResponse
	}

	return providers.GetSchemaResponse{
		Provider:      providers.Schema{},
		DataSources:   map[string]providers.Schema{},
		ResourceTypes: map[string]providers.Schema{},
	}
}

// ProviderSchema is a helper to convert from the internal GetSchemaResponse to
// a ProviderSchema.
func (p *MockProvider) ProviderSchema() *ProviderSchema {
	resp := p.getSchema()

	schema := &ProviderSchema{
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

func (p *MockProvider) PrepareProviderConfig(r providers.PrepareProviderConfigRequest) (resp providers.PrepareProviderConfigResponse) {
	p.Lock()
	defer p.Unlock()

	p.PrepareProviderConfigCalled = true
	p.PrepareProviderConfigRequest = r
	if p.PrepareProviderConfigFn != nil {
		return p.PrepareProviderConfigFn(r)
	}

	if p.PrepareProviderConfigResponse != nil {
		return *p.PrepareProviderConfigResponse
	}

	resp.PreparedConfig = r.Config
	return resp
}

func (p *MockProvider) ValidateResourceTypeConfig(r providers.ValidateResourceTypeConfigRequest) (resp providers.ValidateResourceTypeConfigResponse) {
	p.Lock()
	defer p.Unlock()

	p.ValidateResourceTypeConfigCalled = true
	p.ValidateResourceTypeConfigRequest = r

	// Marshall the value to replicate behavior by the GRPC protocol,
	// and return any relevant errors
	resourceSchema, ok := p.getSchema().ResourceTypes[r.TypeName]
	if !ok {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("no schema found for %q", r.TypeName))
		return resp
	}

	_, err := msgpack.Marshal(r.Config, resourceSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	if p.ValidateResourceTypeConfigFn != nil {
		return p.ValidateResourceTypeConfigFn(r)
	}

	if p.ValidateResourceTypeConfigResponse != nil {
		return *p.ValidateResourceTypeConfigResponse
	}

	return resp
}

func (p *MockProvider) ValidateDataSourceConfig(r providers.ValidateDataSourceConfigRequest) (resp providers.ValidateDataSourceConfigResponse) {
	p.Lock()
	defer p.Unlock()

	p.ValidateDataSourceConfigCalled = true
	p.ValidateDataSourceConfigRequest = r

	// Marshall the value to replicate behavior by the GRPC protocol
	dataSchema, ok := p.getSchema().DataSources[r.TypeName]
	if !ok {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("no schema found for %q", r.TypeName))
		return resp
	}
	_, err := msgpack.Marshal(r.Config, dataSchema.Block.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	if p.ValidateDataSourceConfigFn != nil {
		return p.ValidateDataSourceConfigFn(r)
	}

	if p.ValidateDataSourceConfigResponse != nil {
		return *p.ValidateDataSourceConfigResponse
	}

	return resp
}

func (p *MockProvider) UpgradeResourceState(r providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
	p.Lock()
	defer p.Unlock()

	schema, ok := p.getSchema().ResourceTypes[r.TypeName]
	if !ok {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("no schema found for %q", r.TypeName))
		return resp
	}

	schemaType := schema.Block.ImpliedType()

	p.UpgradeResourceStateCalled = true
	p.UpgradeResourceStateRequest = r

	if p.UpgradeResourceStateFn != nil {
		return p.UpgradeResourceStateFn(r)
	}

	if p.UpgradeResourceStateResponse != nil {
		return *p.UpgradeResourceStateResponse
	}

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

	return resp
}

func (p *MockProvider) Configure(r providers.ConfigureRequest) (resp providers.ConfigureResponse) {
	p.Lock()
	defer p.Unlock()

	p.ConfigureCalled = true
	p.ConfigureRequest = r

	if p.ConfigureFn != nil {
		return p.ConfigureFn(r)
	}

	if p.ConfigureResponse != nil {
		return *p.ConfigureResponse
	}

	return resp
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

func (p *MockProvider) ReadResource(r providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
	p.Lock()
	defer p.Unlock()

	p.ReadResourceCalled = true
	p.ReadResourceRequest = r

	if p.ReadResourceFn != nil {
		return p.ReadResourceFn(r)
	}

	if p.ReadResourceResponse != nil {
		resp = *p.ReadResourceResponse

		// Make sure the NewState conforms to the schema.
		// This isn't always the case for the existing tests.
		schema, ok := p.getSchema().ResourceTypes[r.TypeName]
		if !ok {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("no schema found for %q", r.TypeName))
			return resp
		}

		newState, err := schema.Block.CoerceValue(resp.NewState)
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
		}
		resp.NewState = newState
		return resp
	}

	// otherwise just return the same state we received
	resp.NewState = r.PriorState
	return resp
}

func (p *MockProvider) PlanResourceChange(r providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
	p.Lock()
	defer p.Unlock()

	p.PlanResourceChangeCalled = true
	p.PlanResourceChangeRequest = r

	if p.PlanResourceChangeFn != nil {
		return p.PlanResourceChangeFn(r)
	}

	if p.PlanResourceChangeResponse != nil {
		return *p.PlanResourceChangeResponse
	}

	schema, ok := p.getSchema().ResourceTypes[r.TypeName]
	if !ok {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("no schema found for %q", r.TypeName))
		return resp
	}

	return resp
}

func (p *MockProvider) ApplyResourceChange(r providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
	p.Lock()
	p.ApplyResourceChangeCalled = true
	p.ApplyResourceChangeRequest = r
	p.Unlock()

	if p.ApplyResourceChangeFn != nil {
		return p.ApplyResourceChangeFn(r)
	}

	if p.ApplyResourceChangeResponse != nil {
		return *p.ApplyResourceChangeResponse
	}

	return resp
}

func (p *MockProvider) ImportResourceState(r providers.ImportResourceStateRequest) (resp providers.ImportResourceStateResponse) {
	p.Lock()
	defer p.Unlock()

	p.ImportResourceStateCalled = true
	p.ImportResourceStateRequest = r
	if p.ImportResourceStateFn != nil {
		return p.ImportResourceStateFn(r)
	}

	if p.ImportResourceStateResponse != nil {
		resp = *p.ImportResourceStateResponse
		// fixup the cty value to match the schema
		for i, res := range resp.ImportedResources {
			schema, ok := p.getSchema().ResourceTypes[res.TypeName]
			if !ok {
				resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("no schema found for %q", res.TypeName))
				return resp
			}

			var err error
			res.State, err = schema.Block.CoerceValue(res.State)
			if err != nil {
				resp.Diagnostics = resp.Diagnostics.Append(err)
				return resp
			}

			resp.ImportedResources[i] = res
		}
	}

	return resp
}

func (p *MockProvider) ReadDataSource(r providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
	p.Lock()
	defer p.Unlock()

	p.ReadDataSourceCalled = true
	p.ReadDataSourceRequest = r

	if p.ReadDataSourceFn != nil {
		return p.ReadDataSourceFn(r)
	}

	if p.ReadDataSourceResponse != nil {
		resp = *p.ReadDataSourceResponse
	}

	return resp
}

func (p *MockProvider) Close() error {
	p.CloseCalled = true
	return p.CloseError
}
