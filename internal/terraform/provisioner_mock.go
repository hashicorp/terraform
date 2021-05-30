package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/internal/provisioners"
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
	return p.getSchema()
}

// getSchema is the implementation of GetSchema, which can be called from other
// methods on MockProvisioner that may already be holding the lock.
func (p *MockProvisioner) getSchema() provisioners.GetSchemaResponse {
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
	defer p.Unlock()

	p.ProvisionResourceCalled = true
	p.ProvisionResourceRequest = r
	if p.ProvisionResourceFn != nil {
		fn := p.ProvisionResourceFn
		return fn(r)
	}

	return p.ProvisionResourceResponse
}

func (p *MockProvisioner) Stop() error {
	// We intentionally don't lock in this one because the whole point of this
	// method is to be called concurrently with another operation that can
	// be cancelled. The provisioner itself is responsible for handling
	// any concurrency concerns in this case.

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
