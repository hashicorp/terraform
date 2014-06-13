package terraform

// MockResourceProvider implements ResourceProvider but mocks out all the
// calls for testing purposes.
type MockResourceProvider struct {
	// Anything you want, in case you need to store extra data with the mock.
	Meta interface{}

	ConfigureCalled      bool
	ConfigureConfig      *ResourceConfig
	ConfigureReturnError error
	DiffCalled           bool
	DiffState            *ResourceState
	DiffDesired          *ResourceConfig
	DiffFn               func(*ResourceState, *ResourceConfig) (*ResourceDiff, error)
	DiffReturn           *ResourceDiff
	DiffReturnError      error
	ResourcesCalled      bool
	ResourcesReturn      []ResourceType
	ValidateCalled       bool
	ValidateConfig       *ResourceConfig
	ValidateReturnWarns  []string
	ValidateReturnErrors []error
}

func (p *MockResourceProvider) Validate(c *ResourceConfig) ([]string, []error) {
	p.ValidateCalled = true
	p.ValidateConfig = c
	return p.ValidateReturnWarns, p.ValidateReturnErrors
}

func (p *MockResourceProvider) Configure(c *ResourceConfig) error {
	p.ConfigureCalled = true
	p.ConfigureConfig = c
	return p.ConfigureReturnError
}

func (p *MockResourceProvider) Diff(
	state *ResourceState,
	desired *ResourceConfig) (*ResourceDiff, error) {
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
