package ast

import (
	"testing"
)

func TestBasicScope_impl(t *testing.T) {
	var _ Scope = new(BasicScope)
}

func TestBasicScopeLookupFunc(t *testing.T) {
	scope := &BasicScope{
		FuncMap: map[string]Function{
			"foo": Function{},
		},
	}

	if _, ok := scope.LookupFunc("bar"); ok {
		t.Fatal("should not find bar")
	}
	if _, ok := scope.LookupFunc("foo"); !ok {
		t.Fatal("should find foo")
	}
}

func TestBasicScopeLookupVar(t *testing.T) {
	scope := &BasicScope{
		VarMap: map[string]Variable{
			"foo": Variable{},
		},
	}

	if _, ok := scope.LookupVar("bar"); ok {
		t.Fatal("should not find bar")
	}
	if _, ok := scope.LookupVar("foo"); !ok {
		t.Fatal("should find foo")
	}
}
