package lang

import (
	"github.com/hashicorp/terraform/config/lang/ast"
)

// FixedValueTransform transforms an AST to return a fixed value for
// all interpolations. i.e. you can make "hello ${anything}" always
// turn into "hello foo".
func FixedValueTransform(root ast.Node, Value *ast.LiteralNode) ast.Node {
	// We visit the nodes in top-down order
	result := root
	switch n := result.(type) {
	case *ast.Concat:
		for i, v := range n.Exprs {
			n.Exprs[i] = FixedValueTransform(v, Value)
		}
	case *ast.LiteralNode:
		// We keep it as-is
	default:
		// Anything else we replace
		result = Value
	}

	return result
}
