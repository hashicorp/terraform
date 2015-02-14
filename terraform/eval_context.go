package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/config"
)

// EvalContext is the interface that is given to eval nodes to execute.
type EvalContext interface {
	// Path is the current module path.
	Path() []string

	// Hook is used to call hook methods. The callback is called for each
	// hook and should return the hook action to take and the error.
	Hook(func(Hook) (HookAction, error)) error

	// Input is the UIInput object for interacting with the UI.
	Input() UIInput

	// InitProvider initializes the provider with the given name and
	// returns the implementation of the resource provider or an error.
	//
	// It is an error to initialize the same provider more than once.
	InitProvider(string) (ResourceProvider, error)

	// Provider gets the provider instance with the given name (already
	// initialized) or returns nil if the provider isn't initialized.
	Provider(string) ResourceProvider

	// ConfigureProvider configures the provider with the given
	// configuration. This is a separate context call because this call
	// is used to store the provider configuration for inheritance lookups
	// with ParentProviderConfig().
	ConfigureProvider(string, *ResourceConfig) error
	ParentProviderConfig(string) *ResourceConfig

	// ProviderInput and SetProviderInput are used to configure providers
	// from user input.
	ProviderInput(string) map[string]interface{}
	SetProviderInput(string, map[string]interface{})

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

	// SetVariables sets the variables for interpolation. These variables
	// should not have a "var." prefix. For example: "var.foo" should be
	// "foo" as the key.
	SetVariables(map[string]string)

	// Diff returns the global diff as well as the lock that should
	// be used to modify that diff.
	Diff() (*Diff, *sync.RWMutex)

	// State returns the global state as well as the lock that should
	// be used to modify that state.
	State() (*State, *sync.RWMutex)
}

// MockEvalContext is a mock version of EvalContext that can be used
// for tests.
type MockEvalContext struct {
	HookCalled bool
	HookError  error

	InputCalled bool
	InputInput  UIInput

	InitProviderCalled   bool
	InitProviderName     string
	InitProviderProvider ResourceProvider
	InitProviderError    error

	ProviderCalled   bool
	ProviderName     string
	ProviderProvider ResourceProvider

	ProviderInputCalled bool
	ProviderInputName   string
	ProviderInputConfig map[string]interface{}

	SetProviderInputCalled bool
	SetProviderInputName   string
	SetProviderInputConfig map[string]interface{}

	ConfigureProviderCalled bool
	ConfigureProviderName   string
	ConfigureProviderConfig *ResourceConfig
	ConfigureProviderError  error

	ParentProviderConfigCalled bool
	ParentProviderConfigName   string
	ParentProviderConfigConfig *ResourceConfig

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

	SetVariablesCalled    bool
	SetVariablesVariables map[string]string

	DiffCalled bool
	DiffDiff   *Diff
	DiffLock   *sync.RWMutex

	StateCalled bool
	StateState  *State
	StateLock   *sync.RWMutex
}

func (c *MockEvalContext) Hook(fn func(Hook) (HookAction, error)) error {
	c.HookCalled = true
	return c.HookError
}

func (c *MockEvalContext) Input() UIInput {
	c.InputCalled = true
	return c.InputInput
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

func (c *MockEvalContext) ConfigureProvider(n string, cfg *ResourceConfig) error {
	c.ConfigureProviderCalled = true
	c.ConfigureProviderName = n
	c.ConfigureProviderConfig = cfg
	return c.ConfigureProviderError
}

func (c *MockEvalContext) ParentProviderConfig(n string) *ResourceConfig {
	c.ParentProviderConfigCalled = true
	c.ParentProviderConfigName = n
	return c.ParentProviderConfigConfig
}

func (c *MockEvalContext) ProviderInput(n string) map[string]interface{} {
	c.ProviderInputCalled = true
	c.ProviderInputName = n
	return c.ProviderInputConfig
}

func (c *MockEvalContext) SetProviderInput(n string, cfg map[string]interface{}) {
	c.SetProviderInputCalled = true
	c.SetProviderInputName = n
	c.SetProviderInputConfig = cfg
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

func (c *MockEvalContext) SetVariables(vs map[string]string) {
	c.SetVariablesCalled = true
	c.SetVariablesVariables = vs
}

func (c *MockEvalContext) Diff() (*Diff, *sync.RWMutex) {
	c.DiffCalled = true
	return c.DiffDiff, c.DiffLock
}

func (c *MockEvalContext) State() (*State, *sync.RWMutex) {
	c.StateCalled = true
	return c.StateState, c.StateLock
}
