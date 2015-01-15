package ast

import (
	"testing"
)

func TestConcatType(t *testing.T) {
	c := &Concat{}
	actual, err := c.Type(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != TypeString {
		t.Fatalf("bad: %s", actual)
	}
}

func TestConcatEval(t *testing.T) {
	c := &Concat{
		Exprs: []Node{
			&LiteralNode{Value: "foo"},
			&LiteralNode{Value: "bar"},
		},
	}
	scope := &BasicScope{}

	actual, err := c.Eval(&EvalContext{Scope: scope})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != "foobar" {
		t.Fatalf("bad: %s", actual)
	}
}
