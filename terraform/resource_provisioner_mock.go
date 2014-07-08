package terraform

// MockResourceProvisioner implements ResourceProvisioner but mocks out all the
// calls for testing purposes.
type MockResourceProvisioner struct {
	// Anything you want, in case you need to store extra data with the mock.
	Meta interface{}

	ApplyCalled      bool
	ApplyState       *ResourceState
	ApplyConfig      *ResourceConfig
	ApplyFn          func(*ResourceState, *ResourceConfig) (*ResourceState, error)
	ApplyReturn      *ResourceState
	ApplyReturnError error

	ValidateCalled       bool
	ValidateConfig       *ResourceConfig
	ValidateReturnWarns  []string
	ValidateReturnErrors []error
}

func (p *MockResourceProvisioner) Validate(c *ResourceConfig) ([]string, []error) {
	p.ValidateCalled = true
	p.ValidateConfig = c
	return p.ValidateReturnWarns, p.ValidateReturnErrors
}

func (p *MockResourceProvisioner) Apply(state *ResourceState, c *ResourceConfig) (*ResourceState, error) {
	p.ApplyCalled = true
	p.ApplyState = state
	p.ApplyConfig = c
	if p.ApplyFn != nil {
		return p.ApplyFn(state, c)
	}
	return p.ApplyReturn, p.ApplyReturnError
}
