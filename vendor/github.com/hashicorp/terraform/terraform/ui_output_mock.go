package terraform

// MockUIOutput is an implementation of UIOutput that can be used for tests.
type MockUIOutput struct {
	OutputCalled  bool
	OutputMessage string
	OutputFn      func(string)
}

func (o *MockUIOutput) Output(v string) {
	o.OutputCalled = true
	o.OutputMessage = v
	if o.OutputFn != nil {
		o.OutputFn(v)
	}
}
