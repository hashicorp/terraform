package command

import (
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestGet(t *testing.T) {
	wd := tempWorkingDirFixture(t, "get")
	defer testChdir(t, wd.RootModuleDir())()

	ui := cli.NewMockUi()
	c := &GetCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			WorkingDir:       wd,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "- foo in") {
		t.Fatalf("doesn't look like get: %s", output)
	}
}

func TestGet_multipleArgs(t *testing.T) {
	wd, cleanup := tempWorkingDir(t)
	defer cleanup()
	defer testChdir(t, wd.RootModuleDir())()

	ui := cli.NewMockUi()
	c := &GetCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			WorkingDir:       wd,
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

func TestGet_update(t *testing.T) {
	wd := tempWorkingDirFixture(t, "get")
	defer testChdir(t, wd.RootModuleDir())()

	ui := cli.NewMockUi()
	c := &GetCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			WorkingDir:       wd,
		},
	}

	args := []string{
		"-update",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, `- foo in`) {
		t.Fatalf("doesn't look like get: %s", output)
	}
}
