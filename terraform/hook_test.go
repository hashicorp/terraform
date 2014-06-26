package terraform

import (
	"testing"
)

func TestNilHook_impl(t *testing.T) {
	var _ Hook = new(NilHook)
}
