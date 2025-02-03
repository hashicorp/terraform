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
	enterScope(evalContextScope) EvalContext
	exitScope(evalContextScope)
	Execute(EvalContext, GraphNodeExecutable) tfdiags.Diagnostics
	TargetAddrs() addrs.Set[addrs.Targetable]
	ExcludedAddrs() addrs.Set[addrs.Targetable]
}

// NullGraphWalker is a GraphWalker implementation that does nothing.
// This can be embedded within other GraphWalker implementations for easily
// implementing all the required functions.
type NullGraphWalker struct{}

func (NullGraphWalker) EvalContext() EvalContext                                     { return new(MockEvalContext) }
func (NullGraphWalker) enterScope(evalContextScope) EvalContext                      { return new(MockEvalContext) }
func (NullGraphWalker) exitScope(evalContextScope)                                   {}
func (NullGraphWalker) Execute(EvalContext, GraphNodeExecutable) tfdiags.Diagnostics { return nil }
func (NullGraphWalker) TargetAddrs() addrs.Set[addrs.Targetable]                     { return nil }
func (NullGraphWalker) ExcludedAddrs() addrs.Set[addrs.Targetable]                   { return nil }
