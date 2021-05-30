package command

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/mitchellh/cli"
)

func TestStatePush_empty(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("state-push-good"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "replace.tfstate")

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StatePushCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
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

func TestStatePush_lockedState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("state-push-good"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StatePushCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	unlock, err := testLockState(testDataDir, "local-state.tfstate")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	args := []string{"replace.tfstate"}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d", code)
	}
	if !strings.Contains(ui.ErrorWriter.String(), "Error acquiring the state lock") {
		t.Fatalf("expected a lock error, got: %s", ui.ErrorWriter.String())
	}
}

func TestStatePush_replaceMatch(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("state-push-replace-match"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "replace.tfstate")

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StatePushCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
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
	testCopyDir(t, testFixturePath("state-push-replace-match"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "replace.tfstate")

	// Set up the replacement to come from stdin
	var buf bytes.Buffer
	if err := writeStateForTesting(expected, &buf); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer testStdinPipe(t, &buf)()

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StatePushCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-force", "-"}
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
	testCopyDir(t, testFixturePath("state-push-bad-lineage"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "local-state.tfstate")

	p := testProvider()
	ui := cli.NewMockUi()
	view, _ := testView(t)
	c := &StatePushCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
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
	testCopyDir(t, testFixturePath("state-push-serial-newer"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "local-state.tfstate")

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StatePushCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"replace.tfstate"}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d", code)
	}

	actual := testStateRead(t, "local-state.tfstate")
	if !actual.Equal(expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestStatePush_serialOlder(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("state-push-serial-older"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	expected := testStateRead(t, "replace.tfstate")

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &StatePushCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
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
	td := tempDir(t)
	testCopyDir(t, testFixturePath("inmem-backend"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()
	defer inmem.Reset()

	s := states.NewState()
	statePath := testStateFile(t, s)

	// init the backend
	ui := new(cli.MockUi)
	view, _ := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{Ui: ui, View: view},
	}
	if code := initCmd.Run([]string{}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// create a new workspace
	ui = new(cli.MockUi)
	newCmd := &WorkspaceNewCommand{
		Meta: Meta{Ui: ui, View: view},
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
		Meta: Meta{Ui: ui, View: view},
	}

	args := []string{"-force", statePath}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}
