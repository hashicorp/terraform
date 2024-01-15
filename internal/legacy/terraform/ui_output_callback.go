// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

type CallbackUIOutput struct {
	OutputFn func(string)
}

func (o *CallbackUIOutput) Output(v string) {
	o.OutputFn(v)
}
