package command

import (
	"bytes"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestStatePush_empty(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("state-push-good"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "replace.tfstate")

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StatePushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{"replace.tfstate"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testStateRead(t, "local-state.tfstate")
	if !actual.Equal(expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestStatePush_replaceMatch(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("state-push-replace-match"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "replace.tfstate")

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StatePushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{"replace.tfstate"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testStateRead(t, "local-state.tfstate")
	if !actual.Equal(expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestStatePush_replaceMatchStdin(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("state-push-replace-match"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "replace.tfstate")

	// Setup the replacement to come from stdin
	var buf bytes.Buffer
	if err := terraform.WriteState(expected, &buf); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer testStdinPipe(t, &buf)()

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StatePushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{"-"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testStateRead(t, "local-state.tfstate")
	if !actual.Equal(expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestStatePush_lineageMismatch(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("state-push-bad-lineage"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "local-state.tfstate")

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StatePushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{"replace.tfstate"}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testStateRead(t, "local-state.tfstate")
	if !actual.Equal(expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestStatePush_serialNewer(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("state-push-serial-newer"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "local-state.tfstate")

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StatePushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{"replace.tfstate"}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testStateRead(t, "local-state.tfstate")
	if !actual.Equal(expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestStatePush_serialOlder(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("state-push-serial-older"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "replace.tfstate")

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StatePushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{"replace.tfstate"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testStateRead(t, "local-state.tfstate")
	if !actual.Equal(expected) {
		t.Fatalf("bad: %#v", actual)
	}
}
