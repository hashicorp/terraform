package command

import (
	"testing"

	"github.com/mitchellh/cli"
)

func TestColorizeUi_impl(t *testing.T) {
	var _ cli.Ui = new(ColorizeUi)
}
