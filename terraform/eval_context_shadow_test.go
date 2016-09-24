package terraform

import (
	"testing"
)

func TestShadowEvalContext_impl(t *testing.T) {
	var _ EvalContext = new(shadowEvalContextReal)
	var _ EvalContext = new(shadowEvalContextShadow)
}
