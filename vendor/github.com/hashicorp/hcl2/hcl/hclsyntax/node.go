package hclsyntax

import (
	"github.com/hashicorp/hcl2/hcl"
)

// Node is the abstract type that every AST node implements.
//
// This is a closed interface, so it cannot be implemented from outside of
// this package.
type Node interface {
	// This is the mechanism by which the public-facing walk functions
	// are implemented. Implementations should call the given function
	// for each child node and then replace that node with its return value.
	// The return value might just be the same node, for non-transforming
	// walks.
	walkChildNodes(w internalWalkFunc)

	Range() hcl.Range
}

type internalWalkFunc func(Node)
