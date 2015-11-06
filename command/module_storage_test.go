package command

import (
	"testing"

	"github.com/hashicorp/go-getter"
)

func TestUiModuleStorage_impl(t *testing.T) {
	var _ getter.Storage = new(uiModuleStorage)
}
