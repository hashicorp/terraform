package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// EvalContext is the interface that is given to eval nodes to execute.
type EvalContext interface {
	// Path is the current module path.
	Path() []string

	// InitProvider initializes the provider with the given name and
	// returns the implementation of the resource provider or an error.
	//
	// It is an error to initialize the same provider more than once.
	InitProvider(string) (ResourceProvider, error)

	// Provider gets the provider instance with the given name (already
	// initialized) or returns nil if the provider isn't initialized.
	Provider(string) ResourceProvider

	// InitProvisioner initializes the provisioner with the given name and
	// returns the implementation of the resource provisioner or an error.
	//
	// It is an error to initialize the same provisioner more than once.
	InitProvisioner(string) (ResourceProvisioner, error)

	// Provisioner gets the provisioner instance with the given name (already
	// initialized) or returns nil if the provisioner isn't initialized.
	Provisioner(string) ResourceProvisioner

	// Interpolate takes the given raw configuration and completes
	// the interpolations, returning the processed ResourceConfig.
	//
	// The resource argument is optional. If given, it is the resource
	// that is currently being acted upon.
	Interpolate(*config.RawConfig, *Resource) (*ResourceConfig, error)
}

// MockEvalContext is a mock version of EvalContext that can be used
// for tests.
type MockEvalContext struct {
	InitProviderCalled   bool
	InitProviderName     string
	InitProviderProvider ResourceProvider
	InitProviderError    error

	ProviderCalled   bool
	ProviderName     string
	ProviderProvider ResourceProvider

	InitProvisionerCalled      bool
	InitProvisionerName        string
	InitProvisionerProvisioner ResourceProvisioner
	InitProvisionerError       error

	ProvisionerCalled      bool
	ProvisionerName        string
	ProvisionerProvisioner ResourceProvisioner

	InterpolateCalled       bool
	InterpolateConfig       *config.RawConfig
	InterpolateResource     *Resource
	InterpolateConfigResult *ResourceConfig
	InterpolateError        error

	PathCalled bool
	PathPath   []string
}

func (c *MockEvalContext) InitProvider(n string) (ResourceProvider, error) {
	c.InitProviderCalled = true
	c.InitProviderName = n
	return c.InitProviderProvider, c.InitProviderError
}

func (c *MockEvalContext) Provider(n string) ResourceProvider {
	c.ProviderCalled = true
	c.ProviderName = n
	return c.ProviderProvider
}

func (c *MockEvalContext) InitProvisioner(n string) (ResourceProvisioner, error) {
	c.InitProvisionerCalled = true
	c.InitProvisionerName = n
	return c.InitProvisionerProvisioner, c.InitProvisionerError
}

func (c *MockEvalContext) Provisioner(n string) ResourceProvisioner {
	c.ProvisionerCalled = true
	c.ProvisionerName = n
	return c.ProvisionerProvisioner
}

func (c *MockEvalContext) Interpolate(
	config *config.RawConfig, resource *Resource) (*ResourceConfig, error) {
	c.InterpolateCalled = true
	c.InterpolateConfig = config
	c.InterpolateResource = resource
	return c.InterpolateConfigResult, c.InterpolateError
}

func (c *MockEvalContext) Path() []string {
	c.PathCalled = true
	return c.PathPath
}
