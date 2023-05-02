// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import "sync"

// MockUIOutput is an implementation of UIOutput that can be used for tests.
type MockUIOutput struct {
	sync.Mutex
	OutputCalled  bool
	OutputMessage string
	OutputFn      func(string)
}

func (o *MockUIOutput) Output(v string) {
	o.Lock()
	defer o.Unlock()
	o.OutputCalled = true
	o.OutputMessage = v
	if o.OutputFn != nil {
		o.OutputFn(v)
	}
}
