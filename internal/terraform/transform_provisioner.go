// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

// GraphNodeProvisionerConsumer is an interface that nodes that require
// a provisioner must implement. ProvisionedBy must return the names of the
// provisioners to use.
type GraphNodeProvisionerConsumer interface {
	ProvisionedBy() []string
}
