package terraform

// MockResourceProvider implements ResourceProvider but mocks out all the
// calls for testing purposes.
type MockResourceProvider struct {
	ConfigureCalled         bool
	ConfigureConfig         map[string]interface{}
	ConfigureReturnWarnings []string
	ConfigureReturnError    error
}

func (p *MockResourceProvider) Configure(c map[string]interface{}) ([]string, error) {
	p.ConfigureCalled = true
	p.ConfigureConfig = c
	return p.ConfigureReturnWarnings, p.ConfigureReturnError
}
