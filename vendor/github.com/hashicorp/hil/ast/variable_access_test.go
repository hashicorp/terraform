package ast

import (
	"testing"
)

func TestVariableAccessType(t *testing.T) {
	c := &VariableAccess{Name: "foo"}
	scope := &BasicScope{
		VarMap: map[string]Variable{
			"foo": Variable{Type: TypeString},
		},
	}

	actual, err := c.Type(scope)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != TypeString {
		t.Fatalf("bad: %s", actual)
	}
}

func TestVariableAccessType_invalid(t *testing.T) {
	c := &VariableAccess{Name: "bar"}
	scope := &BasicScope{
		VarMap: map[string]Variable{
			"foo": Variable{Type: TypeString},
		},
	}

	_, err := c.Type(scope)
	if err == nil {
		t.Fatal("should error")
	}
}
