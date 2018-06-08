package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/config"
)

// MockEvalContext is a mock version of EvalContext that can be used
// for tests.
type MockEvalContext struct {
	StoppedCalled bool
	StoppedValue  <-chan struct{}

	HookCalled bool
	HookHook   Hook
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

	CloseProviderCalled   bool
	CloseProviderName     string
	CloseProviderProvider ResourceProvider

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

	InitProvisionerCalled      bool
	InitProvisionerName        string
	InitProvisionerProvisioner ResourceProvisioner
	InitProvisionerError       error

	ProvisionerCalled      bool
	ProvisionerName        string
	ProvisionerProvisioner ResourceProvisioner

	CloseProvisionerCalled      bool
	CloseProvisionerName        string
	CloseProvisionerProvisioner ResourceProvisioner

	InterpolateCalled       bool
	InterpolateConfig       *config.RawConfig
	InterpolateResource     *Resource
	InterpolateConfigResult *ResourceConfig
	InterpolateError        error

	InterpolateProviderCalled       bool
	InterpolateProviderConfig       *config.ProviderConfig
	InterpolateProviderResource     *Resource
	InterpolateProviderConfigResult *ResourceConfig
	InterpolateProviderError        error

	PathCalled bool
	PathPath   []string

	SetVariablesCalled    bool
	SetVariablesModule    string
	SetVariablesVariables map[string]interface{}

	DiffCalled bool
	DiffDiff   *Diff
	DiffLock   *sync.RWMutex

	StateCalled bool
	StateState  *State
	StateLock   *sync.RWMutex
}

func (c *MockEvalContext) Stopped() <-chan struct{} {
	c.StoppedCalled = true
	return c.StoppedValue
}

func (c *MockEvalContext) Hook(fn func(Hook) (HookAction, error)) error {
	c.HookCalled = true
	if c.HookHook != nil {
		if _, err := fn(c.HookHook); err != nil {
			return err
		}
	}

	return c.HookError
}

func (c *MockEvalContext) Input() UIInput {
	c.InputCalled = true
	return c.InputInput
}

func (c *MockEvalContext) InitProvider(t, n string) (ResourceProvider, error) {
	c.InitProviderCalled = true
	c.InitProviderName = n
	return c.InitProviderProvider, c.InitProviderError
}

func (c *MockEvalContext) Provider(n string) ResourceProvider {
	c.ProviderCalled = true
	c.ProviderName = n
	return c.ProviderProvider
}

func (c *MockEvalContext) CloseProvider(n string) error {
	c.CloseProviderCalled = true
	c.CloseProviderName = n
	return nil
}

func (c *MockEvalContext) ConfigureProvider(n string, cfg *ResourceConfig) error {
	c.ConfigureProviderCalled = true
	c.ConfigureProviderName = n
	c.ConfigureProviderConfig = cfg
	return c.ConfigureProviderError
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

func (c *MockEvalContext) CloseProvisioner(n string) error {
	c.CloseProvisionerCalled = true
	c.CloseProvisionerName = n
	return nil
}

func (c *MockEvalContext) Interpolate(
	config *config.RawConfig, resource *Resource) (*ResourceConfig, error) {
	c.InterpolateCalled = true
	c.InterpolateConfig = config
	c.InterpolateResource = resource
	return c.InterpolateConfigResult, c.InterpolateError
}

func (c *MockEvalContext) InterpolateProvider(
	config *config.ProviderConfig, resource *Resource) (*ResourceConfig, error) {
	c.InterpolateProviderCalled = true
	c.InterpolateProviderConfig = config
	c.InterpolateProviderResource = resource
	return c.InterpolateProviderConfigResult, c.InterpolateError
}

func (c *MockEvalContext) Path() []string {
	c.PathCalled = true
	return c.PathPath
}

func (c *MockEvalContext) SetVariables(n string, vs map[string]interface{}) {
	c.SetVariablesCalled = true
	c.SetVariablesModule = n
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
