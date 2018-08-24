package terraform

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/providers"
)

var _ providers.Interface = (*MockProvider)(nil)

// MockProvider implements providers.Interface but mocks out all the
// calls for testing purposes.
type MockProvider struct {
	sync.Mutex

	// Anything you want, in case you need to store extra data with the mock.
	Meta interface{}

	GetSchemaCalled bool
	GetSchemaReturn *ProviderSchema // This is using ProviderSchema directly rather than providers.GetSchemaResponse for compatibility with old tests

	ValidateProviderConfigCalled   bool
	ValidateProviderConfigResponse providers.ValidateProviderConfigResponse
	ValidateProviderConfigRequest  providers.ValidateProviderConfigRequest
	ValidateProviderConfigFn       func(providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse

	ValidateResourceTypeConfigCalled   bool
	ValidateResourceTypeConfigTypeName string
	ValidateResourceTypeConfigResponse providers.ValidateResourceTypeConfigResponse
	ValidateResourceTypeConfigRequest  providers.ValidateResourceTypeConfigRequest
	ValidateResourceTypeConfigFn       func(providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse

	ValidateDataSourceConfigCalled   bool
	ValidateDataSourceConfigTypeName string
	ValidateDataSourceConfigResponse providers.ValidateDataSourceConfigResponse
	ValidateDataSourceConfigRequest  providers.ValidateDataSourceConfigRequest
	ValidateDataSourceConfigFn       func(providers.ValidateDataSourceConfigRequest) providers.ValidateDataSourceConfigResponse

	UpgradeResourceStateCalled   bool
	UpgradeResourceStateTypeName string
	UpgradeResourceStateResponse providers.UpgradeResourceStateResponse
	UpgradeResourceStateRequest  providers.UpgradeResourceStateRequest
	UpgradeResourceStateFn       func(providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse

	ConfigureCalled   bool
	ConfigureResponse providers.ConfigureResponse
	ConfigureRequest  providers.ConfigureRequest
	ConfigureNewFn    func(providers.ConfigureRequest) providers.ConfigureResponse // Named ConfigureNewFn so we can still have the legacy ConfigureFn declared below

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

	ReadDataSourceCalled   bool
	ReadDataSourceResponse providers.ReadDataSourceResponse
	ReadDataSourceRequest  providers.ReadDataSourceRequest
	ReadDataSourceFn       func(providers.ReadDataSourceRequest) providers.ReadDataSourceResponse

	CloseCalled bool
	CloseError  error

	// Legacy callbacks: if these are set, we will shim incoming calls for
	// new-style methods to these old-fashioned terraform.ResourceProvider
	// mock callbacks, for the benefit of older tests that were written against
	// the old mock API.
	ValidateFn  func(c *ResourceConfig) (ws []string, es []error)
	ConfigureFn func(c *ResourceConfig) error
	DiffFn      func(info *InstanceInfo, s *InstanceState, c *ResourceConfig) (*InstanceDiff, error)
	ApplyFn     func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error)
}

func (p *MockProvider) GetSchema() providers.GetSchemaResponse {
	p.Lock()
	defer p.Unlock()

	p.GetSchemaCalled = true
	ret := providers.GetSchemaResponse{
		Provider: providers.Schema{
			Block: p.GetSchemaReturn.Provider,
		},
		DataSources:   map[string]providers.Schema{},
		ResourceTypes: map[string]providers.Schema{},
	}
	for n, s := range p.GetSchemaReturn.DataSources {
		ret.DataSources[n] = providers.Schema{
			Block: s,
		}
	}
	for n, s := range p.GetSchemaReturn.ResourceTypes {
		ret.ResourceTypes[n] = providers.Schema{
			Block: s,
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

func (p *MockProvider) ValidateResourceTypeConfig(r providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
	p.Lock()
	defer p.Unlock()

	p.ValidateResourceTypeConfigCalled = true
	p.ValidateResourceTypeConfigRequest = r

	if p.ValidateFn != nil {
		return providers.ValidateResourceTypeConfigResponse{
			Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Errorf("legacy ValidateFn handling in MockProvider not actually implemented yet")),
		}
	}
	if p.ValidateResourceTypeConfigFn != nil {
		return p.ValidateResourceTypeConfigFn(r)
	}

	return p.ValidateResourceTypeConfigResponse
}

func (p *MockProvider) ValidateDataSourceConfig(r providers.ValidateDataSourceConfigRequest) providers.ValidateDataSourceConfigResponse {
	p.Lock()
	defer p.Unlock()

	p.ValidateDataSourceConfigCalled = true
	p.ValidateDataSourceConfigRequest = r

	if p.ValidateDataSourceConfigFn != nil {
		return p.ValidateDataSourceConfigFn(r)
	}

	return p.ValidateDataSourceConfigResponse
}

func (p *MockProvider) UpgradeResourceState(r providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	p.Lock()
	defer p.Unlock()

	p.UpgradeResourceStateCalled = true
	p.UpgradeResourceStateRequest = r

	if p.UpgradeResourceStateFn != nil {
		return p.UpgradeResourceStateFn(r)
	}

	return p.UpgradeResourceStateResponse
}

func (p *MockProvider) Configure(r providers.ConfigureRequest) providers.ConfigureResponse {
	p.Lock()
	defer p.Unlock()

	p.ConfigureCalled = true
	p.ConfigureRequest = r

	if p.ConfigureFn != nil {
		return providers.ConfigureResponse{
			Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Errorf("legacy ConfigureFn handling in MockProvider not actually implemented yet")),
		}
	}
	if p.ConfigureNewFn != nil {
		return p.ConfigureNewFn(r)
	}

	return p.ConfigureResponse
}

func (p *MockProvider) Stop() error {
	p.Lock()
	defer p.Unlock()

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

	return p.ReadResourceResponse
}

func (p *MockProvider) PlanResourceChange(r providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	p.PlanResourceChangeCalled = true
	p.PlanResourceChangeRequest = r

	if p.DiffFn != nil {
		return providers.PlanResourceChangeResponse{
			Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Errorf("legacy DiffFn handling in MockProvider not actually implemented yet")),
		}
	}
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

	if p.DiffFn != nil {
		return providers.ApplyResourceChangeResponse{
			Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Errorf("legacy ApplyFn handling in MockProvider not actually implemented yet")),
		}
	}
	if p.ApplyResourceChangeFn != nil {
		return p.ApplyResourceChangeFn(r)
	}

	return p.ApplyResourceChangeResponse
}

func (p *MockProvider) ImportResourceState(r providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	p.Lock()
	defer p.Unlock()

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
