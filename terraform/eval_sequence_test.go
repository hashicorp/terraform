package terraform

import (
	"testing"
)

func TestEvalSequence_impl(t *testing.T) {
	var _ EvalNodeFilterable = new(EvalSequence)
}
