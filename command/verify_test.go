package command

import (
	"github.com/mitchellh/cli"
	"testing"
)

func TestVerifyCommand(t *testing.T) {
	ui := new(cli.MockUi)
	c := &VerifyCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		testFixturePath("verify-valid"),
	}

	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestVerifyFailingCommand(t *testing.T) {
	ui := new(cli.MockUi)
	c := &VerifyCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		testFixturePath("verify-invalid"),
	}

	if code := c.Run(args); code == 0 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}
