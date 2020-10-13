package command

import (
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestZeroThirteenUpgrade(t *testing.T) {
	ui := new(cli.MockUi)
	c := &ZeroThirteenUpgradeCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	if code := c.Run([]string{}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "The 0.13upgrade command has been removed.") {
		t.Fatal("unexpected output:", output)
	}
}
