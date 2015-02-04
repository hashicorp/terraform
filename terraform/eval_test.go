package terraform

import (
	"testing"
)

func TestMockEvalContext_impl(t *testing.T) {
	var _ EvalContext = new(MockEvalContext)
}
