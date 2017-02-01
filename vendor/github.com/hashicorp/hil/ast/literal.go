package ast

import (
	"fmt"
	"reflect"
)

// LiteralNode represents a single literal value, such as "foo" or
// 42 or 3.14159. Based on the Type, the Value can be safely cast.
type LiteralNode struct {
	Value interface{}
	Typex Type
	Posx  Pos
}

// NewLiteralNode returns a new literal node representing the given
// literal Go value, which must correspond to one of the primitive types
// supported by HIL. Lists and maps cannot currently be constructed via
// this function.
//
// If an inappropriately-typed value is provided, this function will
// return an error. The main intended use of this function is to produce
// "synthetic" literals from constants in code, where the value type is
// well known at compile time. To easily store these in global variables,
// see also MustNewLiteralNode.
func NewLiteralNode(value interface{}, pos Pos) (*LiteralNode, error) {
	goType := reflect.TypeOf(value)
	var hilType Type

	switch goType.Kind() {
	case reflect.Bool:
		hilType = TypeBool
	case reflect.Int:
		hilType = TypeInt
	case reflect.Float64:
		hilType = TypeFloat
	case reflect.String:
		hilType = TypeString
	default:
		return nil, fmt.Errorf("unsupported literal node type: %T", value)
	}

	return &LiteralNode{
		Value: value,
		Typex: hilType,
		Posx:  pos,
	}, nil
}

// MustNewLiteralNode wraps NewLiteralNode and panics if an error is
// returned, thus allowing valid literal nodes to be easily assigned to
// global variables.
func MustNewLiteralNode(value interface{}, pos Pos) *LiteralNode {
	node, err := NewLiteralNode(value, pos)
	if err != nil {
		panic(err)
	}
	return node
}

func (n *LiteralNode) Accept(v Visitor) Node {
	return v(n)
}

func (n *LiteralNode) Pos() Pos {
	return n.Posx
}

func (n *LiteralNode) GoString() string {
	return fmt.Sprintf("*%#v", *n)
}

func (n *LiteralNode) String() string {
	return fmt.Sprintf("Literal(%s, %v)", n.Typex, n.Value)
}

func (n *LiteralNode) Type(Scope) (Type, error) {
	return n.Typex, nil
}
