package command

import (
	"bytes"
	"os"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/states"
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
	if err := writeStateForTesting(expected, &buf); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer testStdinPipe(t, &buf)()

	p := testProvider()
	ui := new(cli.MockUi)
	c := &StatePushCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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

func TestStatePush_forceRemoteState(t *testing.T) {
	t.Fatalf("FIXME: This test seems to be getting hanged or into an infinite loop")
	td := tempDir(t)
	copy.CopyDir(testFixturePath("inmem-backend"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()
	defer inmem.Reset()

	s := states.NewState()
	statePath := testStateFile(t, s)

	// init the backend
	ui := new(cli.MockUi)
	initCmd := &InitCommand{
		Meta: Meta{Ui: ui},
	}
	if code := initCmd.Run([]string{}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// create a new workspace
	ui = new(cli.MockUi)
	newCmd := &WorkspaceNewCommand{
		Meta: Meta{Ui: ui},
	}
	if code := newCmd.Run([]string{"test"}); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
	}

	// put a dummy state in place, so we have something to force
	b := backend.TestBackendConfig(t, inmem.New(), nil)
	sMgr, err := b.StateMgr("test")
	if err != nil {
		t.Fatal(err)
	}
	if err := sMgr.WriteState(states.NewState()); err != nil {
		t.Fatal(err)
	}
	if err := sMgr.PersistState(); err != nil {
		t.Fatal(err)
	}

	// push our local state to that new workspace
	ui = new(cli.MockUi)
	c := &StatePushCommand{
		Meta: Meta{Ui: ui},
	}

	args := []string{"-force", statePath}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}
