// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package schema

// Equal is an interface that checks for deep equality between two objects.
type Equal interface {
	Equal(interface{}) bool
}
