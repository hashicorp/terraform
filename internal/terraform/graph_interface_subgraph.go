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

// GraphNodePartialExpandedModule is implemented by nodes that represent
// potentially multiple instances of a particular configuration object that
// we haven't fully resolved yet because the expansion of one or more of
// the calling modules was deferred.
type GraphNodePartialExpandedModule interface {
	PartialExpandedModule() addrs.PartialExpandedModule
}

// GraphNodeModuleEvalScope is a dynamic variant of both [GraphNodeModuleInstance]
// and [GraphNodePartialExpandedModule] combined, which a node can implement
// to dynamically choose between:
//
//    - nil, representing no module evaluation scope at all
//    - an [addrs.ModuleInstance], representing an exact module instance with similar effect to implementing [GraphNodeModuleInstance]
//    - an [addrs.PartialExpandedModule], representing a partial-expanded module instance with similar effect to implementing [GraphNodePartialExpandedModule]
//
// When calling a node's [GraphNodeExecutable] implementation, the graph
// walker will pass an [EvalContext] with the appropriate module evaluation
// scope (if any) pre-assigned.
type GraphNodeModuleEvalScope interface {
	ModuleEvalScope() addrs.ModuleEvalScope
}
