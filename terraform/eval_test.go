package terraform

import (
	"testing"
)

func TestMockEvalContext_impl(t *testing.T) {
	var _ EvalContext = new(MockEvalContext)
}

func TestEval(t *testing.T) {
	n := &testEvalAdd{
		Items: []EvalNode{
			&EvalLiteral{Value: 10},
			&EvalLiteral{Value: 32},
		},
	}

	result, err := Eval(n, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if result != 42 {
		t.Fatalf("bad: %#v", result)
	}
}

type testEvalAdd struct {
	Items []EvalNode
}

func (n *testEvalAdd) Args() ([]EvalNode, []EvalType) {
	types := make([]EvalType, len(n.Items))
	for i, _ := range n.Items {
		types[i] = EvalTypeInvalid
	}

	return n.Items, types
}

func (n *testEvalAdd) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	result := 0
	for _, arg := range args {
		result += arg.(int)
	}

	return result, nil
}

func (n *testEvalAdd) Type() EvalType {
	return EvalTypeInvalid
}
