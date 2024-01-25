// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

// GraphNodeModuleInstance says that a node is part of a graph with a
// different path, and the context should be adjusted accordingly.
type GraphNodeModuleInstance interface {
	Path() addrs.ModuleInstance
}

// GraphNodeModulePath is implemented by all referenceable nodes, to indicate
// their configuration path in unexpanded modules.
type GraphNodeModulePath interface {
	ModulePath() addrs.Module
}

// GraphNodePartialExpandedModule says that a node represents an unbounded
// set of objects within an unbounded set of module instances that happen
// to share a known address prefix.
//
// Nodes of this type typically produce placeholder data to support partial
// evaluation despite the full analysis of a module being deferred to a future
// plan when more information will be available. They might also perform
// checks and raise errors when something can be proven to be definitely
// invalid regardless of what the final set of module instances turns out to
// be.
//
// Node types implementing this interface cannot also implement
// [GraphNodeModuleInstance], because it is not possible to evaluate a
// node in two different contexts at once.
type GraphNodePartialExpandedModule interface {
	Path() addrs.PartialExpandedModule
}
