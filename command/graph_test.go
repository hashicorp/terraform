package command

import (
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestGraph(t *testing.T) {
	ui := new(cli.MockUi)
	c := &GraphCommand{
		ContextOpts: testCtxConfig(testProvider()),
		Ui:          ui,
	}

	args := []string{
		testFixturePath("graph"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "digraph {") {
		t.Fatalf("doesn't look like digraph: %s", output)
	}
}

func TestGraph_noArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		ContextOpts: testCtxConfig(testProvider()),
		Ui:          ui,
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestGraph_multipleArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		ContextOpts: testCtxConfig(testProvider()),
		Ui:          ui,
	}

	args := []string{
		"bad",
		"bad",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}
