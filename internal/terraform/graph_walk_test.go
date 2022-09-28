package terraform

import (
	"testing"
)

func TestNullGraphWalker_impl(t *testing.T) {
	var _ GraphWalker = NullGraphWalker{}
}
