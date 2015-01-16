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
