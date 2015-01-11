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
	// VarMap and FuncMap are the mappings of identifiers to functions
	// and variable values.
	VarMap  map[string]Variable
	FuncMap map[string]Function

	// SemanticChecks is a list of additional semantic checks that will be run
	// on the tree prior to executing it. The type checker, identifier checker,
	// etc. will be run before these.
	SemanticChecks []SemanticChecker
}

// SemanticChecker is the type that must be implemented to do a
// semantic check on an AST tree. This will be called with the root node.
type SemanticChecker func(ast.Node) error

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
	Name     string
	ArgTypes []ast.Type
	Callback func([]interface{}) (interface{}, ast.Type, error)
}

// Execute executes the given ast.Node and returns its final value, its
// type, and an error if one exists.
func (e *Engine) Execute(root ast.Node) (interface{}, ast.Type, error) {
	v := &executeVisitor{Engine: e}
	return v.Visit(root)
}

// executeVisitor is the visitor used to do the actual execution of
// a program. Note at this point it is assumed that the types check out
// and the identifiers exist.
type executeVisitor struct {
	Engine *Engine

	stack []*ast.LiteralNode
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
	if len(v.stack) > 0 {
		result = v.stack[len(v.stack)-1]
	} else {
		result = new(ast.LiteralNode)
	}
	resultErr := v.err

	// Clear everything else so we aren't just dangling
	v.stack = nil
	v.err = nil

	return result.Value, result.Type, resultErr
}

func (v *executeVisitor) visit(raw ast.Node) {
	if v.err != nil {
		return
	}

	switch n := raw.(type) {
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

func (v *executeVisitor) visitConcat(n *ast.Concat) {
	// The expressions should all be on the stack in reverse
	// order. So pop them off, reverse their order, and concatenate.
	nodes := make([]*ast.LiteralNode, 0, len(n.Exprs))
	for range n.Exprs {
		nodes = append(nodes, v.stackPop())
	}

	var buf bytes.Buffer
	for i := len(nodes) - 1; i >= 0; i-- {
		buf.WriteString(nodes[i].Value.(string))
	}

	v.stackPush(&ast.LiteralNode{
		Value: buf.String(),
		Type:  ast.TypeString,
	})
}

func (v *executeVisitor) visitLiteral(n *ast.LiteralNode) {
	v.stack = append(v.stack, n)
}

func (v *executeVisitor) visitVariableAccess(n *ast.VariableAccess) {
	// Look up the variable in the map
	variable, ok := v.Engine.VarMap[n.Name]
	if !ok {
		v.err = fmt.Errorf("unknown variable accessed: %s", n.Name)
		return
	}

	v.stack = append(v.stack, &ast.LiteralNode{
		Value: variable.Value,
		Type:  variable.Type,
	})
}

func (v *executeVisitor) stackPush(n *ast.LiteralNode) {
	v.stack = append(v.stack, n)
}

func (v *executeVisitor) stackPop() *ast.LiteralNode {
	var x *ast.LiteralNode
	x, v.stack = v.stack[len(v.stack)-1], v.stack[:len(v.stack)-1]
	return x
}
