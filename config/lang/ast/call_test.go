package ast

import (
	"testing"
)

func TestCallType(t *testing.T) {
	c := &Call{Func: "foo"}
	scope := &BasicScope{
		FuncMap: map[string]Function{
			"foo": Function{ReturnType: TypeString},
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

func TestCallType_invalid(t *testing.T) {
	c := &Call{Func: "bar"}
	scope := &BasicScope{
		FuncMap: map[string]Function{
			"foo": Function{ReturnType: TypeString},
		},
	}

	_, err := c.Type(scope)
	if err == nil {
		t.Fatal("should error")
	}
}

func TestCallEval(t *testing.T) {
	c := &Call{Func: "foo"}
	scope := &BasicScope{
		FuncMap: map[string]Function{
			"foo": Function{
				Callback: func([]interface{}) (interface{}, error) {
					return "42", nil
				},
			},
		},
	}

	actual, err := c.Eval(&EvalContext{Scope: scope})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != "42" {
		t.Fatalf("bad: %s", actual)
	}
}
