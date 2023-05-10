// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

// GraphNodeDynamicExpandable is an interface that nodes can implement
// to signal that they can be expanded at eval-time (hence dynamic).
// These nodes are given the eval context and are expected to return
// a new subgraph.
type GraphNodeDynamicExpandable interface {
	// DynamicExpand returns a new graph which will be treated as the dynamic
	// subgraph of the receiving node.
	//
	// The second return value is of type error for historical reasons;
	// it's valid (and most ideal) for DynamicExpand to return the result
	// of calling ErrWithWarnings on a tfdiags.Diagnostics value instead,
	// in which case the caller will unwrap it and gather the individual
	// diagnostics.
	DynamicExpand(EvalContext) (*Graph, error)
}
