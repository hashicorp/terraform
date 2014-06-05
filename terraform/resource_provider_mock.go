package terraform

import (
	"github.com/mitchellh/mapstructure"
)

// MockResourceProvider implements ResourceProvider but mocks out all the
// calls for testing purposes.
type MockResourceProvider struct {
	// Anything you want, in case you need to store extra data with the mock.
	Meta interface{}

	ConfigureCalled         bool
	ConfigureCommonConfig   ResourceProviderCommonConfig
	ConfigureConfig         map[string]interface{}
	ConfigureReturnWarnings []string
	ConfigureReturnError    error
	DiffCalled              bool
	DiffState               *ResourceState
	DiffDesired             map[string]interface{}
	DiffFn                  func(*ResourceState, map[string]interface{}) (ResourceDiff, error)
	DiffReturn              ResourceDiff
	DiffReturnError         error
	ResourcesCalled         bool
	ResourcesReturn         []ResourceType
}

func (p *MockResourceProvider) Configure(c map[string]interface{}) ([]string, error) {
	p.ConfigureCalled = true
	p.ConfigureConfig = c

	if err := mapstructure.Decode(&p.ConfigureCommonConfig, c); err != nil {
		return nil, err
	}

	return p.ConfigureReturnWarnings, p.ConfigureReturnError
}

func (p *MockResourceProvider) Diff(
	state *ResourceState,
	desired map[string]interface{}) (ResourceDiff, error) {
	p.DiffCalled = true
	p.DiffState = state
	p.DiffDesired = desired
	if p.DiffFn != nil {
		return p.DiffFn(state, desired)
	}

	return p.DiffReturn, p.DiffReturnError
}

func (p *MockResourceProvider) Resources() []ResourceType {
	p.ResourcesCalled = true
	return p.ResourcesReturn
}
