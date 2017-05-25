package cli

import (
	"testing"
)

func TestConcurrentUi_impl(t *testing.T) {
	var _ Ui = new(ConcurrentUi)
}
