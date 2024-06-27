// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// GraphNodeExecutable is the interface that graph nodes must implement to
// enable execution.
//
// Don't type-assert for this interface directly. Instead, use
// [executeGraphNode] which arranges for this interface to be used correctly
// when called for nodes that implement it.
type GraphNodeExecutable interface {
	Execute(EvalContext, walkOperation) tfdiags.Diagnostics
}

// GraphNodeExecutableSema is a variation of [GraphNodeExecutable] for
// situations where the node implementation needs to handle acquiring and
// releasing semaphore slots itself.
//
// This can be helpful for node types that encapsulate a large number of
// sequential steps, to enable the implementation to acquire semaphore
// only for the duration of each nested step, rather than for the entire
// duration of the Execute method. Implementations of this interface should
// hold the given semaphore whenever doing I/O-bound work such as provider
// plugin calls, but need not hold it while performing simple logic.
//
// This interface is intentionally mutually-exclusive with [GraphNodeExecutable].
// Each node type can only implement one of these interfaces.
//
// Don't type-assert for this interface directly. Instead, use
// [executeGraphNode] which arranges for this interface to be used correctly
// when called for nodes that implement it.
type GraphNodeExecutableSema interface {
	Execute(EvalContext, walkOperation, Semaphore) tfdiags.Diagnostics
}

// executeGraphNode executes any behavior defined for the given graph node
// that is expected to run when visiting the node during a graph walk.
//
// If the given vertex does not have any executable behavior then this is
// a safe no-op.
func executeGraphNode(n dag.Vertex, ctx EvalContext, op walkOperation, concurrencySemaphore Semaphore) tfdiags.Diagnostics {
	switch n := n.(type) {
	case GraphNodeExecutable:
		// Nodes that implement this interface expect us to handle concurrency
		// limits automatically, so we'll acquire the semaphore ourselves
		// before calling.
		concurrencySemaphore.Acquire()
		diags := n.Execute(ctx, op)
		concurrencySemaphore.Release()
		return diags
	case GraphNodeExecutableSema:
		// Nodes that implement this interface are signing up to handle the
		// concurrency limits themselves by interacting directly with the
		// concurrency semaphore.
		return n.Execute(ctx, op, concurrencySemaphore)
	default:
		// Nodes that implement none of these interfaces are not executable,
		// so we'll treat them as no-op.
		return nil
	}
}
