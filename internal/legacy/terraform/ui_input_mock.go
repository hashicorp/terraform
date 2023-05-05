// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import "context"

// MockUIInput is an implementation of UIInput that can be used for tests.
type MockUIInput struct {
	InputCalled       bool
	InputOpts         *InputOpts
	InputReturnMap    map[string]string
	InputReturnString string
	InputReturnError  error
	InputFn           func(*InputOpts) (string, error)
}

func (i *MockUIInput) Input(ctx context.Context, opts *InputOpts) (string, error) {
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
