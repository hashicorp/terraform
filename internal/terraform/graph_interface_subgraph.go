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
