package hclsyntax

import (
	"github.com/hashicorp/hcl2/hcl"
)

// VisitFunc is the callback signature for VisitAll.
type VisitFunc func(node Node) hcl.Diagnostics

// VisitAll is a basic way to traverse the AST beginning with a particular
// node. The given function will be called once for each AST node in
// depth-first order, but no context is provided about the shape of the tree.
//
// The VisitFunc may return diagnostics, in which case they will be accumulated
// and returned as a single set.
func VisitAll(node Node, f VisitFunc) hcl.Diagnostics {
	diags := f(node)
	node.walkChildNodes(func(node Node) Node {
		diags = append(diags, VisitAll(node, f)...)
		return node
	})
	return diags
}

// Walker is an interface used with Walk.
type Walker interface {
	Enter(node Node) hcl.Diagnostics
	Exit(node Node) hcl.Diagnostics
}

// Walk is a more complex way to traverse the AST starting with a particular
// node, which provides information about the tree structure via separate
// Enter and Exit functions.
func Walk(node Node, w Walker) hcl.Diagnostics {
	diags := w.Enter(node)
	node.walkChildNodes(func(node Node) Node {
		diags = append(diags, Walk(node, w)...)
		return node
	})
	return diags
}

// Transformer is an interface used with Transform
type Transformer interface {
	// Transform accepts a node and returns a replacement node along with
	// a flag for whether to also visit child nodes. If the flag is false,
	// none of the child nodes will be visited and the TransformExit method
	// will not be called for the node.
	//
	// It is acceptable and appropriate for Transform to return the same node
	// it was given, for situations where no transform is needed.
	Transform(node Node) (Node, bool, hcl.Diagnostics)

	// TransformExit signals the end of transformations of child nodes of the
	// given node. If Transform returned a new node, the given node is the
	// node that was returned, rather than the node that was originally
	// encountered.
	TransformExit(node Node) hcl.Diagnostics
}

// Transform allows for in-place transformations of an AST starting with a
// particular node. The provider Transformer implementation drives the
// transformation process. The return value is the node that replaced the
// given top-level node.
func Transform(node Node, t Transformer) (Node, hcl.Diagnostics) {
	newNode, descend, diags := t.Transform(node)
	if !descend {
		return newNode, diags
	}
	node.walkChildNodes(func(node Node) Node {
		newNode, newDiags := Transform(node, t)
		diags = append(diags, newDiags...)
		return newNode
	})
	diags = append(diags, t.TransformExit(newNode)...)
	return newNode, diags
}
