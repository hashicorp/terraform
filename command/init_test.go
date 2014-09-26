package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchellh/cli"
)

func TestInit(t *testing.T) {
	dir := tempDir(t)

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{
		testFixturePath("init"),
		dir,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	if _, err := os.Stat(filepath.Join(dir, "hello.tf")); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestInit_multipleArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{
		"bad",
		"bad",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestInit_noArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}
