package command

import (
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestStatePull(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create some legacy remote state
	legacyState := testState()
	_, srv := testRemoteState(t, legacyState, 200)
	defer srv.Close()
	testStateFileRemote(t, legacyState)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StatePullCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	expected := "test_instance.foo"
	actual := ui.OutputWriter.String()
	if !strings.Contains(actual, expected) {
		t.Fatalf("expected:\n%s\n\nto include: %q", actual, expected)
	}
}

func TestStatePull_noState(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StatePullCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := ui.OutputWriter.String()
	if actual != "" {
		t.Fatalf("bad: %s", actual)
	}
}
