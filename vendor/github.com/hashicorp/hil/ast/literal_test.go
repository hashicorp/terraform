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
