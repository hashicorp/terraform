package terraform

import (
	"testing"
)

func TestMockEvalContext_impl(t *testing.T) {
	var _ EvalContext = new(MockEvalContext)
}

func TestEval(t *testing.T) {
	var result int
	n := &testEvalAdd{
		Items:  []int{10, 32},
		Result: &result,
	}

	if _, err := Eval(n, nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	if result != 42 {
		t.Fatalf("bad: %#v", result)
	}
}

type testEvalAdd struct {
	Items  []int
	Result *int
}

func (n *testEvalAdd) Eval(ctx EvalContext) (interface{}, error) {
	result := 0
	for _, item := range n.Items {
		result += item
	}

	*n.Result = result
	return nil, nil
}
