package command

import (
	"os"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestGet(t *testing.T) {
	ui := new(cli.MockUi)
	c := &GetCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
			dataDir:     tempDir(t),
		},
	}

	args := []string{
		testFixturePath("get"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "Get: file://") {
		t.Fatalf("doesn't look like get: %s", output)
	}
	if strings.Contains(output, "(update)") {
		t.Fatalf("doesn't look like get: %s", output)
	}
}

func TestGet_multipleArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &GetCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
			dataDir:     tempDir(t),
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

func TestGet_noArgs(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("get")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	ui := new(cli.MockUi)
	c := &GetCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
			dataDir:     tempDir(t),
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "Get: file://") {
		t.Fatalf("doesn't look like get: %s", output)
	}
	if strings.Contains(output, "(update)") {
		t.Fatalf("doesn't look like get: %s", output)
	}
}

func TestGet_update(t *testing.T) {
	ui := new(cli.MockUi)
	c := &GetCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
			dataDir:     tempDir(t),
		},
	}

	args := []string{
		"-update",
		testFixturePath("get"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "Get: file://") {
		t.Fatalf("doesn't look like get: %s", output)
	}
	if !strings.Contains(output, "(update)") {
		t.Fatalf("doesn't look like get: %s", output)
	}
}
