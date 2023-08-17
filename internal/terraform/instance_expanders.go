// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mnptu

// graphNodeExpandsInstances is implemented by nodes that causes instances to
// be registered in the instances.Expander.
type graphNodeExpandsInstances interface {
	expandsInstances()
}
