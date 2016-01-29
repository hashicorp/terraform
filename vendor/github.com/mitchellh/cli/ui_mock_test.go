package cli

import (
	"testing"
)

func TestMockUi_implements(t *testing.T) {
	var _ Ui = new(MockUi)
}
