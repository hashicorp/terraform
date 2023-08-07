// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import "github.com/hashicorp/terraform/internal/addrs"

// nodeExternalReference allows external callers (such as the testing framework)
// to provide the list of references they are making into the graph. This
// ensures that Terraform will not remove any nodes from the graph that might
// not be referenced from within a module but are referenced by the currently
// executing test file.
//
// This should only be added to the graph if we are executing the
// `terraform test` command.
type nodeExternalReference struct {
	ExternalReferences []*addrs.Reference
}

var (
	_ GraphNodeReferencer = (*nodeExternalReference)(nil)
)

// GraphNodeModulePath
func (n *nodeExternalReference) ModulePath() addrs.Module {
	// The external references are always made from test files, which currently
	// execute as if they are in the root module.
	return addrs.RootModule
}

// GraphNodeReferencer
func (n *nodeExternalReference) References() []*addrs.Reference {
	return n.ExternalReferences
}
