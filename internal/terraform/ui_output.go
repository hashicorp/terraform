// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

// UIOutput is the interface that must be implemented to output
// data to the end user.
type UIOutput interface {
	Output(string)
}
