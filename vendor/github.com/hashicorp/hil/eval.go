package hil

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/hashicorp/hil/ast"
)

// EvalConfig is the configuration for evaluating.
type EvalConfig struct {
	// GlobalScope is the global scope of execution for evaluation.
	GlobalScope *ast.BasicScope

	// SemanticChecks is a list of additional semantic checks that will be run
	// on the tree prior to evaluating it. The type checker, identifier checker,
	// etc. will be run before these automatically.
	SemanticChecks []SemanticChecker
}

// SemanticChecker is the type that must be implemented to do a
// semantic check on an AST tree. This will be called with the root node.
type SemanticChecker func(ast.Node) error

// EvaluationResult is a struct returned from the hil.Eval function,
// representing the result of an interpolation. Results are returned in their
// "natural" Go structure rather than in terms of the HIL AST.  For the types
// currently implemented, this means that the Value field can be interpreted as
// the following Go types:
//     TypeInvalid: undefined
//     TypeString:  string
//     TypeList:    []interface{}
//     TypeMap:     map[string]interface{}
//     TypBool:     bool
type EvaluationResult struct {
	Type  EvalType
	Value interface{}
}

// InvalidResult is a structure representing the result of a HIL interpolation
// which has invalid syntax, missing variables, or some other type of error.
// The error is described out of band in the accompanying error return value.
var InvalidResult = EvaluationResult{Type: TypeInvalid, Value: nil}

// errExitUnknown is an internal error that when returned means the result
// is an unknown value. We use this for early exit.
var errExitUnknown = errors.New("unknown value")

func Eval(root ast.Node, config *EvalConfig) (EvaluationResult, error) {
	output, outputType, err := internalEval(root, config)
	if err != nil {
		return InvalidResult, err
	}

	// If the result contains any nested unknowns then the result as a whole
	// is unknown, so that callers only have to deal with "entirely known"
	// or "entirely unknown" as outcomes.
	if ast.IsUnknown(ast.Variable{Type: outputType, Value: output}) {
		outputType = ast.TypeUnknown
		output = UnknownValue
	}

	switch outputType {
	case ast.TypeList:
		val, err := VariableToInterface(ast.Variable{
			Type:  ast.TypeList,
			Value: output,
		})
		return EvaluationResult{
			Type:  TypeList,
			Value: val,
		}, err
	case ast.TypeMap:
		val, err := VariableToInterface(ast.Variable{
			Type:  ast.TypeMap,
			Value: output,
		})
		return EvaluationResult{
			Type:  TypeMap,
			Value: val,
		}, err
	case ast.TypeString:
		return EvaluationResult{
			Type:  TypeString,
			Value: output,
		}, nil
	case ast.TypeBool:
		return EvaluationResult{
			Type:  TypeBool,
			Value: output,
		}, nil
	case ast.TypeUnknown:
		return EvaluationResult{
			Type:  TypeUnknown,
			Value: UnknownValue,
		}, nil
	default:
		return InvalidResult, fmt.Errorf("unknown type %s as interpolation output", outputType)
	}
}

// Eval evaluates the given AST tree and returns its output value, the type
// of the output, and any error that occurred.
func internalEval(root ast.Node, config *EvalConfig) (interface{}, ast.Type, error) {
	// Copy the scope so we can add our builtins
	if config == nil {
		config = new(EvalConfig)
	}
	scope := registerBuiltins(config.GlobalScope)
	implicitMap := map[ast.Type]map[ast.Type]string{
		ast.TypeFloat: {
			ast.TypeInt:    "__builtin_FloatToInt",
			ast.TypeString: "__builtin_FloatToString",
		},
		ast.TypeInt: {
			ast.TypeFloat:  "__builtin_IntToFloat",
			ast.TypeString: "__builtin_IntToString",
		},
		ast.TypeString: {
			ast.TypeInt:   "__builtin_StringToInt",
			ast.TypeFloat: "__builtin_StringToFloat",
			ast.TypeBool:  "__builtin_StringToBool",
		},
		ast.TypeBool: {
			ast.TypeString: "__builtin_BoolToString",
		},
	}

	// Build our own semantic checks that we always run
	tv := &TypeCheck{Scope: scope, Implicit: implicitMap}
	ic := &IdentifierCheck{Scope: scope}

	// Build up the semantic checks for execution
	checks := make(
		[]SemanticChecker,
		len(config.SemanticChecks),
		len(config.SemanticChecks)+2)
	copy(checks, config.SemanticChecks)
	checks = append(checks, ic.Visit)
	checks = append(checks, tv.Visit)

	// Run the semantic checks
	for _, check := range checks {
		if err := check(root); err != nil {
			return nil, ast.TypeInvalid, err
		}
	}

	// Execute
	v := &evalVisitor{Scope: scope}
	return v.Visit(root)
}

// EvalNode is the interface that must be implemented by any ast.Node
// to support evaluation. This will be called in visitor pattern order.
// The result of each call to Eval is automatically pushed onto the
// stack as a LiteralNode. Pop elements off the stack to get child
// values.
type EvalNode interface {
	Eval(ast.Scope, *ast.Stack) (interface{}, ast.Type, error)
}

type evalVisitor struct {
	Scope ast.Scope
	Stack ast.Stack

	err  error
	lock sync.Mutex
}

func (v *evalVisitor) Visit(root ast.Node) (interface{}, ast.Type, error) {
	// Run the actual visitor pattern
	root.Accept(v.visit)

	// Get our result and clear out everything else
	var result *ast.LiteralNode
	if v.Stack.Len() > 0 {
		result = v.Stack.Pop().(*ast.LiteralNode)
	} else {
		result = new(ast.LiteralNode)
	}
	resultErr := v.err
	if resultErr == errExitUnknown {
		// This means the return value is unknown and we used the error
		// as an early exit mechanism. Reset since the value on the stack
		// should be the unknown value.
		resultErr = nil
	}

	// Clear everything else so we aren't just dangling
	v.Stack.Reset()
	v.err = nil

	t, err := result.Type(v.Scope)
	if err != nil {
		return nil, ast.TypeInvalid, err
	}

	return result.Value, t, resultErr
}

func (v *evalVisitor) visit(raw ast.Node) ast.Node {
	if v.err != nil {
		return raw
	}

	en, err := evalNode(raw)
	if err != nil {
		v.err = err
		return raw
	}

	out, outType, err := en.Eval(v.Scope, &v.Stack)
	if err != nil {
		v.err = err
		return raw
	}

	v.Stack.Push(&ast.LiteralNode{
		Value: out,
		Typex: outType,
	})

	if outType == ast.TypeUnknown {
		// Halt immediately
		v.err = errExitUnknown
		return raw
	}

	return raw
}

// evalNode is a private function that returns an EvalNode for built-in
// types as well as any other EvalNode implementations.
func evalNode(raw ast.Node) (EvalNode, error) {
	switch n := raw.(type) {
	case *ast.Index:
		return &evalIndex{n}, nil
	case *ast.Call:
		return &evalCall{n}, nil
	case *ast.Conditional:
		return &evalConditional{n}, nil
	case *ast.Output:
		return &evalOutput{n}, nil
	case *ast.LiteralNode:
		return &evalLiteralNode{n}, nil
	case *ast.VariableAccess:
		return &evalVariableAccess{n}, nil
	default:
		en, ok := n.(EvalNode)
		if !ok {
			return nil, fmt.Errorf("node doesn't support evaluation: %#v", raw)
		}

		return en, nil
	}
}

type evalCall struct{ *ast.Call }

func (v *evalCall) Eval(s ast.Scope, stack *ast.Stack) (interface{}, ast.Type, error) {
	// Look up the function in the map
	function, ok := s.LookupFunc(v.Func)
	if !ok {
		return nil, ast.TypeInvalid, fmt.Errorf(
			"unknown function called: %s", v.Func)
	}

	// The arguments are on the stack in reverse order, so pop them off.
	args := make([]interface{}, len(v.Args))
	for i, _ := range v.Args {
		node := stack.Pop().(*ast.LiteralNode)
		if node.IsUnknown() {
			// If any arguments are unknown then the result is automatically unknown
			return UnknownValue, ast.TypeUnknown, nil
		}
		args[len(v.Args)-1-i] = node.Value
	}

	// Call the function
	result, err := function.Callback(args)
	if err != nil {
		return nil, ast.TypeInvalid, fmt.Errorf("%s: %s", v.Func, err)
	}

	return result, function.ReturnType, nil
}

type evalConditional struct{ *ast.Conditional }

func (v *evalConditional) Eval(s ast.Scope, stack *ast.Stack) (interface{}, ast.Type, error) {
	// On the stack we have literal nodes representing the resulting values
	// of the condition, true and false expressions, but they are in reverse
	// order.
	falseLit := stack.Pop().(*ast.LiteralNode)
	trueLit := stack.Pop().(*ast.LiteralNode)
	condLit := stack.Pop().(*ast.LiteralNode)

	if condLit.IsUnknown() {
		// If our conditional is unknown then our result is also unknown
		return UnknownValue, ast.TypeUnknown, nil
	}

	if condLit.Value.(bool) {
		return trueLit.Value, trueLit.Typex, nil
	} else {
		return falseLit.Value, trueLit.Typex, nil
	}
}

type evalIndex struct{ *ast.Index }

func (v *evalIndex) Eval(scope ast.Scope, stack *ast.Stack) (interface{}, ast.Type, error) {
	key := stack.Pop().(*ast.LiteralNode)
	target := stack.Pop().(*ast.LiteralNode)

	variableName := v.Index.Target.(*ast.VariableAccess).Name

	if key.IsUnknown() {
		// If our key is unknown then our result is also unknown
		return UnknownValue, ast.TypeUnknown, nil
	}

	// For target, we'll accept collections containing unknown values but
	// we still need to catch when the collection itself is unknown, shallowly.
	if target.Typex == ast.TypeUnknown {
		return UnknownValue, ast.TypeUnknown, nil
	}

	switch target.Typex {
	case ast.TypeList:
		return v.evalListIndex(variableName, target.Value, key.Value)
	case ast.TypeMap:
		return v.evalMapIndex(variableName, target.Value, key.Value)
	default:
		return nil, ast.TypeInvalid, fmt.Errorf(
			"target %q for indexing must be ast.TypeList or ast.TypeMap, is %s",
			variableName, target.Typex)
	}
}

func (v *evalIndex) evalListIndex(variableName string, target interface{}, key interface{}) (interface{}, ast.Type, error) {
	// We assume type checking was already done and we can assume that target
	// is a list and key is an int
	list, ok := target.([]ast.Variable)
	if !ok {
		return nil, ast.TypeInvalid, fmt.Errorf(
			"cannot cast target to []Variable, is: %T", target)
	}

	keyInt, ok := key.(int)
	if !ok {
		return nil, ast.TypeInvalid, fmt.Errorf(
			"cannot cast key to int, is: %T", key)
	}

	if len(list) == 0 {
		return nil, ast.TypeInvalid, fmt.Errorf("list is empty")
	}

	if keyInt < 0 || len(list) < keyInt+1 {
		return nil, ast.TypeInvalid, fmt.Errorf(
			"index %d out of range for list %s (max %d)",
			keyInt, variableName, len(list))
	}

	returnVal := list[keyInt].Value
	returnType := list[keyInt].Type
	return returnVal, returnType, nil
}

func (v *evalIndex) evalMapIndex(variableName string, target interface{}, key interface{}) (interface{}, ast.Type, error) {
	// We assume type checking was already done and we can assume that target
	// is a map and key is a string
	vmap, ok := target.(map[string]ast.Variable)
	if !ok {
		return nil, ast.TypeInvalid, fmt.Errorf(
			"cannot cast target to map[string]Variable, is: %T", target)
	}

	keyString, ok := key.(string)
	if !ok {
		return nil, ast.TypeInvalid, fmt.Errorf(
			"cannot cast key to string, is: %T", key)
	}

	if len(vmap) == 0 {
		return nil, ast.TypeInvalid, fmt.Errorf("map is empty")
	}

	value, ok := vmap[keyString]
	if !ok {
		return nil, ast.TypeInvalid, fmt.Errorf(
			"key %q does not exist in map %s", keyString, variableName)
	}

	return value.Value, value.Type, nil
}

type evalOutput struct{ *ast.Output }

func (v *evalOutput) Eval(s ast.Scope, stack *ast.Stack) (interface{}, ast.Type, error) {
	// The expressions should all be on the stack in reverse
	// order. So pop them off, reverse their order, and concatenate.
	nodes := make([]*ast.LiteralNode, 0, len(v.Exprs))
	haveUnknown := false
	for range v.Exprs {
		n := stack.Pop().(*ast.LiteralNode)
		nodes = append(nodes, n)

		// If we have any unknowns then the whole result is unknown
		// (we must deal with this first, because the type checker can
		// skip type conversions in the presence of unknowns, and thus
		// any of our other nodes may be incorrectly typed.)
		if n.IsUnknown() {
			haveUnknown = true
		}
	}

	if haveUnknown {
		return UnknownValue, ast.TypeUnknown, nil
	}

	// Special case the single list and map
	if len(nodes) == 1 {
		switch t := nodes[0].Typex; t {
		case ast.TypeList:
			fallthrough
		case ast.TypeMap:
			fallthrough
		case ast.TypeUnknown:
			return nodes[0].Value, t, nil
		}
	}

	// Otherwise concatenate the strings
	var buf bytes.Buffer
	for i := len(nodes) - 1; i >= 0; i-- {
		if nodes[i].Typex != ast.TypeString {
			return nil, ast.TypeInvalid, fmt.Errorf(
				"invalid output with %s value at index %d: %#v",
				nodes[i].Typex,
				i,
				nodes[i].Value,
			)
		}
		buf.WriteString(nodes[i].Value.(string))
	}

	return buf.String(), ast.TypeString, nil
}

type evalLiteralNode struct{ *ast.LiteralNode }

func (v *evalLiteralNode) Eval(ast.Scope, *ast.Stack) (interface{}, ast.Type, error) {
	return v.Value, v.Typex, nil
}

type evalVariableAccess struct{ *ast.VariableAccess }

func (v *evalVariableAccess) Eval(scope ast.Scope, _ *ast.Stack) (interface{}, ast.Type, error) {
	// Look up the variable in the map
	variable, ok := scope.LookupVar(v.Name)
	if !ok {
		return nil, ast.TypeInvalid, fmt.Errorf(
			"unknown variable accessed: %s", v.Name)
	}

	return variable.Value, variable.Type, nil
}
