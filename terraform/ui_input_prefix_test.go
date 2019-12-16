package terraform

import (
	"testing"
)

func TestPrefixUIInput_impl(t *testing.T) {
	var _ UIInput = new(PrefixUIInput)
}
