// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// GraphWalker is an interface that can be implemented that when used
// with Graph.Walk will invoke the given callbacks under certain events.
type GraphWalker interface {
	EvalContext() EvalContext
	EnterPath(addrs.ModuleInstance) EvalContext
	ExitPath(addrs.ModuleInstance)
	EnterPartialExpandedPath(addrs.PartialExpandedModule) EvalContext
	ExitPartialExpandedPath(addrs.PartialExpandedModule)
	Execute(EvalContext, GraphNodeExecutable) tfdiags.Diagnostics
}

// NullGraphWalker is a GraphWalker implementation that does nothing.
// This can be embedded within other GraphWalker implementations for easily
// implementing all the required functions.
type NullGraphWalker struct{}

var _ GraphWalker = NullGraphWalker{}

func (NullGraphWalker) EvalContext() EvalContext { return new(MockEvalContext) }

func (NullGraphWalker) EnterPath(addrs.ModuleInstance) EvalContext { return new(MockEvalContext) }
func (NullGraphWalker) ExitPath(addrs.ModuleInstance)              {}

func (NullGraphWalker) EnterPartialExpandedPath(addrs.PartialExpandedModule) EvalContext {
	return new(MockEvalContext)
}
func (NullGraphWalker) ExitPartialExpandedPath(addrs.PartialExpandedModule) {}

func (NullGraphWalker) Execute(EvalContext, GraphNodeExecutable) tfdiags.Diagnostics { return nil }
