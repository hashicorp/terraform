package lang

import (
	"fmt"

	"github.com/hashicorp/terraform/config/lang/ast"
)

// LookupType looks up the type of the given node with the given scope.
func LookupType(raw ast.Node, scope *Scope) (ast.Type, error) {
	switch n := raw.(type) {
	case *ast.LiteralNode:
		return typedLiteralNode{n}.Type(scope)
	case *ast.VariableAccess:
		return typedVariableAccess{n}.Type(scope)
	default:
		if t, ok := raw.(TypedNode); ok {
			return t.Type(scope)
		}

		return ast.TypeInvalid, fmt.Errorf(
			"unknown node to get type of: %T", raw)
	}
}

// TypedNode is an interface that custom AST nodes should implement
// if they want to work with LookupType. All the builtin AST nodes have
// implementations of this.
type TypedNode interface {
	Type(*Scope) (ast.Type, error)
}

type typedLiteralNode struct {
	n *ast.LiteralNode
}

func (n typedLiteralNode) Type(s *Scope) (ast.Type, error) {
	return n.n.Type, nil
}

type typedVariableAccess struct {
	n *ast.VariableAccess
}

func (n typedVariableAccess) Type(s *Scope) (ast.Type, error) {
	v, ok := s.LookupVar(n.n.Name)
	if !ok {
		return ast.TypeInvalid, fmt.Errorf(
			"%s: couldn't find variable %s", n.n.Pos(), n.n.Name)
	}

	return v.Type, nil
}
