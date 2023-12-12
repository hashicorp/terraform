// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// GraphWalker is an interface that can be implemented that when used
// with Graph.Walk will invoke the given callbacks under certain events.
type GraphWalker interface {
	EvalContext(context.Context) EvalContext
	EnterPath(context.Context, addrs.ModuleInstance) EvalContext
	ExitPath(addrs.ModuleInstance)
	Execute(EvalContext, GraphNodeExecutable) tfdiags.Diagnostics
}

// NullGraphWalker is a GraphWalker implementation that does nothing.
// This can be embedded within other GraphWalker implementations for easily
// implementing all the required functions.
type NullGraphWalker struct{}

func (NullGraphWalker) EvalContext(context.Context) EvalContext { return new(MockEvalContext) }
func (NullGraphWalker) EnterPath(context.Context, addrs.ModuleInstance) EvalContext {
	return new(MockEvalContext)
}
func (NullGraphWalker) ExitPath(addrs.ModuleInstance)                                {}
func (NullGraphWalker) Execute(EvalContext, GraphNodeExecutable) tfdiags.Diagnostics { return nil }
