package terraform

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/tfdiags"
)

var _ provisioners.Interface = (*MockProvisioner)(nil)

// MockProvisioner implements provisioners.Interface but mocks out all the
// calls for testing purposes.
type MockProvisioner struct {
	sync.Mutex
	// Anything you want, in case you need to store extra data with the mock.
	Meta interface{}

	GetSchemaCalled   bool
	GetSchemaResponse provisioners.GetSchemaResponse

	ValidateProvisionerConfigCalled   bool
	ValidateProvisionerConfigRequest  provisioners.ValidateProvisionerConfigRequest
	ValidateProvisionerConfigResponse provisioners.ValidateProvisionerConfigResponse
	ValidateProvisionerConfigFn       func(provisioners.ValidateProvisionerConfigRequest) provisioners.ValidateProvisionerConfigResponse

	ProvisionResourceCalled   bool
	ProvisionResourceRequest  provisioners.ProvisionResourceRequest
	ProvisionResourceResponse provisioners.ProvisionResourceResponse
	ProvisionResourceFn       func(provisioners.ProvisionResourceRequest) provisioners.ProvisionResourceResponse

	StopCalled   bool
	StopResponse error
	StopFn       func() error

	CloseCalled   bool
	CloseResponse error
	CloseFn       func() error

	// Legacy callbacks: if these are set, we will shim incoming calls for
	// new-style methods to these old-fashioned terraform.ResourceProvider
	// mock callbacks, for the benefit of older tests that were written against
	// the old mock API.
	ApplyFn func(rs *InstanceState, c *ResourceConfig) error
}

func (p *MockProvisioner) GetSchema() provisioners.GetSchemaResponse {
	p.Lock()
	defer p.Unlock()

	p.GetSchemaCalled = true
	return p.GetSchemaResponse
}

func (p *MockProvisioner) ValidateProvisionerConfig(r provisioners.ValidateProvisionerConfigRequest) provisioners.ValidateProvisionerConfigResponse {
	p.Lock()
	defer p.Unlock()

	p.ValidateProvisionerConfigCalled = true
	p.ValidateProvisionerConfigRequest = r
	if p.ValidateProvisionerConfigFn != nil {
		return p.ValidateProvisionerConfigFn(r)
	}
	return p.ValidateProvisionerConfigResponse
}

func (p *MockProvisioner) ProvisionResource(r provisioners.ProvisionResourceRequest) provisioners.ProvisionResourceResponse {
	p.Lock()
	p.ProvisionResourceCalled = true
	p.ProvisionResourceRequest = r
	if p.ApplyFn != nil {
		return provisioners.ProvisionResourceResponse{
			Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Errorf("legacy ApplyFn handling in MockProvisioner not actually implemented yet")),
		}
	}
	if p.ProvisionResourceFn != nil {
		fn := p.ProvisionResourceFn
		p.Unlock()
		return fn(r)
	}

	defer p.Unlock()
	return p.ProvisionResourceResponse
}

func (p *MockProvisioner) Stop() error {
	p.Lock()
	defer p.Unlock()

	p.StopCalled = true
	if p.StopFn != nil {
		return p.StopFn()
	}

	return p.StopResponse
}

func (p *MockProvisioner) Close() error {
	p.Lock()
	defer p.Unlock()

	p.CloseCalled = true
	if p.CloseFn != nil {
		return p.CloseFn()
	}

	return p.CloseResponse
}
