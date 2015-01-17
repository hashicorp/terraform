package command

import (
	"testing"

	"github.com/mitchellh/cli"
)

func TestValidate(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &ValidateCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{
		testFixturePath("validate"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestValidate_tooManyArgs(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &ValidateCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{
		testFixturePath("validate"),
		"too", "many", "things",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestValidate_badConfig(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &ValidateCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{
		testFixturePath("validate-error"),
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}
