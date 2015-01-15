package lang

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/config/lang/ast"
)

// TypeCheck implements ast.Visitor for type checking an AST tree.
// It requires some configuration to look up the type of nodes.
//
// It also optionally will not type error and will insert an implicit
// type conversions for specific types if specified by the Implicit
// field. Note that this is kind of organizationally weird to put into
// this structure but we'd rather do that than duplicate the type checking
// logic multiple times.
type TypeCheck struct {
	Scope *Scope

	// Implicit is a map of implicit type conversions that we can do,
	// and that shouldn't error. The key of the first map is the from type,
	// the key of the second map is the to type, and the final string
	// value is the function to call (which must be registered in the Scope).
	Implicit map[ast.Type]map[ast.Type]string

	stack []ast.Type
	err   error
	lock  sync.Mutex
}

func (v *TypeCheck) Visit(root ast.Node) error {
	v.lock.Lock()
	defer v.lock.Unlock()
	defer v.reset()
	root.Accept(v.visit)
	return v.err
}

func (v *TypeCheck) visit(raw ast.Node) ast.Node {
	if v.err != nil {
		return raw
	}

	switch n := raw.(type) {
	case *ast.Call:
		v.visitCall(n)
	case *ast.Concat:
		v.visitConcat(n)
	case *ast.LiteralNode:
		v.visitLiteral(n)
	case *ast.VariableAccess:
		v.visitVariableAccess(n)
	default:
		v.createErr(n, fmt.Sprintf("unknown node: %#v", raw))
	}

	return raw
}

func (v *TypeCheck) visitCall(n *ast.Call) {
	// Look up the function in the map
	function, ok := v.Scope.LookupFunc(n.Func)
	if !ok {
		v.createErr(n, fmt.Sprintf("unknown function called: %s", n.Func))
		return
	}

	// The arguments are on the stack in reverse order, so pop them off.
	args := make([]ast.Type, len(n.Args))
	for i, _ := range n.Args {
		args[len(n.Args)-1-i] = v.stackPop()
	}

	// Verify the args
	for i, expected := range function.ArgTypes {
		if args[i] != expected {
			cn := v.implicitConversion(args[i], expected, n.Args[i])
			if cn != nil {
				n.Args[i] = cn
				continue
			}

			v.createErr(n, fmt.Sprintf(
				"%s: argument %d should be %s, got %s",
				n.Func, i+1, expected, args[i]))
			return
		}
	}

	// If we're variadic, then verify the types there
	if function.Variadic {
		args = args[len(function.ArgTypes):]
		for i, t := range args {
			if t != function.VariadicType {
				realI := i + len(function.ArgTypes)
				cn := v.implicitConversion(
					t, function.VariadicType, n.Args[realI])
				if cn != nil {
					n.Args[realI] = cn
					continue
				}

				v.createErr(n, fmt.Sprintf(
					"%s: argument %d should be %s, got %s",
					n.Func, realI,
					function.VariadicType, t))
				return
			}
		}
	}

	// Return type
	v.stackPush(function.ReturnType)
}

func (v *TypeCheck) visitConcat(n *ast.Concat) {
	types := make([]ast.Type, len(n.Exprs))
	for i, _ := range n.Exprs {
		types[len(n.Exprs)-1-i] = v.stackPop()
	}

	// All concat args must be strings, so validate that
	for i, t := range types {
		if t != ast.TypeString {
			cn := v.implicitConversion(t, ast.TypeString, n.Exprs[i])
			if cn != nil {
				n.Exprs[i] = cn
				continue
			}

			v.createErr(n, fmt.Sprintf(
				"argument %d must be a string", i+1))
			return
		}
	}

	// This always results in type string
	v.stackPush(ast.TypeString)
}

func (v *TypeCheck) visitLiteral(n *ast.LiteralNode) {
	v.stackPush(n.Type)
}

func (v *TypeCheck) visitVariableAccess(n *ast.VariableAccess) {
	// Look up the variable in the map
	variable, ok := v.Scope.LookupVar(n.Name)
	if !ok {
		v.createErr(n, fmt.Sprintf(
			"unknown variable accessed: %s", n.Name))
		return
	}

	// Add the type to the stack
	v.stackPush(variable.Type)
}

func (v *TypeCheck) createErr(n ast.Node, str string) {
	pos := n.Pos()
	v.err = fmt.Errorf("At column %d, line %d: %s",
		pos.Column, pos.Line, str)
}

func (v *TypeCheck) implicitConversion(
	actual ast.Type, expected ast.Type, n ast.Node) ast.Node {
	if v.Implicit == nil {
		return nil
	}

	fromMap, ok := v.Implicit[actual]
	if !ok {
		return nil
	}

	toFunc, ok := fromMap[expected]
	if !ok {
		return nil
	}

	return &ast.Call{
		Func: toFunc,
		Args: []ast.Node{n},
		Posx: n.Pos(),
	}
}

func (v *TypeCheck) reset() {
	v.stack = nil
	v.err = nil
}

func (v *TypeCheck) stackPush(t ast.Type) {
	v.stack = append(v.stack, t)
}

func (v *TypeCheck) stackPop() ast.Type {
	var x ast.Type
	x, v.stack = v.stack[len(v.stack)-1], v.stack[:len(v.stack)-1]
	return x
}
