package terraform

// MockResourceProvider implements ResourceProvider but mocks out all the
// calls for testing purposes.
type MockResourceProvider struct {
	// Anything you want, in case you need to store extra data with the mock.
	Meta interface{}

	ApplyCalled                  bool
	ApplyState                   *ResourceState
	ApplyDiff                    *ResourceDiff
	ApplyFn                      func(*ResourceState, *ResourceDiff) (*ResourceState, error)
	ApplyReturn                  *ResourceState
	ApplyReturnError             error
	ConfigureCalled              bool
	ConfigureConfig              *ResourceConfig
	ConfigureReturnError         error
	DiffCalled                   bool
	DiffState                    *ResourceState
	DiffDesired                  *ResourceConfig
	DiffFn                       func(*ResourceState, *ResourceConfig) (*ResourceDiff, error)
	DiffReturn                   *ResourceDiff
	DiffReturnError              error
	RefreshCalled                bool
	RefreshState                 *ResourceState
	RefreshFn                    func(*ResourceState) (*ResourceState, error)
	RefreshReturn                *ResourceState
	RefreshReturnError           error
	ResourcesCalled              bool
	ResourcesReturn              []ResourceType
	ValidateCalled               bool
	ValidateConfig               *ResourceConfig
	ValidateReturnWarns          []string
	ValidateReturnErrors         []error
	ValidateResourceCalled       bool
	ValidateResourceType         string
	ValidateResourceConfig       *ResourceConfig
	ValidateResourceReturnWarns  []string
	ValidateResourceReturnErrors []error
}

func (p *MockResourceProvider) Validate(c *ResourceConfig) ([]string, []error) {
	p.ValidateCalled = true
	p.ValidateConfig = c
	return p.ValidateReturnWarns, p.ValidateReturnErrors
}

func (p *MockResourceProvider) ValidateResource(t string, c *ResourceConfig) ([]string, []error) {
	p.ValidateResourceCalled = true
	p.ValidateResourceType = t
	p.ValidateResourceConfig = c
	return p.ValidateResourceReturnWarns, p.ValidateResourceReturnErrors
}

func (p *MockResourceProvider) Configure(c *ResourceConfig) error {
	p.ConfigureCalled = true
	p.ConfigureConfig = c
	return p.ConfigureReturnError
}

func (p *MockResourceProvider) Apply(
	state *ResourceState,
	diff *ResourceDiff) (*ResourceState, error) {
	p.ApplyCalled = true
	p.ApplyState = state
	p.ApplyDiff = diff
	if p.ApplyFn != nil {
		return p.ApplyFn(state, diff)
	}

	return p.ApplyReturn, p.ApplyReturnError
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

func (p *MockResourceProvider) Refresh(
	s *ResourceState) (*ResourceState, error) {
	p.RefreshCalled = true
	p.RefreshState = s

	if p.RefreshFn != nil {
		return p.RefreshFn(s)
	}

	return p.RefreshReturn, p.RefreshReturnError
}

func (p *MockResourceProvider) Resources() []ResourceType {
	p.ResourcesCalled = true
	return p.ResourcesReturn
}
