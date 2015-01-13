package lang

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/config/lang/ast"
)

// Engine is the execution engine for this language. It should be configured
// prior to running Execute.
type Engine struct {
	// GlobalScope is the global scope of execution for this engine.
	GlobalScope *Scope

	// SemanticChecks is a list of additional semantic checks that will be run
	// on the tree prior to executing it. The type checker, identifier checker,
	// etc. will be run before these.
	SemanticChecks []SemanticChecker
}

// SemanticChecker is the type that must be implemented to do a
// semantic check on an AST tree. This will be called with the root node.
type SemanticChecker func(ast.Node) error

// Execute executes the given ast.Node and returns its final value, its
// type, and an error if one exists.
func (e *Engine) Execute(root ast.Node) (interface{}, ast.Type, error) {
	// Run the type checker
	tv := &TypeVisitor{Scope: e.GlobalScope}
	if err := tv.Visit(root); err != nil {
		return nil, ast.TypeInvalid, err
	}

	// Execute
	v := &executeVisitor{Scope: e.GlobalScope}
	return v.Visit(root)
}

// executeVisitor is the visitor used to do the actual execution of
// a program. Note at this point it is assumed that the types check out
// and the identifiers exist.
type executeVisitor struct {
	Scope *Scope

	stack EngineStack
	err   error
	lock  sync.Mutex
}

func (v *executeVisitor) Visit(root ast.Node) (interface{}, ast.Type, error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	// Run the actual visitor pattern
	root.Accept(v.visit)

	// Get our result and clear out everything else
	var result *ast.LiteralNode
	if v.stack.Len() > 0 {
		result = v.stack.Pop()
	} else {
		result = new(ast.LiteralNode)
	}
	resultErr := v.err

	// Clear everything else so we aren't just dangling
	v.stack.Reset()
	v.err = nil

	return result.Value, result.Type, resultErr
}

func (v *executeVisitor) visit(raw ast.Node) {
	if v.err != nil {
		return
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
		v.err = fmt.Errorf("unknown node: %#v", raw)
	}
}

func (v *executeVisitor) visitCall(n *ast.Call) {
	// Look up the function in the map
	function, ok := v.Scope.FuncMap[n.Func]
	if !ok {
		v.err = fmt.Errorf("unknown function called: %s", n.Func)
		return
	}

	// The arguments are on the stack in reverse order, so pop them off.
	args := make([]interface{}, len(n.Args))
	for i, _ := range n.Args {
		node := v.stack.Pop()
		args[len(n.Args)-1-i] = node.Value
	}

	// Call the function
	result, err := function.Callback(args)
	if err != nil {
		v.err = fmt.Errorf("%s: %s", n.Func, err)
		return
	}

	// Push the result
	v.stack.Push(&ast.LiteralNode{
		Value: result,
		Type:  function.ReturnType,
	})
}

func (v *executeVisitor) visitConcat(n *ast.Concat) {
	// The expressions should all be on the stack in reverse
	// order. So pop them off, reverse their order, and concatenate.
	nodes := make([]*ast.LiteralNode, 0, len(n.Exprs))
	for range n.Exprs {
		nodes = append(nodes, v.stack.Pop())
	}

	var buf bytes.Buffer
	for i := len(nodes) - 1; i >= 0; i-- {
		buf.WriteString(nodes[i].Value.(string))
	}

	v.stack.Push(&ast.LiteralNode{
		Value: buf.String(),
		Type:  ast.TypeString,
	})
}

func (v *executeVisitor) visitLiteral(n *ast.LiteralNode) {
	v.stack.Push(n)
}

func (v *executeVisitor) visitVariableAccess(n *ast.VariableAccess) {
	// Look up the variable in the map
	variable, ok := v.Scope.VarMap[n.Name]
	if !ok {
		v.err = fmt.Errorf("unknown variable accessed: %s", n.Name)
		return
	}

	v.stack.Push(&ast.LiteralNode{
		Value: variable.Value,
		Type:  variable.Type,
	})
}

// EngineStack is a stack of ast.LiteralNodes that the Engine keeps track
// of during execution. This is currently backed by a dumb slice, but can be
// replaced with a better data structure at some point in the future if this
// turns out to require optimization.
type EngineStack struct {
	stack []*ast.LiteralNode
}

func (s *EngineStack) Len() int {
	return len(s.stack)
}

func (s *EngineStack) Push(n *ast.LiteralNode) {
	s.stack = append(s.stack, n)
}

func (s *EngineStack) Pop() *ast.LiteralNode {
	x := s.stack[len(s.stack)-1]
	s.stack[len(s.stack)-1] = nil
	s.stack = s.stack[:len(s.stack)-1]
	return x
}

func (s *EngineStack) Reset() {
	s.stack = nil
}

// Scope represents a lookup scope for execution.
type Scope struct {
	// VarMap and FuncMap are the mappings of identifiers to functions
	// and variable values.
	VarMap  map[string]Variable
	FuncMap map[string]Function
}

// Variable is a variable value for execution given as input to the engine.
// It records the value of a variables along with their type.
type Variable struct {
	Value interface{}
	Type  ast.Type
}

// Function defines a function that can be executed by the engine.
// The type checker will validate that the proper types will be called
// to the callback.
type Function struct {
	ArgTypes   []ast.Type
	ReturnType ast.Type
	Callback   func([]interface{}) (interface{}, error)
}

// LookupFunc will look up a variable by name.
// TODO test
func (s *Scope) LookupFunc(n string) (Function, bool) {
	if s == nil {
		return Function{}, false
	}

	v, ok := s.FuncMap[n]
	return v, ok
}

// LookupVar will look up a variable by name.
// TODO test
func (s *Scope) LookupVar(n string) (Variable, bool) {
	if s == nil {
		return Variable{}, false
	}

	v, ok := s.VarMap[n]
	return v, ok
}
