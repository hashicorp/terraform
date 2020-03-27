package command

import (
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestZeroTwelveUpgrade_deprecated(t *testing.T) {
	ui := new(cli.MockUi)
	c := &ZeroTwelveUpgradeCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	if code := c.Run([]string{}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "The 0.12upgrade command is deprecated.") {
		t.Fatal("unexpected output:", output)
	}
}
