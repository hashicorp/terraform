package terraform

// MockUIInput is an implementation of UIInput that can be used for tests.
type MockUIInput struct {
	InputCalled       bool
	InputOpts         *InputOpts
	InputReturnMap    map[string]string
	InputReturnString string
	InputReturnError  error
	InputFn           func(*InputOpts) (string, error)
}

func (i *MockUIInput) Input(opts *InputOpts) (string, error) {
	i.InputCalled = true
	i.InputOpts = opts
	if i.InputFn != nil {
		return i.InputFn(opts)
	}
	if i.InputReturnMap != nil {
		return i.InputReturnMap[opts.Id], i.InputReturnError
	}
	return i.InputReturnString, i.InputReturnError
}
