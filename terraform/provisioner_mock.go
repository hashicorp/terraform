package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/provisioners"
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
