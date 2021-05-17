package command

import (
	"os"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestGet(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("get"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &GetCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			dataDir:          tempDir(t),
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
	ui := new(cli.MockUi)
	c := &GetCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			dataDir:          tempDir(t),
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
	td := tempDir(t)
	testCopyDir(t, testFixturePath("get"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &GetCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			dataDir:          tempDir(t),
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
