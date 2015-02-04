package terraform

import (
	"testing"
)

func TestEvalLiteral_impl(t *testing.T) {
	var _ EvalNode = new(EvalLiteral)
}

func TestEvalLiteralEval(t *testing.T) {
	n := &EvalLiteral{Value: 42}
	actual, err := n.Eval(nil, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != 42 {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestEvalLiteralType(t *testing.T) {
	n := &EvalLiteral{Value: 42, ValueType: EvalTypeConfig}
	if actual := n.Type(); actual != EvalTypeConfig {
		t.Fatalf("bad: %#v", actual)
	}
}
