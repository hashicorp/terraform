package hil

import (
	"fmt"
	"sync"

	"github.com/hashicorp/hil/ast"
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
	Scope ast.Scope

	// Implicit is a map of implicit type conversions that we can do,
	// and that shouldn't error. The key of the first map is the from type,
	// the key of the second map is the to type, and the final string
	// value is the function to call (which must be registered in the Scope).
	Implicit map[ast.Type]map[ast.Type]string

	// Stack of types. This shouldn't be used directly except by implementations
	// of TypeCheckNode.
	Stack []ast.Type

	err  error
	lock sync.Mutex
}

// TypeCheckNode is the interface that must be implemented by any
// ast.Node that wants to support type-checking. If the type checker
// encounters a node that doesn't implement this, it will error.
type TypeCheckNode interface {
	TypeCheck(*TypeCheck) (ast.Node, error)
}

func (v *TypeCheck) Visit(root ast.Node) error {
	v.lock.Lock()
	defer v.lock.Unlock()
	defer v.reset()
	root.Accept(v.visit)

	// If the resulting type is unknown, then just let the whole thing go.
	if v.err == errExitUnknown {
		v.err = nil
	}

	return v.err
}

func (v *TypeCheck) visit(raw ast.Node) ast.Node {
	if v.err != nil {
		return raw
	}

	var result ast.Node
	var err error
	switch n := raw.(type) {
	case *ast.Arithmetic:
		tc := &typeCheckArithmetic{n}
		result, err = tc.TypeCheck(v)
	case *ast.Call:
		tc := &typeCheckCall{n}
		result, err = tc.TypeCheck(v)
	case *ast.Conditional:
		tc := &typeCheckConditional{n}
		result, err = tc.TypeCheck(v)
	case *ast.Index:
		tc := &typeCheckIndex{n}
		result, err = tc.TypeCheck(v)
	case *ast.Output:
		tc := &typeCheckOutput{n}
		result, err = tc.TypeCheck(v)
	case *ast.LiteralNode:
		tc := &typeCheckLiteral{n}
		result, err = tc.TypeCheck(v)
	case *ast.VariableAccess:
		tc := &typeCheckVariableAccess{n}
		result, err = tc.TypeCheck(v)
	default:
		tc, ok := raw.(TypeCheckNode)
		if !ok {
			err = fmt.Errorf("unknown node for type check: %#v", raw)
			break
		}

		result, err = tc.TypeCheck(v)
	}

	if err != nil {
		pos := raw.Pos()
		v.err = fmt.Errorf("At column %d, line %d: %s",
			pos.Column, pos.Line, err)
	}

	if v.StackPeek() == ast.TypeUnknown {
		v.err = errExitUnknown
	}

	return result
}

type typeCheckArithmetic struct {
	n *ast.Arithmetic
}

func (tc *typeCheckArithmetic) TypeCheck(v *TypeCheck) (ast.Node, error) {
	// The arguments are on the stack in reverse order, so pop them off.
	exprs := make([]ast.Type, len(tc.n.Exprs))
	for i, _ := range tc.n.Exprs {
		exprs[len(tc.n.Exprs)-1-i] = v.StackPop()
	}

	switch tc.n.Op {
	case ast.ArithmeticOpLogicalAnd, ast.ArithmeticOpLogicalOr:
		return tc.checkLogical(v, exprs)
	case ast.ArithmeticOpEqual, ast.ArithmeticOpNotEqual,
		ast.ArithmeticOpLessThan, ast.ArithmeticOpGreaterThan,
		ast.ArithmeticOpGreaterThanOrEqual, ast.ArithmeticOpLessThanOrEqual:
		return tc.checkComparison(v, exprs)
	default:
		return tc.checkNumeric(v, exprs)
	}

}

func (tc *typeCheckArithmetic) checkNumeric(v *TypeCheck, exprs []ast.Type) (ast.Node, error) {
	// Determine the resulting type we want. We do this by going over
	// every expression until we find one with a type we recognize.
	// We do this because the first expr might be a string ("var.foo")
	// and we need to know what to implicit to.
	mathFunc := "__builtin_IntMath"
	mathType := ast.TypeInt
	for _, v := range exprs {
		// We assume int math but if we find ANY float, the entire
		// expression turns into floating point math.
		if v == ast.TypeFloat {
			mathFunc = "__builtin_FloatMath"
			mathType = v
			break
		}
	}

	// Verify the args
	for i, arg := range exprs {
		if arg != mathType {
			cn := v.ImplicitConversion(exprs[i], mathType, tc.n.Exprs[i])
			if cn != nil {
				tc.n.Exprs[i] = cn
				continue
			}

			return nil, fmt.Errorf(
				"operand %d should be %s, got %s",
				i+1, mathType, arg)
		}
	}

	// Modulo doesn't work for floats
	if mathType == ast.TypeFloat && tc.n.Op == ast.ArithmeticOpMod {
		return nil, fmt.Errorf("modulo cannot be used with floats")
	}

	// Return type
	v.StackPush(mathType)

	// Replace our node with a call to the proper function. This isn't
	// type checked but we already verified types.
	args := make([]ast.Node, len(tc.n.Exprs)+1)
	args[0] = &ast.LiteralNode{
		Value: tc.n.Op,
		Typex: ast.TypeInt,
		Posx:  tc.n.Pos(),
	}
	copy(args[1:], tc.n.Exprs)
	return &ast.Call{
		Func: mathFunc,
		Args: args,
		Posx: tc.n.Pos(),
	}, nil
}

func (tc *typeCheckArithmetic) checkComparison(v *TypeCheck, exprs []ast.Type) (ast.Node, error) {
	if len(exprs) != 2 {
		// This should never happen, because the parser never produces
		// nodes that violate this.
		return nil, fmt.Errorf(
			"comparison operators must have exactly two operands",
		)
	}

	// The first operand always dictates the type for a comparison.
	compareFunc := ""
	compareType := exprs[0]
	switch compareType {
	case ast.TypeBool:
		compareFunc = "__builtin_BoolCompare"
	case ast.TypeFloat:
		compareFunc = "__builtin_FloatCompare"
	case ast.TypeInt:
		compareFunc = "__builtin_IntCompare"
	case ast.TypeString:
		compareFunc = "__builtin_StringCompare"
	default:
		return nil, fmt.Errorf(
			"comparison operators apply only to bool, float, int, and string",
		)
	}

	// For non-equality comparisons, we will do implicit conversions to
	// integer types if possible. In this case, we need to go through and
	// determine the type of comparison we're doing to enable the implicit
	// conversion.
	if tc.n.Op != ast.ArithmeticOpEqual && tc.n.Op != ast.ArithmeticOpNotEqual {
		compareFunc = "__builtin_IntCompare"
		compareType = ast.TypeInt
		for _, expr := range exprs {
			if expr == ast.TypeFloat {
				compareFunc = "__builtin_FloatCompare"
				compareType = ast.TypeFloat
				break
			}
		}
	}

	// Verify (and possibly, convert) the args
	for i, arg := range exprs {
		if arg != compareType {
			cn := v.ImplicitConversion(exprs[i], compareType, tc.n.Exprs[i])
			if cn != nil {
				tc.n.Exprs[i] = cn
				continue
			}

			return nil, fmt.Errorf(
				"operand %d should be %s, got %s",
				i+1, compareType, arg,
			)
		}
	}

	// Only ints and floats can have the <, >, <= and >= operators applied
	switch tc.n.Op {
	case ast.ArithmeticOpEqual, ast.ArithmeticOpNotEqual:
		// anything goes
	default:
		switch compareType {
		case ast.TypeFloat, ast.TypeInt:
			// fine
		default:
			return nil, fmt.Errorf(
				"<, >, <= and >= may apply only to int and float values",
			)
		}
	}

	// Comparison operators always return bool
	v.StackPush(ast.TypeBool)

	// Replace our node with a call to the proper function. This isn't
	// type checked but we already verified types.
	args := make([]ast.Node, len(tc.n.Exprs)+1)
	args[0] = &ast.LiteralNode{
		Value: tc.n.Op,
		Typex: ast.TypeInt,
		Posx:  tc.n.Pos(),
	}
	copy(args[1:], tc.n.Exprs)
	return &ast.Call{
		Func: compareFunc,
		Args: args,
		Posx: tc.n.Pos(),
	}, nil
}

func (tc *typeCheckArithmetic) checkLogical(v *TypeCheck, exprs []ast.Type) (ast.Node, error) {
	for i, t := range exprs {
		if t != ast.TypeBool {
			cn := v.ImplicitConversion(t, ast.TypeBool, tc.n.Exprs[i])
			if cn == nil {
				return nil, fmt.Errorf(
					"logical operators require boolean operands, not %s",
					t,
				)
			}
			tc.n.Exprs[i] = cn
		}
	}

	// Return type is always boolean
	v.StackPush(ast.TypeBool)

	// Arithmetic nodes are replaced with a call to a built-in function
	args := make([]ast.Node, len(tc.n.Exprs)+1)
	args[0] = &ast.LiteralNode{
		Value: tc.n.Op,
		Typex: ast.TypeInt,
		Posx:  tc.n.Pos(),
	}
	copy(args[1:], tc.n.Exprs)
	return &ast.Call{
		Func: "__builtin_Logical",
		Args: args,
		Posx: tc.n.Pos(),
	}, nil
}

type typeCheckCall struct {
	n *ast.Call
}

func (tc *typeCheckCall) TypeCheck(v *TypeCheck) (ast.Node, error) {
	// Look up the function in the map
	function, ok := v.Scope.LookupFunc(tc.n.Func)
	if !ok {
		return nil, fmt.Errorf("unknown function called: %s", tc.n.Func)
	}

	// The arguments are on the stack in reverse order, so pop them off.
	args := make([]ast.Type, len(tc.n.Args))
	for i, _ := range tc.n.Args {
		args[len(tc.n.Args)-1-i] = v.StackPop()
	}

	// Verify the args
	for i, expected := range function.ArgTypes {
		if expected == ast.TypeAny {
			continue
		}

		if args[i] != expected {
			cn := v.ImplicitConversion(args[i], expected, tc.n.Args[i])
			if cn != nil {
				tc.n.Args[i] = cn
				continue
			}

			return nil, fmt.Errorf(
				"%s: argument %d should be %s, got %s",
				tc.n.Func, i+1, expected.Printable(), args[i].Printable())
		}
	}

	// If we're variadic, then verify the types there
	if function.Variadic && function.VariadicType != ast.TypeAny {
		args = args[len(function.ArgTypes):]
		for i, t := range args {
			if t != function.VariadicType {
				realI := i + len(function.ArgTypes)
				cn := v.ImplicitConversion(
					t, function.VariadicType, tc.n.Args[realI])
				if cn != nil {
					tc.n.Args[realI] = cn
					continue
				}

				return nil, fmt.Errorf(
					"%s: argument %d should be %s, got %s",
					tc.n.Func, realI,
					function.VariadicType.Printable(), t.Printable())
			}
		}
	}

	// Return type
	v.StackPush(function.ReturnType)

	return tc.n, nil
}

type typeCheckConditional struct {
	n *ast.Conditional
}

func (tc *typeCheckConditional) TypeCheck(v *TypeCheck) (ast.Node, error) {
	// On the stack we have the types of the condition, true and false
	// expressions, but they are in reverse order.
	falseType := v.StackPop()
	trueType := v.StackPop()
	condType := v.StackPop()

	if condType != ast.TypeBool {
		cn := v.ImplicitConversion(condType, ast.TypeBool, tc.n.CondExpr)
		if cn == nil {
			return nil, fmt.Errorf(
				"condition must be type bool, not %s", condType.Printable(),
			)
		}
		tc.n.CondExpr = cn
	}

	// The types of the true and false expression must match
	if trueType != falseType {

		// Since passing around stringified versions of other types is
		// common, we pragmatically allow the false expression to dictate
		// the result type when the true expression is a string.
		if trueType == ast.TypeString {
			cn := v.ImplicitConversion(trueType, falseType, tc.n.TrueExpr)
			if cn == nil {
				return nil, fmt.Errorf(
					"true and false expression types must match; have %s and %s",
					trueType.Printable(), falseType.Printable(),
				)
			}
			tc.n.TrueExpr = cn
			trueType = falseType
		} else {
			cn := v.ImplicitConversion(falseType, trueType, tc.n.FalseExpr)
			if cn == nil {
				return nil, fmt.Errorf(
					"true and false expression types must match; have %s and %s",
					trueType.Printable(), falseType.Printable(),
				)
			}
			tc.n.FalseExpr = cn
			falseType = trueType
		}
	}

	// Currently list and map types cannot be used, because we cannot
	// generally assert that their element types are consistent.
	// Such support might be added later, either by improving the type
	// system or restricting usage to only variable and literal expressions,
	// but for now this is simply prohibited because it doesn't seem to
	// be a common enough case to be worth the complexity.
	switch trueType {
	case ast.TypeList:
		return nil, fmt.Errorf(
			"conditional operator cannot be used with list values",
		)
	case ast.TypeMap:
		return nil, fmt.Errorf(
			"conditional operator cannot be used with map values",
		)
	}

	// Result type (guaranteed to also match falseType due to the above)
	v.StackPush(trueType)

	return tc.n, nil
}

type typeCheckOutput struct {
	n *ast.Output
}

func (tc *typeCheckOutput) TypeCheck(v *TypeCheck) (ast.Node, error) {
	n := tc.n
	types := make([]ast.Type, len(n.Exprs))
	for i, _ := range n.Exprs {
		types[len(n.Exprs)-1-i] = v.StackPop()
	}

	// If there is only one argument and it is a list, we evaluate to a list
	if len(types) == 1 {
		switch t := types[0]; t {
		case ast.TypeList:
			fallthrough
		case ast.TypeMap:
			v.StackPush(t)
			return n, nil
		}
	}

	// Otherwise, all concat args must be strings, so validate that
	for i, t := range types {
		if t != ast.TypeString {
			cn := v.ImplicitConversion(t, ast.TypeString, n.Exprs[i])
			if cn != nil {
				n.Exprs[i] = cn
				continue
			}

			return nil, fmt.Errorf(
				"output of an HIL expression must be a string, or a single list (argument %d is %s)", i+1, t)
		}
	}

	// This always results in type string
	v.StackPush(ast.TypeString)

	return n, nil
}

type typeCheckLiteral struct {
	n *ast.LiteralNode
}

func (tc *typeCheckLiteral) TypeCheck(v *TypeCheck) (ast.Node, error) {
	v.StackPush(tc.n.Typex)
	return tc.n, nil
}

type typeCheckVariableAccess struct {
	n *ast.VariableAccess
}

func (tc *typeCheckVariableAccess) TypeCheck(v *TypeCheck) (ast.Node, error) {
	// Look up the variable in the map
	variable, ok := v.Scope.LookupVar(tc.n.Name)
	if !ok {
		return nil, fmt.Errorf(
			"unknown variable accessed: %s", tc.n.Name)
	}

	// Check if the variable contains any unknown types. If so, then
	// mark it as unknown.
	if ast.IsUnknown(variable) {
		v.StackPush(ast.TypeUnknown)
		return tc.n, nil
	}

	// Add the type to the stack
	v.StackPush(variable.Type)

	return tc.n, nil
}

type typeCheckIndex struct {
	n *ast.Index
}

func (tc *typeCheckIndex) TypeCheck(v *TypeCheck) (ast.Node, error) {
	keyType := v.StackPop()
	targetType := v.StackPop()

	// Ensure we have a VariableAccess as the target
	varAccessNode, ok := tc.n.Target.(*ast.VariableAccess)
	if !ok {
		return nil, fmt.Errorf(
			"target of an index must be a VariableAccess node, was %T", tc.n.Target)
	}

	// Get the variable
	variable, ok := v.Scope.LookupVar(varAccessNode.Name)
	if !ok {
		return nil, fmt.Errorf(
			"unknown variable accessed: %s", varAccessNode.Name)
	}

	switch targetType {
	case ast.TypeList:
		if keyType != ast.TypeInt {
			tc.n.Key = v.ImplicitConversion(keyType, ast.TypeInt, tc.n.Key)
			if tc.n.Key == nil {
				return nil, fmt.Errorf(
					"key of an index must be an int, was %s", keyType)
			}
		}

		valType, err := ast.VariableListElementTypesAreHomogenous(
			varAccessNode.Name, variable.Value.([]ast.Variable))
		if err != nil {
			return tc.n, err
		}

		v.StackPush(valType)
		return tc.n, nil
	case ast.TypeMap:
		if keyType != ast.TypeString {
			tc.n.Key = v.ImplicitConversion(keyType, ast.TypeString, tc.n.Key)
			if tc.n.Key == nil {
				return nil, fmt.Errorf(
					"key of an index must be a string, was %s", keyType)
			}
		}

		valType, err := ast.VariableMapValueTypesAreHomogenous(
			varAccessNode.Name, variable.Value.(map[string]ast.Variable))
		if err != nil {
			return tc.n, err
		}

		v.StackPush(valType)
		return tc.n, nil
	default:
		return nil, fmt.Errorf("invalid index operation into non-indexable type: %s", variable.Type)
	}
}

func (v *TypeCheck) ImplicitConversion(
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
	v.Stack = nil
	v.err = nil
}

func (v *TypeCheck) StackPush(t ast.Type) {
	v.Stack = append(v.Stack, t)
}

func (v *TypeCheck) StackPop() ast.Type {
	var x ast.Type
	x, v.Stack = v.Stack[len(v.Stack)-1], v.Stack[:len(v.Stack)-1]
	return x
}

func (v *TypeCheck) StackPeek() ast.Type {
	if len(v.Stack) == 0 {
		return ast.TypeInvalid
	}

	return v.Stack[len(v.Stack)-1]
}
