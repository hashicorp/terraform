package ast

import (
	"testing"
)

func TestLiteralNodeType(t *testing.T) {
	c := &LiteralNode{Typex: TypeString}
	actual, err := c.Type(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != TypeString {
		t.Fatalf("bad: %s", actual)
	}
}

func TestLiteralNodeEval(t *testing.T) {
	c := &LiteralNode{Value: "42", Typex: TypeString}
	scope := &BasicScope{}

	actual, err := c.Eval(&EvalContext{Scope: scope})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != "42" {
		t.Fatalf("bad: %s", actual)
	}
}
